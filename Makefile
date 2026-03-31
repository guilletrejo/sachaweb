# Makefile provides shortcuts for common development tasks.
# Instead of typing "go run ./cmd/api", you type "make run".

.PHONY: run build test vet clean docker-up docker-down docker-reset

# --- Application ---

# run starts the server (requires PostgreSQL to be running).
run:
	go run ./cmd/api

# build compiles the project into a binary.
build:
	go build -o bin/sachaweb ./cmd/api

# test runs all tests recursively.
test:
	go test -v ./...

# vet runs Go's built-in static analysis.
vet:
	go vet ./...

# clean removes build artifacts.
clean:
	rm -rf bin/

# --- Docker (PostgreSQL) ---

# docker-up starts PostgreSQL in the background.
# -d = detached mode (runs in background, returns your terminal).
# --wait = waits until the healthcheck passes before returning.
docker-up:
	docker compose up -d --wait

# docker-down stops PostgreSQL (data is preserved in the volume).
docker-down:
	docker compose down

# docker-reset stops PostgreSQL AND deletes all data (fresh start).
# The -v flag removes named volumes (pgdata), wiping the database.
docker-reset:
	docker compose down -v
