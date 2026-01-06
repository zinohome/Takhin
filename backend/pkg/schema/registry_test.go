// Copyright 2025 Takhin Data, Inc.

package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &Config{
		DataDir:              tempDir,
		DefaultCompatibility: CompatibilityBackward,
		MaxVersions:          100,
		CacheSize:            1000,
	}

	registry, err := NewRegistry(cfg)
	require.NoError(t, err)
	defer registry.Close()

	t.Run("RegisterAndGetSchema", func(t *testing.T) {
		schema := `{"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}`

		registered, err := registry.RegisterSchema("user", schema, SchemaTypeAvro, nil)
		assert.NoError(t, err)
		assert.NotEqual(t, 0, registered.ID)
		assert.Equal(t, "user", registered.Subject)
		assert.Equal(t, 1, registered.Version)

		retrieved, err := registry.GetSchemaByID(registered.ID)
		assert.NoError(t, err)
		assert.Equal(t, schema, retrieved.Schema)
	})

	t.Run("RegisterMultipleVersions", func(t *testing.T) {
		schema1 := `{"type":"record","name":"Product","fields":[{"name":"id","type":"string"}]}`
		schema2 := `{"type":"record","name":"Product","fields":[{"name":"id","type":"string"},{"name":"name","type":"string","default":""}]}`

		v1, err := registry.RegisterSchema("product", schema1, SchemaTypeAvro, nil)
		assert.NoError(t, err)
		assert.Equal(t, 1, v1.Version)

		v2, err := registry.RegisterSchema("product", schema2, SchemaTypeAvro, nil)
		assert.NoError(t, err)
		assert.Equal(t, 2, v2.Version)
	})

	t.Run("GetLatestSchema", func(t *testing.T) {
		schema1 := `{"type":"record","name":"Order","fields":[{"name":"id","type":"string"}]}`
		schema2 := `{"type":"record","name":"Order","fields":[{"name":"id","type":"string"},{"name":"total","type":"double","default":0.0}]}`

		_, err := registry.RegisterSchema("order", schema1, SchemaTypeAvro, nil)
		require.NoError(t, err)

		v2, err := registry.RegisterSchema("order", schema2, SchemaTypeAvro, nil)
		require.NoError(t, err)

		latest, err := registry.GetLatestSchema("order")
		assert.NoError(t, err)
		assert.Equal(t, v2.Version, latest.Version)
		assert.Equal(t, schema2, latest.Schema)
	})

	t.Run("GetAllVersions", func(t *testing.T) {
		subject := "versioned"

		for i := 0; i < 3; i++ {
			schema := `{"type":"string"}`
			_, err := registry.RegisterSchema(subject, schema, SchemaTypeAvro, nil)
			require.NoError(t, err)
		}

		versions, err := registry.GetAllVersions(subject)
		assert.NoError(t, err)
		assert.Equal(t, 3, len(versions))
		assert.Equal(t, []int{1, 2, 3}, versions)
	})

	t.Run("GetSubjects", func(t *testing.T) {
		subjects := []string{"subject1", "subject2", "subject3"}

		for _, subject := range subjects {
			schema := `{"type":"string"}`
			_, err := registry.RegisterSchema(subject, schema, SchemaTypeAvro, nil)
			require.NoError(t, err)
		}

		allSubjects, err := registry.GetSubjects()
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(allSubjects), 3)

		for _, subject := range subjects {
			assert.Contains(t, allSubjects, subject)
		}
	})

	t.Run("DeleteSchemaVersion", func(t *testing.T) {
		schema := `{"type":"string"}`
		registered, err := registry.RegisterSchema("deletable", schema, SchemaTypeAvro, nil)
		require.NoError(t, err)

		err = registry.DeleteSchemaVersion("deletable", registered.Version)
		assert.NoError(t, err)

		_, err = registry.GetSchemaBySubjectVersion("deletable", registered.Version)
		assert.Error(t, err)
	})

	t.Run("DeleteSubject", func(t *testing.T) {
		subject := "delete-subject"

		for i := 0; i < 3; i++ {
			schema := `{"type":"string"}`
			_, err := registry.RegisterSchema(subject, schema, SchemaTypeAvro, nil)
			require.NoError(t, err)
		}

		versions, err := registry.DeleteSubject(subject)
		assert.NoError(t, err)
		assert.Equal(t, []int{1, 2, 3}, versions)

		_, err = registry.GetAllVersions(subject)
		assert.Error(t, err)
	})

	t.Run("SetAndGetCompatibility", func(t *testing.T) {
		subject := "compat-test"

		err := registry.SetCompatibility(subject, CompatibilityFull)
		assert.NoError(t, err)

		mode, err := registry.GetCompatibility(subject)
		assert.NoError(t, err)
		assert.Equal(t, CompatibilityFull, mode)
	})

	t.Run("TestCompatibility", func(t *testing.T) {
		subject := "test-compat"
		schema1 := `{"type":"record","name":"Test","fields":[{"name":"field1","type":"string"},{"name":"field2","type":"string"}]}`
		schema2 := `{"type":"record","name":"Test","fields":[{"name":"field1","type":"string"},{"name":"field2","type":"string"},{"name":"field3","type":"string","default":""}]}`
		schema3 := `{"type":"record","name":"Test","fields":[{"name":"field1","type":"string"}]}`

		_, err := registry.RegisterSchema(subject, schema1, SchemaTypeAvro, nil)
		require.NoError(t, err)

		compatible, err := registry.TestCompatibility(subject, schema2, SchemaTypeAvro, 0)
		assert.NoError(t, err)
		assert.True(t, compatible)

		compatible, err = registry.TestCompatibility(subject, schema3, SchemaTypeAvro, 0)
		assert.NoError(t, err)
		assert.False(t, compatible)
	})

	t.Run("IncompatibleSchemaRejection", func(t *testing.T) {
		subject := "reject-test"
		schema1 := `{"type":"record","name":"Test","fields":[{"name":"field1","type":"string"},{"name":"field2","type":"string"}]}`
		schema2 := `{"type":"record","name":"Test","fields":[{"name":"field1","type":"string"}]}`

		_, err := registry.RegisterSchema(subject, schema1, SchemaTypeAvro, nil)
		require.NoError(t, err)

		_, err = registry.RegisterSchema(subject, schema2, SchemaTypeAvro, nil)
		assert.Error(t, err)
		schemaErr, ok := err.(*SchemaError)
		assert.True(t, ok)
		assert.Equal(t, ErrCodeIncompatibleSchema, schemaErr.ErrorCode)
	})

	t.Run("InvalidSchemaRejection", func(t *testing.T) {
		invalidSchema := `{invalid json}`

		_, err := registry.RegisterSchema("invalid", invalidSchema, SchemaTypeJSON, nil)
		assert.Error(t, err)
		schemaErr, ok := err.(*SchemaError)
		assert.True(t, ok)
		assert.Equal(t, ErrCodeInvalidSchema, schemaErr.ErrorCode)
	})
}

