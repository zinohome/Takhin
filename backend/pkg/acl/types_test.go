// Copyright 2025 Takhin Data, Inc.

package acl

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEntry(t *testing.T) {
	tests := []struct {
		name          string
		principal     string
		host          string
		resourceType  ResourceType
		resourceName  string
		patternType   ResourcePatternType
		operation     Operation
		permission    PermissionType
		expectError   bool
		errorContains string
	}{
		{
			name:         "valid entry",
			principal:    "User:alice",
			host:         "192.168.1.1",
			resourceType: ResourceTypeTopic,
			resourceName: "test-topic",
			patternType:  PatternTypeLiteral,
			operation:    OperationRead,
			permission:   PermissionTypeAllow,
			expectError:  false,
		},
		{
			name:          "empty principal",
			principal:     "",
			host:          "*",
			resourceType:  ResourceTypeTopic,
			resourceName:  "test-topic",
			patternType:   PatternTypeLiteral,
			operation:     OperationRead,
			permission:    PermissionTypeAllow,
			expectError:   true,
			errorContains: "principal cannot be empty",
		},
		{
			name:          "invalid principal prefix",
			principal:     "alice",
			host:          "*",
			resourceType:  ResourceTypeTopic,
			resourceName:  "test-topic",
			patternType:   PatternTypeLiteral,
			operation:     OperationRead,
			permission:    PermissionTypeAllow,
			expectError:   true,
			errorContains: "must start with 'User:' prefix",
		},
		{
			name:          "unknown resource type",
			principal:     "User:alice",
			host:          "*",
			resourceType:  ResourceTypeUnknown,
			resourceName:  "test-topic",
			patternType:   PatternTypeLiteral,
			operation:     OperationRead,
			permission:    PermissionTypeAllow,
			expectError:   true,
			errorContains: "invalid resource type",
		},
		{
			name:          "empty resource name",
			principal:     "User:alice",
			host:          "*",
			resourceType:  ResourceTypeTopic,
			resourceName:  "",
			patternType:   PatternTypeLiteral,
			operation:     OperationRead,
			permission:    PermissionTypeAllow,
			expectError:   true,
			errorContains: "resource name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := NewEntry(tt.principal, tt.host, tt.resourceType,
				tt.resourceName, tt.patternType, tt.operation, tt.permission)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Nil(t, entry)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, entry)
				assert.Equal(t, tt.principal, entry.Principal)
				if tt.host == "" {
					assert.Equal(t, "*", entry.Host)
				} else {
					assert.Equal(t, tt.host, entry.Host)
				}
			}
		})
	}
}

