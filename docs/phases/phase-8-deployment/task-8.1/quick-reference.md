# Task 8.1: Multi-Platform Build - Quick Reference

## Build Commands

### Local Development
```bash
# Build for current platform
task backend:build

# Build all platforms
task backend:build:all

# Build with custom version
VERSION=v1.2.3 task backend:build:all

# Build via script directly
./scripts/build-all.sh v1.0.0
```

### Testing Release
```bash
# Validate GoReleaser config
task backend:release:check
goreleaser check

# Create snapshot release (no publish)
task backend:release:snapshot
goreleaser release --snapshot --clean --skip=sign
```

### Manual Cross-Compilation
```bash
cd backend

# Linux AMD64
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o takhin-linux-amd64 ./cmd/takhin

# Linux ARM64
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o takhin-linux-arm64 ./cmd/takhin

# macOS Intel
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o takhin-darwin-amd64 ./cmd/takhin

# macOS Apple Silicon
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o takhin-darwin-arm64 ./cmd/takhin

# Windows
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o takhin-windows-amd64.exe ./cmd/takhin
```

## Release Workflow

### Creating a Release
```bash
# 1. Update version
vim backend/cmd/takhin/version.go

# 2. Commit changes
git add .
git commit -m "chore: bump version to v1.2.3"

# 3. Create and push tag
git tag -a v1.2.3 -m "Release v1.2.3"
git push origin v1.2.3

# 4. GitHub Actions automatically:
#    - Builds all platforms
#    - Creates GitHub Release
#    - Publishes Docker images
#    - Updates Homebrew tap
```

### Manual Release (if needed)
```bash
# Export GitHub token
export GITHUB_TOKEN=ghp_xxxxx

# Run GoReleaser
goreleaser release --clean

# Dry run first
goreleaser release --skip=publish --clean
```

## Docker Images

### Building Multi-Arch Images
```bash
# Setup buildx
docker buildx create --name multiarch --use
docker buildx inspect --bootstrap

# Build for multiple platforms
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t ghcr.io/takhin/takhin:latest \
  -f deployments/Dockerfile.goreleaser \
  --push \
  .
```

### Using Published Images
```bash
# Pull latest
docker pull ghcr.io/takhin/takhin:latest

# Pull specific version
docker pull ghcr.io/takhin/takhin:v1.2.3

# Pull specific architecture
docker pull ghcr.io/takhin/takhin:v1.2.3-arm64

# Run container
docker run -p 9092:9092 ghcr.io/takhin/takhin:latest
```

## Package Installation

### Homebrew (macOS)
```bash
# Install from tap
brew install takhin/tap/takhin

# Upgrade
brew upgrade takhin

# Check version
takhin --version
```

### Debian/Ubuntu
```bash
# Download package
wget https://github.com/takhin/takhin/releases/download/v1.0.0/takhin_1.0.0_amd64.deb

# Install
sudo dpkg -i takhin_1.0.0_amd64.deb

# Fix dependencies if needed
sudo apt-get install -f

# Start service
sudo systemctl start takhin
sudo systemctl enable takhin
```

### RHEL/CentOS/Fedora
```bash
# Download package
wget https://github.com/takhin/takhin/releases/download/v1.0.0/takhin_1.0.0_x86_64.rpm

# Install
sudo rpm -i takhin_1.0.0_x86_64.rpm

# Or with yum
sudo yum install takhin_1.0.0_x86_64.rpm

# Start service
sudo systemctl start takhin
sudo systemctl enable takhin
```

### Alpine Linux
```bash
# Download package
wget https://github.com/takhin/takhin/releases/download/v1.0.0/takhin_1.0.0_x86_64.apk

# Install
apk add --allow-untrusted takhin_1.0.0_x86_64.apk

# Start service
rc-update add takhin default
rc-service takhin start
```

### Windows
```bash
# Download ZIP
curl -LO https://github.com/takhin/takhin/releases/download/v1.0.0/takhin_1.0.0_windows_x86_64.zip

# Extract
unzip takhin_1.0.0_windows_x86_64.zip

# Add to PATH or run directly
.\takhin.exe --version
```

## CI/CD Workflows

### Build Workflow Triggers
- Push to `main` or `develop`
- Pull requests to `main` or `develop`
- Changes in `backend/**` or `.github/workflows/build.yml`

### Release Workflow Triggers
- Tag push matching `v*` pattern (e.g., v1.0.0, v2.1.3-beta)