func TestRegistryWithReferences(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &Config{
		DataDir:              tempDir,
		DefaultCompatibility: CompatibilityBackward,
		CacheSize:            1000,
	}

	registry, err := NewRegistry(cfg)
	require.NoError(t, err)
	defer registry.Close()

	t.Run("RegisterSchemaWithReferences", func(t *testing.T) {
		baseSchema := `{"type":"record","name":"Address","fields":[{"name":"street","type":"string"}]}`
		_, err := registry.RegisterSchema("address", baseSchema, SchemaTypeAvro, nil)
		require.NoError(t, err)

		userSchema := `{"type":"record","name":"User","fields":[{"name":"name","type":"string"}]}`
		refs := []SchemaReference{
			{Name: "Address", Subject: "address", Version: 1},
		}

		registered, err := registry.RegisterSchema("user-with-ref", userSchema, SchemaTypeAvro, refs)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(registered.References))
		assert.Equal(t, "address", registered.References[0].Subject)
	})
}

func TestRegistryCache(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &Config{
		DataDir:              tempDir,
		DefaultCompatibility: CompatibilityBackward,
		CacheSize:            2,
	}

	registry, err := NewRegistry(cfg)
	require.NoError(t, err)
	defer registry.Close()

	t.Run("CacheHit", func(t *testing.T) {
		schema := `{"type":"string"}`
		registered, err := registry.RegisterSchema("cache-test", schema, SchemaTypeAvro, nil)
		require.NoError(t, err)

		retrieved1, err := registry.GetSchemaByID(registered.ID)
		assert.NoError(t, err)

		retrieved2, err := registry.GetSchemaByID(registered.ID)
		assert.NoError(t, err)

		assert.Equal(t, retrieved1, retrieved2)
	})
}

func TestRegistryDefaultCompatibility(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &Config{
		DataDir:              tempDir,
		DefaultCompatibility: CompatibilityFull,
		CacheSize:            1000,
	}

	registry, err := NewRegistry(cfg)
	require.NoError(t, err)
	defer registry.Close()

	t.Run("UsesDefaultCompatibility", func(t *testing.T) {
		subject := "default-compat"

		mode, err := registry.GetCompatibility(subject)
		assert.NoError(t, err)
		assert.Equal(t, CompatibilityFull, mode)
	})
}

func TestRegistryVersioning(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &Config{
		DataDir:              tempDir,
		DefaultCompatibility: CompatibilityBackward,
		CacheSize:            1000,
	}

	registry, err := NewRegistry(cfg)
	require.NoError(t, err)
	defer registry.Close()

	t.Run("VersionIncrement", func(t *testing.T) {
		subject := "version-test"

		for i := 1; i <= 5; i++ {
			schema := `{"type":"string"}`
			registered, err := registry.RegisterSchema(subject, schema, SchemaTypeAvro, nil)
			assert.NoError(t, err)
			assert.Equal(t, i, registered.Version)
		}
	})

	t.Run("GetSpecificVersion", func(t *testing.T) {
		subject := "specific-version"

		schemas := []string{
			`{"type":"string"}`,
			`{"type":"int"}`,
			`{"type":"long"}`,
		}

		for _, schema := range schemas {
			_, err := registry.RegisterSchema(subject, schema, SchemaTypeAvro, nil)
			require.NoError(t, err)
		}

		retrieved, err := registry.GetSchemaBySubjectVersion(subject, 2)
		assert.NoError(t, err)
		assert.Equal(t, 2, retrieved.Version)
		assert.Equal(t, schemas[1], retrieved.Schema)
	})
}
