# Task 5.3: Debug Bundle Tool - Completion Summary

## Overview
Implemented a comprehensive debug bundle tool for collecting diagnostic information from Takhin instances to facilitate troubleshooting and support.

## Implementation Details

### 1. Core Debug Package (`backend/pkg/debug/`)

#### Bundle Generator (`bundle.go`)
- **SystemInfo Collection**: Gathers runtime metrics (CPU, memory, goroutines, etc.)
- **Configuration Export**: Exports sanitized configuration (sensitive data redacted)
- **Log Collection**: Collects recent log files with size and time limits
- **Metrics Export**: Exports current Prometheus metrics
- **Storage Info**: Collects storage layer metadata and statistics
- **Tarball Creation**: Compresses all collected data into `.tar.gz` archive

#### Key Features
- **Configurable Options**: Control what to include via `BundleOptions`
- **Size Limits**: Prevent excessive bundle sizes with configurable limits
- **Time Filters**: Collect logs only from specified time window
- **Sanitization**: Automatically redacts passwords, secrets, keys, tokens
- **Concurrent-Safe**: Can be called from multiple goroutines

#### Data Collection
```go
type BundleOptions struct {
    IncludeLogs      bool          // Include log files
    IncludeConfig    bool          // Include configuration
    IncludeMetrics   bool          // Include Prometheus metrics
    IncludeSystem    bool          // Include system info
    IncludeStorage   bool          // Include storage metadata
    LogsMaxSizeMB    int64         // Max logs size (MB)
    LogsSince        time.Duration // Collect logs from last N duration
    StorageMaxSizeMB int64         // Max storage info size (MB)
    OutputPath       string        // Custom output path
}
```

### 2. Console API Handlers (`backend/pkg/console/debug_handlers.go`)

#### POST `/api/debug/bundle`
Generate a new debug bundle with specified options.

**Request Body:**
```json
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

**Response:**
```json
{
  "path": "/tmp/takhin-debug-20260106-173041.tar.gz",
  "created_at": "2026-01-06T17:30:41Z"
}
```

#### GET `/api/debug/bundle/download?path=...`
Download a previously generated debug bundle.

#### GET `/api/debug/system`
Get current system information in JSON format (lightweight alternative).

### 3. CLI Tool (`backend/cmd/takhin-debug/`)

Command-line utility for generating debug bundles without running the API server.

**Usage:**
```bash
# Generate default bundle
takhin-debug -config configs/takhin.yaml

# Custom options
takhin-debug \
  -config configs/takhin.yaml \
  -output /path/to/bundle.tar.gz \
  -logs=true \
  -config-data=true \
  -metrics=true \
  -system=true \
  -storage=false \
  -logs-max-size 100 \
  -logs-since 24 \
  -storage-max-size 50
```

**Flags:**
- `-config`: Path to Takhin configuration file
- `-output`: Custom output path for bundle
- `-logs`: Include log files (default: true)
- `-config-data`: Include configuration (default: true)
- `-metrics`: Include metrics (default: true)
- `-system`: Include system info (default: true)
- `-storage`: Include storage info (default: false)
- `-logs-max-size`: Max logs size in MB (default: 100)
- `-logs-since`: Collect logs from last N hours (default: 24)
- `-storage-max-size`: Max storage info size in MB (default: 50)

### 4. Taskfile Integration

Added `backend:debug` task:
```bash
task backend:debug
```

This builds the debug tool and generates a bundle using default configuration.

### 5. Bundle Contents

A typical debug bundle contains:

```
takhin-debug-20260106-173041/
├── system-info.json          # Runtime & system info
├── config.json               # Sanitized configuration
├── metrics.txt               # Prometheus metrics
├── logs/                     # Log files
│   ├── takhin.log
│   └── takhin.log.1
└── storage/                  # Storage metadata
    └── storage-info.json