func TestEntryMatches(t *testing.T) {
	tests := []struct {
		name         string
		entry        *Entry
		principal    string
		host         string
		resourceType ResourceType
		resourceName string
		operation    Operation
		expected     bool
	}{
		{
			name: "exact match",
			entry: &Entry{
				Principal:      "User:alice",
				Host:           "192.168.1.1",
				ResourceType:   ResourceTypeTopic,
				ResourceName:   "test-topic",
				PatternType:    PatternTypeLiteral,
				Operation:      OperationRead,
				PermissionType: PermissionTypeAllow,
			},
			principal:    "User:alice",
			host:         "192.168.1.1",
			resourceType: ResourceTypeTopic,
			resourceName: "test-topic",
			operation:    OperationRead,
			expected:     true,
		},
		{
			name: "wildcard principal",
			entry: &Entry{
				Principal:      "User:*",
				Host:           "*",
				ResourceType:   ResourceTypeTopic,
				ResourceName:   "test-topic",
				PatternType:    PatternTypeLiteral,
				Operation:      OperationRead,
				PermissionType: PermissionTypeAllow,
			},
			principal:    "User:alice",
			host:         "192.168.1.1",
			resourceType: ResourceTypeTopic,
			resourceName: "test-topic",
			operation:    OperationRead,
			expected:     true,
		},
		{
			name: "prefix pattern match",
			entry: &Entry{
				Principal:      "User:alice",
				Host:           "*",
				ResourceType:   ResourceTypeTopic,
				ResourceName:   "test-",
				PatternType:    PatternTypePrefix,
				Operation:      OperationRead,
				PermissionType: PermissionTypeAllow,
			},
			principal:    "User:alice",
			host:         "192.168.1.1",
			resourceType: ResourceTypeTopic,
			resourceName: "test-topic-1",
			operation:    OperationRead,
			expected:     true,
		},
		{
			name: "prefix pattern no match",
			entry: &Entry{
				Principal:      "User:alice",
				Host:           "*",
				ResourceType:   ResourceTypeTopic,
				ResourceName:   "test-",
				PatternType:    PatternTypePrefix,
				Operation:      OperationRead,
				PermissionType: PermissionTypeAllow,
			},
			principal:    "User:alice",
			host:         "192.168.1.1",
			resourceType: ResourceTypeTopic,
			resourceName: "prod-topic",
			operation:    OperationRead,
			expected:     false,
		},
		{
			name: "operation all matches any operation",
			entry: &Entry{
				Principal:      "User:alice",
				Host:           "*",
				ResourceType:   ResourceTypeTopic,
				ResourceName:   "test-topic",
				PatternType:    PatternTypeLiteral,
				Operation:      OperationAll,
				PermissionType: PermissionTypeAllow,
			},
			principal:    "User:alice",
			host:         "192.168.1.1",
			resourceType: ResourceTypeTopic,
			resourceName: "test-topic",
			operation:    OperationWrite,
			expected:     true,
		},
		{
			name: "different principal no match",
			entry: &Entry{
				Principal:      "User:alice",
				Host:           "*",
				ResourceType:   ResourceTypeTopic,
				ResourceName:   "test-topic",
				PatternType:    PatternTypeLiteral,
				Operation:      OperationRead,
				PermissionType: PermissionTypeAllow,
			},
			principal:    "User:bob",
			host:         "192.168.1.1",
			resourceType: ResourceTypeTopic,
			resourceName: "test-topic",
			operation:    OperationRead,
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.entry.Matches(tt.principal, tt.host, tt.resourceType,
				tt.resourceName, tt.operation)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilter(t *testing.T) {
	alice := "User:alice"
	topicType := ResourceTypeTopic
	testTopic := "test-topic"
	literal := PatternTypeLiteral
	readOp := OperationRead
	allow := PermissionTypeAllow

	entry := &Entry{
		Principal:      alice,
		Host:           "*",
		ResourceType:   topicType,
		ResourceName:   testTopic,
		PatternType:    literal,
		Operation:      readOp,
		PermissionType: allow,
	}

	tests := []struct {
		name     string
		filter   *Filter
		expected bool
	}{
		{
			name:     "empty filter matches all",
			filter:   &Filter{},
			expected: true,
		},
		{
			name: "filter by principal",
			filter: &Filter{
				Principal: &alice,
			},
			expected: true,
		},
		{
			name: "filter by principal no match",
			filter: &Filter{
				Principal: func() *string { s := "User:bob"; return &s }(),
			},
			expected: false,
		},
		{
			name: "filter by resource type",
			filter: &Filter{
				ResourceType: &topicType,
			},
			expected: true,
		},
		{
			name: "filter by resource name",
			filter: &Filter{
				ResourceName: &testTopic,
			},
			expected: true,
		},
		{
			name: "filter multiple criteria match",
			filter: &Filter{
				Principal:    &alice,
				ResourceType: &topicType,
				Operation:    &readOp,
			},
			expected: true,
		},
		{
			name: "filter multiple criteria no match",
			filter: &Filter{
				Principal: &alice,
				Operation: func() *Operation { op := OperationWrite; return &op }(),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.filter.Matches(entry)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResourceTypeString(t *testing.T) {
	tests := []struct {
		resourceType ResourceType
		expected     string
	}{
		{ResourceTypeTopic, "Topic"},
		{ResourceTypeGroup, "Group"},
		{ResourceTypeCluster, "Cluster"},
		{ResourceTypeUnknown, "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.resourceType.String())
		})
	}
}

func TestOperationString(t *testing.T) {
	tests := []struct {
		operation Operation
		expected  string
	}{
		{OperationAll, "All"},
		{OperationRead, "Read"},
		{OperationWrite, "Write"},
		{OperationCreate, "Create"},
		{OperationDelete, "Delete"},
		{OperationAlter, "Alter"},
		{OperationDescribe, "Describe"},
		{OperationUnknown, "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.operation.String())
		})
	}
}

func TestEntryKey(t *testing.T) {
	entry, err := NewEntry("User:alice", "192.168.1.1", ResourceTypeTopic,
		"test-topic", PatternTypeLiteral, OperationRead, PermissionTypeAllow)
	require.NoError(t, err)

	key := entry.Key()
	assert.NotEmpty(t, key)
	assert.Contains(t, key, "User:alice")
	assert.Contains(t, key, "192.168.1.1")
	assert.Contains(t, key, "test-topic")

	// Create another entry with same properties
	entry2, err := NewEntry("User:alice", "192.168.1.1", ResourceTypeTopic,
		"test-topic", PatternTypeLiteral, OperationRead, PermissionTypeAllow)
	require.NoError(t, err)

	// Keys should be identical
	assert.Equal(t, key, entry2.Key())

	// Entry with different operation should have different key
	entry3, err := NewEntry("User:alice", "192.168.1.1", ResourceTypeTopic,
		"test-topic", PatternTypeLiteral, OperationWrite, PermissionTypeAllow)
	require.NoError(t, err)
	assert.NotEqual(t, key, entry3.Key())
}
