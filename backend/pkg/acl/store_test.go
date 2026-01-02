// Copyright 2025 Takhin Data, Inc.

package acl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreAddAndList(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	entry := Entry{
		Principal:      "User:alice",
		Host:           "*",
		ResourceType:   ResourceTypeTopic,
		ResourceName:   "test-topic",
		PatternType:    PatternTypeLiteral,
		Operation:      OperationRead,
		PermissionType: PermissionTypeAllow,
	}

	err := store.Add(entry)
	require.NoError(t, err)

	// Try to add duplicate
	err = store.Add(entry)
	assert.Error(t, err)

	// List all entries
	entries := store.GetAll()
	assert.Len(t, entries, 1)
	assert.Equal(t, entry, entries[0])
}

func TestStoreDelete(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	entry1 := Entry{
		Principal:      "User:alice",
		Host:           "*",
		ResourceType:   ResourceTypeTopic,
		ResourceName:   "test-topic",
		PatternType:    PatternTypeLiteral,
		Operation:      OperationRead,
		PermissionType: PermissionTypeAllow,
	}

	entry2 := Entry{
		Principal:      "User:bob",
		Host:           "*",
		ResourceType:   ResourceTypeTopic,
		ResourceName:   "test-topic",
		PatternType:    PatternTypeLiteral,
		Operation:      OperationWrite,
		PermissionType: PermissionTypeAllow,
	}

	require.NoError(t, store.Add(entry1))
	require.NoError(t, store.Add(entry2))

	// Delete entries for alice
	principal := "User:alice"
	filter := Filter{
		ResourceFilter: ResourceFilter{
			ResourceType: ResourceTypeTopic,
		},
		AccessFilter: AccessFilter{
			Principal: &principal,
		},
	}

	deleted, err := store.Delete(filter)
	require.NoError(t, err)
	assert.Equal(t, 1, deleted)

	// Verify only bob's entry remains
	entries := store.GetAll()
	assert.Len(t, entries, 1)
	assert.Equal(t, entry2, entries[0])
}

func TestStoreListWithFilter(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	entries := []Entry{
		{
			Principal:      "User:alice",
			Host:           "*",
			ResourceType:   ResourceTypeTopic,
			ResourceName:   "topic1",
			PatternType:    PatternTypeLiteral,
			Operation:      OperationRead,
			PermissionType: PermissionTypeAllow,
		},
		{
			Principal:      "User:alice",
			Host:           "*",
			ResourceType:   ResourceTypeTopic,
			ResourceName:   "topic2",
			PatternType:    PatternTypeLiteral,
			Operation:      OperationWrite,
			PermissionType: PermissionTypeAllow,
		},
		{
			Principal:      "User:bob",
			Host:           "*",
			ResourceType:   ResourceTypeGroup,
			ResourceName:   "group1",
			PatternType:    PatternTypeLiteral,
			Operation:      OperationRead,
			PermissionType: PermissionTypeAllow,
		},
	}

	for _, entry := range entries {
		require.NoError(t, store.Add(entry))
	}

	// Filter by principal
	principal := "User:alice"
	filter := Filter{
		ResourceFilter: ResourceFilter{
			ResourceType: ResourceTypeTopic,
		},
		AccessFilter: AccessFilter{
			Principal: &principal,
		},
	}

	filtered := store.List(filter)
	assert.Len(t, filtered, 2)

	// Filter by resource type
	filter = Filter{
		ResourceFilter: ResourceFilter{
			ResourceType: ResourceTypeGroup,
		},
		AccessFilter: AccessFilter{},
	}

	filtered = store.List(filter)
	assert.Len(t, filtered, 1)
	assert.Equal(t, "User:bob", filtered[0].Principal)
}

func TestStoreSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	entry := Entry{
		Principal:      "User:alice",
		Host:           "192.168.1.1",
		ResourceType:   ResourceTypeTopic,
		ResourceName:   "test-topic",
		PatternType:    PatternTypeLiteral,
		Operation:      OperationRead,
		PermissionType: PermissionTypeAllow,
	}

	require.NoError(t, store.Add(entry))
	require.NoError(t, store.Save())

	// Create new store and load
	store2 := NewStore(tmpDir)
	require.NoError(t, store2.Load())

	entries := store2.GetAll()
	assert.Len(t, entries, 1)
	assert.Equal(t, entry, entries[0])
}

func TestStoreLoadNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	// Should not error when loading non-existent file
	err := store.Load()
	assert.NoError(t, err)
	assert.Len(t, store.GetAll(), 0)
}

func TestStoreSaveCreateDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	aclDir := filepath.Join(tmpDir, "nested", "acl")
	store := NewStore(aclDir)

	entry := Entry{
		Principal:      "User:alice",
		Host:           "*",
		ResourceType:   ResourceTypeTopic,
		ResourceName:   "test-topic",
		PatternType:    PatternTypeLiteral,
		Operation:      OperationRead,
		PermissionType: PermissionTypeAllow,
	}

	require.NoError(t, store.Add(entry))
	require.NoError(t, store.Save())

	// Verify directory was created
	_, err := os.Stat(aclDir)
	assert.NoError(t, err)
}

func TestPatternMatching(t *testing.T) {
	tests := []struct {
		name         string
		entryName    string
		patternType  PatternType
		resourceName string
		expected     bool
	}{
		{
			name:         "literal exact match",
			entryName:    "test-topic",
			patternType:  PatternTypeLiteral,
			resourceName: "test-topic",
			expected:     true,
		},
		{
			name:         "literal no match",
			entryName:    "test-topic",
			patternType:  PatternTypeLiteral,
			resourceName: "other-topic",
			expected:     false,
		},
		{
			name:         "prefix match",
			entryName:    "test-",
			patternType:  PatternTypePrefixed,
			resourceName: "test-topic",
			expected:     true,
		},
		{
			name:         "prefix no match",
			entryName:    "test-",
			patternType:  PatternTypePrefixed,
			resourceName: "prod-topic",
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesResource(tt.entryName, tt.patternType, tt.resourceName)
			assert.Equal(t, tt.expected, result)
		})
	}
}
