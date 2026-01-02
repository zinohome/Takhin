// Copyright 2025 Takhin Data, Inc.

package main

import (
	"fmt"
	"time"

	storagelog "github.com/takhin-data/takhin/pkg/storage/log"
)

func main() {
	// Create a log
	logInstance, err := storagelog.NewLog(storagelog.LogConfig{
		Dir:            "./data/example-snapshot-log",
		MaxSegmentSize: 1024 * 1024, // 1MB segments
	})
	if err != nil {
		panic(err)
	}
	defer logInstance.Close()

	// Write some data
	fmt.Println("Writing data to log...")
	for i := 0; i < 100; i++ {
		key := []byte(fmt.Sprintf("key-%d", i))
		value := []byte(fmt.Sprintf("value-%d with some data", i))
		offset, err := logInstance.Append(key, value)
		if err != nil {
			panic(err)
		}
		if i%10 == 0 {
			fmt.Printf("Written offset %d\n", offset)
		}
	}

	hwm := logInstance.HighWaterMark()
	fmt.Printf("Log high water mark: %d\n", hwm)

	// Create snapshot manager
	fmt.Println("\nCreating snapshot manager...")
	snapshotManager, err := storagelog.NewSnapshotManager("./data/example-snapshot-log")
	if err != nil {
		panic(err)
	}

	// Create a snapshot
	fmt.Println("Creating snapshot...")
	snapshot, err := snapshotManager.CreateSnapshot(logInstance)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Snapshot created: %s\n", snapshot.ID)
	fmt.Printf("  Timestamp: %s\n", snapshot.Timestamp.Format(time.RFC3339))
	fmt.Printf("  High Water Mark: %d\n", snapshot.HighWaterMark)
	fmt.Printf("  Segments: %d\n", snapshot.NumSegments)
	fmt.Printf("  Total Size: %d bytes\n", snapshot.TotalSize)

	// List all snapshots
	fmt.Println("\nListing all snapshots...")
	snapshots := snapshotManager.ListSnapshots()
	for i, s := range snapshots {
		fmt.Printf("  %d. %s (HWM: %d, Size: %d bytes, Time: %s)\n",
			i+1, s.ID, s.HighWaterMark, s.TotalSize,
			s.Timestamp.Format(time.RFC3339))
	}

	// Simulate creating more data
	fmt.Println("\nWriting more data...")
	for i := 100; i < 150; i++ {
		key := []byte(fmt.Sprintf("key-%d", i))
		value := []byte(fmt.Sprintf("value-%d with some data", i))
		_, err := logInstance.Append(key, value)
		if err != nil {
			panic(err)
		}
	}

	newHWM := logInstance.HighWaterMark()
	fmt.Printf("New high water mark: %d\n", newHWM)

	// Restore from snapshot
	fmt.Println("\nRestoring from snapshot to new location...")
	restorePath := "./data/restored-snapshot-log"
	err = snapshotManager.RestoreSnapshot(snapshot.ID, restorePath)
	if err != nil {
		panic(err)
	}

	// Verify restored log
	restoredLog, err := storagelog.NewLog(storagelog.LogConfig{
		Dir:            restorePath,
		MaxSegmentSize: 1024 * 1024,
	})
	if err != nil {
		panic(err)
	}
	defer restoredLog.Close()

	restoredHWM := restoredLog.HighWaterMark()
	fmt.Printf("Restored log high water mark: %d\n", restoredHWM)
	fmt.Printf("Original snapshot HWM: %d\n", snapshot.HighWaterMark)

	if restoredHWM == snapshot.HighWaterMark {
		fmt.Println("✓ Restoration successful!")
	} else {
		fmt.Println("✗ Restoration mismatch!")
	}

	// Read some data from restored log
	fmt.Println("\nReading sample data from restored log...")
	for i := int64(0); i < 5; i++ {
		record, err := restoredLog.Read(i)
		if err != nil {
			panic(err)
		}
		fmt.Printf("  Offset %d: %s = %s\n", record.Offset, string(record.Key), string(record.Value))
	}

	// Demonstrate cleanup
	fmt.Println("\nDemonstrating cleanup with retention policy...")
	config := storagelog.SnapshotConfig{
		MaxSnapshots:  3,
		RetentionTime: 24 * time.Hour,
		MinInterval:   1 * time.Hour,
	}

	deleted, err := snapshotManager.CleanupSnapshots(config)
	if err != nil {
		fmt.Printf("Warning during cleanup: %v\n", err)
	}
	fmt.Printf("Deleted %d old snapshots\n", deleted)

	// Show remaining snapshots
	remainingSnapshots := snapshotManager.ListSnapshots()
	fmt.Printf("Remaining snapshots: %d\n", len(remainingSnapshots))

	// Get total size of all snapshots
	totalSize, err := snapshotManager.Size()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Total snapshot storage: %d bytes (%.2f MB)\n", totalSize, float64(totalSize)/(1024*1024))

	fmt.Println("\nSnapshot example completed successfully!")
}
