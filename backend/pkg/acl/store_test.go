// Copyright 2025 Takhin Data, Inc.

package acl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreCreateAndLoad(t *testing.T) {
	tmpDir := t.TempDir()

	// Create store
	store, err := NewStore(tmpDir)
	require.NoError(t, err)
	assert.NotNil(t, store)

	// Verify ACL directory created
	aclDir := filepath.Join(tmpDir, "acls")
	_, err = os.Stat(aclDir)
	assert.NoError(t, err)
}

func TestStoreAddAndList(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStore(tmpDir)
	require.NoError(t, err)

	// Create test entry
	entry, err := NewEntry("User:alice", "*", ResourceTypeTopic,
		"test-topic", PatternTypeLiteral, OperationRead, PermissionTypeAllow)
	require.NoError(t, err)

	// Add entry
	err = store.Add(entry)
	assert.NoError(t, err)

	// List all entries
	entries := store.List(&Filter{})
	assert.Len(t, entries, 1)
	assert.Equal(t, "User:alice", entries[0].Principal)
	assert.Equal(t, "test-topic", entries[0].ResourceName)
}

func TestStoreAddMultiple(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStore(tmpDir)
	require.NoError(t, err)

	// Add multiple entries
	entries := []struct {
		principal    string
		resourceName string
		operation    Operation
	}{
		{"User:alice", "topic-1", OperationRead},
		{"User:alice", "topic-1", OperationWrite},
		{"User:bob", "topic-2", OperationRead},
	}

	for _, e := range entries {
		entry, err := NewEntry(e.principal, "*", ResourceTypeTopic,
			e.resourceName, PatternTypeLiteral, e.operation, PermissionTypeAllow)
		require.NoError(t, err)
		err = store.Add(entry)
		assert.NoError(t, err)
	}

	// List all
	allEntries := store.List(&Filter{})
	assert.Len(t, allEntries, 3)

	// Filter by principal
	alice := "User:alice"
	aliceEntries := store.List(&Filter{Principal: &alice})
	assert.Len(t, aliceEntries, 2)

	// Filter by resource name
	topic1 := "topic-1"
	topic1Entries := store.List(&Filter{ResourceName: &topic1})
	assert.Len(t, topic1Entries, 2)
}

func TestStoreDelete(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStore(tmpDir)
	require.NoError(t, err)

	// Add entries
	entry1, _ := NewEntry("User:alice", "*", ResourceTypeTopic,
		"topic-1", PatternTypeLiteral, OperationRead, PermissionTypeAllow)
	entry2, _ := NewEntry("User:alice", "*", ResourceTypeTopic,
		"topic-2", PatternTypeLiteral, OperationRead, PermissionTypeAllow)
	entry3, _ := NewEntry("User:bob", "*", ResourceTypeTopic,
		"topic-1", PatternTypeLiteral, OperationRead, PermissionTypeAllow)

	store.Add(entry1)
	store.Add(entry2)
	store.Add(entry3)

	assert.Equal(t, 3, store.Count())

	// Delete alice's entries
	alice := "User:alice"
	deleted, err := store.Delete(&Filter{Principal: &alice})
	assert.NoError(t, err)
	assert.Equal(t, 2, deleted)

	// Verify only bob's entry remains
	remaining := store.List(&Filter{})
	assert.Len(t, remaining, 1)
	assert.Equal(t, "User:bob", remaining[0].Principal)
}

func TestStoreCheck(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStore(tmpDir)
	require.NoError(t, err)

	// Add allow entry for alice to read topic-1
	allow, _ := NewEntry("User:alice", "*", ResourceTypeTopic,
		"topic-1", PatternTypeLiteral, OperationRead, PermissionTypeAllow)
	store.Add(allow)

	// Add deny entry for bob to read topic-1
	deny, _ := NewEntry("User:bob", "*", ResourceTypeTopic,
		"topic-1", PatternTypeLiteral, OperationRead, PermissionTypeDeny)
	store.Add(deny)

	tests := []struct {
		name         string
		principal    string
		resourceName string
		operation    Operation
		expected     bool
	}{
		{
			name:         "alice can read topic-1",
			principal:    "User:alice",
			resourceName: "topic-1",
			operation:    OperationRead,
			expected:     true,
		},
		{
			name:         "bob denied to read topic-1",
			principal:    "User:bob",
			resourceName: "topic-1",
			operation:    OperationRead,
			expected:     false,
		},
		{
			name:         "alice cannot write topic-1",
			principal:    "User:alice",
			resourceName: "topic-1",
			operation:    OperationWrite,
			expected:     false,
		},
		{
			name:         "charlie has no access",
			principal:    "User:charlie",
			resourceName: "topic-1",
			operation:    OperationRead,
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := store.Check(tt.principal, "*", ResourceTypeTopic,
				tt.resourceName, tt.operation)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStorePersistence(t *testing.T) {
	tmpDir := t.TempDir()

	// Create store and add entries
	store1, err := NewStore(tmpDir)
	require.NoError(t, err)

	entry, _ := NewEntry("User:alice", "*", ResourceTypeTopic,
		"test-topic", PatternTypeLiteral, OperationRead, PermissionTypeAllow)
	err = store1.Add(entry)
	assert.NoError(t, err)

	// Create new store with same directory
	store2, err := NewStore(tmpDir)
	require.NoError(t, err)

	// Verify entry loaded from disk
	entries := store2.List(&Filter{})
	assert.Len(t, entries, 1)
	assert.Equal(t, "User:alice", entries[0].Principal)
	assert.Equal(t, "test-topic", entries[0].ResourceName)
}

func TestStoreDenyPrecedence(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStore(tmpDir)
	require.NoError(t, err)

	// Add both allow and deny for same resource
	allow, _ := NewEntry("User:alice", "*", ResourceTypeTopic,
		"topic-1", PatternTypeLiteral, OperationRead, PermissionTypeAllow)
	deny, _ := NewEntry("User:alice", "*", ResourceTypeTopic,
		"topic-1", PatternTypeLiteral, OperationRead, PermissionTypeDeny)

	store.Add(allow)
	store.Add(deny)

	// Deny should take precedence
	result := store.Check("User:alice", "*", ResourceTypeTopic, "topic-1", OperationRead)
	assert.False(t, result)
}

func TestStoreWildcardPattern(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStore(tmpDir)
	require.NoError(t, err)

	// Add wildcard principal entry
	entry, _ := NewEntry("User:*", "*", ResourceTypeTopic,
		"public-topic", PatternTypeLiteral, OperationRead, PermissionTypeAllow)
	store.Add(entry)

	// Any user can access
	assert.True(t, store.Check("User:alice", "*", ResourceTypeTopic, "public-topic", OperationRead))
	assert.True(t, store.Check("User:bob", "*", ResourceTypeTopic, "public-topic", OperationRead))
}

func TestStorePrefixPattern(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStore(tmpDir)
	require.NoError(t, err)

	// Add prefix pattern entry
	entry, _ := NewEntry("User:alice", "*", ResourceTypeTopic,
		"test-", PatternTypePrefix, OperationRead, PermissionTypeAllow)
	store.Add(entry)

	// Should match topics with test- prefix
	assert.True(t, store.Check("User:alice", "*", ResourceTypeTopic, "test-topic-1", OperationRead))
	assert.True(t, store.Check("User:alice", "*", ResourceTypeTopic, "test-topic-2", OperationRead))

	// Should not match other topics
	assert.False(t, store.Check("User:alice", "*", ResourceTypeTopic, "prod-topic", OperationRead))
}
