// Copyright 2025 Takhin Data, Inc.

package schema

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// Storage handles persistent storage of schemas
type Storage interface {
	SaveSchema(schema *Schema) error
	GetSchema(id int) (*Schema, error)
	GetSchemaBySubjectVersion(subject string, version int) (*Schema, error)
	GetLatestSchema(subject string) (*Schema, error)
	GetAllVersions(subject string) ([]int, error)
	GetSubjects() ([]string, error)
	DeleteSchemaVersion(subject string, version int) error
	DeleteSubject(subject string) ([]int, error)
	SaveCompatibility(subject string, mode CompatibilityMode) error
	GetCompatibility(subject string) (CompatibilityMode, error)
	Close() error
}

// FileStorage implements Storage using filesystem
type FileStorage struct {
	dataDir          string
	mu               sync.RWMutex
	schemasByID      map[int]*Schema
	schemasBySubject map[string]map[int]*Schema // subject -> version -> schema
	compatibility    map[string]CompatibilityMode
	nextID           int
}

// NewFileStorage creates a new file-based storage
func NewFileStorage(dataDir string) (*FileStorage, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	fs := &FileStorage{
		dataDir:          dataDir,
		schemasByID:      make(map[int]*Schema),
		schemasBySubject: make(map[string]map[int]*Schema),
		compatibility:    make(map[string]CompatibilityMode),
		nextID:           1,
	}

	if err := fs.load(); err != nil {
		return nil, fmt.Errorf("failed to load schemas: %w", err)
	}

	return fs, nil
}

// SaveSchema persists a schema
func (fs *FileStorage) SaveSchema(schema *Schema) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if schema.ID == 0 {
		schema.ID = fs.nextID
		fs.nextID++
	}

	fs.schemasByID[schema.ID] = schema

	if _, exists := fs.schemasBySubject[schema.Subject]; !exists {
		fs.schemasBySubject[schema.Subject] = make(map[int]*Schema)
	}
	fs.schemasBySubject[schema.Subject][schema.Version] = schema

	return fs.persist()
}

// GetSchema retrieves a schema by ID
func (fs *FileStorage) GetSchema(id int) (*Schema, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	schema, exists := fs.schemasByID[id]
	if !exists {
		return nil, NewSchemaError(ErrCodeSchemaNotFound, fmt.Sprintf("schema with ID %d not found", id))
	}

	return schema, nil
}

// GetSchemaBySubjectVersion retrieves a schema by subject and version
func (fs *FileStorage) GetSchemaBySubjectVersion(subject string, version int) (*Schema, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	versions, exists := fs.schemasBySubject[subject]
	if !exists {
		return nil, NewSchemaError(ErrCodeSubjectNotFound, fmt.Sprintf("subject %s not found", subject))
	}

	schema, exists := versions[version]
	if !exists {
		return nil, NewSchemaError(ErrCodeVersionNotFound, fmt.Sprintf("version %d not found for subject %s", version, subject))
	}

	return schema, nil
}

// GetLatestSchema retrieves the latest schema for a subject
func (fs *FileStorage) GetLatestSchema(subject string) (*Schema, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	versions, exists := fs.schemasBySubject[subject]
	if !exists || len(versions) == 0 {
		return nil, NewSchemaError(ErrCodeSubjectNotFound, fmt.Sprintf("subject %s not found", subject))
	}

	maxVersion := 0
	for v := range versions {
		if v > maxVersion {
			maxVersion = v
		}
	}

	return versions[maxVersion], nil
}

// GetAllVersions retrieves all versions for a subject
func (fs *FileStorage) GetAllVersions(subject string) ([]int, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	versions, exists := fs.schemasBySubject[subject]
	if !exists {
		return nil, NewSchemaError(ErrCodeSubjectNotFound, fmt.Sprintf("subject %s not found", subject))
	}

	result := make([]int, 0, len(versions))
	for v := range versions {
		result = append(result, v)
	}
	sort.Ints(result)

	return result, nil
}

