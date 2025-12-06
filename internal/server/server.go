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
	router     *http.ServeMux
	listenAddr string
}

// New creates a new Server with all routes configured.
func New(cfg *config.Config) *Server {

	router := http.NewServeMux()
	// Static files - serve original-webapp from root
	staticFS := http.FileServer(http.Dir(cfg.StaticDir))

	// OCA Directory endpoint (most specific path first)
	router.HandleFunc("GET /netflix/speedtest/v2", oca.HandleDirectory)

	// Speed test endpoints - use exact path matching to avoid conflicts
	// The {range...} wildcard captures everything after /speedtest/range/
	speedHandler, err := speed.NewSpeedTestHandler()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create speed test handler")
	}
	router.HandleFunc("GET /speedtest/range/{range}", speedHandler.HandleDownloadRange)
	router.HandleFunc("POST /speedtest/range/{range}", speedHandler.HandleUploadRange)
	router.HandleFunc("GET /speedtest", speedHandler.HandleDownload)
	router.HandleFunc("POST /speedtest", speedHandler.HandleUpload)

	// Telemetry endpoint
	router.HandleFunc("POST /telemetry/cl2", telemetry.HandleIngest)

	// Static files (catch-all for webapp assets) - must be last
	router.Handle("/", staticFS)

	if cfg.EnablePprof {
		log.Info().Msg("Pprof enabled")
		router.HandleFunc("/debug/pprof/", pprof.Index)
		router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		router.HandleFunc("/debug/pprof/profile", pprof.Profile)
		router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		router.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}

	if cfg.HttpLogging {
		log.Info().Msg("HTTP logging enabled")
		router = LoggingMiddleware(router).(*http.ServeMux)
	}

	return &Server{
		router:     router,
		listenAddr: cfg.ListenAddr,
	}
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe() error {
	log.Info().
		Str("listen_addr", s.listenAddr).
		Msg("Starting server")

	return http.ListenAndServe(s.listenAddr, s.router)
}
