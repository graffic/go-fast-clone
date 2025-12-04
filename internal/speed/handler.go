// Package speed handles the speed test endpoints for download, upload, and latency.
package speed

import (
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	ReadBufferSize  uint64 = 30 * 1024 * 1024
	MaxPayloadBytes uint64 = 25 * 1024 * 1024
)

var (
	randomBuffer = make([]byte, ReadBufferSize)
)

// Init initializes the random buffer used for speed tests.
func Init() {
	// Use a seeded random generator to fill the buffer
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Read(randomBuffer)
}

// HandleDownload handles GET /speedtest/ for download speed tests without ranges
func HandleDownload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatUint(MaxPayloadBytes, 10))

	_, err := w.Write(randomBuffer[:MaxPayloadBytes])
	if err != nil {
		log.Error().Err(err).Msg("Failed to write download response")
	}
}

// HandleDownloadRange handles GET /speedtest/range/0-{N} for ranged download.
func HandleDownloadRange(w http.ResponseWriter, r *http.Request) {
	rangePath := r.PathValue("range")

	// Expected format: "0-{N}"
	parts := strings.Split(rangePath, "-")
	if len(parts) != 2 || parts[0] != "0" {
		http.Error(w, "Invalid range format. Expected 0-{N}", http.StatusBadRequest)
		return
	}

	endByte, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		http.Error(w, "Invalid range size", http.StatusBadRequest)
		return
	}

	// Size to return is N+1 bytes (from 0 to N inclusive)
	totalSize := endByte + 1

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatUint(totalSize, 10))

	// Allow CORS (already handled by middleware, but specific headers might be needed if not covered)
	// The middleware handles Access-Control-Allow-Origin: *

	remaining := totalSize

	for remaining > 0 {
		// Determine how much to write: min(ReadBufferSize, remaining, what's left in buffer)
		toWrite := ReadBufferSize
		if remaining < toWrite {
			toWrite = remaining
		}

		n, err := w.Write(randomBuffer[:toWrite])
		if err != nil {
			log.Error().Err(err).Msg("Failed to write download range response")
			return
		}
		remaining -= uint64(n)
	}
}

// HandleUpload handles POST /speedtest/ for upload without ranges
func HandleUpload(w http.ResponseWriter, r *http.Request) {
	_, _ = io.Copy(io.Discard, r.Body)
	r.Body.Close()
	w.WriteHeader(http.StatusOK)
}

// HandleUploadRange handles POST /speedtest/range/0-{N} for ranged upload and latency.
func HandleUploadRange(w http.ResponseWriter, r *http.Request) {
	rangePath := r.PathValue("range")

	if rangePath != "0-0" {
		_, _ = io.Copy(io.Discard, r.Body)
		r.Body.Close()
	}

	w.WriteHeader(http.StatusOK)
}
