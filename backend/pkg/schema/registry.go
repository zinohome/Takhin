// Copyright 2025 Takhin Data, Inc.

package schema

import (
	"fmt"
	"sync"
	"time"
)

// Registry manages schema registration and retrieval
type Registry struct {
	storage              Storage
	compatibilityChecker CompatibilityChecker
	validator            SchemaValidator
	defaultCompatibility CompatibilityMode
	mu                   sync.RWMutex
	cache                *SchemaCache
}

// NewRegistry creates a new schema registry
func NewRegistry(cfg *Config) (*Registry, error) {
	storage, err := NewFileStorage(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}

	defaultCompat := cfg.DefaultCompatibility
	if defaultCompat == "" {
		defaultCompat = CompatibilityBackward
	}

	cacheSize := cfg.CacheSize
	if cacheSize <= 0 {
		cacheSize = 1000
	}

	return &Registry{
		storage:              storage,
		compatibilityChecker: NewDefaultCompatibilityChecker(),
		validator:            &DefaultSchemaValidator{},
		defaultCompatibility: defaultCompat,
		cache:                NewSchemaCache(cacheSize),
	}, nil
}

// RegisterSchema registers a new schema version
func (r *Registry) RegisterSchema(subject, schema string, schemaType SchemaType, references []SchemaReference) (*Schema, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.validator.Validate(schema, schemaType); err != nil {
		return nil, err
	}

	existingSchemas, err := r.getExistingSchemasForCompatibility(subject)
	if err != nil && err.(*SchemaError).ErrorCode != ErrCodeSubjectNotFound {
		return nil, err
	}

	compatMode, err := r.storage.GetCompatibility(subject)
	if err != nil {
		compatMode = r.defaultCompatibility
	}

	if err := r.compatibilityChecker.CheckCompatibility(schema, existingSchemas, compatMode); err != nil {
		return nil, err
	}

	nextVersion := r.getNextVersion(subject)

	newSchema := &Schema{
		Subject:    subject,
		Version:    nextVersion,
		SchemaType: schemaType,
		Schema:     schema,
		References: references,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := r.storage.SaveSchema(newSchema); err != nil {
		return nil, fmt.Errorf("failed to save schema: %w", err)
	}

	r.cache.Put(newSchema.ID, newSchema)

	return newSchema, nil
}

// GetSchemaByID retrieves a schema by its ID
func (r *Registry) GetSchemaByID(id int) (*Schema, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if cached := r.cache.Get(id); cached != nil {
		return cached, nil
	}

	schema, err := r.storage.GetSchema(id)
	if err != nil {
		return nil, err
	}

	r.cache.Put(id, schema)
	return schema, nil
}

// GetSchemaBySubjectVersion retrieves a schema by subject and version
func (r *Registry) GetSchemaBySubjectVersion(subject string, version int) (*Schema, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.storage.GetSchemaBySubjectVersion(subject, version)
}

// GetLatestSchema retrieves the latest schema for a subject
func (r *Registry) GetLatestSchema(subject string) (*Schema, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.storage.GetLatestSchema(subject)
}

// GetAllVersions retrieves all versions for a subject
func (r *Registry) GetAllVersions(subject string) ([]int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.storage.GetAllVersions(subject)
}

// GetSubjects retrieves all subjects
func (r *Registry) GetSubjects() ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.storage.GetSubjects()
}

// DeleteSchemaVersion deletes a specific schema version
func (r *Registry) DeleteSchemaVersion(subject string, version int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	schema, err := r.storage.GetSchemaBySubjectVersion(subject, version)
	if err != nil {
		return err
	}

	r.cache.Delete(schema.ID)
	return r.storage.DeleteSchemaVersion(subject, version)
}

