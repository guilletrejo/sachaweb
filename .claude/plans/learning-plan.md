# SachaWeb — Backend Development Learning Project

## Context

You want to learn backend development for a role at companies like MercadoLibre. You know Go basics (structs, slices, sorting, table-driven tests) but haven't built an HTTP server or worked with databases, auth, or deployment. This project teaches every major backend concept through building a **mini e-commerce platform** (simplified MercadoLibre) with a **learning dashboard** that shows what's happening in the backend in real-time.

**Quick answer to your question:** Backend is NOT just for websites. Backend systems power mobile apps, IoT devices, other services (microservices), CLI tools, batch jobs, event pipelines, and third-party APIs. MercadoLibre's backend serves their website, iOS/Android apps, seller tools, payment processing, shipping, fraud detection, and recommendation engines — all from Go/Java/Node services. The HTTP server is just the interface; the business logic behind it is what matters.

---

## Project: "SachaWeb" — 10 Phases

Each phase builds on the previous. Every line of backend code will be explained.

### Phase 1: Foundation — First HTTP Server
- `net/http`, `http.Handler` interface, `http.HandlerFunc`, Go 1.22 enhanced routing
- `GET /health` + `GET /products` (hardcoded)
- JSON responses, environment-based config
- **Files:** `cmd/api/main.go`, `internal/handler/health.go`, `internal/handler/product.go`, `internal/model/product.go`, `internal/config/config.go`, `Makefile`

### Phase 2: REST API — CRUD for Products
- Full CRUD: GET/POST/PUT/DELETE for products
- HTTP status codes, input validation, error handling
- In-memory storage with `sync.RWMutex` (teaches concurrency)
- Interface-based repository pattern (swap implementations later)
- **Files:** `internal/service/product.go`, `internal/repository/product.go`, `internal/model/errors.go`

### Phase 3: Database — PostgreSQL
- SQL, migrations, connection pooling, parameterized queries, `context.Context`
- Replace in-memory store with PostgreSQL (repository pattern pays off)
- Docker Compose for local Postgres
- **Libraries:** `pgx/v5`, `sqlx`, `golang-migrate/migrate`
- **Files:** `migrations/`, `internal/repository/product_postgres.go`, `docker-compose.yml`

### Phase 4: Authentication — JWT & Middleware
- User registration/login, password hashing (bcrypt), JWT tokens
- Middleware pattern: `func(http.Handler) http.Handler`
- Protected vs public routes
- **Libraries:** `golang.org/x/crypto/bcrypt`, `golang-jwt/jwt/v5`
- **Files:** `internal/handler/user.go`, `internal/service/user.go`, `internal/server/middleware/auth.go`

### Phase 5: Business Logic — Cart, Orders, Chi Router
- Shopping cart, checkout with database transactions
- SOLID principles in practice, price handling (integer cents)
- Migrate to Chi router for route groups and middleware stacking
- **Library:** `go-chi/chi/v5`
- **Files:** `internal/handler/cart.go`, `internal/handler/order.go`, `internal/service/order.go`

### Phase 6: Testing — Comprehensive Strategy
- Unit tests (mock repositories), integration tests (real DB), HTTP handler tests
- Test helpers, factories, build tags for test separation
- `httptest` package, table-driven tests for handlers
- **Files:** `*_test.go` files throughout, `internal/testutil/`

### Phase 7: Caching & Performance — Redis
- Cache-aside pattern, TTL, invalidation
- Pagination (offset + cursor-based), rate limiting
- Database indexes, `EXPLAIN ANALYZE`
- **Library:** `redis/go-redis/v9`
- **Files:** `internal/cache/`, `internal/server/middleware/ratelimit.go`

### Phase 8: Observability — Logging, Metrics, Learning Dashboard
- Structured logging with `log/slog` (stdlib)
- Request ID middleware, metrics endpoint, enhanced health checks
- **Learning Dashboard** at `/dashboard`: live request log, cache hit/miss ratios, DB query stats, errors — all via Server-Sent Events (SSE)
- **Files:** `internal/observability/`, `internal/handler/dashboard.go`, `static/dashboard/`

### Phase 9: Containerization — Docker
- Multi-stage Docker build (final image ~15MB)
- Full Docker Compose: Go app + PostgreSQL + Redis
- `docker compose up` = entire stack running
- **Files:** `Dockerfile`, `.dockerignore`, updated `docker-compose.yml`

### Phase 10: Deployment — Cloud & CI/CD
- GitHub Actions: lint + test on PR, deploy on merge
- Deploy to **Fly.io** (3 free VMs) or **Render** (free tier)
- PostgreSQL on **Neon** (free tier), Redis on **Upstash** (free tier)
- **Files:** `.github/workflows/ci.yml`, `.github/workflows/deploy.yml`, `fly.toml`

---

## Technology Choices

| Choice | Why |
|--------|-----|
| **`net/http` first, Chi later (Phase 5)** | Understand what handlers and middleware actually are before using a router |
| **PostgreSQL over MongoDB** | Relational DBs are standard in enterprise (MercadoLibre uses them). Products, orders, users = relations |
| **`pgx` + `sqlx` over GORM** | Write real SQL, understand your queries. ORMs hide too much |
| **Manual mocks over mockgen** | Go interfaces make mocking natural — no framework needed |
| **`log/slog` (stdlib)** | Built into Go since 1.21, no external dependency needed |

## Free Hosting Stack

| Service | Provider | Free Tier |
|---------|----------|-----------|
| **App hosting** | Fly.io | 3 shared VMs |
| **PostgreSQL** | Neon | 0.5 GB, autosuspend |
| **Redis** | Upstash | 10K commands/day |
| **CI/CD** | GitHub Actions | 2000 min/month |

---

## Project Structure (grows incrementally)

```
sachaweb/
  cmd/api/main.go                    — entry point, wires everything
  internal/
    config/config.go                 — env-based configuration
    model/                           — domain types (Product, User, Cart, Order)
    handler/                         — HTTP handlers (one per resource)
    service/                         — business logic layer
    repository/                      — database access layer
    server/
      server.go                      — route setup
      middleware/                     — auth, logging, rate limiting, etc.
    cache/                           — Redis cache wrapper
    observability/                   — metrics, event bus
    testutil/                        — test helpers
  migrations/                        — SQL migration files
  static/dashboard/                  — learning dashboard UI
  Dockerfile
  docker-compose.yml
  Makefile
  .github/workflows/
```

**Dependency flow (strict, one-directional):**
```
handler → service → repository → database
             ↓
           model (used by all layers)
```

---

## Project Details

- **Name:** sachaweb
- **Location:** `/home/gtrejo/sachaweb/`
- **Go module:** `github.com/guilletrejo/sachaweb`
- **GitHub user:** `guilletrejo`
- **Docker:** available for local dev (PostgreSQL, Redis via Docker Compose)

## Execution Plan

- **Phase 1** is what we build first in this session — a working HTTP server with health check and products endpoint
- Each subsequent phase = one or two conversations
- At the end of each phase: code compiles, tests pass, you can demo with `curl`, you understand every line
- We'll create a GitHub repo after Phase 1 is working

## Verification

After each phase:
1. `go build ./...` — compiles
2. `go test ./...` — tests pass
3. `curl` commands to demo endpoints
4. `go vet ./...` — no issues
5. You can explain what every backend file does
