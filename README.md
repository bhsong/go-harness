# Go Harness

A learning project that builds an LLM harness from scratch using only the Go standard library.

Reference: https://github.com/IsuruMaduranga/harness-engineering-101

## Current stage

Stage 0: development environment and executable entry point.

The current program prints a startup message.
API calls and .env loading will be implemented in Go v1.

## Roadmap

- v1: Messages and Anthropic Messages API
- v2: Tools and agent loop
- v3: Multi-agent
- v4: Background work and steering
- v5: Todo management

## Local configuration

Copy .env.example to .env when setting up a fresh checkout.
Fill in the API key and model ID before running the future v1 implementation.
The .env file must remain untracked.

## Run

```bash
go run .
```

## Check

```bash
go fmt ./...
go vet ./...
go test ./...
```

## Build

```bash
mkdir -p bin
go build -o bin/go-harness .
./bin/go-harness
```
