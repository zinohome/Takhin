# Task 1.3 - Snapshot Architecture

## System Overview

```
┌─────────────────────────────────────────────────────────────┐
│                     Takhin Storage Layer                     │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌────────────┐          ┌──────────────────┐              │
│  │    Log     │◄────────►│ SnapshotManager  │              │
│  │ Instance   │          │                  │              │
│  └────────────┘          └──────────────────┘              │
│       │                           │                         │
│       │ has                       │ manages                 │
│       ▼                           ▼                         │
│  ┌────────────┐          ┌──────────────────┐              │
│  │  Segments  │          │    Snapshots     │              │
│  │            │          │                  │              │
│  │ • .log     │──copy───►│ • snapshot-1/    │              │
│  │ • .index   │          │ • snapshot-2/    │              │
│  │ • .timeindex│         │ • snapshots.json │              │
│  └────────────┘          └──────────────────┘              │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Component Interaction

```
┌──────────────┐
│   Client     │
│  Application │
└──────┬───────┘
       │
       │ 1. CreateSnapshot(log)
       ▼
┌──────────────────────────┐
│   SnapshotManager        │
├──────────────────────────┤
│ • Acquire locks          │
│ • Flush segments         │──────┐
│ • Copy files             │      │
│ • Update metadata        │      │
└──────────────────────────┘      │
       │                          │
       │ 2. Create snapshot dir   │
       ▼                          │
┌──────────────────────────┐      │
│  File System             │      │
├──────────────────────────┤      │
│ log-dir/                 │      │
│ ├── .snapshots/          │◄─────┘
│ │   ├── snapshot-xxx/    │
│ │   │   ├── .log         │
│ │   │   ├── .index       │
│ │   │   └── .timeindex   │
│ │   └── snapshots.json   │
│ ├── {offset}.log         │
│ ├── {offset}.index       │
│ └── {offset}.timeindex   │
└──────────────────────────┘
```

## Snapshot Creation Flow

```
┌─────────┐
│  Start  │
└────┬────┘
     │
     ▼
┌─────────────────────────┐
│ Lock Log (Read Lock)    │
└────┬────────────────────┘
     │
     ▼
┌─────────────────────────┐
│ Collect Metadata        │
│ • High Water Mark       │
│ • Segment Offsets       │
│ • Total Size            │
└────┬────────────────────┘
     │
     ▼
┌─────────────────────────┐
│ Create Snapshot Dir     │
│ .snapshots/snapshot-ID  │
└────┬────────────────────┘
     │
     ▼
┌─────────────────────────┐
│ For Each Segment:       │
│ 1. Flush to disk        │
│ 2. Copy .log file       │
│ 3. Copy .index file     │
│ 4. Copy .timeindex file │
└────┬────────────────────┘
     │
     ▼
┌─────────────────────────┐
│ Update Metadata         │
│ Add to snapshots.json   │
└────┬────────────────────┘
     │
     ▼
┌─────────────────────────┐
│ Release Lock            │
└────┬────────────────────┘
     │
     ▼
┌─────────┐
│   End   │
└─────────┘
```

## Snapshot Restoration Flow

```
┌─────────┐
│  Start  │
└────┬────┘
     │
     ▼
┌─────────────────────────┐
│ Lookup Snapshot by ID   │
└────┬────────────────────┘
     │
     ▼
┌─────────────────────────┐
│ Verify Snapshot Exists  │
└────┬────────────────────┘
     │
     ▼
┌─────────────────────────┐
│ Create Target Directory │
└────┬────────────────────┘
     │
     ▼
┌─────────────────────────┐
│ For Each File:          │
│ Copy from snapshot      │
│ to target directory     │
└────┬────────────────────┘
     │
     ▼
┌─────────────────────────┐
│ Log Loading (NewLog)    │
│ 1. Scan for .log files  │
│ 2. Load segments        │
│ 3. Set active segment   │
└────┬────────────────────┘
     │
     ▼
