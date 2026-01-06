#!/bin/bash
# Build script for multi-platform binaries
# Usage: ./scripts/build-all.sh [version]

set -e

VERSION="${1:-dev}"
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)
BUILD_DIR="build"

LDFLAGS="-s -w -X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.Date=${DATE}"
CLI_LDFLAGS="-s -w -X github.com/takhin/takhin/pkg/cli.Version=${VERSION} -X github.com/takhin/takhin/pkg/cli.Commit=${COMMIT} -X github.com/takhin/takhin/pkg/cli.BuildDate=${DATE}"

# Clean build directory
rm -rf "${BUILD_DIR}"
mkdir -p "${BUILD_DIR}"

# Platform configurations
PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

echo "Building Takhin ${VERSION} for multiple platforms..."
echo "Commit: ${COMMIT}"
echo "Date: ${DATE}"
echo ""

cd backend

for platform in "${PLATFORMS[@]}"; do
    IFS='/' read -r -a array <<< "$platform"
    GOOS="${array[0]}"
    GOARCH="${array[1]}"
    
    OUTPUT_NAME="../${BUILD_DIR}/takhin-${GOOS}-${GOARCH}"
    CONSOLE_NAME="../${BUILD_DIR}/takhin-console-${GOOS}-${GOARCH}"
    CLI_NAME="../${BUILD_DIR}/takhin-cli-${GOOS}-${GOARCH}"
    
    if [ "$GOOS" = "windows" ]; then
        OUTPUT_NAME="${OUTPUT_NAME}.exe"
        CONSOLE_NAME="${CONSOLE_NAME}.exe"
        CLI_NAME="${CLI_NAME}.exe"
    fi
    
    echo "Building for ${GOOS}/${GOARCH}..."
    
    # Build Takhin
    CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH go build \
        -ldflags="${LDFLAGS}" \
        -o "${OUTPUT_NAME}" \
        ./cmd/takhin
    
    # Build Console
    CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH go build \
        -ldflags="${LDFLAGS}" \
        -o "${CONSOLE_NAME}" \
        ./cmd/console
    
    # Build CLI
    CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH go build \
        -ldflags="${CLI_LDFLAGS}" \
        -o "${CLI_NAME}" \
        ./cmd/takhin-cli
    
    echo "✓ Built ${GOOS}/${GOARCH}"
done

cd ..

echo ""
echo "Build complete! Binaries in ${BUILD_DIR}/"
ls -lh "${BUILD_DIR}/"
