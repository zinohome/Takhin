# Task 6.5 - S3 Tiered Storage: Quick Reference

## 🎯 Quick Start

### Enable Tiered Storage
```yaml
# configs/takhin.yaml
storage:
  tiered:
    enabled: true
    s3:
      bucket: "my-takhin-bucket"
      region: "us-east-1"
    cold:
      age:
        hours: 168  # Archive after 7 days
```

### Environment Variables
```bash
export TAKHIN_STORAGE_TIERED_ENABLED=true
export TAKHIN_STORAGE_TIERED_S3_BUCKET=my-bucket
export TAKHIN_STORAGE_TIERED_S3_REGION=us-west-2
```

---

## 📦 Core Components

### S3 Client
```go
import "github.com/takhin-data/takhin/pkg/storage/tiered"

client, _ := tiered.NewS3Client(ctx, tiered.S3Config{
    Region:   "us-east-1",
    Bucket:   "my-bucket",
    Prefix:   "segments",
    Endpoint: "", // Empty for AWS, or MinIO URL
})

// Upload
client.UploadFile(ctx, "/local/path/segment.log", "topic-0/segment.log")

// Download
client.DownloadFile(ctx, "topic-0/segment.log", "/local/path/segment.log")

// Check existence
exists, _ := client.FileExists(ctx, "topic-0/segment.log")
```

### Tiered Storage Manager
```go
ts, _ := tiered.NewTieredStorage(ctx, tiered.TieredStorageConfig{
    DataDir:            "/data",
    S3Config:           s3Config,
    ColdAgeThreshold:   168 * time.Hour,
    AutoArchiveEnabled: true,
})
defer ts.Close()

// Archive
ts.ArchiveSegment(ctx, "topic-0/00000000000000000000.log")

// Restore
ts.RestoreSegment(ctx, "topic-0/00000000000000000000.log")

// Check status
policy := ts.GetSegmentPolicy("topic-0/00000000000000000000.log")
archived := ts.IsSegmentArchived("topic-0/00000000000000000000.log")

// Statistics
stats := ts.GetStats()
// Returns: total_segments, hot_segments, cold_segments, archived_segments, total_size_bytes
```

---

## 🌐 REST API

### Get Statistics
```bash
GET /api/v1/tiered/stats
```
```json
{
  "total_segments": 1500,
  "hot_segments": 800,
  "cold_segments": 300,
  "archived_segments": 300,
  "total_size_bytes": 157286400
}
```

### Archive Segment
```bash
POST /api/v1/tiered/archive
Content-Type: application/json

{
  "segment_path": "topic-0/00000000000000000000.log"
}
```

### Restore Segment
```bash
POST /api/v1/tiered/restore
Content-Type: application/json

{
  "segment_path": "topic-0/00000000000000000000.log"
}
```

### Check Segment Status
```bash
GET /api/v1/tiered/segments/{segment_path}
```
```json
{
  "segment_path": "topic-0/00000000000000000000.log",
  "policy": "cold",
  "is_archived": true
}
```

---

## ⚙️ Configuration Reference

### Full Configuration
```yaml
storage:
  tiered:
    enabled: false                       # Enable tiered storage
    s3:
      bucket: ""                         # S3 bucket name (REQUIRED)
      region: "us-east-1"                # AWS region
      prefix: "takhin-segments"          # S3 key prefix
      endpoint: ""                       # Custom endpoint (for MinIO)
    cold:
      age:
        hours: 168                       # Archive segments older than 7 days
    warm:
      age:
        hours: 72                        # Mark as warm after 3 days (placeholder)
    archive:
      interval:
        minutes: 60                      # Run archive policy every hour
    local:
      cache:
        size:
          mb: 10240                      # 10GB local cache limit
    auto:
      archive:
        enabled: true                    # Enable automatic archiving
```

### Environment Variable Mapping
| YAML Path | Environment Variable | Default |
|-----------|---------------------|---------|
| `storage.tiered.enabled` | `TAKHIN_STORAGE_TIERED_ENABLED` | `false` |
| `storage.tiered.s3.bucket` | `TAKHIN_STORAGE_TIERED_S3_BUCKET` | `""` |
| `storage.tiered.s3.region` | `TAKHIN_STORAGE_TIERED_S3_REGION` | `"us-east-1"` |
| `storage.tiered.s3.prefix` | `TAKHIN_STORAGE_TIERED_S3_PREFIX` | `"takhin-segments"` |
| `storage.tiered.s3.endpoint` | `TAKHIN_STORAGE_TIERED_S3_ENDPOINT` | `""` |
| `storage.tiered.cold.age.hours` | `TAKHIN_STORAGE_TIERED_COLD_AGE_HOURS` | `168` |
| `storage.tiered.archive.interval.minutes` | `TAKHIN_STORAGE_TIERED_ARCHIVE_INTERVAL_MINUTES` | `60` |

