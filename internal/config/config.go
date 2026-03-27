// Package config handles application configuration.
// In production systems, configuration comes from environment variables,
// NOT hardcoded values. This lets you run the same binary in different
// environments (dev, staging, production) with different settings.
package config

import "os"

// Config holds all configuration for the application.
// Each field maps to an environment variable.
type Config struct {
	Port string // The port the HTTP server listens on (e.g., "8080")
}

// Load reads configuration from environment variables.
// If an environment variable is not set, it falls back to a default value.
// This is the simplest config pattern — later phases might use a library
// like "envconfig" for more complex needs, but this is how it starts.
func Load() Config {
	return Config{
		Port: getEnv("PORT", "8080"),
	}
}

// getEnv reads an environment variable or returns a default value.
// This is a tiny helper, but it makes Load() much cleaner to read.
func getEnv(key, fallback string) string {
	// os.Getenv returns "" if the variable is not set.
	// We treat "" as "not set" and use the fallback.
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
