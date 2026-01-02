// Copyright 2025 Takhin Data, Inc.

package acl

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAuthorizer(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		Enabled:      true,
		DataDir:      tmpDir,
		CacheEnabled: true,
		CacheTTL:     time.Minute,
		CacheSize:    100,
	}

	auth, err := NewAuthorizer(cfg)
	require.NoError(t, err)
	assert.NotNil(t, auth)
	assert.True(t, auth.IsEnabled())
}

func TestAuthorizerEnableDisable(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		Enabled: false,
		DataDir: tmpDir,
	}

	auth, err := NewAuthorizer(cfg)
	require.NoError(t, err)
	assert.False(t, auth.IsEnabled())

	auth.Enable()
	assert.True(t, auth.IsEnabled())

	auth.Disable()
	assert.False(t, auth.IsEnabled())
}

func TestAuthorizerDisabledAllowsAll(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		Enabled: false,
		DataDir: tmpDir,
	}

	auth, err := NewAuthorizer(cfg)
	require.NoError(t, err)

	// Even without any ACL entries, everything is allowed when disabled
	allowed := auth.Authorize("User:alice", "*", ResourceTypeTopic, "test-topic", OperationRead)
	assert.True(t, allowed)
}

func TestAuthorizerAuthorize(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		Enabled: true,
		DataDir: tmpDir,
	}

	auth, err := NewAuthorizer(cfg)
	require.NoError(t, err)

	// Add ACL entry
	entry, _ := NewEntry("User:alice", "*", ResourceTypeTopic,
		"test-topic", PatternTypeLiteral, OperationRead, PermissionTypeAllow)
	err = auth.AddACL(entry)
	require.NoError(t, err)

	// Test authorization
	tests := []struct {
		name      string
		principal string
		resource  string
		operation Operation
		expected  bool
	}{
		{
			name:      "alice can read test-topic",
			principal: "User:alice",
			resource:  "test-topic",
			operation: OperationRead,
			expected:  true,
		},
		{
			name:      "alice cannot write test-topic",
			principal: "User:alice",
			resource:  "test-topic",
			operation: OperationWrite,
			expected:  false,
		},
		{
			name:      "bob cannot read test-topic",
			principal: "User:bob",
			resource:  "test-topic",
			operation: OperationRead,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := auth.Authorize(tt.principal, "*", ResourceTypeTopic, tt.resource, tt.operation)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAuthorizerWithCache(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		Enabled:      true,
		DataDir:      tmpDir,
		CacheEnabled: true,
		CacheTTL:     time.Minute,
		CacheSize:    100,
	}

	auth, err := NewAuthorizer(cfg)
	require.NoError(t, err)

	// Add ACL entry
	entry, _ := NewEntry("User:alice", "*", ResourceTypeTopic,
		"test-topic", PatternTypeLiteral, OperationRead, PermissionTypeAllow)
	err = auth.AddACL(entry)
	require.NoError(t, err)

	// First call - cache miss
	result1 := auth.Authorize("User:alice", "*", ResourceTypeTopic, "test-topic", OperationRead)
	assert.True(t, result1)

	stats1 := auth.Stats()
	assert.Equal(t, int64(1), stats1.CacheMissCount)
	assert.Equal(t, int64(0), stats1.CacheHitCount)

	// Second call - cache hit
	result2 := auth.Authorize("User:alice", "*", ResourceTypeTopic, "test-topic", OperationRead)
	assert.True(t, result2)

	stats2 := auth.Stats()
	assert.Equal(t, int64(1), stats2.CacheMissCount)
	assert.Equal(t, int64(1), stats2.CacheHitCount)
}

func TestAuthorizerCacheClearedOnACLChange(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		Enabled:      true,
		DataDir:      tmpDir,
		CacheEnabled: true,
		CacheTTL:     time.Minute,
		CacheSize:    100,
	}

	auth, err := NewAuthorizer(cfg)
	require.NoError(t, err)

	// Add initial ACL
	entry1, _ := NewEntry("User:alice", "*", ResourceTypeTopic,
		"test-topic", PatternTypeLiteral, OperationRead, PermissionTypeAllow)
	auth.AddACL(entry1)

	// Authorize - fills cache
	auth.Authorize("User:alice", "*", ResourceTypeTopic, "test-topic", OperationRead)

	// Add another ACL - cache should be cleared
	entry2, _ := NewEntry("User:bob", "*", ResourceTypeTopic,
		"test-topic", PatternTypeLiteral, OperationRead, PermissionTypeAllow)
	auth.AddACL(entry2)

	// Next authorize should be cache miss
	auth.Authorize("User:alice", "*", ResourceTypeTopic, "test-topic", OperationRead)

	stats := auth.Stats()
	assert.Equal(t, int64(2), stats.CacheMissCount) // Both calls were cache misses
}

