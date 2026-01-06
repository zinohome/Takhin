// Copyright 2025 Takhin Data, Inc.

package schema

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileStorage(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := NewFileStorage(tempDir)
	require.NoError(t, err)
	defer storage.Close()

	t.Run("SaveAndGetSchema", func(t *testing.T) {
		schema := &Schema{
			Subject:    "test-subject",
			Version:    1,
			SchemaType: SchemaTypeAvro,
			Schema:     `{"type":"record","name":"Test","fields":[{"name":"field1","type":"string"}]}`,
		}

		err := storage.SaveSchema(schema)
		assert.NoError(t, err)
		assert.NotEqual(t, 0, schema.ID)

		retrieved, err := storage.GetSchema(schema.ID)
		assert.NoError(t, err)
		assert.Equal(t, schema.Subject, retrieved.Subject)
		assert.Equal(t, schema.Version, retrieved.Version)
		assert.Equal(t, schema.Schema, retrieved.Schema)
	})

	t.Run("GetSchemaBySubjectVersion", func(t *testing.T) {
		schema := &Schema{
			Subject:    "test-subject-2",
			Version:    1,
			SchemaType: SchemaTypeJSON,
			Schema:     `{"type":"object","properties":{"name":{"type":"string"}}}`,
		}

		err := storage.SaveSchema(schema)
		require.NoError(t, err)

		retrieved, err := storage.GetSchemaBySubjectVersion("test-subject-2", 1)
		assert.NoError(t, err)
		assert.Equal(t, schema.Subject, retrieved.Subject)
		assert.Equal(t, schema.Version, retrieved.Version)
	})

	t.Run("GetLatestSchema", func(t *testing.T) {
		subject := "test-subject-3"

		schema1 := &Schema{
			Subject:    subject,
			Version:    1,
			SchemaType: SchemaTypeAvro,
			Schema:     `{"type":"string"}`,
		}
		err := storage.SaveSchema(schema1)
		require.NoError(t, err)

		schema2 := &Schema{
			Subject:    subject,
			Version:    2,
			SchemaType: SchemaTypeAvro,
			Schema:     `{"type":"int"}`,
		}
		err = storage.SaveSchema(schema2)
		require.NoError(t, err)

		latest, err := storage.GetLatestSchema(subject)
		assert.NoError(t, err)
		assert.Equal(t, 2, latest.Version)
		assert.Equal(t, schema2.Schema, latest.Schema)
	})

	t.Run("GetAllVersions", func(t *testing.T) {
		subject := "test-subject-4"

		for i := 1; i <= 3; i++ {
			schema := &Schema{
				Subject:    subject,
				Version:    i,
				SchemaType: SchemaTypeAvro,
				Schema:     `{"type":"string"}`,
			}
			err := storage.SaveSchema(schema)
			require.NoError(t, err)
		}

		versions, err := storage.GetAllVersions(subject)
		assert.NoError(t, err)
		assert.Equal(t, []int{1, 2, 3}, versions)
	})

	t.Run("DeleteSchemaVersion", func(t *testing.T) {
		subject := "test-subject-5"

		schema := &Schema{
			Subject:    subject,
			Version:    1,
			SchemaType: SchemaTypeAvro,
			Schema:     `{"type":"string"}`,
		}
		err := storage.SaveSchema(schema)
		require.NoError(t, err)

		err = storage.DeleteSchemaVersion(subject, 1)
		assert.NoError(t, err)

		_, err = storage.GetSchemaBySubjectVersion(subject, 1)
		assert.Error(t, err)
	})

	t.Run("DeleteSubject", func(t *testing.T) {
		subject := "test-subject-6"

		for i := 1; i <= 3; i++ {
			schema := &Schema{
				Subject:    subject,
				Version:    i,
				SchemaType: SchemaTypeAvro,
				Schema:     `{"type":"string"}`,
			}
			err := storage.SaveSchema(schema)
			require.NoError(t, err)
		}

		versions, err := storage.DeleteSubject(subject)
		assert.NoError(t, err)
		assert.Equal(t, []int{1, 2, 3}, versions)

		_, err = storage.GetAllVersions(subject)
		assert.Error(t, err)
	})

	t.Run("SaveAndGetCompatibility", func(t *testing.T) {
		subject := "test-subject-7"

		err := storage.SaveCompatibility(subject, CompatibilityBackward)
		assert.NoError(t, err)

		mode, err := storage.GetCompatibility(subject)
		assert.NoError(t, err)
		assert.Equal(t, CompatibilityBackward, mode)
	})

	t.Run("Persistence", func(t *testing.T) {
		tempDir2 := t.TempDir()
		storage2, err := NewFileStorage(tempDir2)
		require.NoError(t, err)

		schema := &Schema{
			Subject:    "persist-test",
			Version:    1,
			SchemaType: SchemaTypeAvro,
			Schema:     `{"type":"string"}`,
		}
		err = storage2.SaveSchema(schema)
		require.NoError(t, err)

		storage2.Close()

		storage3, err := NewFileStorage(tempDir2)
		require.NoError(t, err)
		defer storage3.Close()

		retrieved, err := storage3.GetSchemaBySubjectVersion("persist-test", 1)
		assert.NoError(t, err)
		assert.Equal(t, schema.Schema, retrieved.Schema)
	})
}

