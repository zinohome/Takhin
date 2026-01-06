// Copyright 2025 Takhin Data, Inc.

package mempool

// Package mempool provides memory pooling for buffers and records to reduce GC pressure.
//
// It includes:
// - BufferPool: Pools byte slices in various size buckets (512B to 16MB)
// - RecordPool: Pools log.Record objects
// - RecordBatchPool: Pools slices of log.Record pointers
//
// Usage example:
//
//	// Get a buffer
//	buf := mempool.GetBuffer(4096)
//	defer mempool.PutBuffer(buf)
//
//	// Get a record
//	record := mempool.GetRecord()
//	defer mempool.PutRecord(record)
//
//	// Get a record batch
//	batch := mempool.GetRecordBatch(100)
//	defer mempool.PutRecordBatch(batch)
