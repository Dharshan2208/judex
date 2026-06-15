GO       := go
BIN_DIR  := bin
GO_FLAGS := -ldflags="-s -w"

IMAGES := python cpp c java go

.PHONY: help build build-api build-worker images run-api run-worker dev redis \
        compose-up compose-down test lint vet fmt clean

help: ## List all available targets with descriptions
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Docker Sandbox Images

images: $(addprefix image-, $(IMAGES)) ## Build all sandbox Docker images

image-%:
	docker build -t compiler-$* -f docker/$*/Dockerfile docker/$*

# Run

run-api: ## Start the API server (development)
	$(GO) run ./cmd/api

run-worker: ## Start the worker (development)
	$(GO) run ./cmd/worker

# Docker Compose
compose-up: ## Start all services via Docker Compose
	mkdir -p /app/temp
	docker compose up --build
	@echo "Services started. Run 'make compose-down' to stop."

compose-down: ## Stop all Docker Compose services
	docker compose down

# Code Quality

test: ## Run all tests with race detection
	$(GO) test ./... -v -race -count=1

vet: ## Run go vet
	$(GO) vet ./...

lint: ## Run golangci-lint (requires golangci-lint to be installed)
	golangci-lint run ./...

clean: ## Remove build artifacts
	rm -rf $(BIN_DIR)/
