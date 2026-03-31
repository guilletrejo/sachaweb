// Package config handles application configuration.
// In production systems, configuration comes from environment variables,
// NOT hardcoded values. This lets you run the same binary in different
// environments (dev, staging, production) with different settings.
package config

import "os"

// Config holds all configuration for the application.
// Each field maps to an environment variable.
type Config struct {
	Port        string // The port the HTTP server listens on (e.g., "8080")
	DatabaseURL string // PostgreSQL connection string
	JWTSecret   string // Secret key for signing JWT tokens
}

// Load reads configuration from environment variables.
//
// WHAT IS A DATABASE URL?
// It's a single string that contains everything needed to connect to a database:
//   postgres://user:password@host:port/dbname?sslmode=disable
//
// Breaking it down:
//   postgres://           → the protocol (like https:// for websites)
//   sachaweb:sachaweb     → username:password
//   @localhost:5432       → host:port (localhost because Docker maps to your machine)
//   /sachaweb             → database name
//   ?sslmode=disable      → connection options (no SSL for local dev)
//
// In production, this would be something like:
//   postgres://app_user:s3cur3p4ss@db.neon.tech:5432/sachaweb_prod?sslmode=require
//
// If the password contains special characters like @ : / #, they must be
// percent-encoded (e.g., p@ss → p%40ss). Cloud providers give you the
// pre-encoded URL, so you rarely do this manually.
//
// The DATABASE_URL environment variable is a standard convention used by
// Heroku, Fly.io, Render, Railway, and most cloud platforms.
func Load() Config {
	return Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://sachaweb:sachaweb@localhost:5432/sachaweb?sslmode=disable"),
		// JWT_SECRET should be a long, random string in production.
		// The default is fine for local development ONLY.
		// In production: JWT_SECRET=$(openssl rand -hex 32)
		JWTSecret: getEnv("JWT_SECRET", "sachaweb-dev-secret-change-in-production"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
