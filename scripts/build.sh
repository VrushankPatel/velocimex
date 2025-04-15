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
    --help)
      echo "Usage: ./build.sh [options]"
      echo ""
      echo "Options:"
      echo "  --skip-tests     Skip running tests"
      echo "  --skip-coverage  Skip generating coverage report"
      echo "  --obfuscate      Obfuscate Go code using garble (must be installed)"
      echo "  --compress       Compress binary using upx (must be installed)"
      echo "  --help           Show this help message"
      exit 0
      ;;
  esac
done

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
go mod download

# Run tests if not skipped
if [[ $SKIP_TESTS -eq 0 ]]; then
  echo "Running tests..."
  
  if [[ $SKIP_COVERAGE -eq 0 ]]; then
    echo "Generating coverage report..."
    go test -race -coverprofile=coverage.out -covermode=atomic ./...
    go tool cover -html=coverage.out -o coverage.html
    mv coverage.html "$BUILD_DIR/coverage.html"
    echo "Coverage report generated at $BUILD_DIR/coverage.html"
  else
    go test -race ./...
  fi
fi

# Build the binary
echo "Building binary..."
if [[ $OBFUSCATE -eq 1 ]]; then
  if ! command -v garble &> /dev/null; then
    echo "Warning: garble not found, skipping obfuscation"
    go build -o "$BUILD_DIR/$BINARY_NAME" ./cmd/velocimex
  else
    echo "Obfuscating code with garble..."
    garble -seed=random build -o "$BUILD_DIR/$BINARY_NAME" ./cmd/velocimex
  fi
else
  go build -o "$BUILD_DIR/$BINARY_NAME" ./cmd/velocimex
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
