# VectorDB Makefile
# High-performance vector database build and development tools

.PHONY: build run test clean fmt lint help demo load-test

# Build configuration
BINARY_NAME := vectordb
BUILD_DIR := ./bin
CMD_DIR := ./cmd
GO_FILES := $(shell find . -name "*.go" -type f)

# Default target
help: ## Show this help message
	@echo "VectorDB - High-Performance Vector Database"
	@echo "==========================================="
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build targets
build: $(BUILD_DIR)/$(BINARY_NAME) ## Build the vectordb binary

$(BUILD_DIR)/$(BINARY_NAME): $(GO_FILES)
	@echo "🔨 Building VectorDB..."
	@mkdir -p $(BUILD_DIR)
	@go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "✅ Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

build-dev: ## Build with debug information
	@echo "🔨 Building VectorDB (development mode)..."
	@mkdir -p $(BUILD_DIR)
	@go build -race -o $(BUILD_DIR)/$(BINARY_NAME)-dev $(CMD_DIR)
	@echo "✅ Development build complete: $(BUILD_DIR)/$(BINARY_NAME)-dev"

# Run targets
run: build ## Build and run VectorDB with default settings
	@echo "🚀 Starting VectorDB..."
	@$(BUILD_DIR)/$(BINARY_NAME)

run-dev: build-dev ## Build and run VectorDB in development mode
	@echo "🚀 Starting VectorDB (development mode)..."
	@$(BUILD_DIR)/$(BINARY_NAME)-dev -dimensions 128 -port 8080

run-custom: build ## Run with custom settings (example)
	@echo "🚀 Starting VectorDB with custom configuration..."
	@$(BUILD_DIR)/$(BINARY_NAME) -port 9090 -dimensions 256 -data ./custom_data

# Development targets
fmt: ## Format Go code
	@echo "🎨 Formatting code..."
	@go fmt ./...
	@echo "✅ Code formatted"

lint: ## Run linting tools
	@echo "🔍 Running linters..."
	@command -v golangci-lint >/dev/null 2>&1 || { echo "Installing golangci-lint..."; go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; }
	@golangci-lint run
	@echo "✅ Linting complete"

vet: ## Run go vet
	@echo "🔍 Running go vet..."
	@go vet ./...
	@echo "✅ Vet complete"

# Testing and demo targets
test: ## Run unit tests
	@echo "🧪 Running tests..."
	@go test -v ./...
	@echo "✅ Tests complete"

test-race: ## Run tests with race detection
	@echo "🧪 Running tests with race detection..."
	@go test -race -v ./...
	@echo "✅ Race tests complete"

benchmark: ## Run benchmark tests
	@echo "📊 Running benchmarks..."
	@go test -bench=. -benchmem ./...

demo: build ## Run the interactive demo
	@echo "🎬 Starting demo..."
	@if ! pgrep -f "$(BINARY_NAME)" > /dev/null; then \
		echo "Starting VectorDB server..."; \
		$(BUILD_DIR)/$(BINARY_NAME) & \
		sleep 3; \
		echo "Running demo script..."; \
		./examples/demo.sh; \
		echo "Stopping server..."; \
		pkill -f "$(BINARY_NAME)"; \
	else \
		echo "VectorDB server is already running"; \
		./examples/demo.sh; \
	fi

load-test: build ## Run load testing
	@echo "⚡ Starting load test..."
	@if ! pgrep -f "$(BINARY_NAME)" > /dev/null; then \
		echo "Starting VectorDB server..."; \
		$(BUILD_DIR)/$(BINARY_NAME) & \
		sleep 3; \
		echo "Running load test..."; \
		./examples/load_test.sh; \
		echo "Stopping server..."; \
		pkill -f "$(BINARY_NAME)"; \
	else \
		echo "VectorDB server is already running"; \
		./examples/load_test.sh; \
	fi

