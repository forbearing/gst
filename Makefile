.PHONY: check build vet modernize lint shadow help

# Default target
help:
	@echo "Available commands:"
	@echo "  check      - Run all code quality checks"
	@echo "  build      - Build the project"
	@echo "  vet        - Run go vet"
	@echo "  modernize  - Run modernize"
	@echo "  lint       - Run golangci-lint"
	@echo "  shadow     - Run shadow analysis"
	@echo "  help       - Show this help message"

# Run all code quality checks
check: build vet modernize lint shadow
	@echo "All checks passed successfully!"

# Build the project
build:
	@echo "Running go build..."
	go build ./...

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