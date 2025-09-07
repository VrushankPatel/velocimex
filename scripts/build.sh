#!/bin/bash
set -e

# Build script for Velocimex HFT ecosystem

# Configuration
GO_MIN_VERSION="1.16"
BUILD_DIR="./build"
BINARY_NAME="velocimex"
SKIP_TESTS=0
SKIP_COVERAGE=0
OBFUSCATE=0
COMPRESS=0
TIMEOUT_SECONDS=300  # 5 minutes default timeout

# Process command line arguments
for arg in "$@"; do
  case $arg in
    --skip-tests)
      SKIP_TESTS=1
      shift
      ;;
    --skip-coverage)
      SKIP_COVERAGE=1
      shift
      ;;
    --obfuscate)
      OBFUSCATE=1
      shift
      ;;
    --compress)
      COMPRESS=1
      shift
      ;;
    --timeout=*)
      TIMEOUT_SECONDS="${arg#*=}"
      shift
      ;;
    --help)
      echo "Usage: ./build.sh [options]"
      echo ""
      echo "Options:"
      echo "  --skip-tests     Skip running tests"
      echo "  --skip-coverage  Skip generating coverage report"
      echo "  --obfuscate      Obfuscate Go code using garble (must be installed)"
      echo "  --compress       Compress binary using upx (must be installed)"
      echo "  --timeout=N      Set timeout in seconds (default: 300)"
      echo "  --help           Show this help message"
      exit 0
      ;;
  esac
done

# Function to run commands with timeout (macOS compatible)
run_with_timeout() {
    local timeout=$1
    shift
    
    echo "Running command with ${timeout}s timeout: $*"
    
    # Run command in background
    "$@" &
    local cmd_pid=$!
    
    # Monitor command with timeout
    local count=0
    while [ $count -lt $timeout ]; do
        if ! kill -0 "$cmd_pid" 2>/dev/null; then
            # Command finished, get exit code
            wait "$cmd_pid"
            local exit_code=$?
            if [ $exit_code -ne 0 ]; then
                echo "Error: Command failed with exit code $exit_code"
                exit $exit_code
            fi
            return 0
        fi
        sleep 1
        count=$((count + 1))
    done
    
    # Timeout reached, kill the process
    echo "Error: Command timed out after $timeout seconds"
    kill -TERM "$cmd_pid" 2>/dev/null
    sleep 2
    kill -KILL "$cmd_pid" 2>/dev/null
    exit 1
}

# Create build directory if it doesn't exist
mkdir -p "$BUILD_DIR"

# Check Go version
echo "Checking Go version..."
go_version=$(go version | awk '{print $3}' | sed 's/go//')
if [[ "$(printf '%s\n' "$GO_MIN_VERSION" "$go_version" | sort -V | head -n1)" != "$GO_MIN_VERSION" ]]; then
  echo "Error: Go version $GO_MIN_VERSION or higher is required (found $go_version)"
  exit 1
fi

# Download dependencies
echo "Downloading dependencies..."
run_with_timeout $TIMEOUT_SECONDS go mod download

# Run tests if not skipped
if [[ $SKIP_TESTS -eq 0 ]]; then
  echo "Running tests..."
  
  if [[ $SKIP_COVERAGE -eq 0 ]]; then
    echo "Generating coverage report..."
    run_with_timeout $TIMEOUT_SECONDS go test -race -coverprofile=coverage.out -covermode=atomic ./...
    go tool cover -html=coverage.out -o coverage.html
    mv coverage.html "$BUILD_DIR/coverage.html"
    echo "Coverage report generated at $BUILD_DIR/coverage.html"
  else
    run_with_timeout $TIMEOUT_SECONDS go test -race ./...
  fi
fi

# Build the binary
echo "Building binary..."
if [[ $OBFUSCATE -eq 1 ]]; then
  if ! command -v garble &> /dev/null; then
    echo "Warning: garble not found, skipping obfuscation"
    run_with_timeout $TIMEOUT_SECONDS go build -o "$BUILD_DIR/$BINARY_NAME" ./cmd/velocimex
  else
    echo "Obfuscating code with garble..."
    run_with_timeout $TIMEOUT_SECONDS garble -seed=random build -o "$BUILD_DIR/$BINARY_NAME" ./cmd/velocimex
  fi
else
  run_with_timeout $TIMEOUT_SECONDS go build -o "$BUILD_DIR/$BINARY_NAME" ./cmd/velocimex
fi

# Compress the binary if requested
if [[ $COMPRESS -eq 1 ]]; then
  if ! command -v upx &> /dev/null; then
    echo "Warning: upx not found, skipping compression"
  else
    echo "Compressing binary with upx..."
    upx --best "$BUILD_DIR/$BINARY_NAME"
  fi
fi

# Copy UI files to build directory
echo "Copying UI files..."
mkdir -p "$BUILD_DIR/ui"
cp -r ui/* "$BUILD_DIR/ui/"

# Copy configuration file
echo "Copying configuration file..."
cp config.yaml "$BUILD_DIR/config.yaml"

echo "Build completed successfully!"
echo "Binary location: $BUILD_DIR/$BINARY_NAME"
echo ""
echo "To run the application:"
echo "  cd $BUILD_DIR"
echo "  ./$BINARY_NAME --config config.yaml"
