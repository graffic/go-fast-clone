// Package server provides HTTP server setup, routing, and middleware.
package server

import (
	"net/http"

	"github.com/felixge/httpsnoop"
	"github.com/rs/zerolog/log"
)

// LoggingMiddleware logs the request and response.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m := httpsnoop.CaptureMetrics(next, w, r)
		log.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("code", m.Code).
			Int("received_bytes", int(r.ContentLength)).
			Int64("sent_bytes", m.Written).
			Dur("duration_ms", m.Duration).
			Msg("Request processed")
	})
}
