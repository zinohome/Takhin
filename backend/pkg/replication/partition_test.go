// Copyright 2025 Takhin Data, Inc.

package replication

import (
"testing"

"github.com/stretchr/testify/assert"
"github.com/stretchr/testify/require"
"github.com/takhin-data/takhin/pkg/storage/log"
)

func TestNewPartition(t *testing.T) {
	config := PartitionConfig{
		TopicName:   "test-topic",
		PartitionID: 0,
		Leader:      1,
		Replicas:    []int32{1, 2, 3},
		LogConfig: log.LogConfig{
			Dir:            t.TempDir(),
			MaxSegmentSize: 1024 * 1024,
		},
	}
	
	partition, err := NewPartition(config)
	require.NoError(t, err)
	defer partition.Close()
	
	assert.Equal(t, "test-topic", partition.TopicName)
	assert.Equal(t, int32(0), partition.PartitionID)
	assert.Equal(t, int32(1), partition.Leader)
}

func TestPartitionAppendAndRead(t *testing.T) {
	config := PartitionConfig{
		TopicName:   "test-topic",
		PartitionID: 0,
		Leader:      1,
		Replicas:    []int32{1, 2, 3},
		LogConfig: log.LogConfig{
			Dir:            t.TempDir(),
			MaxSegmentSize: 1024 * 1024,
		},
	}
	
	partition, err := NewPartition(config)
	require.NoError(t, err)
	defer partition.Close()
	
	offset, err := partition.Append([]byte("key1"), []byte("value1"))
	require.NoError(t, err)
	assert.Equal(t, int64(0), offset)
	
	record, err := partition.Read(0)
	require.NoError(t, err)
	assert.Equal(t, []byte("key1"), record.Key)
}
