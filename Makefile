.PHONY: fmt lint test

fmt:
	@echo "Formatting Go files..."
	@go fmt ./...
	@echo "Formatting complete"

lint:
	@echo "Running linters..."
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not installed. Install with: brew install golangci-lint"; exit 1; }
	@golangci-lint run ./... --fix
	@echo "Linting complete"

test:
	@echo "Running tests..."
	@go test -v ./...
	@echo "Tests complete"
