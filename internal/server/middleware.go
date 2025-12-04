// Package server provides HTTP server setup, routing, and middleware.
package server

import (
	"net/http"

	"github.com/felixge/httpsnoop"
	"github.com/rs/zerolog/log"
)

// CORSMiddleware adds CORS headers to all responses.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, hasOriginHeader := r.Header["Origin"]

		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" && hasOriginHeader {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

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
