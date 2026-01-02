// Copyright 2025 Takhin Data, Inc.

package acl

import (
	"testing"
)

// BenchmarkAuthorizerDisabled benchmarks authorization when disabled
func BenchmarkAuthorizerDisabled(b *testing.B) {
	tmpDir := b.TempDir()
	store := NewStore(tmpDir)
	authorizer := NewAuthorizer(store, false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		authorizer.Authorize("User:alice", "192.168.1.1", ResourceTypeTopic, "test-topic", OperationRead)
	}
}

// BenchmarkAuthorizerSingleACL benchmarks authorization with a single ACL entry
func BenchmarkAuthorizerSingleACL(b *testing.B) {
	tmpDir := b.TempDir()
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
	store.Add(entry)

	authorizer := NewAuthorizer(store, true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		authorizer.Authorize("User:alice", "192.168.1.1", ResourceTypeTopic, "test-topic", OperationRead)
	}
}

// BenchmarkAuthorizerMultipleACLs benchmarks authorization with multiple ACL entries
func BenchmarkAuthorizerMultipleACLs(b *testing.B) {
	tmpDir := b.TempDir()
	store := NewStore(tmpDir)

	// Add 100 ACL entries
	for i := 0; i < 100; i++ {
		entry := Entry{
			Principal:      "User:alice",
			Host:           "*",
			ResourceType:   ResourceTypeTopic,
			ResourceName:   "test-topic",
			PatternType:    PatternTypeLiteral,
			Operation:      Operation(i % 8),
			PermissionType: PermissionTypeAllow,
		}
		store.Add(entry)
	}

	authorizer := NewAuthorizer(store, true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		authorizer.Authorize("User:alice", "192.168.1.1", ResourceTypeTopic, "test-topic", OperationRead)
	}
}

// BenchmarkAuthorizerPrefixPattern benchmarks authorization with prefix patterns
func BenchmarkAuthorizerPrefixPattern(b *testing.B) {
	tmpDir := b.TempDir()
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
	store.Add(entry)

	authorizer := NewAuthorizer(store, true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		authorizer.Authorize("User:alice", "192.168.1.1", ResourceTypeTopic, "test-topic-123", OperationRead)
	}
}

// BenchmarkStoreAdd benchmarks adding ACL entries
func BenchmarkStoreAdd(b *testing.B) {
	tmpDir := b.TempDir()
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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Clear store for each iteration
		store.entries = make([]Entry, 0)
		store.Add(entry)
	}
}

// BenchmarkStoreList benchmarks listing ACL entries
func BenchmarkStoreList(b *testing.B) {
	tmpDir := b.TempDir()
	store := NewStore(tmpDir)

	// Add 100 ACL entries
	for i := 0; i < 100; i++ {
		entry := Entry{
			Principal:      "User:alice",
			Host:           "*",
			ResourceType:   ResourceTypeTopic,
			ResourceName:   "test-topic",
			PatternType:    PatternTypeLiteral,
			Operation:      Operation(i % 8),
			PermissionType: PermissionTypeAllow,
		}
		store.Add(entry)
	}

	filter := Filter{
		ResourceFilter: ResourceFilter{
			ResourceType: ResourceTypeTopic,
		},
		AccessFilter: AccessFilter{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.List(filter)
	}
}

// BenchmarkStoreDelete benchmarks deleting ACL entries
func BenchmarkStoreDelete(b *testing.B) {
	tmpDir := b.TempDir()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		store := NewStore(tmpDir)

		// Add entries
		for j := 0; j < 10; j++ {
			entry := Entry{
				Principal:      "User:alice",
				Host:           "*",
				ResourceType:   ResourceTypeTopic,
				ResourceName:   "test-topic",
				PatternType:    PatternTypeLiteral,
				Operation:      Operation(j % 8),
				PermissionType: PermissionTypeAllow,
			}
			store.Add(entry)
		}

		principal := "User:alice"
		filter := Filter{
			ResourceFilter: ResourceFilter{
				ResourceType: ResourceTypeTopic,
			},
			AccessFilter: AccessFilter{
				Principal: &principal,
			},
		}
		b.StartTimer()

		store.Delete(filter)
	}
}

// BenchmarkStoreSave benchmarks saving ACL entries to disk
func BenchmarkStoreSave(b *testing.B) {
	tmpDir := b.TempDir()
	store := NewStore(tmpDir)

	// Add 100 ACL entries
	for i := 0; i < 100; i++ {
		entry := Entry{
			Principal:      "User:alice",
			Host:           "*",
			ResourceType:   ResourceTypeTopic,
			ResourceName:   "test-topic",
			PatternType:    PatternTypeLiteral,
			Operation:      Operation(i % 8),
			PermissionType: PermissionTypeAllow,
		}
		store.Add(entry)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Save()
	}
}

// BenchmarkStoreLoad benchmarks loading ACL entries from disk
func BenchmarkStoreLoad(b *testing.B) {
	tmpDir := b.TempDir()
	store := NewStore(tmpDir)

	// Add and save 100 ACL entries
	for i := 0; i < 100; i++ {
		entry := Entry{
			Principal:      "User:alice",
			Host:           "*",
			ResourceType:   ResourceTypeTopic,
			ResourceName:   "test-topic",
			PatternType:    PatternTypeLiteral,
			Operation:      Operation(i % 8),
			PermissionType: PermissionTypeAllow,
		}
		store.Add(entry)
	}
	store.Save()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		newStore := NewStore(tmpDir)
		newStore.Load()
	}
}
