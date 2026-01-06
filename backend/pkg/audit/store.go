// Copyright 2025 Takhin Data, Inc.

package audit

import (
	"sync"
	"time"
)

// Store provides in-memory queryable storage for audit logs
type Store struct {
	mu         sync.RWMutex
	events     []*Event
	retention  int64 // Retention period in milliseconds
	indexByPrincipal map[string][]*Event
	indexByResource  map[string][]*Event
}

// NewStore creates a new audit log store
func NewStore(retentionMs int64) *Store {
	return &Store{
		events:           make([]*Event, 0, 10000),
		retention:        retentionMs,
		indexByPrincipal: make(map[string][]*Event),
		indexByResource:  make(map[string][]*Event),
	}
}

// Add adds an event to the store
func (s *Store) Add(event *Event) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.events = append(s.events, event)

	// Update indices
	if event.Principal != "" {
		s.indexByPrincipal[event.Principal] = append(s.indexByPrincipal[event.Principal], event)
	}
	if event.ResourceName != "" {
		key := event.ResourceType + ":" + event.ResourceName
		s.indexByResource[key] = append(s.indexByResource[key], event)
	}
}

// Query queries events with filters
func (s *Store) Query(filter Filter) []*Event {
	s.mu.RLock()
	defer s.mu.RUnlock()

	results := make([]*Event, 0)

	// Use indices if possible
	var candidates []*Event
	if len(filter.Principals) == 1 {
		candidates = s.indexByPrincipal[filter.Principals[0]]
	} else if filter.ResourceType != "" && filter.ResourceName != "" {
		key := filter.ResourceType + ":" + filter.ResourceName
		candidates = s.indexByResource[key]
	} else {
		candidates = s.events
	}

	// Apply filters
	for _, event := range candidates {
		if s.matches(event, filter) {
			results = append(results, event)
		}
	}

	// Apply limit and offset
	if filter.Offset > 0 && filter.Offset < len(results) {
		results = results[filter.Offset:]
	}
	if filter.Limit > 0 && filter.Limit < len(results) {
		results = results[:filter.Limit]
	}

	return results
}

// matches checks if an event matches the filter
func (s *Store) matches(event *Event, filter Filter) bool {
	// Time range filter
	if filter.StartTime != nil && event.Timestamp.Before(*filter.StartTime) {
		return false
	}
	if filter.EndTime != nil && event.Timestamp.After(*filter.EndTime) {
		return false
	}

	// Event type filter
	if len(filter.EventTypes) > 0 {
		found := false
		for _, et := range filter.EventTypes {
			if event.EventType == et {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Principal filter
	if len(filter.Principals) > 0 {
		found := false
		for _, p := range filter.Principals {
			if event.Principal == p {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Resource filter
	if filter.ResourceType != "" && event.ResourceType != filter.ResourceType {
		return false
	}
	if filter.ResourceName != "" && event.ResourceName != filter.ResourceName {
		return false
	}

	// Result filter
	if filter.Result != "" && event.Result != filter.Result {
		return false
	}

	// Severity filter
	if filter.Severity != "" && event.Severity != filter.Severity {
		return false
	}

	return true
}

// Cleanup removes old events based on retention period
func (s *Store) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-time.Duration(s.retention) * time.Millisecond)

	// Find the first event to keep
	keepFrom := -1
	for i, event := range s.events {
		if event.Timestamp.After(cutoff) {
			keepFrom = i
			break
		}
	}

	// All events are old
	if keepFrom == -1 {
		s.events = make([]*Event, 0, 10000)
		s.indexByPrincipal = make(map[string][]*Event)
		s.indexByResource = make(map[string][]*Event)
		return
	}

	// No old events
	if keepFrom == 0 {
		return
	}

	// Remove old events
	s.events = s.events[keepFrom:]

	// Rebuild indices
	s.indexByPrincipal = make(map[string][]*Event)
	s.indexByResource = make(map[string][]*Event)

	for _, event := range s.events {
		if event.Principal != "" {
			s.indexByPrincipal[event.Principal] = append(s.indexByPrincipal[event.Principal], event)
		}
		if event.ResourceName != "" {
			key := event.ResourceType + ":" + event.ResourceName
			s.indexByResource[key] = append(s.indexByResource[key], event)
		}
	}
}

// Count returns the total number of events in the store
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.events)
}

// Clear removes all events from the store
func (s *Store) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.events = make([]*Event, 0, 10000)
	s.indexByPrincipal = make(map[string][]*Event)
	s.indexByResource = make(map[string][]*Event)
}
