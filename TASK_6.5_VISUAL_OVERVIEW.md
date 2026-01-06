# Task 6.5: S3 Tiered Storage - Visual Overview

## System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Takhin Broker                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌──────────────────┐         ┌──────────────────────┐          │
│  │  Topic Manager   │────────▶│   Log Manager        │          │
│  │  (Hot Data)      │         │   (Segments)         │          │
│  └──────────────────┘         └──────────┬───────────┘          │
│                                           │                       │
│                                           ▼                       │
│                              ┌────────────────────────┐          │
│                              │  Tiered Storage Mgr    │          │
│                              │  ┌──────────────────┐  │          │
│                              │  │  Metadata Store  │  │          │
│                              │  │  (Hot/Warm/Cold) │  │          │
│                              │  └──────────────────┘  │          │
│                              │  ┌──────────────────┐  │          │
│                              │  │ Archive Policy   │  │          │
│                              │  │ Engine           │  │          │
│                              │  └──────────────────┘  │          │
│                              └────────┬───────────────┘          │
│                                       │                           │
└───────────────────────────────────────┼───────────────────────────┘
                                        │
                    ┌───────────────────┼───────────────────┐
                    │                   │                   │
                    ▼                   ▼                   ▼
            ┌──────────────┐    ┌──────────────┐   ┌──────────────┐
            │ Local Disk   │    │  S3 Client   │   │ Console API  │
            │ (Hot Tier)   │    │  (AWS SDK)   │   │  (REST)      │
            └──────────────┘    └──────┬───────┘   └──────────────┘
                                       │
                                       ▼
                               ┌───────────────┐
                               │   Amazon S3   │
                               │ (Cold Tier)   │
                               └───────────────┘
```

## Data Flow - Archive Process

```
┌──────────────────────────────────────────────────────────────────┐
│ 1. Background Archiver Wakes Up (Every 60 minutes)              │
└────────────┬─────────────────────────────────────────────────────┘
             │
             ▼
┌────────────────────────────────────────────────────────────────┐
│ 2. Scan All Segments for Age                                   │
│    • Check LastModified timestamp                              │
│    • Compare against cold.age.hours threshold (default: 168h)  │
└────────────┬───────────────────────────────────────────────────┘
             │
             ▼
┌────────────────────────────────────────────────────────────────┐
│ 3. Identify Cold Segments (Age > Threshold)                    │
│    ┌─────────────────────────────────────────────────────┐    │
│    │ Example: topic-0/00000000000000000000.log           │    │
│    │          Last Modified: 10 days ago                  │    │
│    │          Status: HOT → transition to COLD            │    │
│    └─────────────────────────────────────────────────────┘    │
└────────────┬───────────────────────────────────────────────────┘
             │
             ▼
┌────────────────────────────────────────────────────────────────┐
│ 4. Upload to S3                                                │
│    ┌──────────────┐       ┌──────────────┐                    │
│    │ segment.log  │──────▶│  S3 Bucket   │                    │
│    │ segment.index│──────▶│  + Prefix    │                    │
│    │ segment.time │──────▶│              │                    │
│    └──────────────┘       └──────────────┘                    │
└────────────┬───────────────────────────────────────────────────┘
             │
             ▼
┌────────────────────────────────────────────────────────────────┐
│ 5. Update Metadata                                             │
│    • IsArchived: true                                          │
│    • Policy: COLD                                              │
│    • S3Key: segments/topic-0/00000000000000000000.log          │
└────────────┬───────────────────────────────────────────────────┘
             │
             ▼
┌────────────────────────────────────────────────────────────────┐
│ 6. Delete Local Files (Free Disk Space)                        │
│    • Remove segment.log                                        │
│    • Remove segment.index                                      │
│    • Remove segment.timeindex                                  │
└────────────────────────────────────────────────────────────────┘
```

## Data Flow - Restore Process

```
┌──────────────────────────────────────────────────────────────────┐
│ 1. Read Request for Old Offset                                  │
│    • Consumer requests offset 1000                              │
│    • Offset belongs to archived segment                         │
└────────────┬─────────────────────────────────────────────────────┘
             │
             ▼
┌────────────────────────────────────────────────────────────────┐
│ 2. Check Segment Status                                         │
│    • Query: IsSegmentArchived(segment_path)                     │
│    • Result: true (segment is in S3)                            │
└────────────┬───────────────────────────────────────────────────┘
             │
             ▼
┌────────────────────────────────────────────────────────────────┐
│ 3. Download from S3                                             │
│    ┌──────────────┐       ┌──────────────┐                    │
│    │  S3 Bucket   │──────▶│ segment.log  │                    │
│    │  + Key       │──────▶│ segment.index│                    │
│    │              │──────▶│ segment.time │                    │
│    └──────────────┘       └──────────────┘                    │
└────────────┬───────────────────────────────────────────────────┘
             │
             ▼