// DeleteSubject deletes all versions of a subject
func (r *Registry) DeleteSubject(subject string) ([]int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	schemas, _ := r.storage.GetAllVersions(subject)
	for _, version := range schemas {
		if schema, err := r.storage.GetSchemaBySubjectVersion(subject, version); err == nil {
			r.cache.Delete(schema.ID)
		}
	}

	return r.storage.DeleteSubject(subject)
}

// SetCompatibility sets the compatibility mode for a subject
func (r *Registry) SetCompatibility(subject string, mode CompatibilityMode) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.storage.SaveCompatibility(subject, mode)
}

// GetCompatibility retrieves the compatibility mode for a subject
func (r *Registry) GetCompatibility(subject string) (CompatibilityMode, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	mode, err := r.storage.GetCompatibility(subject)
	if err != nil {
		return r.defaultCompatibility, nil
	}
	return mode, nil
}

// TestCompatibility tests if a schema is compatible without registering it
func (r *Registry) TestCompatibility(subject, schema string, schemaType SchemaType, version int) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if err := r.validator.Validate(schema, schemaType); err != nil {
		return false, err
	}

	var existingSchemas []string
	if version > 0 {
		existingSchema, err := r.storage.GetSchemaBySubjectVersion(subject, version)
		if err != nil {
			return false, err
		}
		existingSchemas = []string{existingSchema.Schema}
	} else {
		var err error
		existingSchemas, err = r.getExistingSchemasForCompatibility(subject)
		if err != nil && err.(*SchemaError).ErrorCode != ErrCodeSubjectNotFound {
			return false, err
		}
	}

	compatMode, err := r.storage.GetCompatibility(subject)
	if err != nil {
		compatMode = r.defaultCompatibility
	}

	if err := r.compatibilityChecker.CheckCompatibility(schema, existingSchemas, compatMode); err != nil {
		return false, nil
	}

	return true, nil
}

// Close closes the registry
func (r *Registry) Close() error {
	return r.storage.Close()
}

// getExistingSchemasForCompatibility retrieves existing schemas for compatibility checking
func (r *Registry) getExistingSchemasForCompatibility(subject string) ([]string, error) {
	versions, err := r.storage.GetAllVersions(subject)
	if err != nil {
		return nil, err
	}

	schemas := make([]string, 0, len(versions))
	for _, version := range versions {
		schema, err := r.storage.GetSchemaBySubjectVersion(subject, version)
		if err != nil {
			continue
		}
		schemas = append(schemas, schema.Schema)
	}

	return schemas, nil
}

// getNextVersion determines the next version number for a subject
func (r *Registry) getNextVersion(subject string) int {
	versions, err := r.storage.GetAllVersions(subject)
	if err != nil || len(versions) == 0 {
		return 1
	}

	maxVersion := 0
	for _, v := range versions {
		if v > maxVersion {
			maxVersion = v
		}
	}

	return maxVersion + 1
}

// SchemaCache provides LRU caching for schemas
type SchemaCache struct {
	mu      sync.RWMutex
	cache   map[int]*Schema
	lruList []int
	maxSize int
}

// NewSchemaCache creates a new schema cache
func NewSchemaCache(maxSize int) *SchemaCache {
	return &SchemaCache{
		cache:   make(map[int]*Schema),
		lruList: make([]int, 0, maxSize),
		maxSize: maxSize,
	}
}

// Get retrieves a schema from cache
func (c *SchemaCache) Get(id int) *Schema {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.cache[id]
}

// Put adds a schema to cache
func (c *SchemaCache) Put(id int, schema *Schema) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.cache) >= c.maxSize && c.cache[id] == nil {
		oldest := c.lruList[0]
		delete(c.cache, oldest)
		c.lruList = c.lruList[1:]
	}

	c.cache[id] = schema
	c.lruList = append(c.lruList, id)
}

// Delete removes a schema from cache
func (c *SchemaCache) Delete(id int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.cache, id)
	for i, cachedID := range c.lruList {
		if cachedID == id {
			c.lruList = append(c.lruList[:i], c.lruList[i+1:]...)
			break
		}
	}
}
