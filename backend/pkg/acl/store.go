// Copyright 2025 Takhin Data, Inc.

package acl

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Store manages ACL persistence and in-memory cache
type Store struct {
	mu      sync.RWMutex
	entries map[string]*Entry // Key: Entry.Key() -> Entry
	dataDir string
}

// NewStore creates a new ACL store
func NewStore(dataDir string) (*Store, error) {
	store := &Store{
		entries: make(map[string]*Entry),
		dataDir: dataDir,
	}

	// Create ACL directory if it doesn't exist
	aclDir := filepath.Join(dataDir, "acls")
	if err := os.MkdirAll(aclDir, 0755); err != nil {
		return nil, fmt.Errorf("create ACL directory: %w", err)
	}

	// Load existing ACLs from disk
	if err := store.load(); err != nil {
		return nil, fmt.Errorf("load ACLs: %w", err)
	}

	return store, nil
}

// Add adds a new ACL entry
func (s *Store) Add(entry *Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := entry.Key()
	s.entries[key] = entry

	// Persist to disk
	if err := s.persist(); err != nil {
		return fmt.Errorf("persist ACL: %w", err)
	}

	return nil
}

// Delete removes an ACL entry
func (s *Store) Delete(filter *Filter) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	deleted := 0
	for key, entry := range s.entries {
		if filter.Matches(entry) {
			delete(s.entries, key)
			deleted++
		}
	}

	if deleted > 0 {
		// Persist to disk
		if err := s.persist(); err != nil {
			return deleted, fmt.Errorf("persist ACL: %w", err)
		}
	}

	return deleted, nil
}

// List returns all ACL entries matching the filter
func (s *Store) List(filter *Filter) []*Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Entry
	for _, entry := range s.entries {
		if filter.Matches(entry) {
			// Create a copy to avoid concurrent modification
			entryCopy := *entry
			result = append(result, &entryCopy)
		}
	}

	return result
}

// Check checks if an operation is authorized
func (s *Store) Check(principal, host string, resourceType ResourceType,
	resourceName string, operation Operation) bool {

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Deny takes precedence over allow
	var hasAllow bool
	for _, entry := range s.entries {
		if entry.Matches(principal, host, resourceType, resourceName, operation) {
			if entry.PermissionType == PermissionTypeDeny {
				return false
			}
			if entry.PermissionType == PermissionTypeAllow {
				hasAllow = true
			}
		}
	}

	return hasAllow
}

// persist writes all ACL entries to disk
func (s *Store) persist() error {
	aclFile := filepath.Join(s.dataDir, "acls", "acls.json")

	// Convert map to slice for JSON serialization
	entries := make([]*Entry, 0, len(s.entries))
	for _, entry := range s.entries {
		entries = append(entries, entry)
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal ACLs: %w", err)
	}

	// Write to temp file first, then rename for atomic write
	tempFile := aclFile + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}

	if err := os.Rename(tempFile, aclFile); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}

	return nil
}

// load reads ACL entries from disk
func (s *Store) load() error {
	aclFile := filepath.Join(s.dataDir, "acls", "acls.json")

	// If file doesn't exist, start with empty ACL
	if _, err := os.Stat(aclFile); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(aclFile)
	if err != nil {
		return fmt.Errorf("read ACL file: %w", err)
	}

	var entries []*Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("unmarshal ACLs: %w", err)
	}

	// Load into map
	for _, entry := range entries {
		s.entries[entry.Key()] = entry
	}

	return nil
}

// Count returns the total number of ACL entries
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entries)
}
