// Copyright 2025 Takhin Data, Inc.

package handler

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/takhin-data/takhin/pkg/config"
	"github.com/takhin-data/takhin/pkg/storage/topic"
)

func TestHandlerWithCoordinator(t *testing.T) {
	dir := t.TempDir()

	topicMgr := topic.NewManager(dir, 1024*1024)
	cfg := &config.Config{}
	h := New(cfg, topicMgr)

	// Verify coordinator is initialized
	assert.NotNil(t, h.coordinator)

	// Test basic coordinator operations
	group := h.coordinator.GetOrCreateGroup("test-group", "consumer")
	assert.NotNil(t, group)
	assert.Equal(t, "test-group", group.ID)

	// Test offset operations
	err := h.coordinator.CommitOffset("test-group", "test-topic", 0, 100, "metadata")
	require.NoError(t, err)

	offset, exists := h.coordinator.FetchOffset("test-group", "test-topic", 0)
	require.True(t, exists)
	assert.Equal(t, int64(100), offset.Offset)
	assert.Equal(t, "metadata", offset.Metadata)
}
