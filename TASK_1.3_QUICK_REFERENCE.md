# Task 1.3 Snapshot Support - Quick Reference

## Quick Start

```go
import "github.com/takhin-data/takhin/pkg/storage/log"

// Create snapshot manager
sm, _ := log.NewSnapshotManager("/data/log-dir")

// Create snapshot
snapshot, _ := sm.CreateSnapshot(logInstance)

// Restore snapshot
sm.RestoreSnapshot(snapshot.ID, "/data/restore-dir")

// List snapshots
snapshots := sm.ListSnapshots() // sorted by time, newest first

// Cleanup old snapshots
config := log.SnapshotConfig{
    MaxSnapshots:  5,
    RetentionTime: 24 * time.Hour,
}
deleted, _ := sm.CleanupSnapshots(config)
```

## Key Functions

| Function | Purpose | Returns |
|----------|---------|---------|
| `NewSnapshotManager(dir)` | Create manager | `*SnapshotManager, error` |
| `CreateSnapshot(log)` | Create new snapshot | `*Snapshot, error` |
| `RestoreSnapshot(id, dir)` | Restore to directory | `error` |
| `ListSnapshots()` | Get all snapshots | `[]*Snapshot` |
| `GetSnapshot(id)` | Get specific snapshot | `*Snapshot` |
| `DeleteSnapshot(id)` | Delete snapshot | `error` |
| `CleanupSnapshots(config)` | Apply retention policy | `int, error` |
| `Size()` | Total snapshot storage | `int64, error` |

## Snapshot Metadata

```go
type Snapshot struct {
    ID             string    // e.g., "snapshot-1735998123456789000"
    Timestamp      time.Time // Creation time
    BaseOffset     int64     // First offset
    HighWaterMark  int64     // Last offset + 1
    NumSegments    int       // Number of segments
    TotalSize      int64     // Total bytes
    SegmentOffsets []int64   // Base offsets
}
```

## Configuration

```go
type SnapshotConfig struct {
    MaxSnapshots   int           // Keep N most recent (default: 5)
    RetentionTime  time.Duration // Delete older than (default: 24h)
    MinInterval    time.Duration // Min time between (default: 1h)
}

// Get defaults
config := log.DefaultSnapshotConfig()
```

## Directory Structure

```
log-dir/
├── .snapshots/
│   ├── snapshot-{timestamp-1}/
│   │   ├── {offset}.log
│   │   ├── {offset}.index
│   │   └── {offset}.timeindex
│   ├── snapshot-{timestamp-2}/
│   │   └── ...
│   └── snapshots.json          # Metadata
├── {offset}.log
├── {offset}.index
└── {offset}.timeindex
```

## Common Patterns

### Periodic Snapshots
```go
ticker := time.NewTicker(6 * time.Hour)
for range ticker.C {
    if snapshot, err := sm.CreateSnapshot(log); err == nil {
        fmt.Printf("Created snapshot: %s\n", snapshot.ID)
    }
    sm.CleanupSnapshots(config)
}
```

### Disaster Recovery
```go
// Restore to temp location
tempDir := "/tmp/recovered-log"
err := sm.RestoreSnapshot(latestSnapshotID, tempDir)

// Open recovered log
recovered, err := log.NewLog(log.LogConfig{
    Dir:            tempDir,
    MaxSegmentSize: 1024 * 1024,
})
```

### Health Check
```go
snapshots := sm.ListSnapshots()
if len(snapshots) == 0 {
    alert("No snapshots available!")
}

totalSize, _ := sm.Size()
if totalSize > maxAllowedSize {
    sm.CleanupSnapshots(config)
}
```

## Testing

```bash
# Run all snapshot tests
go test ./pkg/storage/log -run TestSnapshot -v

# Run specific test
go test ./pkg/storage/log -run TestSnapshotManager_CreateSnapshot -v

# Run with race detector
go test ./pkg/storage/log -run TestSnapshot -race

# Run example
cd backend/pkg/storage/log/example
go run snapshot_example.go
```

## Files

- Implementation: `backend/pkg/storage/log/snapshot.go`
- Tests: `backend/pkg/storage/log/snapshot_test.go`
- Example: `backend/pkg/storage/log/example/snapshot_example.go`
- Documentation: `TASK_1.3_SNAPSHOT_COMPLETION.md`
- Summary: `TASK_1.3_COMPLETION_SUMMARY.md`

## Acceptance Criteria ✅

- ✅ Snapshot creation with metadata
- ✅ Fast snapshot restoration
- ✅ Configurable cleanup policies
- ✅ Comprehensive test coverage (18 tests)
- ✅ Complete documentation

## Status

**Complete** | Priority: P1 - Medium | Time: ~4 hours
