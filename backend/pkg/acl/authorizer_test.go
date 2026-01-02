// Copyright 2025 Takhin Data, Inc.

package acl

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthorizerDisabled(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	authorizer := NewAuthorizer(store, false)

	// When disabled, should allow everything
	allowed := authorizer.Authorize("User:alice", "192.168.1.1", ResourceTypeTopic, "test-topic", OperationRead)
	assert.True(t, allowed)
}

func TestAuthorizerBasicAllow(t *testing.T) {
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

	require.NoError(t, store.Add(entry))

	authorizer := NewAuthorizer(store, true)

	// Should allow alice to read test-topic
	allowed := authorizer.Authorize("User:alice", "192.168.1.1", ResourceTypeTopic, "test-topic", OperationRead)
	assert.True(t, allowed)

	// Should deny alice to write test-topic (no permission)
	allowed = authorizer.Authorize("User:alice", "192.168.1.1", ResourceTypeTopic, "test-topic", OperationWrite)
	assert.False(t, allowed)

	// Should deny bob to read test-topic (no permission)
	allowed = authorizer.Authorize("User:bob", "192.168.1.1", ResourceTypeTopic, "test-topic", OperationRead)
	assert.False(t, allowed)
}

func TestAuthorizerDenyTakesPrecedence(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	allowEntry := Entry{
		Principal:      "User:alice",
		Host:           "*",
		ResourceType:   ResourceTypeTopic,
		ResourceName:   "test-topic",
		PatternType:    PatternTypeLiteral,
		Operation:      OperationAll,
		PermissionType: PermissionTypeAllow,
	}

	denyEntry := Entry{
		Principal:      "User:alice",
		Host:           "*",
		ResourceType:   ResourceTypeTopic,
		ResourceName:   "test-topic",
		PatternType:    PatternTypeLiteral,
		Operation:      OperationWrite,
		PermissionType: PermissionTypeDeny,
	}

	require.NoError(t, store.Add(allowEntry))
	require.NoError(t, store.Add(denyEntry))

	authorizer := NewAuthorizer(store, true)

	// Should allow read (only allow exists)
	allowed := authorizer.Authorize("User:alice", "192.168.1.1", ResourceTypeTopic, "test-topic", OperationRead)
	assert.True(t, allowed)

	// Should deny write (explicit deny)
	allowed = authorizer.Authorize("User:alice", "192.168.1.1", ResourceTypeTopic, "test-topic", OperationWrite)
	assert.False(t, allowed)
}

func TestAuthorizerWildcardPrincipal(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	entry := Entry{
		Principal:      "*",
		Host:           "*",
		ResourceType:   ResourceTypeTopic,
		ResourceName:   "public-topic",
		PatternType:    PatternTypeLiteral,
		Operation:      OperationRead,
		PermissionType: PermissionTypeAllow,
	}

	require.NoError(t, store.Add(entry))

	authorizer := NewAuthorizer(store, true)

	// Should allow any user to read public-topic
	allowed := authorizer.Authorize("User:alice", "192.168.1.1", ResourceTypeTopic, "public-topic", OperationRead)
	assert.True(t, allowed)

	allowed = authorizer.Authorize("User:bob", "192.168.1.1", ResourceTypeTopic, "public-topic", OperationRead)
	assert.True(t, allowed)
}

func TestAuthorizerHostFilter(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	entry := Entry{
		Principal:      "User:alice",
		Host:           "192.168.1.100",
		ResourceType:   ResourceTypeTopic,
		ResourceName:   "secure-topic",
		PatternType:    PatternTypeLiteral,
		Operation:      OperationRead,
		PermissionType: PermissionTypeAllow,
	}

	require.NoError(t, store.Add(entry))

	authorizer := NewAuthorizer(store, true)

	// Should allow from specific host
	allowed := authorizer.Authorize("User:alice", "192.168.1.100", ResourceTypeTopic, "secure-topic", OperationRead)
	assert.True(t, allowed)

	// Should deny from different host
	allowed = authorizer.Authorize("User:alice", "192.168.1.200", ResourceTypeTopic, "secure-topic", OperationRead)
	assert.False(t, allowed)
}

