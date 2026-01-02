// Copyright 2025 Takhin Data, Inc.

package acl

import (
	"sync/atomic"
	"time"
)

// Authorizer provides ACL-based authorization with caching
type Authorizer struct {
	store          *Store
	enabled        atomic.Bool
	cache          *authCache
	cacheEnabled   bool
	cacheTTL       time.Duration
	denyCount      atomic.Int64
	allowCount     atomic.Int64
	cacheHitCount  atomic.Int64
	cacheMissCount atomic.Int64
}

// Config holds Authorizer configuration
type Config struct {
	Enabled      bool
	DataDir      string
	CacheEnabled bool
	CacheTTL     time.Duration
	CacheSize    int
}

// NewAuthorizer creates a new ACL authorizer
func NewAuthorizer(cfg Config) (*Authorizer, error) {
	store, err := NewStore(cfg.DataDir)
	if err != nil {
		return nil, err
	}

	var cache *authCache
	if cfg.CacheEnabled {
		if cfg.CacheTTL == 0 {
			cfg.CacheTTL = 5 * time.Minute
		}
		if cfg.CacheSize == 0 {
			cfg.CacheSize = 10000
		}
		cache = newAuthCache(cfg.CacheSize, cfg.CacheTTL)
	}

	a := &Authorizer{
		store:        store,
		cache:        cache,
		cacheEnabled: cfg.CacheEnabled,
		cacheTTL:     cfg.CacheTTL,
	}
	a.enabled.Store(cfg.Enabled)

	return a, nil
}

// Enable enables ACL authorization
func (a *Authorizer) Enable() {
	a.enabled.Store(true)
}

// Disable disables ACL authorization
func (a *Authorizer) Disable() {
	a.enabled.Store(false)
}

// IsEnabled returns whether ACL is enabled
func (a *Authorizer) IsEnabled() bool {
	return a.enabled.Load()
}

// Authorize checks if the principal is authorized to perform the operation
func (a *Authorizer) Authorize(principal, host string, resourceType ResourceType,
	resourceName string, operation Operation) bool {

	// If ACL is disabled, allow everything
	if !a.enabled.Load() {
		return true
	}

	// Check cache first if enabled
	if a.cacheEnabled && a.cache != nil {
		if allowed, ok := a.cache.get(principal, host, resourceType, resourceName, operation); ok {
			a.cacheHitCount.Add(1)
			if allowed {
				a.allowCount.Add(1)
			} else {
				a.denyCount.Add(1)
			}
			return allowed
		}
		a.cacheMissCount.Add(1)
	}

	// Check store
	allowed := a.store.Check(principal, host, resourceType, resourceName, operation)

	// Update cache
	if a.cacheEnabled && a.cache != nil {
		a.cache.put(principal, host, resourceType, resourceName, operation, allowed)
	}

	// Update metrics
	if allowed {
		a.allowCount.Add(1)
	} else {
		a.denyCount.Add(1)
	}

	return allowed
}

// AddACL adds a new ACL entry
func (a *Authorizer) AddACL(entry *Entry) error {
	// Clear cache when ACL changes
	if a.cacheEnabled && a.cache != nil {
		a.cache.clear()
	}
	return a.store.Add(entry)
}

// DeleteACL removes ACL entries matching the filter
func (a *Authorizer) DeleteACL(filter *Filter) (int, error) {
	// Clear cache when ACL changes
	if a.cacheEnabled && a.cache != nil {
		a.cache.clear()
	}
	return a.store.Delete(filter)
}

// ListACL returns ACL entries matching the filter
func (a *Authorizer) ListACL(filter *Filter) []*Entry {
	return a.store.List(filter)
}

// Stats returns authorization statistics
func (a *Authorizer) Stats() AuthStats {
	return AuthStats{
		Enabled:        a.enabled.Load(),
		TotalACLs:      a.store.Count(),
		AllowCount:     a.allowCount.Load(),
		DenyCount:      a.denyCount.Load(),
		CacheHitCount:  a.cacheHitCount.Load(),
		CacheMissCount: a.cacheMissCount.Load(),
	}
}

// AuthStats holds authorization statistics
type AuthStats struct {
	Enabled        bool
	TotalACLs      int
	AllowCount     int64
	DenyCount      int64
	CacheHitCount  int64
	CacheMissCount int64
}