---

## 🏗️ Storage Policies

### Hot
- Recently modified segments
- Kept locally on disk
- Fastest access
- Default policy for new segments

### Warm
- Placeholder for future implementation
- Could use reduced replication
- Still local but with different characteristics

### Cold
- Old segments (> cold.age.hours)
- Archived to S3
- Restored on-demand
- Local copy deleted after successful upload

---

## 🔍 Common Operations

### Manual Archive Old Segments
```bash
# Using REST API
curl -X POST http://localhost:8080/api/v1/tiered/archive \
  -H "Content-Type: application/json" \
  -d '{"segment_path": "my-topic-0/00000000000000000000.log"}'
```

### Check Archive Status
```bash
curl http://localhost:8080/api/v1/tiered/segments/my-topic-0%2F00000000000000000000.log | jq
```

### View Statistics
```bash
curl http://localhost:8080/api/v1/tiered/stats | jq
```

### Restore for Reading
```bash
curl -X POST http://localhost:8080/api/v1/tiered/restore \
  -H "Content-Type: application/json" \
  -d '{"segment_path": "my-topic-0/00000000000000000000.log"}'
```

---

## 🧪 Testing

### Run Tests
```bash
cd backend
go test ./pkg/storage/tiered/... -v
```

### Run Benchmarks
```bash
go test ./pkg/storage/tiered -bench=. -benchmem
```

### Test with MinIO (Local S3)
```bash
# Start MinIO
docker run -p 9000:9000 -p 9001:9001 \
  -e MINIO_ROOT_USER=minioadmin \
  -e MINIO_ROOT_PASSWORD=minioadmin \
  quay.io/minio/minio server /data --console-address ":9001"

# Configure Takhin
storage:
  tiered:
    enabled: true
    s3:
      bucket: "takhin-test"
      region: "us-east-1"
      endpoint: "http://localhost:9000"

# Set AWS credentials
export AWS_ACCESS_KEY_ID=minioadmin
export AWS_SECRET_ACCESS_KEY=minioadmin
```

---

## 📊 Monitoring

### Recommended Metrics (Future)
```promql
# Total segments by policy
tiered_storage_segments_total{policy="hot|warm|cold"}

# Archived segments
tiered_storage_archived_segments_total

# Operations
tiered_storage_archive_operations_total{result="success|failure"}
tiered_storage_restore_operations_total{result="success|failure"}

# Data transfer
tiered_storage_s3_bytes_uploaded
tiered_storage_s3_bytes_downloaded
```

### Current Statistics API
```bash
curl http://localhost:8080/api/v1/tiered/stats
```

---

## 🚨 Troubleshooting

### Archive Fails
**Symptom**: Archive operation returns error
**Check**:
1. S3 bucket exists and is accessible
2. AWS credentials are configured
3. Network connectivity to S3
4. Segment file exists locally

### Restore Fails
**Symptom**: Cannot restore archived segment
**Check**:
1. Segment was successfully archived
2. S3 object exists
3. Local disk has space
4. Permissions on data directory

### High S3 Costs
**Issue**: Excessive S3 requests
**Solutions**:
1. Increase `cold.age.hours` threshold
2. Reduce `archive.interval.minutes`
3. Implement local cache (future)
4. Use S3 Lifecycle policies

---

## 🔗 Related Files

- **Implementation**: `backend/pkg/storage/tiered/`
- **Configuration**: `backend/pkg/config/config.go`
- **API Handlers**: `backend/pkg/console/tiered_handlers.go`
- **Config File**: `backend/configs/takhin.yaml`
- **Tests**: `backend/pkg/storage/tiered/*_test.go`

---

## 📚 See Also

- [Full Completion Summary](TASK_6.5_S3_COMPLETION.md)
- [Storage Architecture](docs/architecture/)
- [Configuration Guide](configs/takhin.yaml)
- [AWS S3 Documentation](https://docs.aws.amazon.com/s3/)
