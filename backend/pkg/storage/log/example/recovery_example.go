package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	storageLog "github.com/takhin-data/takhin/pkg/storage/log"
)

// Example demonstrating the storage recovery mechanism
func main() {
	// Create a temporary directory for this example
	dataDir := "/tmp/takhin-recovery-example"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}
	defer os.RemoveAll(dataDir)

	fmt.Println("=== Storage Recovery Example ===")

	// Step 1: Create a log and write some data
	fmt.Println("Step 1: Creating log and writing data...")
	originalLog, err := storageLog.NewLog(storageLog.LogConfig{
		Dir:            dataDir,
		MaxSegmentSize: 1024,
	})
	if err != nil {
		log.Fatalf("Failed to create log: %v", err)
	}

	// Write records
	for i := 0; i < 50; i++ {
		key := []byte(fmt.Sprintf("key-%d", i))
		value := []byte(fmt.Sprintf("value-%d with some data", i))
		_, err := originalLog.Append(key, value)
		if err != nil {
			log.Fatalf("Failed to append: %v", err)
		}
	}

	hwm := originalLog.HighWaterMark()
	fmt.Printf("  Written %d records (HWM: %d)\n", hwm, hwm)
	fmt.Printf("  Number of segments: %d\n\n", originalLog.NumSegments())

	// Step 2: Simulate corruption by truncating indexes
	fmt.Println("Step 2: Simulating index corruption...")
	segments := originalLog.GetSegments()
	originalLog.Close()

	// Corrupt the indexes of all segments
	for _, segInfo := range segments {
		indexPath := filepath.Join(dataDir, fmt.Sprintf("%020d.index", segInfo.BaseOffset))
		timeIndexPath := filepath.Join(dataDir, fmt.Sprintf("%020d.timeindex", segInfo.BaseOffset))

		// Truncate indexes to simulate corruption
		if err := os.Truncate(indexPath, 0); err != nil {
			log.Printf("  Warning: Failed to truncate %s: %v", indexPath, err)
		}
		if err := os.Truncate(timeIndexPath, 0); err != nil {
			log.Printf("  Warning: Failed to truncate %s: %v", timeIndexPath, err)
		}
	}
	fmt.Printf("  Corrupted indexes for %d segments\n\n", len(segments))

	// Step 3: Recover the log
	fmt.Println("Step 3: Recovering log from directory...")
	recoveredLog, err := storageLog.RecoverFromDirectory(dataDir, 1024)
	if err != nil {
		fmt.Printf("  Recovery completed with warnings: %v\n", err)
	} else {
		fmt.Println("  Recovery completed successfully!")
	}

	// Perform full recovery with detailed results
	recovery := storageLog.NewLogRecovery(recoveredLog)
	result, err := recovery.RecoverLog()
	if err != nil {
		fmt.Printf("  Recovery had errors: %v\n", err)
	}

	fmt.Println("\nRecovery Results:")
	fmt.Printf("  Records recovered: %d\n", result.RecordsRecovered)
	fmt.Printf("  Records truncated: %d\n", result.RecordsTruncated)
	fmt.Printf("  Index rebuilt: %v\n", result.IndexRebuilt)
	fmt.Printf("  Time index rebuilt: %v\n", result.TimeIndexRebuilt)
	fmt.Printf("  Corruption detected: %v\n", result.CorruptionDetected)
	if len(result.Errors) > 0 {
		fmt.Printf("  Errors encountered: %d\n", len(result.Errors))
		for i, err := range result.Errors {
			fmt.Printf("    %d. %v\n", i+1, err)
		}
	}

	// Step 4: Verify recovered data
	fmt.Println("\nStep 4: Verifying recovered data...")
	recoveredHWM := recoveredLog.HighWaterMark()
	fmt.Printf("  Recovered HWM: %d (original: %d)\n", recoveredHWM, hwm)
	fmt.Printf("  Number of segments: %d\n", recoveredLog.NumSegments())

	// Read some records to verify
	fmt.Println("\nSample recovered records:")
	for i := int64(0); i < 5; i++ {
		record, err := recoveredLog.Read(i)
		if err != nil {
			fmt.Printf("  Offset %d: Error - %v\n", i, err)
		} else {
			fmt.Printf("  Offset %d: Key=%s, Value=%s\n", i, string(record.Key), string(record.Value))
		}
	}

	// Step 5: Continue writing to recovered log
	fmt.Println("\nStep 5: Writing new records to recovered log...")
	newRecords := 10
	for i := 0; i < newRecords; i++ {
		key := []byte(fmt.Sprintf("new-key-%d", i))
		value := []byte(fmt.Sprintf("new-value-%d", i))
		offset, err := recoveredLog.Append(key, value)
		if err != nil {
			log.Fatalf("Failed to append to recovered log: %v", err)
		}
		if i == 0 {
			fmt.Printf("  First new offset: %d\n", offset)
		}
	}
	fmt.Printf("  Written %d new records\n", newRecords)
	fmt.Printf("  New HWM: %d\n", recoveredLog.HighWaterMark())

	recoveredLog.Close()
	fmt.Println("\n=== Example completed successfully ===")
}
