// Package config handles configuration loading from environment variables.
package config

import (
	"os"
)

// Config holds all application configuration.
type Config struct {
	// ListenAddr is the address:port the server listens on.
	ListenAddr string

	// StaticDir is the path to the static files directory (original-webapp).
	StaticDir string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		ListenAddr: getEnv("LISTEN_ADDR", ":8080"),
		StaticDir:  getEnv("STATIC_DIR", "original-webapp"),
	}
}

// getEnv retrieves an environment variable or returns a default value.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
