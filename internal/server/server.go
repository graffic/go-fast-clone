// Package server provides HTTP server setup, routing, and middleware.
package server

import (
	"net/http"
	"net/http/pprof"

	"github.com/rs/zerolog/log"

	"fastclone/internal/config"
	"fastclone/internal/oca"
	"fastclone/internal/speed"
	"fastclone/internal/telemetry"
)

// Server holds the HTTP server and its dependencies.
type Server struct {
	cfg    *config.Config
	router *http.ServeMux
}

// New creates a new Server with all routes configured.
func New(cfg *config.Config) *Server {
	s := &Server{
		cfg:    cfg,
		router: http.NewServeMux(),
	}
	s.setupRoutes()
	return s
}

// setupRoutes configures all HTTP routes.
func (s *Server) setupRoutes() {
	// Static files - serve original-webapp from root
	staticFS := http.FileServer(http.Dir(s.cfg.StaticDir))

	// OCA Directory endpoint (most specific path first)
	s.router.HandleFunc("GET /netflix/speedtest/v2", oca.HandleDirectory)

	// Speed test endpoints - use exact path matching to avoid conflicts
	// The {range...} wildcard captures everything after /speedtest/range/
	s.router.HandleFunc("GET /speedtest/range/{range}", speed.HandleDownloadRange)
	s.router.HandleFunc("POST /speedtest/range/{range}", speed.HandleUploadRange)
	s.router.HandleFunc("GET /speedtest", speed.HandleDownload)
	s.router.HandleFunc("POST /speedtest", speed.HandleUpload)

	// Telemetry endpoint
	s.router.HandleFunc("POST /telemetry/cl2", telemetry.HandleIngest)

	// Static files (catch-all for webapp assets) - must be last
	s.router.Handle("/", staticFS)

	if s.cfg.EnablePprof {
		log.Info().Msg("Pprof enabled")
		s.router.HandleFunc("/debug/pprof/", pprof.Index)
		s.router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		s.router.HandleFunc("/debug/pprof/profile", pprof.Profile)
		s.router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		s.router.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}
}

// Handler returns the HTTP handler with all middleware applied.
func (s *Server) Handler() http.Handler {
	if s.cfg.HttpLogging {
		log.Info().Msg("HTTP logging enabled")
		return LoggingMiddleware(s.router)
	}
	return s.router
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe() error {
	log.Info().
		Str("listen_addr", s.cfg.ListenAddr).Str("static_dir", s.cfg.StaticDir).
		Msg("Starting server")

	return http.ListenAndServe(s.cfg.ListenAddr, s.Handler())
}