┌─────────┐
│   End   │
└─────────┘
```

## Cleanup Policy Flow

```
┌─────────┐
│  Start  │
└────┬────┘
     │
     ▼
┌─────────────────────────┐
│ Sort Snapshots by Time  │
│ (Newest First)          │
└────┬────────────────────┘
     │
     ▼
┌─────────────────────────┐
│ For Each Snapshot:      │
│ Check Policies          │
└────┬────────────────────┘
     │
     ├──► Policy 1: Index >= MaxSnapshots?
     │    Yes: Mark for deletion
     │
     ├──► Policy 2: Age > RetentionTime?
     │    Yes: Mark for deletion
     │
     ▼
┌─────────────────────────┐
│ Delete Marked Snapshots │
│ • Remove directory      │
│ • Update metadata       │
└────┬────────────────────┘
     │
     ▼
┌─────────────────────────┐
│ Persist Updated         │
│ Metadata to JSON        │
└────┬────────────────────┘
     │
     ▼
┌─────────┐
│   End   │
└─────────┘
```

## Data Structures

```
SnapshotManager
├── logDir: string
├── snapshotDir: string (.snapshots)
├── metadata: SnapshotMetadata
│   ├── snapshots: []*Snapshot
│   └── mu: RWMutex
└── mu: RWMutex

Snapshot
├── ID: string (snapshot-{timestamp})
├── Timestamp: time.Time
├── BaseOffset: int64
├── HighWaterMark: int64
├── NumSegments: int
├── TotalSize: int64
└── SegmentOffsets: []int64

SnapshotConfig
├── MaxSnapshots: int (5)
├── RetentionTime: time.Duration (24h)
└── MinInterval: time.Duration (1h)
```

## File Layout

```
log-directory/
├── .snapshots/                    # Snapshot storage
│   ├── snapshot-1735998123000/    # Snapshot 1
│   │   ├── 00000000000000000000.log
│   │   ├── 00000000000000000000.index
│   │   ├── 00000000000000000000.timeindex
│   │   ├── 00000000000000001000.log
│   │   ├── 00000000000000001000.index
│   │   └── 00000000000000001000.timeindex
│   ├── snapshot-1735998456000/    # Snapshot 2
│   │   └── ...
│   └── snapshots.json             # Metadata
├── 00000000000000000000.log       # Active log
├── 00000000000000000000.index
├── 00000000000000000000.timeindex
├── 00000000000000002000.log
├── 00000000000000002000.index
└── 00000000000000002000.timeindex
```

## Thread Safety

```
┌────────────────────────────────────────────────────────┐
│                  Concurrency Model                     │
├────────────────────────────────────────────────────────┤
│                                                         │
│  CreateSnapshot:                                        │
│  ┌────────────────────────────────────────┐            │
│  │ sm.mu.Lock()              [Write Lock] │            │
│  │   ├─ log.mu.RLock()       [Read Lock]  │            │
│  │   ├─ Copy files                        │            │
│  │   ├─ metadata.mu.Lock()   [Write Lock] │            │
│  │   └─ Update metadata                   │            │
│  └────────────────────────────────────────┘            │
│                                                         │
│  RestoreSnapshot:                                       │
│  ┌────────────────────────────────────────┐            │
│  │ sm.mu.RLock()             [Read Lock]  │            │
│  │   └─ Copy files from snapshot          │            │
│  └────────────────────────────────────────┘            │
│                                                         │
│  CleanupSnapshots:                                      │
│  ┌────────────────────────────────────────┐            │
│  │ sm.mu.Lock()              [Write Lock] │            │
│  │   ├─ metadata.mu.Lock()   [Write Lock] │            │
│  │   ├─ Sort and filter                   │            │
│  │   └─ Delete files                      │            │
│  └────────────────────────────────────────┘            │
│                                                         │
└────────────────────────────────────────────────────────┘
```

## Performance Characteristics

| Operation | Time Complexity | Space Complexity | I/O Operations |
|-----------|----------------|------------------|----------------|
| CreateSnapshot | O(n) | O(n) | n file copies |
| RestoreSnapshot | O(n) | O(n) | n file copies |
| ListSnapshots | O(m) | O(m) | 1 read |
| GetSnapshot | O(m) | O(1) | 1 read |
| DeleteSnapshot | O(m + n) | O(1) | n file deletes |
| CleanupSnapshots | O(m + k×n) | O(m) | k×n file deletes |

*where n = log size, m = # snapshots, k = # snapshots to delete*

## Integration Points

```
┌─────────────────────────────────────────────────────────┐
│                    Integration Layer                     │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  Topic Manager                                           │
│  ┌────────────────────────────────────────┐             │
│  │ CreateTopicSnapshot(topic, partition)  │             │
│  │   └─► SnapshotManager.CreateSnapshot() │             │
│  └────────────────────────────────────────┘             │
│                                                          │
│  Console API                                             │
│  ┌────────────────────────────────────────┐             │
│  │ POST /api/snapshots                    │             │
│  │ GET  /api/snapshots                    │             │
│  │ POST /api/snapshots/{id}/restore       │             │
│  │ DELETE /api/snapshots/{id}             │             │
│  └────────────────────────────────────────┘             │
│                                                          │
│  Scheduler (Future)                                      │
│  ┌────────────────────────────────────────┐             │
│  │ Periodic Snapshot Creation             │             │
│  │ Automatic Cleanup                      │             │
│  └────────────────────────────────────────┘             │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

