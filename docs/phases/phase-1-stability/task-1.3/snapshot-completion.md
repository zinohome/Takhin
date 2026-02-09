# Snapshot Support - Task 1.3

## Overview

The snapshot functionality provides point-in-time backup and fast recovery capabilities for the storage layer. Snapshots capture the complete state of a log including all segments, indexes, and metadata, enabling quick restoration in case of data corruption or system failure.

## Features

### 1. Snapshot Creation
- **Point-in-time capture**: Creates an immutable copy of the log at a specific moment
- **Multi-segment support**: Handles logs with multiple segments correctly
- **Metadata tracking**: Records high water mark, segment offsets, and size information
- **Atomic operations**: Ensures snapshot consistency during creation

### 2. Snapshot Restoration
- **Fast recovery**: Quickly restore logs from snapshots without replay
- **Flexible targets**: Restore to any directory location
- **Validation**: Verifies snapshot integrity during restoration
- **Complete state**: Restores all segments, indexes, and time indexes

### 3. Snapshot Cleanup
Automatic cleanup based on configurable policies:
- **Max snapshots**: Keep only N most recent snapshots
- **Retention time**: Delete snapshots older than specified duration
- **Manual deletion**: Remove specific snapshots by ID

## Architecture

### Components

```
SnapshotManager
├── Snapshot metadata (JSON)
├── Snapshot directories
│   ├── snapshot-{timestamp-1}/
│   │   ├── {offset}.log
│   │   ├── {offset}.index
│   │   └── {offset}.timeindex
│   └── snapshot-{timestamp-2}/
│       └── ...
└── snapshots.json (metadata file)
```

### Key Types

**Snapshot**: Represents a single snapshot with metadata
```go
type Snapshot struct {
    ID             string    // Unique identifier
    Timestamp      time.Time // Creation time
    BaseOffset     int64     // First offset in log
    HighWaterMark  int64     // Last offset + 1
    NumSegments    int       // Number of segments
    TotalSize      int64     // Total size in bytes
    SegmentOffsets []int64   // Base offset of each segment
}
```

**SnapshotManager**: Manages snapshot lifecycle
```go
type SnapshotManager struct {
    logDir      string           // Original log directory
    snapshotDir string           // Snapshot storage directory
    metadata    *SnapshotMetadata
}
```

**SnapshotConfig**: Cleanup policy configuration
```go
type SnapshotConfig struct {
    MaxSnapshots   int           // Maximum snapshots to keep (default: 5)
    RetentionTime  time.Duration // Max age (default: 24h)
    MinInterval    time.Duration // Min time between snapshots (default: 1h)
}
```

## Usage

### Creating a Snapshot

```go
// Create snapshot manager for a log directory
sm, err := log.NewSnapshotManager("/data/log")
if err != nil {
    log.Fatal(err)
}

// Create a snapshot
snapshot, err := sm.CreateSnapshot(logInstance)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Created snapshot: %s\n", snapshot.ID)
fmt.Printf("High Water Mark: %d\n", snapshot.HighWaterMark)
fmt.Printf("Size: %d bytes\n", snapshot.TotalSize)
```

### Restoring from Snapshot

```go
// Restore to a new directory
err := sm.RestoreSnapshot(snapshot.ID, "/data/restored-log")
if err != nil {
    log.Fatal(err)
}

// Open the restored log
restoredLog, err := log.NewLog(log.LogConfig{
    Dir:            "/data/restored-log",
    MaxSegmentSize: 1024 * 1024,
})
```

### Listing Snapshots

```go
// Get all snapshots (sorted by timestamp, newest first)
snapshots := sm.ListSnapshots()
for _, s := range snapshots {
    fmt.Printf("%s: HWM=%d, Size=%d, Time=%s\n",
        s.ID, s.HighWaterMark, s.TotalSize, s.Timestamp)
}

// Get specific snapshot
snapshot := sm.GetSnapshot("snapshot-123456")
if snapshot != nil {
    fmt.Printf("Found: %s\n", snapshot.ID)
}
```

