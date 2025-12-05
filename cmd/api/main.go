// Fast.com Clone Backend
//
// This is the main entry point for the speed test backend server.
// It serves the original webapp static files and provides API endpoints
// for speed testing.
//
// Usage:
//
//	LISTEN_ADDR=:8080 STATIC_DIR=original-webapp go run ./cmd/api
//
// Environment Variables:
//   - LISTEN_ADDR: Address to listen on (default: ":8080")
//   - STATIC_DIR: Path to static files directory (default: "original-webapp")
package main

import (
	"os"

	"fastclone/internal/config"
	"fastclone/internal/server"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	cfg := config.Load()

	setupLogging()

	srv := server.New(cfg)

	log.Fatal().Err(srv.ListenAndServe()).Msg("Server error")
}

func setupLogging() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
}
