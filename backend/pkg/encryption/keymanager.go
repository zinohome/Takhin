// Copyright 2025 Takhin Data, Inc.

package encryption

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// KeyManager manages encryption keys
type KeyManager interface {
	// GetKey retrieves the encryption key by ID
	GetKey(keyID string) ([]byte, error)
	
	// GetCurrentKey returns the currently active encryption key and its ID
	GetCurrentKey() (keyID string, key []byte, err error)
	
	// RotateKey creates a new encryption key
	RotateKey() (keyID string, key []byte, err error)
}

// fileKeyManager stores keys in files
type fileKeyManager struct {
	keyDir      string
	currentID   string
	keys        map[string][]byte
	mu          sync.RWMutex
}

// FileKeyManagerConfig configures the file-based key manager
type FileKeyManagerConfig struct {
	KeyDir     string
	KeySize    int // 16 for AES-128, 32 for AES-256/ChaCha20
}

// NewFileKeyManager creates a file-based key manager
func NewFileKeyManager(config FileKeyManagerConfig) (KeyManager, error) {
	if err := os.MkdirAll(config.KeyDir, 0700); err != nil {
		return nil, fmt.Errorf("create key directory: %w", err)
	}

	km := &fileKeyManager{
		keyDir: config.KeyDir,
		keys:   make(map[string][]byte),
	}

	// Load existing keys
	if err := km.loadKeys(); err != nil {
		return nil, fmt.Errorf("load keys: %w", err)
	}

	// Create initial key if none exist
	if len(km.keys) == 0 {
		keyID, _, err := km.RotateKey()
		if err != nil {
			return nil, fmt.Errorf("create initial key: %w", err)
		}
		km.currentID = keyID
	}

	return km, nil
}

func (km *fileKeyManager) loadKeys() error {
	entries, err := os.ReadDir(km.keyDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read key directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if filepath.Ext(name) != ".key" {
			continue
		}

		keyID := name[:len(name)-4]
		keyPath := filepath.Join(km.keyDir, name)

		keyData, err := os.ReadFile(keyPath)
		if err != nil {
			return fmt.Errorf("read key file %s: %w", name, err)
		}

		key, err := base64.StdEncoding.DecodeString(string(keyData))
		if err != nil {
			return fmt.Errorf("decode key %s: %w", keyID, err)
		}

		km.keys[keyID] = key

		// The most recent key is the current one
		if km.currentID == "" || keyID > km.currentID {
			km.currentID = keyID
		}
	}

	return nil
}

func (km *fileKeyManager) GetKey(keyID string) ([]byte, error) {
	km.mu.RLock()
	defer km.mu.RUnlock()

	key, ok := km.keys[keyID]
	if !ok {
		return nil, fmt.Errorf("key not found: %s", keyID)
	}

	// Return a copy
	keyCopy := make([]byte, len(key))
	copy(keyCopy, key)
	return keyCopy, nil
}

func (km *fileKeyManager) GetCurrentKey() (string, []byte, error) {
	km.mu.RLock()
	defer km.mu.RUnlock()

	if km.currentID == "" {
		return "", nil, fmt.Errorf("no current key")
	}

	key, ok := km.keys[km.currentID]
	if !ok {
		return "", nil, fmt.Errorf("current key not found: %s", km.currentID)
	}

	keyCopy := make([]byte, len(key))
	copy(keyCopy, key)
	return km.currentID, keyCopy, nil
}

func (km *fileKeyManager) RotateKey() (string, []byte, error) {
	km.mu.Lock()
	defer km.mu.Unlock()

	// Generate new key
	key := make([]byte, 32) // Default to 256-bit
	if _, err := rand.Read(key); err != nil {
		return "", nil, fmt.Errorf("generate key: %w", err)
	}

	// Generate key ID (timestamp-based for ordering)
	keyID := fmt.Sprintf("key-%d", time.Now().UnixNano())

	// Save to file
	keyPath := filepath.Join(km.keyDir, keyID+".key")
	encoded := base64.StdEncoding.EncodeToString(key)
	
	if err := os.WriteFile(keyPath, []byte(encoded), 0600); err != nil {
		return "", nil, fmt.Errorf("write key file: %w", err)
	}

	// Store in memory
	km.keys[keyID] = key
	km.currentID = keyID

	keyCopy := make([]byte, len(key))
	copy(keyCopy, key)
	return keyID, keyCopy, nil
}

// staticKeyManager is a simple key manager with a single static key
type staticKeyManager struct {
	keyID string
	key   []byte
}

// NewStaticKeyManager creates a key manager with a single static key
func NewStaticKeyManager(keyID string, key []byte) KeyManager {
	keyCopy := make([]byte, len(key))
	copy(keyCopy, key)
	
	return &staticKeyManager{
		keyID: keyID,
		key:   keyCopy,
	}
}

func (km *staticKeyManager) GetKey(keyID string) ([]byte, error) {
	if keyID != km.keyID {
		return nil, fmt.Errorf("key not found: %s", keyID)
	}
	
	keyCopy := make([]byte, len(km.key))
	copy(keyCopy, km.key)
	return keyCopy, nil
}

func (km *staticKeyManager) GetCurrentKey() (string, []byte, error) {
	keyCopy := make([]byte, len(km.key))
	copy(keyCopy, km.key)
	return km.keyID, keyCopy, nil
}

func (km *staticKeyManager) RotateKey() (string, []byte, error) {
	return "", nil, fmt.Errorf("key rotation not supported in static key manager")
}