func TestAuthorizerStats(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		Enabled: true,
		DataDir: tmpDir,
	}

	auth, err := NewAuthorizer(cfg)
	require.NoError(t, err)

	// Add ACL entries
	allow, _ := NewEntry("User:alice", "*", ResourceTypeTopic,
		"topic-1", PatternTypeLiteral, OperationRead, PermissionTypeAllow)
	deny, _ := NewEntry("User:bob", "*", ResourceTypeTopic,
		"topic-1", PatternTypeLiteral, OperationRead, PermissionTypeDeny)

	auth.AddACL(allow)
	auth.AddACL(deny)

	// Test authorization multiple times
	auth.Authorize("User:alice", "*", ResourceTypeTopic, "topic-1", OperationRead) // allow
	auth.Authorize("User:bob", "*", ResourceTypeTopic, "topic-1", OperationRead)   // deny
	auth.Authorize("User:charlie", "*", ResourceTypeTopic, "topic-1", OperationRead) // deny (no ACL)

	stats := auth.Stats()
	assert.True(t, stats.Enabled)
	assert.Equal(t, 2, stats.TotalACLs)
	assert.Equal(t, int64(1), stats.AllowCount)
	assert.Equal(t, int64(2), stats.DenyCount)
}

func TestAuthorizerListACL(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		Enabled: true,
		DataDir: tmpDir,
	}

	auth, err := NewAuthorizer(cfg)
	require.NoError(t, err)

	// Add multiple ACL entries
	entries := []struct {
		principal string
		resource  string
	}{
		{"User:alice", "topic-1"},
		{"User:alice", "topic-2"},
		{"User:bob", "topic-1"},
	}

	for _, e := range entries {
		entry, _ := NewEntry(e.principal, "*", ResourceTypeTopic,
			e.resource, PatternTypeLiteral, OperationRead, PermissionTypeAllow)
		auth.AddACL(entry)
	}

	// List all ACLs
	allACLs := auth.ListACL(&Filter{})
	assert.Len(t, allACLs, 3)

	// Filter by principal
	alice := "User:alice"
	aliceACLs := auth.ListACL(&Filter{Principal: &alice})
	assert.Len(t, aliceACLs, 2)
}

func TestAuthorizerDeleteACL(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		Enabled: true,
		DataDir: tmpDir,
	}

	auth, err := NewAuthorizer(cfg)
	require.NoError(t, err)

	// Add ACL entries
	entry1, _ := NewEntry("User:alice", "*", ResourceTypeTopic,
		"topic-1", PatternTypeLiteral, OperationRead, PermissionTypeAllow)
	entry2, _ := NewEntry("User:alice", "*", ResourceTypeTopic,
		"topic-2", PatternTypeLiteral, OperationRead, PermissionTypeAllow)
	entry3, _ := NewEntry("User:bob", "*", ResourceTypeTopic,
		"topic-1", PatternTypeLiteral, OperationRead, PermissionTypeAllow)

	auth.AddACL(entry1)
	auth.AddACL(entry2)
	auth.AddACL(entry3)

	// Delete alice's ACLs
	alice := "User:alice"
	deleted, err := auth.DeleteACL(&Filter{Principal: &alice})
	assert.NoError(t, err)
	assert.Equal(t, 2, deleted)

	// Verify remaining ACLs
	remaining := auth.ListACL(&Filter{})
	assert.Len(t, remaining, 1)
	assert.Equal(t, "User:bob", remaining[0].Principal)
}

func TestAuthorizerOperationAll(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		Enabled: true,
		DataDir: tmpDir,
	}

	auth, err := NewAuthorizer(cfg)
	require.NoError(t, err)

	// Add ACL with OperationAll
	entry, _ := NewEntry("User:alice", "*", ResourceTypeTopic,
		"test-topic", PatternTypeLiteral, OperationAll, PermissionTypeAllow)
	auth.AddACL(entry)

	// Alice should be able to perform any operation
	assert.True(t, auth.Authorize("User:alice", "*", ResourceTypeTopic, "test-topic", OperationRead))
	assert.True(t, auth.Authorize("User:alice", "*", ResourceTypeTopic, "test-topic", OperationWrite))
	assert.True(t, auth.Authorize("User:alice", "*", ResourceTypeTopic, "test-topic", OperationDelete))
	assert.True(t, auth.Authorize("User:alice", "*", ResourceTypeTopic, "test-topic", OperationCreate))
}

func TestAuthorizerCacheExpiration(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		Enabled:      true,
		DataDir:      tmpDir,
		CacheEnabled: true,
		CacheTTL:     100 * time.Millisecond, // Short TTL for testing
		CacheSize:    100,
	}

	auth, err := NewAuthorizer(cfg)
	require.NoError(t, err)

	// Add ACL entry
	entry, _ := NewEntry("User:alice", "*", ResourceTypeTopic,
		"test-topic", PatternTypeLiteral, OperationRead, PermissionTypeAllow)
	auth.AddACL(entry)

	// First call - cache miss
	auth.Authorize("User:alice", "*", ResourceTypeTopic, "test-topic", OperationRead)

	// Second call immediately - cache hit
	auth.Authorize("User:alice", "*", ResourceTypeTopic, "test-topic", OperationRead)

	stats1 := auth.Stats()
	assert.Equal(t, int64(1), stats1.CacheMissCount)
	assert.Equal(t, int64(1), stats1.CacheHitCount)

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Third call after expiration - cache miss
	auth.Authorize("User:alice", "*", ResourceTypeTopic, "test-topic", OperationRead)

	stats2 := auth.Stats()
	assert.Equal(t, int64(2), stats2.CacheMissCount)
	assert.Equal(t, int64(1), stats2.CacheHitCount)
}
