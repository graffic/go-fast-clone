// Package config handles configuration loading from environment variables.
package config

import (
	"os"

	"github.com/rs/zerolog/log"
)

// Config holds all application configuration.
type Config struct {
	// ListenAddr is the address:port the server listens on.
	ListenAddr string
	// StaticDir is the path to the static files directory (original-webapp).
	StaticDir string
	// HttpLogging flag to enable logging of http requests
	HttpLogging bool
	// EnablePprof flag to add debug/pprof endpoints for profiling
	EnablePprof bool
	// RandomDataFile path to the file in a tmpfs filesystem to use as source for speed tests
	RandomDataFile string
}

func (c *Config) Log() {
	log.Info().
		Str("listen_addr", c.ListenAddr).
		Str("static_dir", c.StaticDir).
		Bool("http_logging", c.HttpLogging).
		Bool("enable_pprof", c.EnablePprof).
		Str("random_data_file", c.RandomDataFile).
		Msg("Configuration loaded")
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	config := &Config{
		ListenAddr:     getEnv("LISTEN_ADDR", ":8080"),
		StaticDir:      getEnv("STATIC_DIR", "original-webapp"),
		HttpLogging:    getEnv("HTTP_LOGGING", "false") == "true",
		EnablePprof:    getEnv("ENABLE_PPROF", "false") == "true",
		RandomDataFile: getEnv("RANDOM_DATA_FILE", "/dev/shm/random_data"),
	}

	config.Log()

	return config
}

// getEnv retrieves an environment variable or returns a default value.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