python-demo: ## Run Python client demo (requires requests, numpy)
	@echo "🐍 Running Python client demo..."
	@if ! command -v python3 >/dev/null 2>&1; then \
		echo "❌ Python 3 is required"; \
		exit 1; \
	fi
	@python3 -c "import requests, numpy" 2>/dev/null || { echo "❌ Please install: pip install requests numpy"; exit 1; }
	@if ! pgrep -f "$(BINARY_NAME)" > /dev/null; then \
		echo "Starting VectorDB server..."; \
		$(BUILD_DIR)/$(BINARY_NAME) & \
		sleep 3; \
		echo "Running Python demo..."; \
		python3 ./examples/python_client.py; \
		echo "Stopping server..."; \
		pkill -f "$(BINARY_NAME)"; \
	else \
		echo "VectorDB server is already running"; \
		python3 ./examples/python_client.py; \
	fi
# Maintenance targets
clean: ## Clean build artifacts
	@echo "🧹 Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -rf ./vectordb_data
	@rm -rf ./custom_data
	@go clean
	@echo "✅ Clean complete"

deps: ## Download and tidy dependencies
	@echo "📦 Managing dependencies..."
	@go mod download
	@go mod tidy
	@echo "✅ Dependencies updated"

install: build ## Install binary to GOPATH/bin
	@echo "📦 Installing VectorDB..."
	@go install $(CMD_DIR)
	@echo "✅ VectorDB installed to GOPATH/bin"

# Release targets
release: clean fmt vet test build ## Prepare a release build
	@echo "🚀 Release build complete"
	@echo "Binary: $(BUILD_DIR)/$(BINARY_NAME)"
	@echo "Size: $$(du -h $(BUILD_DIR)/$(BINARY_NAME) | cut -f1)"

# Development workflow
dev: clean fmt vet build run-dev ## Complete development workflow

# Quick quality check
check: fmt vet test ## Run formatting, vetting, and tests

# Performance profiling
profile: build ## Run with CPU profiling
	@echo "📊 Running with CPU profiling..."
	@$(BUILD_DIR)/$(BINARY_NAME) -cpuprofile=cpu.prof &
	@sleep 10
	@pkill -f "$(BINARY_NAME)"
	@echo "CPU profile saved to cpu.prof"
	@echo "Analyze with: go tool pprof cpu.prof"

# Show project statistics
stats: ## Show project statistics
	@echo "📈 VectorDB Project Statistics"
	@echo "=============================="
	@echo "Go files: $$(find . -name '*.go' | wc -l)"
	@echo "Lines of code: $$(find . -name '*.go' -exec wc -l {} + | tail -1 | awk '{print $$1}')"
	@echo "Package size: $$(du -sh . | cut -f1)"
	@echo ""
	@echo "Git statistics:"
	@git log --oneline | wc -l | awk '{print "Commits: " $$1}'
	@git ls-files | wc -l | awk '{print "Files tracked: " $$1}'

# Generate documentation
docs: ## Generate and serve documentation
	@echo "📚 Generating documentation..."
	@command -v godoc >/dev/null 2>&1 || { echo "Installing godoc..."; go install golang.org/x/tools/cmd/godoc@latest; }
	@echo "Starting documentation server at http://localhost:6060"
	@godoc -http=:6060

# Show current status
status: ## Show current project status
	@echo "📊 VectorDB Status"
	@echo "=================="
	@echo "Build status: $$(if [ -f $(BUILD_DIR)/$(BINARY_NAME) ]; then echo "✅ Built"; else echo "❌ Not built"; fi)"
	@echo "Server status: $$(if pgrep -f "$(BINARY_NAME)" > /dev/null; then echo "🟢 Running"; else echo "🔴 Stopped"; fi)"
	@echo "Git status: $$(git status --porcelain | wc -l) uncommitted changes"
	@echo ""
	@if [ -f $(BUILD_DIR)/$(BINARY_NAME) ]; then \
		echo "Binary info:"; \
		echo "  Size: $$(du -h $(BUILD_DIR)/$(BINARY_NAME) | cut -f1)"; \
		echo "  Modified: $$(stat -f %Sm $(BUILD_DIR)/$(BINARY_NAME))"; \
	fi
