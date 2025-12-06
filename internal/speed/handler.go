// Package speed handles the speed test endpoints for download, upload, and latency.
package speed

import (
	"io"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	ReadBufferSize  int64 = 30 * 1024 * 1024
	MaxPayloadBytes int64 = 25 * 1024 * 1024
)

type SpeedTestHandler struct {
	randomBufferFile string
}

func NewSpeedTestHandler(randomBufferFile string) (*SpeedTestHandler, error) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	heavyData := make([]byte, ReadBufferSize)
	r.Read(heavyData)
	err := os.WriteFile(randomBufferFile, heavyData, 0644)
	if err != nil {
		return nil, err
	}
	log.Info().
		Str("random_buffer_file", randomBufferFile).
		Int64("read_buffer_size", ReadBufferSize).
		Int64("max_payload_bytes", MaxPayloadBytes).
		Msg("Speed test handler initialized")

	return &SpeedTestHandler{
		randomBufferFile: randomBufferFile,
	}, nil
}

func (h *SpeedTestHandler) ServeContent(w http.ResponseWriter, r *http.Request, amount int64) {
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(amount, 10))

	f, _ := os.Open(h.randomBufferFile)
	defer f.Close()

	// This triggers sendfile() because io.CopyN creates a LimitedReader wrapping the OS file
	// which net/http detects and optimizes.
	io.CopyN(w, f, amount)
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
