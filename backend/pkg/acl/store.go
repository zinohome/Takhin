// Copyright 2025 Takhin Data, Inc.

package acl

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Store manages ACL entries with thread-safe operations
type Store struct {
	mu      sync.RWMutex
	entries []Entry
	dataDir string
}

// NewStore creates a new ACL store
func NewStore(dataDir string) *Store {
	return &Store{
		entries: make([]Entry, 0),
		dataDir: dataDir,
	}
}

// Load loads ACL entries from disk
func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	aclFile := filepath.Join(s.dataDir, "acls.json")
	data, err := os.ReadFile(aclFile)
	if err != nil {
		if os.IsNotExist(err) {
			// No ACL file yet, start with empty
			return nil
		}
		return fmt.Errorf("read ACL file: %w", err)
	}

	if err := json.Unmarshal(data, &s.entries); err != nil {
		return fmt.Errorf("unmarshal ACL entries: %w", err)
	}

	return nil
}

// Save persists ACL entries to disk
func (s *Store) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	aclFile := filepath.Join(s.dataDir, "acls.json")

	// Ensure directory exists
	if err := os.MkdirAll(s.dataDir, 0755); err != nil {
		return fmt.Errorf("create ACL directory: %w", err)
	}

	data, err := json.MarshalIndent(s.entries, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal ACL entries: %w", err)
	}

	if err := os.WriteFile(aclFile, data, 0644); err != nil {
		return fmt.Errorf("write ACL file: %w", err)
	}

	return nil
}

// Add adds a new ACL entry
func (s *Store) Add(entry Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for duplicate
	for _, e := range s.entries {
		if entryEquals(e, entry) {
			return fmt.Errorf("ACL entry already exists")
		}
	}

	s.entries = append(s.entries, entry)
	return nil
}

// Delete removes ACL entries matching the filter
func (s *Store) Delete(filter Filter) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var remaining []Entry
	deleted := 0

	for _, entry := range s.entries {
		if !matchesFilter(entry, filter) {
			remaining = append(remaining, entry)
		} else {
			deleted++
		}
	}

	s.entries = remaining
	return deleted, nil
}

// List returns all ACL entries matching the filter
func (s *Store) List(filter Filter) []Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []Entry
	for _, entry := range s.entries {
		if matchesFilter(entry, filter) {
			result = append(result, entry)
		}
	}

	return result
}

// GetAll returns all ACL entries
func (s *Store) GetAll() []Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Entry, len(s.entries))
	copy(result, s.entries)
	return result
}

// entryEquals checks if two ACL entries are equal
func entryEquals(a, b Entry) bool {
	return a.Principal == b.Principal &&
		a.Host == b.Host &&
		a.ResourceType == b.ResourceType &&
		a.ResourceName == b.ResourceName &&
		a.PatternType == b.PatternType &&
		a.Operation == b.Operation &&
		a.PermissionType == b.PermissionType
}

// matchesFilter checks if an entry matches the filter
func matchesFilter(entry Entry, filter Filter) bool {
	// Check resource filter
	if entry.ResourceType != filter.ResourceFilter.ResourceType {
		return false
	}

	if filter.ResourceFilter.ResourceName != nil {
		if entry.ResourceName != *filter.ResourceFilter.ResourceName {
			return false
		}
	}

	if filter.ResourceFilter.PatternType != nil {
		if entry.PatternType != *filter.ResourceFilter.PatternType {
			return false
		}
	}

	// Check access filter
	if filter.AccessFilter.Principal != nil {
		if entry.Principal != *filter.AccessFilter.Principal {
			return false
		}
	}

	if filter.AccessFilter.Host != nil {
		if entry.Host != *filter.AccessFilter.Host {
			return false
		}
	}

	if filter.AccessFilter.Operation != nil {
		if entry.Operation != *filter.AccessFilter.Operation {
			return false
		}
	}

	if filter.AccessFilter.PermissionType != nil {
		if entry.PermissionType != *filter.AccessFilter.PermissionType {
			return false
		}
	}

	return true
}

// matchesResource checks if a resource name matches an ACL entry
func matchesResource(entryName string, patternType PatternType, resourceName string) bool {
	switch patternType {
	case PatternTypeLiteral:
		return entryName == resourceName
	case PatternTypePrefixed:
		return strings.HasPrefix(resourceName, entryName)
	default:
		return false
	}
}