func TestAuthorizerPrefixPattern(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	entry := Entry{
		Principal:      "User:alice",
		Host:           "*",
		ResourceType:   ResourceTypeTopic,
		ResourceName:   "test-",
		PatternType:    PatternTypePrefixed,
		Operation:      OperationRead,
		PermissionType: PermissionTypeAllow,
	}

	require.NoError(t, store.Add(entry))

	authorizer := NewAuthorizer(store, true)

	// Should allow topics with prefix
	allowed := authorizer.Authorize("User:alice", "192.168.1.1", ResourceTypeTopic, "test-topic1", OperationRead)
	assert.True(t, allowed)

	allowed = authorizer.Authorize("User:alice", "192.168.1.1", ResourceTypeTopic, "test-topic2", OperationRead)
	assert.True(t, allowed)

	// Should deny topics without prefix
	allowed = authorizer.Authorize("User:alice", "192.168.1.1", ResourceTypeTopic, "prod-topic", OperationRead)
	assert.False(t, allowed)
}

func TestAuthorizerOperationAll(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	entry := Entry{
		Principal:      "User:admin",
		Host:           "*",
		ResourceType:   ResourceTypeTopic,
		ResourceName:   "admin-topic",
		PatternType:    PatternTypeLiteral,
		Operation:      OperationAll,
		PermissionType: PermissionTypeAllow,
	}

	require.NoError(t, store.Add(entry))

	authorizer := NewAuthorizer(store, true)

	// Should allow all operations
	operations := []Operation{
		OperationRead,
		OperationWrite,
		OperationDelete,
		OperationCreate,
		OperationAlter,
		OperationDescribe,
	}

	for _, op := range operations {
		allowed := authorizer.Authorize("User:admin", "192.168.1.1", ResourceTypeTopic, "admin-topic", op)
		assert.True(t, allowed, "Should allow operation: %s", op)
	}
}

func TestAuthorizerGroupResource(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	entry := Entry{
		Principal:      "User:consumer",
		Host:           "*",
		ResourceType:   ResourceTypeGroup,
		ResourceName:   "my-group",
		PatternType:    PatternTypeLiteral,
		Operation:      OperationRead,
		PermissionType: PermissionTypeAllow,
	}

	require.NoError(t, store.Add(entry))

	authorizer := NewAuthorizer(store, true)

	// Should allow group access
	allowed := authorizer.Authorize("User:consumer", "192.168.1.1", ResourceTypeGroup, "my-group", OperationRead)
	assert.True(t, allowed)

	// Should deny different resource type
	allowed = authorizer.Authorize("User:consumer", "192.168.1.1", ResourceTypeTopic, "my-group", OperationRead)
	assert.False(t, allowed)
}

func TestAuthorizerComplexScenario(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	entries := []Entry{
		// Alice can read all test- topics
		{
			Principal:      "User:alice",
			Host:           "*",
			ResourceType:   ResourceTypeTopic,
			ResourceName:   "test-",
			PatternType:    PatternTypePrefixed,
			Operation:      OperationRead,
			PermissionType: PermissionTypeAllow,
		},
		// Alice can write to test-alice-* topics
		{
			Principal:      "User:alice",
			Host:           "*",
			ResourceType:   ResourceTypeTopic,
			ResourceName:   "test-alice-",
			PatternType:    PatternTypePrefixed,
			Operation:      OperationWrite,
			PermissionType: PermissionTypeAllow,
		},
		// Deny alice from sensitive topic
		{
			Principal:      "User:alice",
			Host:           "*",
			ResourceType:   ResourceTypeTopic,
			ResourceName:   "test-sensitive",
			PatternType:    PatternTypeLiteral,
			Operation:      OperationRead,
			PermissionType: PermissionTypeDeny,
		},
	}

	for _, entry := range entries {
		require.NoError(t, store.Add(entry))
	}

	authorizer := NewAuthorizer(store, true)

	// Can read test- topics
	allowed := authorizer.Authorize("User:alice", "192.168.1.1", ResourceTypeTopic, "test-topic1", OperationRead)
	assert.True(t, allowed)

	// Cannot read test-sensitive (explicit deny)
	allowed = authorizer.Authorize("User:alice", "192.168.1.1", ResourceTypeTopic, "test-sensitive", OperationRead)
	assert.False(t, allowed)

	// Can write to test-alice-* topics
	allowed = authorizer.Authorize("User:alice", "192.168.1.1", ResourceTypeTopic, "test-alice-private", OperationWrite)
	assert.True(t, allowed)

	// Cannot write to test-bob-* topics
	allowed = authorizer.Authorize("User:alice", "192.168.1.1", ResourceTypeTopic, "test-bob-private", OperationWrite)
	assert.False(t, allowed)
}
