package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/forbearing/gst/internal/clioutput"
	"github.com/spf13/cobra"
)

// dockerCmd represents the docker command
var dockerCmd = &cobra.Command{
	Use:   "docker",
	Short: "Docker management for the application",
	Long: `Docker management for the application.

This command provides subcommands to:
- Generate Dockerfile
- Build Docker image using best practices
- Support both development and production builds
- Use minimal base images for production

Examples:
  gg docker gen                # Generate Dockerfile only
  gg docker build              # Build with default settings
  gg docker build --tag myapp:v1.0   # Build with custom tag
  gg docker build --dev        # Build development image
  gg docker build --no-cache   # Build without using cache`,
}

// dockerGenCmd represents the docker gen command
var dockerGenCmd = &cobra.Command{
	Use:   "gen",
	Short: "Generate Dockerfile for the application",
	Long: `Generate Dockerfile for the application using best practices.

This command will generate a Dockerfile with:
- Multi-stage build for production
- Development mode with hot reload support
- Security best practices
- Minimal base images

Examples:
  gg docker gen                # Generate production Dockerfile
  gg docker gen --dev          # Generate development Dockerfile`,
	RunE: dockerGenRun,
}

// dockerBuildCmd represents the docker build command
var dockerBuildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build Docker image for the application",
	Long: `Build Docker image for the application using best practices.

This command will:
- Generate a Dockerfile if it doesn't exist
- Build a Docker image using multi-stage build
- Support both development and production builds
- Use minimal base images for production

Examples:
  gg docker build              # Build with default settings
  gg docker build --tag myapp:v1.0   # Build with custom tag
  gg docker build --dev        # Build development image
  gg docker build --no-cache   # Build without using cache`,
	RunE: dockerBuildRun,
}

var (
	dockerTag     string
	dockerDev     bool
	dockerNoCache bool
	dockerPush    bool
)

const dockerBuildFailureMessage = "Docker build failed: %v"

func init() {
	// Add subcommands to docker command
	dockerCmd.AddCommand(dockerGenCmd, dockerBuildCmd)

	// Flags for docker gen command
	dockerGenCmd.Flags().BoolVar(&dockerDev, "dev", false, "Generate development Dockerfile with hot reload support")

	// Flags for docker build command
	dockerBuildCmd.Flags().StringVarP(&dockerTag, "tag", "t", "", "Docker image tag (default: auto-generated from module name)")
	dockerBuildCmd.Flags().BoolVar(&dockerDev, "dev", false, "Build development image with hot reload support")
	dockerBuildCmd.Flags().BoolVar(&dockerNoCache, "no-cache", false, "Build without using cache")
	dockerBuildCmd.Flags().BoolVar(&dockerPush, "push", false, "Push image to registry after build")
}

// dockerGenRun generates Dockerfile only
func dockerGenRun(cmd *cobra.Command, args []string) error {
	// Generate Dockerfile
	if err := generateDockerfile(); err != nil {
		return fmt.Errorf("failed to generate Dockerfile: %w", err)
	}

	clioutput.Success("", "Dockerfile generated successfully at: ./Dockerfile")
	return nil
}

// dockerBuildRun builds Docker image
func dockerBuildRun(cmd *cobra.Command, args []string) error {
	// Check if Docker is installed
	if err := checkDockerInstalled(); err != nil {
		return fmt.Errorf("docker is not installed or not accessible: %w", err)
	}

	// Generate Dockerfile if it doesn't exist
	if err := generateDockerfile(); err != nil {
		return fmt.Errorf("failed to generate Dockerfile: %w", err)
	}

	// Determine image tag
	tag := dockerTag
	if tag == "" {
		var err error
		tag, err = getDefaultImageTag()
		if err != nil {
			return fmt.Errorf("failed to determine image tag: %w", err)
		}
	}

	// Build Docker image (skip if Docker is not available)
	if err := buildDockerImage(tag); err != nil {
		clioutput.Warn("", dockerBuildFailureMessage, err)
		clioutput.Success("", "Dockerfile generated successfully at: ./Dockerfile")
		return nil
	}

	// Push image if requested
	if dockerPush {
		if err := pushDockerImage(tag); err != nil {
			return fmt.Errorf("failed to push Docker image: %w", err)
		}
	}

	clioutput.Success("", "Docker image built successfully: %s", tag)
	if dockerPush {
		clioutput.Success("", "Docker image pushed successfully: %s", tag)
	}

	return nil
}