## Error Handling

```
┌─────────────────────────────────────────────────────────┐
│                   Error Categories                       │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  1. File System Errors                                   │
│     • Disk full                                          │
│     • Permission denied                                  │
│     • Directory not found                                │
│     └─► Cleanup partial snapshot                        │
│                                                          │
│  2. Validation Errors                                    │
│     • Snapshot not found                                 │
│     • Invalid snapshot ID                                │
│     └─► Return descriptive error                        │
│                                                          │
│  3. Concurrency Errors                                   │
│     • Lock timeout (handled by mutex)                    │
│     └─► Automatic retry or fail                         │
│                                                          │
│  4. Data Integrity Errors                                │
│     • Corrupted metadata                                 │
│     • Missing files                                      │
│     └─► Return error, suggest recovery                  │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

## Testing Strategy

```
┌─────────────────────────────────────────────────────────┐
│                    Test Coverage                         │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  Unit Tests (18 test cases)                              │
│  ├─ Create/Restore/Delete operations                     │
│  ├─ Cleanup policies (MaxSnapshots, RetentionTime)       │
│  ├─ Edge cases (empty log, multi-segment)                │
│  ├─ Concurrent operations                                │
│  └─ Metadata persistence                                 │
│                                                          │
│  Integration Tests                                       │
│  ├─ Topic Manager integration                            │
│  ├─ Log recovery integration                             │
│  └─ End-to-end snapshot workflow                         │
│                                                          │
│  Performance Tests                                       │
│  ├─ Large log snapshots                                  │
│  ├─ Many snapshots cleanup                               │
│  └─ Concurrent snapshot creation                         │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

## Deployment Considerations

1. **Disk Space**: Ensure sufficient space for N×log_size snapshots
2. **I/O Performance**: Snapshot creation is I/O intensive
3. **Retention Policy**: Configure based on storage capacity
4. **Monitoring**: Track snapshot count, size, age
5. **Backup Strategy**: Consider offsite backup of snapshots
6. **Recovery Testing**: Regularly test restoration process

## Summary

The snapshot system provides:
- ✅ Point-in-time backup capability
- ✅ Fast recovery without replay
- ✅ Flexible cleanup policies
- ✅ Thread-safe operations
- ✅ Complete test coverage
- ✅ Production-ready implementation