// GetSubjects retrieves all subjects
func (fs *FileStorage) GetSubjects() ([]string, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	subjects := make([]string, 0, len(fs.schemasBySubject))
	for subject := range fs.schemasBySubject {
		subjects = append(subjects, subject)
	}
	sort.Strings(subjects)

	return subjects, nil
}

// DeleteSchemaVersion deletes a specific version of a schema
func (fs *FileStorage) DeleteSchemaVersion(subject string, version int) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	versions, exists := fs.schemasBySubject[subject]
	if !exists {
		return NewSchemaError(ErrCodeSubjectNotFound, fmt.Sprintf("subject %s not found", subject))
	}

	schema, exists := versions[version]
	if !exists {
		return NewSchemaError(ErrCodeVersionNotFound, fmt.Sprintf("version %d not found for subject %s", version, subject))
	}

	delete(versions, version)
	delete(fs.schemasByID, schema.ID)

	if len(versions) == 0 {
		delete(fs.schemasBySubject, subject)
	}

	return fs.persist()
}

// DeleteSubject deletes all versions of a subject
func (fs *FileStorage) DeleteSubject(subject string) ([]int, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	versions, exists := fs.schemasBySubject[subject]
	if !exists {
		return nil, NewSchemaError(ErrCodeSubjectNotFound, fmt.Sprintf("subject %s not found", subject))
	}

	deletedVersions := make([]int, 0, len(versions))
	for v, schema := range versions {
		deletedVersions = append(deletedVersions, v)
		delete(fs.schemasByID, schema.ID)
	}

	delete(fs.schemasBySubject, subject)
	delete(fs.compatibility, subject)

	sort.Ints(deletedVersions)

	return deletedVersions, fs.persist()
}

// SaveCompatibility saves compatibility mode for a subject
func (fs *FileStorage) SaveCompatibility(subject string, mode CompatibilityMode) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	fs.compatibility[subject] = mode
	return fs.persist()
}

// GetCompatibility retrieves compatibility mode for a subject
func (fs *FileStorage) GetCompatibility(subject string) (CompatibilityMode, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	if mode, exists := fs.compatibility[subject]; exists {
		return mode, nil
	}

	return "", NewSchemaError(ErrCodeSubjectLevelNotFound, fmt.Sprintf("compatibility level not found for subject %s", subject))
}

// persist writes data to disk
func (fs *FileStorage) persist() error {
	data := struct {
		Schemas       map[int]*Schema              `json:"schemas"`
		Subjects      map[string]map[int]*Schema   `json:"subjects"`
		Compatibility map[string]CompatibilityMode `json:"compatibility"`
		NextID        int                          `json:"nextId"`
	}{
		Schemas:       fs.schemasByID,
		Subjects:      fs.schemasBySubject,
		Compatibility: fs.compatibility,
		NextID:        fs.nextID,
	}

	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	filePath := filepath.Join(fs.dataDir, "schemas.json")
	if err := os.WriteFile(filePath, bytes, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// load reads data from disk
func (fs *FileStorage) load() error {
	filePath := filepath.Join(fs.dataDir, "schemas.json")

	bytes, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read file: %w", err)
	}

	var data struct {
		Schemas       map[int]*Schema              `json:"schemas"`
		Subjects      map[string]map[int]*Schema   `json:"subjects"`
		Compatibility map[string]CompatibilityMode `json:"compatibility"`
		NextID        int                          `json:"nextId"`
	}

	if err := json.Unmarshal(bytes, &data); err != nil {
		return fmt.Errorf("failed to unmarshal data: %w", err)
	}

	fs.schemasByID = data.Schemas
	fs.schemasBySubject = data.Subjects
	fs.compatibility = data.Compatibility
	fs.nextID = data.NextID

	if fs.schemasByID == nil {
		fs.schemasByID = make(map[int]*Schema)
	}
	if fs.schemasBySubject == nil {
		fs.schemasBySubject = make(map[string]map[int]*Schema)
	}
	if fs.compatibility == nil {
		fs.compatibility = make(map[string]CompatibilityMode)
	}

	return nil
}

// Close closes the storage
func (fs *FileStorage) Close() error {
	return nil
}
