// Copyright 2025 Takhin Data, Inc.

package encryption

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyManager_StaticKeyManager(t *testing.T) {
	keyID := "test-key"
	key := []byte("0123456789abcdef0123456789abcdef")

	km := NewStaticKeyManager(keyID, key)

	// Test GetCurrentKey
	currentID, currentKey, err := km.GetCurrentKey()
	require.NoError(t, err)
	assert.Equal(t, keyID, currentID)
	assert.Equal(t, key, currentKey)

	// Test GetKey
	retrievedKey, err := km.GetKey(keyID)
	require.NoError(t, err)
	assert.Equal(t, key, retrievedKey)

	// Test GetKey with wrong ID
	_, err = km.GetKey("wrong-id")
	assert.Error(t, err)

	// Test RotateKey (should fail)
	_, _, err = km.RotateKey()
	assert.Error(t, err)
}

func TestKeyManager_FileKeyManager(t *testing.T) {
	tmpDir := t.TempDir()
	keyDir := filepath.Join(tmpDir, "keys")

	// Create key manager
	km, err := NewFileKeyManager(FileKeyManagerConfig{
		KeyDir:  keyDir,
		KeySize: 32,
	})
	require.NoError(t, err)

	// Should have created initial key
	keyID1, key1, err := km.GetCurrentKey()
	require.NoError(t, err)
	assert.NotEmpty(t, keyID1)
	assert.Len(t, key1, 32)

	// Test GetKey
	retrievedKey, err := km.GetKey(keyID1)
	require.NoError(t, err)
	assert.Equal(t, key1, retrievedKey)

	// Test key rotation
	keyID2, key2, err := km.RotateKey()
	require.NoError(t, err)
	assert.NotEqual(t, keyID1, keyID2)
	assert.NotEqual(t, key1, key2)

	// Current key should be the new one
	currentID, currentKey, err := km.GetCurrentKey()
	require.NoError(t, err)
	assert.Equal(t, keyID2, currentID)
	assert.Equal(t, key2, currentKey)

	// Old key should still be accessible
	oldKey, err := km.GetKey(keyID1)
	require.NoError(t, err)
	assert.Equal(t, key1, oldKey)
}

func TestKeyManager_FileKeyManager_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	keyDir := filepath.Join(tmpDir, "keys")

	// Create first key manager and generate some keys
	km1, err := NewFileKeyManager(FileKeyManagerConfig{
		KeyDir:  keyDir,
		KeySize: 32,
	})
	require.NoError(t, err)

	keyID1, key1, err := km1.GetCurrentKey()
	require.NoError(t, err)

	keyID2, key2, err := km1.RotateKey()
	require.NoError(t, err)

	// Create second key manager from same directory
	km2, err := NewFileKeyManager(FileKeyManagerConfig{
		KeyDir:  keyDir,
		KeySize: 32,
	})
	require.NoError(t, err)

	// Should load the most recent key as current
	currentID, currentKey, err := km2.GetCurrentKey()
	require.NoError(t, err)
	assert.Equal(t, keyID2, currentID)
	assert.Equal(t, key2, currentKey)

	// Both keys should be accessible
	retrievedKey1, err := km2.GetKey(keyID1)
	require.NoError(t, err)
	assert.Equal(t, key1, retrievedKey1)

	retrievedKey2, err := km2.GetKey(keyID2)
	require.NoError(t, err)
	assert.Equal(t, key2, retrievedKey2)
}

func TestKeyManager_FileKeyManager_KeyFiles(t *testing.T) {
	tmpDir := t.TempDir()
	keyDir := filepath.Join(tmpDir, "keys")

	km, err := NewFileKeyManager(FileKeyManagerConfig{
		KeyDir:  keyDir,
		KeySize: 32,
	})
	require.NoError(t, err)

	keyID, _, err := km.GetCurrentKey()
	require.NoError(t, err)

	// Check that key file exists
	keyFile := filepath.Join(keyDir, keyID+".key")
	_, err = os.Stat(keyFile)
	require.NoError(t, err)

	// Check file permissions (should be 0600)
	info, err := os.Stat(keyFile)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestKeyManager_FileKeyManager_InvalidKeyDir(t *testing.T) {
	_, err := NewFileKeyManager(FileKeyManagerConfig{
		KeyDir:  "/invalid/path/that/should/not/exist",
		KeySize: 32,
	})
	// Should fail to create directory (depending on permissions)
	// On most systems this will error
	if err == nil {
		// If it doesn't error, it means we have permission to create there
		// Clean up
		os.RemoveAll("/invalid/path")
	}
}

func TestKeyManager_FileKeyManager_NonExistentKey(t *testing.T) {
	tmpDir := t.TempDir()
	keyDir := filepath.Join(tmpDir, "keys")

	km, err := NewFileKeyManager(FileKeyManagerConfig{
		KeyDir:  keyDir,
		KeySize: 32,
	})
	require.NoError(t, err)

	_, err = km.GetKey("non-existent-key")
	assert.Error(t, err)
}

func TestKeyManager_FileKeyManager_MultipleRotations(t *testing.T) {
	tmpDir := t.TempDir()
	keyDir := filepath.Join(tmpDir, "keys")

	km, err := NewFileKeyManager(FileKeyManagerConfig{
		KeyDir:  keyDir,
		KeySize: 32,
	})
	require.NoError(t, err)

	// Perform multiple rotations
	numRotations := 5
	keyIDs := make([]string, numRotations)
	keys := make([][]byte, numRotations)

	for i := 0; i < numRotations; i++ {
		keyID, key, err := km.RotateKey()
		require.NoError(t, err)
		keyIDs[i] = keyID
		keys[i] = key
	}

	// Current should be the last one
	currentID, currentKey, err := km.GetCurrentKey()
	require.NoError(t, err)
	assert.Equal(t, keyIDs[numRotations-1], currentID)
	assert.Equal(t, keys[numRotations-1], currentKey)

	// All keys should be accessible
	for i := 0; i < numRotations; i++ {
		retrievedKey, err := km.GetKey(keyIDs[i])
		require.NoError(t, err)
		assert.Equal(t, keys[i], retrievedKey)
	}
}

func TestKeyManager_FileKeyManager_KeyCopy(t *testing.T) {
	tmpDir := t.TempDir()
	keyDir := filepath.Join(tmpDir, "keys")

	km, err := NewFileKeyManager(FileKeyManagerConfig{
		KeyDir:  keyDir,
		KeySize: 32,
	})
	require.NoError(t, err)

	keyID, key1, err := km.GetCurrentKey()
	require.NoError(t, err)

	// Modify the returned key
	key1[0] = 0xFF

	// Get the key again - should not be modified
	key2, err := km.GetKey(keyID)
	require.NoError(t, err)
	assert.NotEqual(t, key1[0], key2[0])
}
