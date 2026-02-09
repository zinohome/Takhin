# Debug Bundle Tool - Quick Reference

## Quick Start

### CLI Tool
```bash
# Build
task backend:build

# Generate bundle with defaults
./build/takhin-debug -config configs/takhin.yaml

# Custom bundle
./build/takhin-debug \
  -config configs/takhin.yaml \
  -output /tmp/my-bundle.tar.gz \
  -logs-max-size 50 \
  -logs-since 12
```

### API Endpoints

#### Generate Bundle
```bash
POST /api/debug/bundle
Content-Type: application/json
Authorization: Bearer <api-key>

{
  "include_logs": true,
  "include_config": true,
  "include_metrics": true,
  "include_system": true,
  "include_storage": false,
  "logs_max_size_mb": 100,
  "logs_since_hours": 24,
  "storage_max_size_mb": 50
}
```

#### Download Bundle
```bash
GET /api/debug/bundle/download?path=/tmp/takhin-debug-xyz.tar.gz
Authorization: Bearer <api-key>
```

#### Get System Info Only
```bash
GET /api/debug/system
Authorization: Bearer <api-key>
```

## Bundle Contents

```
takhin-debug-<timestamp>.tar.gz
├── system-info.json      # Runtime & system metrics
├── config.json           # Sanitized configuration
├── metrics.txt           # Prometheus metrics snapshot
├── logs/                 # Recent log files
│   └── *.log
└── storage/              # Storage layer metadata
    └── storage-info.json
```

## Configuration Options

### CLI Flags
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-config` | string | `configs/takhin.yaml` | Path to config file |
| `-output` | string | auto | Output path for bundle |
| `-logs` | bool | `true` | Include log files |
| `-config-data` | bool | `true` | Include configuration |
| `-metrics` | bool | `true` | Include metrics |
| `-system` | bool | `true` | Include system info |
| `-storage` | bool | `false` | Include storage info |
| `-logs-max-size` | int | `100` | Max logs size (MB) |
| `-logs-since` | int | `24` | Hours of logs to collect |
| `-storage-max-size` | int | `50` | Max storage info (MB) |

### API Request Options
```json
{
  "include_logs": true,       // Collect log files
  "include_config": true,     // Export configuration
  "include_metrics": true,    // Export metrics
  "include_system": true,     // Collect system info
  "include_storage": false,   // Collect storage metadata
  "logs_max_size_mb": 100,    // Max total log size
  "logs_since_hours": 24,     // Hours of logs
  "storage_max_size_mb": 50   // Max storage info size
}
```

## Security Features

### Automatic Redaction
The following are automatically redacted:
- Passwords → `***REDACTED***`
- Secrets → `***REDACTED***`
- API Keys → `***REDACTED***`
- Tokens → `***REDACTED***`
- Credentials → `***REDACTED***`

### Safe Environment Variables
Only `TAKHIN_*` prefixed variables are included, and sensitive values are sanitized.

## Common Use Cases

### 1. Production Issue Report
```bash
# Generate comprehensive bundle
takhin-debug -config /etc/takhin/takhin.yaml \
  -output /tmp/issue-report.tar.gz \
  -logs-max-size 200 \
  -logs-since 48
```

### 2. Performance Investigation
```bash
# Focus on recent metrics and system state
curl -X POST http://localhost:8080/api/debug/bundle \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "include_logs": false,
    "include_config": false,
    "include_metrics": true,
    "include_system": true
  }'
```

### 3. Configuration Verification
```bash
# Export only sanitized config
curl http://localhost:8080/api/debug/system \
  -H "Authorization: Bearer $API_KEY" | jq .
```

## Taskfile Commands

```bash
# Build debug tool
task backend:build

# Generate debug bundle
task backend:debug

# Run tests
go test -v ./pkg/debug/...
```

## Programmatic Usage

```go
import (
    "context"
    "time"
    
    "github.com/takhin-data/takhin/pkg/config"
    "github.com/takhin-data/takhin/pkg/debug"
    "github.com/takhin-data/takhin/pkg/logger"
)

// Create bundle generator
cfg := &config.Config{...}
log := logger.Default()
bundle := debug.NewBundle(cfg, log)

// Generate bundle
opts := &debug.BundleOptions{
    IncludeLogs:      true,
    IncludeConfig:    true,
    IncludeMetrics:   true,
    IncludeSystem:    true,
    LogsMaxSizeMB:    100,
    LogsSince:        24 * time.Hour,
}

path, err := bundle.Generate(context.Background(), opts)
if err != nil {
    log.Error("failed to generate bundle", "error", err)
}
```

## Troubleshooting

### Bundle Too Large
```bash
# Reduce log collection
takhin-debug -logs-max-size 50 -logs-since 12

# Skip storage info
takhin-debug -storage=false
```

### Missing Logs
Logs are collected from these directories (in order):
1. `./logs` (relative to working directory)
2. `/var/log/takhin`
3. `<data-dir>/../logs`

Ensure logs are in one of these locations or adjust log collection logic.

### Permission Denied
Ensure the process has read access to:
- Configuration file
- Log directories
- Storage data directory
- Output directory

## Output Format

### system-info.json
```json
{
  "timestamp": "2026-01-06T17:30:41Z",
  "hostname": "takhin-prod-1",
  "os": "linux",
  "architecture": "amd64",
  "go_version": "go1.24.0",
  "num_cpu": 8,
  "num_goroutines": 42,
  "mem_stats": {
    "alloc": 5242880,
    "total_alloc": 10485760,
    "sys": 73400320,
    "num_gc": 12
  }
}
```

### storage-info.json
```json
{
  "data_dir": "/data/takhin",
  "total_size_bytes": 1073741824,
  "total_size_mb": 1024,
  "scanned_at": "2026-01-06T17:30:41Z"
}
```

## File Locations

### Source Code
- Core: `backend/pkg/debug/bundle.go`
- Tests: `backend/pkg/debug/bundle_test.go`
- API: `backend/pkg/console/debug_handlers.go`
- CLI: `backend/cmd/takhin-debug/main.go`

### Documentation
- Summary: `TASK_5.3_DEBUG_BUNDLE_COMPLETION.md`
- Quick Ref: `TASK_5.3_DEBUG_BUNDLE_QUICK_REFERENCE.md` (this file)

## Testing

```bash
# Run all tests
go test -v ./pkg/debug/...

# Run with coverage
go test -v -cover ./pkg/debug/...

# Run specific test
go test -v -run TestBundleGenerate ./pkg/debug/...
```

## Best Practices

1. **Regular Collection**: Schedule periodic bundle generation for baseline
2. **Size Management**: Use size limits to prevent disk exhaustion
3. **Secure Transfer**: Encrypt bundles when sending over network
4. **Retention Policy**: Delete old bundles to save disk space
5. **Access Control**: Restrict bundle generation to authorized users only

## Support

For issues or questions:
1. Check logs in `/var/log/takhin/`
2. Verify configuration with `GET /api/debug/system`
3. Test with minimal options first
4. Contact support with generated bundle