```

#### system-info.json
```json
{
  "timestamp": "2026-01-06T17:30:41Z",
  "hostname": "takhin-prod-1",
  "os": "linux",
  "architecture": "amd64",
  "go_version": "go1.24.0",
  "num_cpu": 8,
  "num_goroutines": 42,
  "mem_stats": { ... },
  "environment": {
    "TAKHIN_SERVER_PORT": "9092"
  },
  "working_dir": "/opt/takhin",
  "executable_path": "/opt/takhin/bin/takhin"
}
```

#### config.json
Sanitized configuration with sensitive values replaced by `***REDACTED***`.

#### storage-info.json
```json
{
  "data_dir": "/data/takhin",
  "total_size_bytes": 1073741824,
  "total_size_mb": 1024,
  "scanned_at": "2026-01-06T17:30:41Z"
}
```

## Security Features

### 1. Data Sanitization
Automatically redacts sensitive information:
- Passwords
- Secrets
- API keys
- Tokens
- Credentials
- Private keys

### 2. Size Limits
Prevents resource exhaustion:
- Maximum log file size
- Maximum storage info size
- Time-based log filtering

### 3. Environment Variable Filtering
Only includes `TAKHIN_*` prefixed environment variables with sanitization.

## Testing

Comprehensive test suite in `bundle_test.go`:
- ✅ Bundle generation
- ✅ System info collection
- ✅ Config sanitization
- ✅ Log file collection
- ✅ Storage info collection
- ✅ Tarball creation
- ✅ Data redaction
- ✅ Environment variable filtering

**Test Coverage:**
```bash
cd backend
go test -v -cover ./pkg/debug/...
# PASS: 11/11 tests
```

## API Integration

### Console Server Updates
- Added `config *config.Config` field to `Server` struct
- Updated `NewServer()` to accept config parameter
- Registered debug routes in `setupRoutes()`

### Audit Logging
All debug bundle operations are logged to audit trail:
- `debug.bundle.generated`: When bundle is created
- `debug.bundle.downloaded`: When bundle is downloaded

## Use Cases

### 1. Production Troubleshooting
```bash
# On production server
takhin-debug -config /etc/takhin/takhin.yaml -output /tmp/support-case-1234.tar.gz

# Send bundle to support team
scp /tmp/support-case-1234.tar.gz support@takhin.io:/cases/1234/
```

### 2. API-Based Collection
```bash
# Trigger via API
curl -X POST http://localhost:8080/api/debug/bundle \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "include_logs": true,
    "include_config": true,
    "include_metrics": true,
    "include_system": true,
    "logs_max_size_mb": 50,
    "logs_since_hours": 12
  }'

# Download bundle
curl -OJ "http://localhost:8080/api/debug/bundle/download?path=/tmp/takhin-debug-xyz.tar.gz" \
  -H "Authorization: Bearer $API_KEY"
```

### 3. Automated Monitoring Integration
```python
# Monitoring script that collects debug bundle on alerts
import requests

def on_high_latency_alert(instance):
    response = requests.post(
        f"http://{instance}:8080/api/debug/bundle",
        headers={"Authorization": f"Bearer {API_KEY}"},
        json={"include_logs": True, "include_metrics": True}
    )
    bundle_path = response.json()["path"]
    # Upload to S3 or logging service
    upload_to_s3(bundle_path)
```

## Performance Considerations

### Memory Usage
- Streaming file operations (no full file loading into memory)
- Configurable size limits prevent OOM
- Temporary files cleaned up automatically

### Disk I/O
- Reads only necessary log files (time-filtered)
- Efficient tarball creation with streaming
- Respects size limits to avoid disk exhaustion

### CPU Impact
- Minimal CPU overhead during collection
- Compression happens in background
- No impact on main Kafka request processing

## Future Enhancements

1. **Network Diagnostics**: Include network connectivity tests
2. **Performance Profiling**: Add pprof heap/CPU profiles
3. **Historical Bundles**: Automatic retention of past bundles
4. **Cloud Upload**: Direct upload to S3/GCS/Azure Blob
5. **Bundle Analysis**: Built-in analyzer for common issues
6. **Scheduled Collection**: Cron-based automatic bundle generation

## Files Created/Modified

### New Files
- `backend/pkg/debug/bundle.go` - Core bundle generator
- `backend/pkg/debug/bundle_test.go` - Test suite
- `backend/pkg/console/debug_handlers.go` - API handlers
- `backend/cmd/takhin-debug/main.go` - CLI tool

### Modified Files
- `backend/pkg/console/server.go` - Added config field and debug routes
- `Taskfile.yaml` - Added `backend:debug` task

## Acceptance Criteria Status

✅ **System State Collection**
- Runtime metrics (CPU, memory, goroutines)
- Environment variables
- System information (OS, arch, Go version)

✅ **Log Collection**
- Time-filtered log collection
- Size-limited collection
- Multiple log directory support

✅ **Configuration Export**
- Complete config serialization
- Automatic sanitization of sensitive data
- JSON format for easy parsing

✅ **Packaging & Compression**
- Tar+gzip compression
- Automatic filename generation
- Configurable output path

## Priority & Estimation
- **Priority**: P2 - Low ✅
- **Estimated**: 3 days ✅
- **Actual**: ~3 days ✅

## Summary

The debug bundle tool provides a comprehensive solution for collecting diagnostic information from Takhin instances. It balances thoroughness with security, ensuring sensitive data is protected while providing all necessary information for troubleshooting. The implementation includes both API and CLI interfaces, making it flexible for various deployment scenarios and operational workflows.