### Cleanup Policies

```go
// Configure cleanup policy
config := log.SnapshotConfig{
    MaxSnapshots:  3,              // Keep only 3 most recent
    RetentionTime: 24 * time.Hour, // Delete older than 24 hours
    MinInterval:   1 * time.Hour,  // Minimum 1h between snapshots
}

// Run cleanup
deleted, err := sm.CleanupSnapshots(config)
fmt.Printf("Deleted %d old snapshots\n", deleted)
```

### Manual Deletion

```go
// Delete specific snapshot
err := sm.DeleteSnapshot("snapshot-123456")
if err != nil {
    log.Printf("Delete failed: %v", err)
}
```

## Implementation Details

### Snapshot Creation Process

1. **Lock acquisition**: Acquires read lock on log to ensure consistency
2. **Metadata collection**: Gathers segment information and metrics
3. **Directory creation**: Creates snapshot subdirectory
4. **File copying**: Copies all segment files (.log, .index, .timeindex)
5. **Metadata update**: Updates and persists snapshot metadata
6. **Validation**: Verifies snapshot integrity

### Restoration Process

1. **Snapshot lookup**: Validates snapshot exists
2. **Directory preparation**: Creates target directory if needed
3. **File copying**: Copies all files from snapshot to target
4. **Integrity check**: Verifies all files copied successfully
5. **Log initialization**: Target can now be opened as a regular log

### Cleanup Process

1. **Sort snapshots**: Orders by timestamp (newest first)
2. **Apply policies**: 
   - Keep first N (MaxSnapshots)
   - Remove older than RetentionTime
3. **Delete files**: Removes snapshot directories
4. **Update metadata**: Persists updated snapshot list

## Performance Characteristics

### Space Complexity
- **Snapshot size**: Equal to log size at snapshot time
- **Overhead**: ~100 bytes per snapshot for metadata
- **Storage**: O(N × log_size) where N = number of snapshots

### Time Complexity
- **Create**: O(log_size) - linear in log data size
- **Restore**: O(log_size) - linear file copy
- **List**: O(N) where N = number of snapshots
- **Cleanup**: O(N) for metadata + file deletion

### Optimization Strategies

1. **Incremental snapshots**: Future enhancement for delta-based snapshots
2. **Compression**: Compress snapshot files to reduce storage
3. **Hard links**: Use hard links instead of copies on same filesystem
4. **Async creation**: Create snapshots in background without blocking log

## Testing

### Test Coverage

All tests in `snapshot_test.go`:

1. **Basic Operations**
   - Create snapshot
   - Restore snapshot
   - List snapshots
   - Get specific snapshot
   - Delete snapshot

2. **Cleanup Policies**
   - Max snapshots limit
   - Retention time policy
   - Combined policies

3. **Edge Cases**
   - Empty log snapshots
   - Multi-segment logs
   - Non-existent snapshot restoration
   - Concurrent snapshot creation

4. **Persistence**
   - Metadata persistence across restarts
   - Snapshot manager reload

### Running Tests

```bash
# Run all snapshot tests
go test ./pkg/storage/log -run TestSnapshot -v

# Run specific test
go test ./pkg/storage/log -run TestSnapshotManager_CreateSnapshot -v

# Run with race detector
go test ./pkg/storage/log -run TestSnapshot -race
```

### Example Program

Run the example to see snapshot functionality in action:

```bash
cd backend/pkg/storage/log/example
go run snapshot_example.go
```

## Configuration Best Practices

### Production Settings

```go
config := log.SnapshotConfig{
    MaxSnapshots:  5,                // Keep last 5 snapshots
    RetentionTime: 7 * 24 * time.Hour, // 1 week retention
    MinInterval:   6 * time.Hour,      // Snapshot every 6 hours
}
```

