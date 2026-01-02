// Copyright 2025 Takhin Data, Inc.

package acl

import (
	"fmt"
	"sync"
	"time"
)

// authCache implements an LRU cache for authorization decisions
type authCache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
	ttl     time.Duration
	maxSize int
}

type cacheEntry struct {
	allowed   bool
	timestamp time.Time
}

func newAuthCache(maxSize int, ttl time.Duration) *authCache {
	return &authCache{
		entries: make(map[string]*cacheEntry),
		ttl:     ttl,
		maxSize: maxSize,
	}
}

func (c *authCache) key(principal, host string, resourceType ResourceType,
	resourceName string, operation Operation) string {
	return fmt.Sprintf("%s|%s|%d|%s|%d", principal, host, resourceType, resourceName, operation)
}

func (c *authCache) get(principal, host string, resourceType ResourceType,
	resourceName string, operation Operation) (bool, bool) {

	c.mu.RLock()
	defer c.mu.RUnlock()

	key := c.key(principal, host, resourceType, resourceName, operation)
	entry, ok := c.entries[key]
	if !ok {
		return false, false
	}

	// Check if entry has expired
	if time.Since(entry.timestamp) > c.ttl {
		return false, false
	}

	return entry.allowed, true
}

func (c *authCache) put(principal, host string, resourceType ResourceType,
	resourceName string, operation Operation, allowed bool) {

	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict oldest entries if cache is full
	if len(c.entries) >= c.maxSize {
		c.evictOldest()
	}

	key := c.key(principal, host, resourceType, resourceName, operation)
	c.entries[key] = &cacheEntry{
		allowed:   allowed,
		timestamp: time.Now(),
	}
}

func (c *authCache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*cacheEntry)
}

func (c *authCache) evictOldest() {
	// Find and remove oldest entry
	var oldestKey string
	var oldestTime time.Time
	first := true

	for key, entry := range c.entries {
		if first || entry.timestamp.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.timestamp
			first = false
		}
	}

	if oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}