func TestSchemaValidator(t *testing.T) {
	validator := &DefaultSchemaValidator{}

	t.Run("ValidJSON", func(t *testing.T) {
		schema := `{"type":"object","properties":{"name":{"type":"string"}}}`
		err := validator.Validate(schema, SchemaTypeJSON)
		assert.NoError(t, err)
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		schema := `{invalid json}`
		err := validator.Validate(schema, SchemaTypeJSON)
		assert.Error(t, err)
	})

	t.Run("ValidAvro", func(t *testing.T) {
		schema := `{"type":"record","name":"Test","fields":[{"name":"field1","type":"string"}]}`
		err := validator.Validate(schema, SchemaTypeAvro)
		assert.NoError(t, err)
	})

	t.Run("InvalidAvro", func(t *testing.T) {
		schema := `{"name":"Test","fields":[{"name":"field1","type":"string"}]}`
		err := validator.Validate(schema, SchemaTypeAvro)
		assert.Error(t, err)
	})

	t.Run("ValidProtobuf", func(t *testing.T) {
		schema := `syntax = "proto3"; message Test { string field1 = 1; }`
		err := validator.Validate(schema, SchemaTypeProtobuf)
		assert.NoError(t, err)
	})

	t.Run("EmptyProtobuf", func(t *testing.T) {
		schema := ``
		err := validator.Validate(schema, SchemaTypeProtobuf)
		assert.Error(t, err)
	})
}

func TestCompatibilityChecker(t *testing.T) {
	checker := NewDefaultCompatibilityChecker()

	avroSchema1 := `{"type":"record","name":"Test","fields":[{"name":"field1","type":"string"}]}`
	avroSchema2 := `{"type":"record","name":"Test","fields":[{"name":"field1","type":"string"},{"name":"field2","type":"string","default":""}]}`
	avroSchema3 := `{"type":"record","name":"Test","fields":[{"name":"field1","type":"string"},{"name":"field2","type":"string"}]}`

	t.Run("BackwardCompatible", func(t *testing.T) {
		err := checker.CheckCompatibility(avroSchema2, []string{avroSchema1}, CompatibilityBackward)
		assert.NoError(t, err)
	})

	t.Run("BackwardIncompatible", func(t *testing.T) {
		err := checker.CheckCompatibility(avroSchema1, []string{avroSchema2}, CompatibilityBackward)
		assert.Error(t, err)
	})

	t.Run("ForwardCompatible", func(t *testing.T) {
		err := checker.CheckCompatibility(avroSchema2, []string{avroSchema1}, CompatibilityForward)
		assert.NoError(t, err)
	})

	t.Run("ForwardIncompatible", func(t *testing.T) {
		err := checker.CheckCompatibility(avroSchema3, []string{avroSchema1}, CompatibilityForward)
		assert.Error(t, err)
	})

	t.Run("FullCompatible", func(t *testing.T) {
		err := checker.CheckCompatibility(avroSchema2, []string{avroSchema1}, CompatibilityFull)
		assert.NoError(t, err)
	})

	t.Run("FullIncompatible", func(t *testing.T) {
		err := checker.CheckCompatibility(avroSchema3, []string{avroSchema1}, CompatibilityFull)
		assert.Error(t, err)
	})

	t.Run("NoneMode", func(t *testing.T) {
		err := checker.CheckCompatibility(avroSchema3, []string{avroSchema1}, CompatibilityNone)
		assert.NoError(t, err)
	})
}