### Development Settings

```go
config := log.SnapshotConfig{
    MaxSnapshots:  3,                // Keep last 3 snapshots
    RetentionTime: 24 * time.Hour,   // 1 day retention
    MinInterval:   1 * time.Hour,    // Hourly snapshots OK
}
```

### High-frequency Settings

```go
config := log.SnapshotConfig{
    MaxSnapshots:  10,               // More snapshots for safety
    RetentionTime: 48 * time.Hour,   // 2 days retention
    MinInterval:   30 * time.Minute, // More frequent snapshots
}
```

## Error Handling

### Common Errors

1. **Insufficient disk space**: Snapshot creation fails if not enough space
2. **File permission errors**: Requires read/write access to directories
3. **Corrupted snapshots**: Restoration fails with validation error
4. **Concurrent access**: Handled by mutex locks

### Error Recovery

```go
snapshot, err := sm.CreateSnapshot(log)
if err != nil {
    if errors.Is(err, os.ErrPermission) {
        // Handle permission error
    } else if errors.Is(err, os.ErrNotExist) {
        // Handle missing directory
    }
    // Cleanup partial snapshot
    return fmt.Errorf("snapshot failed: %w", err)
}
```

## Integration with Takhin

### Topic Manager Integration

```go
// In topic/manager.go
func (m *Manager) CreateTopicSnapshot(topicName string) error {
    topic, exists := m.GetTopic(topicName)
    if !exists {
        return fmt.Errorf("topic not found")
    }
    
    for partitionID, log := range topic.Partitions {
        sm, err := log.NewSnapshotManager(log.Dir())
        if err != nil {
            return err
        }
        
        _, err = sm.CreateSnapshot(log)
        if err != nil {
            return fmt.Errorf("partition %d: %w", partitionID, err)
        }
    }
    
    return nil
}
```

### Automated Snapshot Scheduling

```go
// Background goroutine for periodic snapshots
func (m *Manager) StartSnapshotScheduler(config log.SnapshotConfig) {
    ticker := time.NewTicker(config.MinInterval)
    defer ticker.Stop()
    
    for range ticker.C {
        for _, topic := range m.topics {
            // Create snapshots for each partition
            // Run cleanup based on policy
        }
    }
}
```

## Future Enhancements

1. **Incremental snapshots**: Only copy changed segments
2. **Compression**: Compress snapshot files (gzip, zstd)
3. **Remote storage**: S3, Azure Blob, GCS integration
4. **Parallel operations**: Speed up snapshot creation/restoration
5. **Encryption**: Encrypt snapshots at rest
6. **Checksum validation**: Verify snapshot integrity with checksums
7. **Differential restore**: Restore only specific segments

## References

- Storage Layer: `backend/pkg/storage/log/`
- Log Implementation: `log.go`
- Segment Implementation: `segment.go`
- Recovery Mechanism: `recovery.go`
- Test Suite: `snapshot_test.go`
- Example Code: `example/snapshot_example.go`

## Acceptance Criteria ✓

- ✅ **Snapshot Creation**: Implemented with metadata tracking
- ✅ **Snapshot Restoration**: Fast recovery to any directory
- ✅ **Cleanup Strategy**: Configurable policies (max count, retention time)
- ✅ **Test Coverage**: Comprehensive test suite with 18 test cases
- ✅ **Documentation**: Complete usage guide and examples
- ✅ **Thread Safety**: Mutex-protected concurrent operations
- ✅ **Error Handling**: Robust error handling and validation

## Summary

The snapshot feature provides a robust backup and recovery mechanism for Takhin's storage layer. It enables:

- **Fast recovery** from snapshots without log replay
- **Flexible policies** for automatic cleanup
- **Production-ready** with comprehensive testing
- **Easy integration** with existing storage components

Priority: **P1 - Medium** | Status: **Complete** | Estimate: **2-3 days**