// checkDockerInstalled checks if Docker is installed
func checkDockerInstalled() error {
	_, err := exec.LookPath("docker")
	if err != nil {
		clioutput.Warn("", "Docker is not installed or not in PATH")
		return fmt.Errorf("docker command not found")
	}
	return nil
}

// generateDockerfile generates a Dockerfile if it doesn't exist
func generateDockerfile() error {
	dockerfilePath := "Dockerfile"
	if fileExists(dockerfilePath) {
		clioutput.Item("", "Dockerfile already exists, skipping generation")
		return nil
	}

	clioutput.Section("Generate Dockerfile")

	// Get module name for binary name
	moduleName, err := getModuleName()
	if err != nil {
		return err
	}

	binaryName := filepath.Base(moduleName)

	dockerfileContent := generateDockerfileContent(binaryName)

	if err := os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0o600); err != nil {
		clioutput.Error("", "Failed to write Dockerfile: %v", err)
		return fmt.Errorf("failed to write Dockerfile: %w", err)
	}

	clioutput.Success("", "Dockerfile generated successfully")
	return nil
}

// generateDockerfileContent generates the Dockerfile content based on best practices
func generateDockerfileContent(binaryName string) string {
	if dockerDev {
		return `# syntax=docker/dockerfile:1

# Development Dockerfile with hot reload support
FROM golang:1.25-alpine AS base

# Install air for hot reload
RUN go install github.com/air-verse/air@latest

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Expose port
EXPOSE 8080

# Use air for hot reload
CMD ["air"]
`
	}

	return fmt.Sprintf(`# syntax=docker/dockerfile:1

# Multi-stage build for production
# Build stage
FROM golang:1.25-alpine AS builder

# Install git and ca-certificates (needed for go mod download)
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
# CGO_ENABLED=0: Build a statically linked binary
# GOOS=linux: Build for Linux
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o %s .

# Production stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/%s .

# Change ownership to non-root user
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./%s"]
`, binaryName, binaryName, binaryName)
}

// getDefaultImageTag generates a default image tag based on module name
func getDefaultImageTag() (string, error) {
	moduleName, err := getModuleName()
	if err != nil {
		return "", err
	}

	// Extract the last part of the module path as image name
	imageName := filepath.Base(moduleName)
	// Clean the image name to be Docker-compatible
	imageName = strings.ToLower(strings.ReplaceAll(imageName, "_", "-"))

	return fmt.Sprintf("%s:latest", imageName), nil
}

// buildDockerImage builds the Docker image
func buildDockerImage(tag string) error {
	clioutput.Section("Build Docker Image")

	args := []string{"build"}

	if dockerNoCache {
		args = append(args, "--no-cache")
	}

	args = append(args, "-t", tag, ".")

	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker build failed: %w", err)
	}

	return nil
}

// pushDockerImage pushes the Docker image to registry
func pushDockerImage(tag string) error {
	clioutput.Section("Push Docker Image")

	cmd := exec.Command("docker", "push", tag)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		clioutput.Error("", "Failed to push Docker image: %v", err)
		return fmt.Errorf("docker push failed: %w", err)
	}

	clioutput.Success("", "Docker image pushed successfully: %s", tag)
	return nil
}

// getModuleName reads the module name from go.mod file
func getModuleName() (string, error) {
	content, err := os.ReadFile("go.mod")
	if err != nil {
		return "", fmt.Errorf("failed to read go.mod: %w", err)
	}

	lines := strings.SplitSeq(string(content), "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module")), nil
		}
	}

	return "", fmt.Errorf("module name not found in go.mod")
}
