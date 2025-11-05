.PHONY: check build format vet modernize lint shadow test testv fix install help

# Default target
help:
	@echo "Available commands:"
	@echo "  check      - Run all code quality checks"
	@echo "  build      - Build the project"
	@echo "  format     - Format code with gofumpt"
	@echo "  vet        - Run go vet"
	@echo "  modernize  - Run modernize"
	@echo "  lint       - Run golangci-lint"
	@echo "  shadow     - Run shadow analysis"
	@echo "  test       - Run unit tests (simple output)"
	@echo "  testv      - Run unit tests with verbose output"
	@echo "  fix        - Auto-fix code issues (gofumpt, golangci-lint, shadow, modernize)"
	@echo "  install    - Install gg command (go install ./cmd/gg)"
	@echo "  help       - Show this help message"

# Run all code quality checks
check: build format vet modernize lint shadow
	@echo "All checks passed successfully!"

# Build the project
build:
	@echo "Running go build..."
	go build ./...

# Format code with gofumpt
format:
	@echo "Running gofumpt..."
	gofumpt -l -w .

# Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...

# Run modernize
modernize:
	@echo "Running modernize..."
	modernize ./...

# Run golangci-lint
lint:
	@echo "Running golangci-lint..."
	golangci-lint run ./...

# Run shadow analysis
shadow:
	@echo "Running shadow analysis..."
	shadow ./...

# Run unit tests
test:
	@echo "Running unit tests..."
	go test ./model/...
	go test ./util/...
	go test ./dsl
	go test ./client
	go test ./internal/codegen/gen/
	go test ./module/helloworld

# Run unit tests with verbose output
testv:
	@echo "Running unit tests with verbose output..."
	go test -v ./model/...
	go test -v ./util/...
	go test -v ./dsl
	go test -v ./client
	go test -v ./internal/codegen/gen/
	go test -v ./module/helloworld

# Auto-fix code issues
fix:
	@echo "Running auto-fix tools..."
	@echo "Running gofumpt..."
	gofumpt -l -w .
	@echo "Running golangci-lint --fix..."
	golangci-lint run --fix ./...
	@echo "Running shadow -fix..."
	shadow -fix ./...
	@echo "Running modernize -fix..."
	modernize -fix ./...
	@echo "All auto-fix operations completed!"

# Install gg command
install:
	@echo "Installing gg command..."
	go install ./cmd/gg
