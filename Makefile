.PHONY: build run test clean install help

# Binary name
BINARY_NAME=go3mf

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(BINARY_NAME) .
	@echo "Build complete: ./$(BINARY_NAME)"

# Run the application with arguments (usage: make run ARGS="combine example/config.yaml")
run: build
	@echo "Running $(BINARY_NAME)..."
	@./$(BINARY_NAME) $(ARGS)

# Run all tests
test:
	@echo "Running tests..."
	@go test ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -cover ./...

# Run tests with verbose output
test-verbose:
	@echo "Running tests (verbose)..."
	@go test -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@go clean
	@echo "Clean complete"

# Install the application
install:
	@echo "Installing $(BINARY_NAME)..."
	@go install .
	@echo "Install complete"

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...

# Run all checks (fmt, vet, test)
check: fmt vet test
	@echo "All checks passed!"

# Show help
help:
	@echo "Available targets:"
	@echo "  build          - Build the application"
	@echo "  run            - Build and run the application (use ARGS=\"...\" to pass arguments)"
	@echo "  test           - Run all tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  test-verbose   - Run tests with verbose output"
	@echo "  clean          - Remove build artifacts"
	@echo "  install        - Install the application"
	@echo "  fmt            - Format code"
	@echo "  vet            - Run go vet"
	@echo "  check          - Run fmt, vet, and test"
	@echo "  help           - Show this help message"
	@echo ""
	@echo "Examples:"
	@echo "  make build"
	@echo "  make run ARGS=\"version\""
	@echo "  make run ARGS=\"combine example/config.yaml\""
	@echo "  make run ARGS=\"inspect output.3mf\""
	@echo "  make test"
