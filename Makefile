.PHONY: help build test clean start-infra stop-infra run-worker run-starter

help:
	@echo "Available targets:"
	@echo "  make build          - Build worker and starter binaries"
	@echo "  make test           - Run all tests"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make start-infra    - Start Docker infrastructure (Temporal, WireMock)"
	@echo "  make stop-infra     - Stop Docker infrastructure"
	@echo "  make run-worker     - Run the Temporal worker"
	@echo "  make run-starter    - Run the workflow starter"
	@echo "  make all            - Build and test"

build:
	@echo "Building binaries..."
	@mkdir -p bin
	@go build -o bin/worker ./worker/worker.go
	@go build -o bin/starter ./starter/starter.go
	@echo "Build complete! Binaries in ./bin/"

test:
	@echo "Running tests..."
	@go test ./tests/... -v -cover

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@go clean -testcache
	@echo "Clean complete!"

start-infra:
	@echo "Starting Docker infrastructure..."
	@docker-compose up -d
	@echo "Waiting for services to be ready..."
	@sleep 10
	@echo "Infrastructure started!"
	@echo "  Temporal UI: http://localhost:8080"
	@echo "  Temporal Server: localhost:7233"
	@echo "  WireMock: http://localhost:8081"

stop-infra:
	@echo "Stopping Docker infrastructure..."
	@docker-compose down
	@echo "Infrastructure stopped!"

run-worker: build
	@echo "Starting Temporal worker..."
	@./bin/worker

run-starter: build
	@echo "Starting workflow..."
	@./bin/starter

all: build test

# Development helpers
deps:
	@echo "Installing dependencies..."
	@go mod tidy
	@go mod download

fmt:
	@echo "Formatting code..."
	@go fmt ./...

lint:
	@echo "Running linter..."
	@golangci-lint run || echo "Install golangci-lint for linting support"

coverage:
	@echo "Running tests with coverage..."
	@go test ./tests/... -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"
