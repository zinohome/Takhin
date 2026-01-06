// Copyright 2025 Takhin Data, Inc.

package encryption

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryption_NoOp(t *testing.T) {
	encryptor, err := NewEncryptor(AlgorithmNone, nil)
	require.NoError(t, err)
	assert.Equal(t, AlgorithmNone, encryptor.Algorithm())
	assert.Equal(t, 0, encryptor.Overhead())

	plaintext := []byte("hello world")
	ciphertext, err := encryptor.Encrypt(plaintext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, ciphertext)

	decrypted, err := encryptor.Decrypt(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncryption_AES128GCM(t *testing.T) {
	key := make([]byte, 16)
	_, err := rand.Read(key)
	require.NoError(t, err)

	encryptor, err := NewEncryptor(AlgorithmAES128GCM, key)
	require.NoError(t, err)
	assert.Equal(t, AlgorithmAES128GCM, encryptor.Algorithm())
	assert.Greater(t, encryptor.Overhead(), 0)

	plaintext := []byte("sensitive data")
	ciphertext, err := encryptor.Encrypt(plaintext)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, ciphertext)
	assert.Greater(t, len(ciphertext), len(plaintext))

	decrypted, err := encryptor.Decrypt(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncryption_AES256GCM(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)

	encryptor, err := NewEncryptor(AlgorithmAES256GCM, key)
	require.NoError(t, err)
	assert.Equal(t, AlgorithmAES256GCM, encryptor.Algorithm())

	plaintext := []byte("top secret information")
	ciphertext, err := encryptor.Encrypt(plaintext)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, ciphertext)

	decrypted, err := encryptor.Decrypt(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncryption_ChaCha20(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)

	encryptor, err := NewEncryptor(AlgorithmChaCha20, key)
	require.NoError(t, err)
	assert.Equal(t, AlgorithmChaCha20, encryptor.Algorithm())

	plaintext := []byte("chacha20 encrypted data")
	ciphertext, err := encryptor.Encrypt(plaintext)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, ciphertext)

	decrypted, err := encryptor.Decrypt(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncryption_InvalidKeySize(t *testing.T) {
	tests := []struct {
		name      string
		algorithm Algorithm
		keySize   int
		wantError bool
	}{
		{"AES-128 valid", AlgorithmAES128GCM, 16, false},
		{"AES-128 invalid", AlgorithmAES128GCM, 32, true},
		{"AES-256 valid", AlgorithmAES256GCM, 32, false},
		{"AES-256 invalid", AlgorithmAES256GCM, 16, true},
		{"ChaCha20 valid", AlgorithmChaCha20, 32, false},
		{"ChaCha20 invalid", AlgorithmChaCha20, 16, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := make([]byte, tt.keySize)
			_, err := NewEncryptor(tt.algorithm, key)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEncryption_WrongKeyDecryption(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	rand.Read(key1)
	rand.Read(key2)

	encryptor1, err := NewEncryptor(AlgorithmAES256GCM, key1)
	require.NoError(t, err)

	encryptor2, err := NewEncryptor(AlgorithmAES256GCM, key2)
	require.NoError(t, err)

	plaintext := []byte("encrypted message")
	ciphertext, err := encryptor1.Encrypt(plaintext)
	require.NoError(t, err)

	// Try to decrypt with wrong key
	_, err = encryptor2.Decrypt(ciphertext)
	assert.Error(t, err)
}

func TestEncryption_LargeData(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	encryptor, err := NewEncryptor(AlgorithmAES256GCM, key)
	require.NoError(t, err)

	// Test with 1MB of data
	plaintext := make([]byte, 1024*1024)
	rand.Read(plaintext)

	ciphertext, err := encryptor.Encrypt(plaintext)
	require.NoError(t, err)

	decrypted, err := encryptor.Decrypt(ciphertext)
	require.NoError(t, err)

	assert.True(t, bytes.Equal(plaintext, decrypted))
}

func TestEncryption_EmptyData(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	encryptor, err := NewEncryptor(AlgorithmAES256GCM, key)
	require.NoError(t, err)

	plaintext := []byte{}
	ciphertext, err := encryptor.Encrypt(plaintext)
	require.NoError(t, err)

	decrypted, err := encryptor.Decrypt(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncryption_NonceUniqueness(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	encryptor, err := NewEncryptor(AlgorithmAES256GCM, key)
	require.NoError(t, err)

	plaintext := []byte("test data")
	
	// Encrypt the same plaintext multiple times
	ciphertexts := make([][]byte, 10)
	for i := 0; i < 10; i++ {
		ct, err := encryptor.Encrypt(plaintext)
		require.NoError(t, err)
		ciphertexts[i] = ct
	}

	// All ciphertexts should be different due to unique nonces
	for i := 0; i < len(ciphertexts); i++ {
		for j := i + 1; j < len(ciphertexts); j++ {
			assert.False(t, bytes.Equal(ciphertexts[i], ciphertexts[j]))
		}
	}
}

func BenchmarkEncryption_AES256GCM_1KB(b *testing.B) {
	key := make([]byte, 32)
	rand.Read(key)
	encryptor, _ := NewEncryptor(AlgorithmAES256GCM, key)
	
	data := make([]byte, 1024)
	rand.Read(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ciphertext, _ := encryptor.Encrypt(data)
		_ = ciphertext
	}
}

func BenchmarkEncryption_AES256GCM_1MB(b *testing.B) {
	key := make([]byte, 32)
	rand.Read(key)
	encryptor, _ := NewEncryptor(AlgorithmAES256GCM, key)
	
	data := make([]byte, 1024*1024)
	rand.Read(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ciphertext, _ := encryptor.Encrypt(data)
		_ = ciphertext
	}
}

func BenchmarkEncryption_ChaCha20_1KB(b *testing.B) {
	key := make([]byte, 32)
	rand.Read(key)
	encryptor, _ := NewEncryptor(AlgorithmChaCha20, key)
	
	data := make([]byte, 1024)
	rand.Read(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ciphertext, _ := encryptor.Encrypt(data)
		_ = ciphertext
	}
}

func BenchmarkDecryption_AES256GCM_1KB(b *testing.B) {
	key := make([]byte, 32)
	rand.Read(key)
	encryptor, _ := NewEncryptor(AlgorithmAES256GCM, key)
	
	data := make([]byte, 1024)
	rand.Read(data)
	ciphertext, _ := encryptor.Encrypt(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		plaintext, _ := encryptor.Decrypt(ciphertext)
		_ = plaintext
	}
}