### Manual Workflow Dispatch
```bash
# Via GitHub CLI
gh workflow run build.yml

# Via GitHub web interface
# Go to Actions → Build → Run workflow
```

## Verification

### Check Build Output
```bash
# List built binaries
ls -lh build/

# Check binary size
du -h build/takhin-*

# Verify binary works
./build/takhin-linux-amd64 --version

# Check dependencies (should be none)
ldd build/takhin-linux-amd64  # Output: "not a dynamic executable"
```

### Test Release Artifacts
```bash
# After GoReleaser snapshot
ls -lh dist/

# Verify checksums
cd dist
sha256sum -c checksums.txt

# Test tarball extraction
tar -tzf takhin_*_linux_amd64.tar.gz
tar -xzf takhin_*_linux_amd64.tar.gz
./takhin --version
```

### Verify Docker Images
```bash
# Inspect image
docker inspect ghcr.io/takhin/takhin:latest

# Check image size
docker images | grep takhin

# Verify multi-arch support
docker manifest inspect ghcr.io/takhin/takhin:latest
```

## Troubleshooting

### Build Failures

**Issue: Go module errors**
```bash
cd backend
go mod download
go mod tidy
```

**Issue: Missing dependencies**
```bash
# Install GoReleaser
go install github.com/goreleaser/goreleaser@latest

# Install Task
brew install go-task/tap/go-task
```

**Issue: Docker buildx not found**
```bash
docker buildx create --name multiarch --use
docker buildx inspect --bootstrap
```

### Release Failures

**Issue: GitHub token expired**
```bash
# Generate new token at https://github.com/settings/tokens
export GITHUB_TOKEN=ghp_newtoken
```

**Issue: Tag already exists**
```bash
# Delete tag locally and remotely
git tag -d v1.0.0
git push origin :refs/tags/v1.0.0
```

**Issue: GPG signing fails**
```bash
# Check GPG key
gpg --list-secret-keys

# Set GPG_FINGERPRINT
export GPG_FINGERPRINT=your-key-fingerprint
```

## Configuration Files

### `.goreleaser.yml`
Main configuration for all platforms, packages, and release settings.

### `.github/workflows/build.yml`
CI workflow for multi-platform builds on every commit.

### `.github/workflows/release.yml`
CD workflow for automated releases on tags.

### `scripts/build-all.sh`
Local script for building all platforms.

## Environment Variables

### Build Variables
```bash
VERSION=v1.2.3        # Version to embed
GOOS=linux           # Target OS
GOARCH=amd64         # Target architecture
CGO_ENABLED=0        # Disable CGO for static builds
```

### Release Variables
```bash
GITHUB_TOKEN=ghp_xxx          # GitHub API token
GPG_FINGERPRINT=ABCD1234      # GPG key for signing
REGISTRY=ghcr.io              # Docker registry
```

## Platform-Specific Notes

### Linux
- Static binaries (no glibc dependency)
- Works on any Linux distribution
- systemd service files included in packages

### macOS
- Universal binaries not yet supported (separate amd64/arm64)
- Homebrew tap for easy installation
- Code signing not yet implemented

### Windows
- Requires Windows Server 2016+ or Windows 10+
- No service wrapper yet (planned)
- Must add to PATH manually

## Performance Tips

### Faster Local Builds
```bash
# Build only what you need
cd backend
go build -o ../build/takhin ./cmd/takhin

# Skip tests
go build -v ./cmd/takhin

# Use build cache
export GOCACHE=/tmp/go-cache
```

### Faster CI Builds
- Use GitHub Actions cache for Go modules
- Enable parallel builds in matrix
- Use self-hosted runners for better performance

## Support Platforms

| OS      | Architecture | Status | Package Format |
|---------|--------------|--------|----------------|
| Linux   | amd64        | ✅      | deb, rpm, apk  |
| Linux   | arm64        | ✅      | deb, rpm, apk  |
| macOS   | amd64        | ✅      | brew, tar.gz   |
| macOS   | arm64        | ✅      | brew, tar.gz   |
| Windows | amd64        | ✅      | zip            |

## Links

- GoReleaser Config: `.goreleaser.yml`
- Build Workflow: `.github/workflows/build.yml`
- Release Workflow: `.github/workflows/release.yml`
- Build Script: `scripts/build-all.sh`
- Full Documentation: `TASK_8.1_MULTI_PLATFORM_BUILD.md`
