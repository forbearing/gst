.PHONY: check build format vet modernize lint shadow nilness lostcancel stringintconv test testv fix install help

# Tool versions - must match go.mod exactly
GOLANGCI_LINT_VERSION := v2.6.2
GOFUMPT_VERSION := v0.9.2
GOTOOLS_VERSION := v0.38.0

# Default target
help:
	@echo "Available commands:"
	@echo "  check          - Run all code quality checks"
	@echo "  build          - Build the project"
	@echo "  format         - Format code with gofumpt"
	@echo "  vet            - Run go vet"
	@echo "  lint           - Run golangci-lint"
	@echo "  modernize      - Run modernize"
	@echo "  nilness        - Run nilness analysis"
	@echo "  shadow         - Run shadow analysis"
	@echo "  lostcancel     - Run lostcancel analysis"
	@echo "  stringintconv  - Run stringintconv analysis"
	@echo "  test           - Run unit tests (simple output)"
	@echo "  testv          - Run unit tests with verbose output"
	@echo "  fix            - Auto-fix code issues (gofumpt, golangci-lint, shadow, modernize)"
	@echo "  install        - Install gg command and development tools"
	@echo "  help           - Show this help message"

# Run all code quality checks
# Order matches make install tool installation order
check: build lint format modernize nilness shadow lostcancel stringintconv vet
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

# Run nilness analysis
nilness:
	@echo "Running nilness analysis..."
	nilness ./...

# Run shadow analysis
shadow:
	@echo "Running shadow analysis..."
	shadow ./...

# Run lostcancel analysis
lostcancel:
	@echo "Running lostcancel analysis..."
	lostcancel ./...

# Run stringintconv analysis
stringintconv:
	@echo "Running stringintconv analysis..."
	stringintconv ./...

# Run unit tests
test:
	@echo "Running unit tests..."
	go test ./model/...
	go test ./util/...
	go test ./dsl
	go test ./client
	go test ./database/...
	go test ./internal/codegen/gen/
	go test ./module/helloworld

# Run unit tests with verbose output
testv:
	@echo "Running unit tests with verbose output..."
	go test -v ./model/...
	go test -v ./util/...
	go test -v ./dsl
	go test -v ./client
	go test -v ./database/...
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

# Install gg command and development tools
# Versions are defined at the top of this file and must match go.mod exactly
install:
	@echo "Installing development tools from go.mod..."
	@echo "Installing golangci-lint@$(GOLANGCI_LINT_VERSION)..."
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	@echo "Installing gofumpt@$(GOFUMPT_VERSION)..."
	@go install mvdan.cc/gofumpt@$(GOFUMPT_VERSION)
	@echo "Installing modernize@$(GOTOOLS_VERSION)..."
	@go install golang.org/x/tools/go/analysis/passes/modernize/cmd/modernize@$(GOTOOLS_VERSION)
	@echo "Installing nilness@$(GOTOOLS_VERSION)..."
	@go install golang.org/x/tools/go/analysis/passes/nilness/cmd/nilness@$(GOTOOLS_VERSION)
	@echo "Installing shadow@$(GOTOOLS_VERSION)..."
	@go install golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow@$(GOTOOLS_VERSION)
	@echo "Installing lostcancel@$(GOTOOLS_VERSION)..."
	@go install golang.org/x/tools/go/analysis/passes/lostcancel/cmd/lostcancel@$(GOTOOLS_VERSION)
	@echo "Installing stringintconv@$(GOTOOLS_VERSION)..."
	@go install golang.org/x/tools/go/analysis/passes/stringintconv/cmd/stringintconv@$(GOTOOLS_VERSION)
	@echo "Installing gg command..."
	@go install ./cmd/gg
	@echo "Installation completed!"
