# Makefile provides shortcuts for common development tasks.
# Instead of typing "go run ./cmd/api", you type "make run".
# This is standard practice in Go projects — every developer on the
# team uses the same commands regardless of their setup.

# .PHONY tells make that these targets are commands, not files.
# Without this, if you had a file called "build" in the directory,
# make would think the target is already up-to-date and skip it.
.PHONY: run build test vet clean

# run starts the server in development mode.
# "go run" compiles and runs in one step (doesn't produce a binary file).
run:
	go run ./cmd/api

# build compiles the project into a binary at ./bin/sachaweb.
# The -o flag specifies the output path.
# In production, you ship the binary, not the source code.
build:
	go build -o bin/sachaweb ./cmd/api

# test runs all tests in the project recursively.
# ./... means "this package and all sub-packages."
# -v is verbose mode — shows each test name and result.
test:
	go test -v ./...

# vet runs Go's built-in static analysis tool.
# It catches common mistakes like unreachable code, wrong printf
# format strings, and suspicious constructs that compile but are
# probably bugs.
vet:
	go vet ./...

# clean removes build artifacts.
clean:
	rm -rf bin/
