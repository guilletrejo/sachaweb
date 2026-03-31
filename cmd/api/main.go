package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/guilletrejo/sachaweb/internal/config"
	"github.com/guilletrejo/sachaweb/internal/handler"
	"github.com/guilletrejo/sachaweb/internal/repository"
	"github.com/guilletrejo/sachaweb/internal/server/middleware"
	"github.com/guilletrejo/sachaweb/internal/service"
	"github.com/guilletrejo/sachaweb/migrations"
)

func main() {
	// Step 1: Load configuration from environment variables.
	cfg := config.Load()

	// Step 2: Connect to PostgreSQL.
	//
	// WHAT IS context.Background()?
	// It's the "root" context — it never expires and is never cancelled.
	// Use it for startup/shutdown operations that shouldn't be tied to
	// any specific HTTP request. Once the server is running, each request
	// gets its own context via r.Context().
	//
	// WHY A 10-SECOND TIMEOUT?
	// If PostgreSQL is down or unreachable, we don't want to hang forever.
	// context.WithTimeout creates a context that automatically cancels
	// after the given duration. If the connection isn't established in
	// 10 seconds, it fails fast instead of waiting indefinitely.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := connectDB(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	// defer pool.Close() ensures the connection pool is cleaned up
	// when main() exits. This releases all database connections.
	defer pool.Close()

	// Step 3: Run database migrations.
	// This creates the products table if it doesn't exist.
	// On subsequent startups, it's a no-op (IF NOT EXISTS).
	if err := runMigrations(ctx, pool); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Step 4: Create the REPOSITORY layer.
	productRepo := repository.NewPostgresProductRepo(pool)
	userRepo := repository.NewPostgresUserRepo(pool)

	// Step 5: Create the SERVICE layer.
	productService := service.NewProductService(productRepo)
	userService := service.NewUserService(userRepo, cfg.JWTSecret)

	// Step 6: Create the HANDLER layer.
	productHandler := handler.NewProductHandler(productService)
	userHandler := handler.NewUserHandler(userService)

	// Step 7: Create the auth middleware.
	//
	// THE MIDDLEWARE PATTERN: func(http.Handler) http.Handler
	// auth is a function that takes a handler and returns a new handler
	// that checks the JWT token first. If the token is valid, it calls
	// the original handler. If not, it returns 401 and stops.
	//
	// Usage: mux.Handle("POST /products", auth(productHandler.HandleCreate()))
	// This means: "when POST /products arrives, first run auth, then HandleCreate"
	auth := middleware.Auth(userService)

	// Step 8: Register routes.
	mux := http.NewServeMux()

	// PUBLIC routes — anyone can access these.
	mux.HandleFunc("GET /health", handler.HandleHealth())
	mux.HandleFunc("GET /products", productHandler.HandleList())
	mux.HandleFunc("GET /products/{id}", productHandler.HandleGet())
	mux.HandleFunc("POST /register", userHandler.HandleRegister())
	mux.HandleFunc("POST /login", userHandler.HandleLogin())

	// PROTECTED routes — require a valid JWT token.
	// Notice: Handle (not HandleFunc) because auth() returns http.Handler.
	// The auth middleware wraps the handler: auth checks token → handler runs.
	//
	// WHO CAN DO WHAT:
	//   Anyone can browse products (GET)
	//   Only logged-in users can create/update/delete products
	//   This is how real e-commerce works — customers browse, sellers manage
	mux.Handle("POST /products", auth(productHandler.HandleCreate()))
	mux.Handle("PUT /products/{id}", auth(productHandler.HandleUpdate()))
	mux.Handle("DELETE /products/{id}", auth(productHandler.HandleDelete()))

	// Step 9: Start server.
	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Starting server on %s (PostgreSQL connected)", addr)
	log.Printf("Public endpoints:")
	log.Printf("  GET    /health")
	log.Printf("  GET    /products")
	log.Printf("  GET    /products/{id}")
	log.Printf("  POST   /register")
	log.Printf("  POST   /login")
	log.Printf("Protected endpoints (require JWT):")
	log.Printf("  POST   /products")
	log.Printf("  PUT    /products/{id}")
	log.Printf("  DELETE /products/{id}")

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

// connectDB creates a connection pool to PostgreSQL.
//
// WHAT IS pgxpool.ParseConfig?
// It parses the DATABASE_URL into a configuration struct, which you can
// then customize before creating the pool. This is where you'd set:
// - MaxConns: maximum number of connections (default: 4 per CPU core)
// - MinConns: minimum idle connections to keep warm
// - MaxConnLifetime: how long a connection lives before being recycled
//
// For now, we just set MaxConns to 10 (plenty for local development).
// In production, you'd tune this based on your traffic and DB capacity.
func connectDB(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing database URL: %w", err)
	}

	// MaxConns limits how many simultaneous database connections the pool
	// maintains. Each connection uses memory on both the app and DB side.
	// 10 is fine for local dev. Production might use 20-50 depending on
	// the database server's capacity.
	poolConfig.MaxConns = 10

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("creating connection pool: %w", err)
	}

	// Ping verifies the connection actually works (not just that parsing succeeded).
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	log.Printf("Connected to PostgreSQL (max connections: %d)", poolConfig.MaxConns)
	return pool, nil
}

// runMigrations reads embedded SQL files and executes them against the database.
//
// This is a simple migration runner: it reads each .up.sql file in order
// and executes it. The SQL uses IF NOT EXISTS / IF NOT EXISTS, so running
// it multiple times is safe (idempotent).
//
// In larger projects, you'd use a migration library like golang-migrate that
// tracks which migrations have been applied. For our few migrations, this
// simple approach works perfectly.
func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	// List of migration files in order. Each new phase adds to this list.
	migrationFiles := []string{
		"000001_create_products_table.up.sql",
		"000002_create_users_table.up.sql",
	}

	for _, file := range migrationFiles {
		sql, err := migrations.FS.ReadFile(file)
		if err != nil {
			return fmt.Errorf("reading migration %s: %w", file, err)
		}
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("executing migration %s: %w", file, err)
		}
	}

	log.Println("Database migrations applied successfully")
	return nil
}