func TestSchemaCache(t *testing.T) {
	cache := NewSchemaCache(3)

	schema1 := &Schema{ID: 1, Subject: "test1", Version: 1}
	schema2 := &Schema{ID: 2, Subject: "test2", Version: 1}
	schema3 := &Schema{ID: 3, Subject: "test3", Version: 1}
	schema4 := &Schema{ID: 4, Subject: "test4", Version: 1}

	t.Run("PutAndGet", func(t *testing.T) {
		cache.Put(1, schema1)
		retrieved := cache.Get(1)
		assert.NotNil(t, retrieved)
		assert.Equal(t, schema1.ID, retrieved.ID)
	})

	t.Run("LRUEviction", func(t *testing.T) {
		cache.Put(2, schema2)
		cache.Put(3, schema3)
		cache.Put(4, schema4)

		assert.Nil(t, cache.Get(1))
		assert.NotNil(t, cache.Get(2))
		assert.NotNil(t, cache.Get(3))
		assert.NotNil(t, cache.Get(4))
	})

	t.Run("Delete", func(t *testing.T) {
		cache.Delete(2)
		assert.Nil(t, cache.Get(2))
	})
}

func TestStorageErrors(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := NewFileStorage(tempDir)
	require.NoError(t, err)
	defer storage.Close()

	t.Run("SubjectNotFound", func(t *testing.T) {
		_, err := storage.GetSchemaBySubjectVersion("non-existent", 1)
		assert.Error(t, err)
		schemaErr, ok := err.(*SchemaError)
		assert.True(t, ok)
		assert.Equal(t, ErrCodeSubjectNotFound, schemaErr.ErrorCode)
	})

	t.Run("VersionNotFound", func(t *testing.T) {
		schema := &Schema{
			Subject:    "test",
			Version:    1,
			SchemaType: SchemaTypeAvro,
			Schema:     `{"type":"string"}`,
		}
		err := storage.SaveSchema(schema)
		require.NoError(t, err)

		_, err = storage.GetSchemaBySubjectVersion("test", 99)
		assert.Error(t, err)
		schemaErr, ok := err.(*SchemaError)
		assert.True(t, ok)
		assert.Equal(t, ErrCodeVersionNotFound, schemaErr.ErrorCode)
	})

	t.Run("SchemaNotFound", func(t *testing.T) {
		_, err := storage.GetSchema(99999)
		assert.Error(t, err)
		schemaErr, ok := err.(*SchemaError)
		assert.True(t, ok)
		assert.Equal(t, ErrCodeSchemaNotFound, schemaErr.ErrorCode)
	})
}

func TestStorageFilePermissions(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := NewFileStorage(tempDir)
	require.NoError(t, err)

	schema := &Schema{
		Subject:    "test",
		Version:    1,
		SchemaType: SchemaTypeAvro,
		Schema:     `{"type":"string"}`,
	}
	err = storage.SaveSchema(schema)
	require.NoError(t, err)

	storage.Close()

	filePath := filepath.Join(tempDir, "schemas.json")
	info, err := os.Stat(filePath)
	require.NoError(t, err)

	assert.Equal(t, os.FileMode(0644), info.Mode().Perm())
}