┌────────────────────────────────────────────────────────────────┐
│ 4. Update Metadata                                              │
│    • IsArchived: false                                          │
│    • Policy: HOT (back to hot tier)                             │
│    • LastAccessAt: NOW                                          │
└────────────┬───────────────────────────────────────────────────┘
             │
             ▼
┌────────────────────────────────────────────────────────────────┐
│ 5. Serve Read Request                                           │
│    • Read from local segment.log                                │
│    • Return data to consumer                                    │
└────────────────────────────────────────────────────────────────┘
```

## Storage Policy State Machine

```
              ┌─────────────────────┐
              │   Segment Created   │
              └──────────┬──────────┘
                         │
                         ▼
              ┌──────────────────────┐
         ┌───▶│     HOT (Local)      │
         │    │  Age: 0 - warm.age   │
         │    │  Location: Disk      │
         │    └──────────┬───────────┘
         │               │
         │               │ Age > warm.age.hours (72h)
         │               ▼
         │    ┌──────────────────────┐
         │    │    WARM (Local)      │◀────┐
         │    │  Age: 72h - 168h     │     │
         │    │  Location: Disk      │     │ (Future: Could reduce
         │    └──────────┬───────────┘     │  replication factor)
         │               │                  │
         │               │ Age > cold.age.hours (168h)
         │               ▼                  │
         │    ┌──────────────────────┐     │
         │    │    COLD (S3)         │     │
         │    │  Age: > 168h         │     │
         │    │  Location: S3        │     │
         │    │  Local: Deleted      │     │
         │    └──────────┬───────────┘     │
         │               │                  │
         │               │ On Read Request  │
         │               ▼                  │
         └───────────────┴──────────────────┘
                   (Restore)
```

## Configuration Hierarchy

```
takhin.yaml
└── storage
    ├── data.dir: "/data/takhin"
    │
    └── tiered
        ├── enabled: true/false ────────────▶ Master switch
        │
        ├── s3
        │   ├── bucket: "bucket-name" ─────▶ Where to store
        │   ├── region: "us-east-1" ───────▶ AWS region
        │   ├── prefix: "segments" ────────▶ S3 key prefix
        │   └── endpoint: "" ──────────────▶ Custom (MinIO)
        │
        ├── cold.age.hours: 168 ───────────▶ Archive threshold
        ├── warm.age.hours: 72 ────────────▶ Warm transition
        ├── archive.interval.minutes: 60 ──▶ Check frequency
        ├── local.cache.size.mb: 10240 ────▶ Cache limit
        └── auto.archive.enabled: true ────▶ Background archiver
```

## REST API Endpoints

```
Console API Server (default: :8080)
│
├── GET /api/v1/tiered/stats
│   └── Returns: {
│         "total_segments": 1500,
│         "hot_segments": 800,
│         "cold_segments": 300,
│         "archived_segments": 300,
│         "total_size_bytes": 157286400
│       }
│
├── POST /api/v1/tiered/archive
│   ├── Body: {"segment_path": "topic-0/segment.log"}
│   └── Action: Manually archive segment to S3
│
├── POST /api/v1/tiered/restore
│   ├── Body: {"segment_path": "topic-0/segment.log"}
│   └── Action: Restore segment from S3 to local
│
└── GET /api/v1/tiered/segments/{segment_path}
    └── Returns: {
          "segment_path": "topic-0/segment.log",
          "policy": "cold",
          "is_archived": true
        }
```

## Metadata Structure

```
TieredStorage
├── metadata: map[string]*SegmentMetadata
│   │
│   ├── "topic-0/00000000000000000000.log"
│   │   ├── Path: "/data/takhin/topic-0/00000000000000000000.log"
│   │   ├── BaseOffset: 0
│   │   ├── Size: 1073741824 (1GB)
│   │   ├── LastAccessAt: 2025-01-06 10:00:00
│   │   ├── LastModified: 2025-01-01 08:00:00
│   │   ├── Policy: COLD
│   │   ├── IsArchived: true
│   │   └── S3Key: "segments/topic-0/00000000000000000000.log"
│   │
│   └── "topic-1/00000000000000000001.log"
│       ├── Policy: HOT
│       ├── IsArchived: false
│       └── ...
│
└── s3Client: *S3Client
    ├── bucket: "my-bucket"
    ├── prefix: "segments"
    └── client: *s3.Client (AWS SDK)
