// Package speed handles the speed test endpoints for download, upload, and latency.
package speed

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/sys/unix"
)

const (
	ReadBufferSize  int64 = 30 * 1024 * 1024
	MaxPayloadBytes int64 = 25 * 1024 * 1024
)

type SpeedTestHandler struct {
	memFdPath string
	// memFile keeps the original file descriptor open
	memFile *os.File
}

func NewSpeedTestHandler() (*SpeedTestHandler, error) {
	name := "fast_clone_speed_test"
	fd, err := unix.MemfdCreate(name, unix.MFD_CLOEXEC)
	if err != nil {
		return nil, err
	}

	// Wrap the fd in an os.File to use convenient Write methods
	// We pass the fd directly. os.NewFile does not duplicate it, just wraps it.
	f := os.NewFile(uintptr(fd), name)

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	heavyData := make([]byte, ReadBufferSize)
	r.Read(heavyData)

	if _, err := f.Write(heavyData); err != nil {
		f.Close()
		return nil, err
	}

	// Path to access this FD via /proc filesystem.
	// Opening this path creates a new file description with its own offset.
	memFdPath := fmt.Sprintf("/proc/self/fd/%d", fd)

	log.Info().
		Str("memfd_path", memFdPath).
		Int64("read_buffer_size", ReadBufferSize).
		Int64("max_payload_bytes", MaxPayloadBytes).
		Msg("Speed test handler (memfd) initialized")

	return &SpeedTestHandler{
		memFdPath: memFdPath,
		memFile:   f,
	}, nil
}

func (h *SpeedTestHandler) ServeContent(w http.ResponseWriter, r *http.Request, amount int64) {
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(amount, 10))

	// Open a fresh file descriptor/description for this request
	f, err := os.Open(h.memFdPath)
	if err != nil {
		log.Error().Err(err).Msg("failed to open memfd path")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	// io.CopyN creates a LimitedReader.
	// net/http optimizes this by unwrapping it and using sendfile if the underlying reader is *os.File
	io.CopyN(w, f, amount)
	// Can errors happen here? Yes, they are usually on connection closed before finishing to write.
}

// HandleDownload handles GET /speedtest/ for download speed tests without ranges
func (h *SpeedTestHandler) HandleDownload(w http.ResponseWriter, r *http.Request) {
	h.ServeContent(w, r, MaxPayloadBytes)
}

// HandleDownloadRange handles GET /speedtest/range/0-{N} for ranged download.
func (h *SpeedTestHandler) HandleDownloadRange(w http.ResponseWriter, r *http.Request) {
	rangePath := r.PathValue("range")

	if rangePath == "0-0" {
		return
	}

	// Expected format: "0-{N}"
	parts := strings.Split(rangePath, "-")
	if len(parts) != 2 || parts[0] != "0" {
		http.Error(w, "Invalid range format. Expected 0-{N}", http.StatusBadRequest)
		return
	}

	endByte, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || endByte >= MaxPayloadBytes {
		http.Error(w, "Invalid range size", http.StatusBadRequest)
		return
	}

	// Size to return is N+1 bytes (from 0 to N inclusive)
	h.ServeContent(w, r, endByte+1)
}

// HandleUpload handles POST /speedtest/ for upload without ranges
func (h *SpeedTestHandler) HandleUpload(w http.ResponseWriter, r *http.Request) {
	_, _ = io.Copy(io.Discard, r.Body)
	r.Body.Close()
	w.WriteHeader(http.StatusOK)
}

// HandleUploadRange handles POST /speedtest/range/0-{N} for ranged upload and latency.
func (h *SpeedTestHandler) HandleUploadRange(w http.ResponseWriter, r *http.Request) {
	rangePath := r.PathValue("range")

	if rangePath != "0-0" {
		_, _ = io.Copy(io.Discard, r.Body)
		r.Body.Close()
	}

	w.WriteHeader(http.StatusOK)
}