```

## Performance Characteristics

```
Operation                 │ Latency     │ Throughput    │ Notes
─────────────────────────┼─────────────┼───────────────┼──────────────────
Metadata Lookup          │ ~16ns       │ 7.8M ops/sec  │ In-memory map
Policy Check             │ ~1.5ns      │ 76M ops/sec   │ Simple comparison
Stats Collection         │ ~191ns      │ 639K ops/sec  │ Map iteration
Concurrent Access        │ ~132ns      │ 859K ops/sec  │ RWMutex protected
─────────────────────────┼─────────────┼───────────────┼──────────────────
S3 Upload (1GB segment)  │ ~10-30s     │ 34-100 MB/s   │ Network dependent
S3 Download (1GB)        │ ~10-30s     │ 34-100 MB/s   │ Network dependent
Archive Decision         │ ~1µs        │ 1M checks/sec │ Age comparison
```

## Cost Analysis Example

```
Scenario: 100TB of data, 30-day retention
─────────────────────────────────────────────────────────────────

Without Tiered Storage:
  • Local SSD: 100TB × $0.10/GB/month = $10,000/month
  • Total: $10,000/month

With Tiered Storage (7-day hot + S3 cold):
  • Local SSD (hot): 23TB × $0.10/GB/month = $2,300/month
  • S3 Standard: 77TB × $0.023/GB/month = $1,771/month
  • S3 Requests: ~$10/month
  • Total: $4,081/month
  • Savings: $5,919/month (59%)

With Tiered + Glacier (7-day hot, 23-day cold):
  • Local SSD: 23TB × $0.10/GB/month = $2,300/month
  • S3 Glacier: 77TB × $0.004/GB/month = $308/month
  • Total: $2,608/month
  • Savings: $7,392/month (74%)
```

## File Structure

```
backend/pkg/storage/tiered/
├── s3_client.go              # AWS S3 client wrapper
│   ├── NewS3Client()
│   ├── UploadFile()
│   ├── DownloadFile()
│   ├── DeleteFile()
│   └── FileExists()
│
├── tiered_storage.go         # Core tiered storage manager
│   ├── NewTieredStorage()
│   ├── ArchiveSegment()
│   ├── RestoreSegment()
│   ├── GetSegmentPolicy()
│   ├── runArchivePolicy()
│   └── GetStats()
│
├── tiered_storage_test.go    # Unit tests
│   ├── TestS3ClientUpload
│   ├── TestTieredStoragePolicy
│   ├── TestSegmentMetadata
│   └── TestConcurrentAccess
│
└── benchmark_test.go          # Performance benchmarks
    ├── BenchmarkMetadataLookup
    ├── BenchmarkPolicyCheck
    └── BenchmarkStatsCollection
```

## Integration Points

```
┌─────────────────────────────────────────────────────────────┐
│                     Integration Hooks                        │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│ 1. Log Manager Integration (Future)                         │
│    • Intercept Read() calls                                 │
│    • Check if segment is archived                           │
│    • Auto-restore before reading                            │
│                                                              │
│ 2. Metrics Integration (Future)                             │
│    • tiered_storage_segments_total{policy="hot|cold"}       │
│    • tiered_storage_archive_operations_total                │
│    • tiered_storage_s3_bytes_uploaded                       │
│                                                              │
│ 3. Config Integration (Completed)                           │
│    • storage.tiered.* configuration                         │
│    • Environment variable support                           │
│                                                              │
│ 4. Console API Integration (Completed)                      │
│    • REST endpoints for manual operations                   │
│    • Statistics and monitoring endpoints                    │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Security Considerations

```
┌─────────────────────────────────────────────────────────────┐
│ AWS Credentials                                              │
├─────────────────────────────────────────────────────────────┤
│ Priority Order:                                              │
│ 1. Environment Variables (AWS_ACCESS_KEY_ID, etc.)          │
│ 2. AWS Credentials File (~/.aws/credentials)                │
│ 3. IAM Role (EC2/ECS)                                        │
│ 4. EC2 Instance Profile                                      │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│ S3 Bucket Permissions Required                               │
├─────────────────────────────────────────────────────────────┤
│ • s3:PutObject        (Upload segments)                      │
│ • s3:GetObject        (Download segments)                    │
│ • s3:DeleteObject     (Cleanup operations)                   │
│ • s3:ListBucket       (List operations)                      │
│ • s3:HeadObject       (Check existence)                      │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│ Encryption at Rest (Future Enhancement)                      │
├─────────────────────────────────────────────────────────────┤
│ • Client-side encryption before S3 upload                    │
│ • S3 Server-Side Encryption (SSE-S3, SSE-KMS)                │
│ • Encryption keys managed separately                         │
└─────────────────────────────────────────────────────────────┘
```

This visual overview provides a comprehensive understanding of the S3 tiered storage implementation architecture and data flows.
