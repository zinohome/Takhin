// Copyright 2025 Takhin Data, Inc.

package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"

	"golang.org/x/crypto/chacha20poly1305"

	"github.com/takhin-data/takhin/pkg/mempool"
)

// Algorithm represents an encryption algorithm
type Algorithm string

const (
	AlgorithmNone      Algorithm = "none"
	AlgorithmAES256GCM Algorithm = "aes-256-gcm"
	AlgorithmAES128GCM Algorithm = "aes-128-gcm"
	AlgorithmChaCha20  Algorithm = "chacha20-poly1305"
)

// Encryptor handles data encryption/decryption
type Encryptor interface {
	// Encrypt encrypts plaintext and returns ciphertext
	Encrypt(plaintext []byte) ([]byte, error)
	
	// Decrypt decrypts ciphertext and returns plaintext
	Decrypt(ciphertext []byte) ([]byte, error)
	
	// Algorithm returns the encryption algorithm used
	Algorithm() Algorithm
	
	// Overhead returns the encryption overhead in bytes
	Overhead() int
}

// noopEncryptor implements Encryptor with no encryption
type noopEncryptor struct{}

func (n *noopEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
	result := mempool.GetBuffer(len(plaintext))
	copy(result, plaintext)
	return result, nil
}

func (n *noopEncryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	result := mempool.GetBuffer(len(ciphertext))
	copy(result, ciphertext)
	return result, nil
}

func (n *noopEncryptor) Algorithm() Algorithm {
	return AlgorithmNone
}

func (n *noopEncryptor) Overhead() int {
	return 0
}

// aesGCMEncryptor implements AES-GCM encryption
type aesGCMEncryptor struct {
	aead      cipher.AEAD
	algorithm Algorithm
}

func newAESGCMEncryptor(key []byte, algorithm Algorithm) (*aesGCMEncryptor, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	return &aesGCMEncryptor{
		aead:      aead,
		algorithm: algorithm,
	}, nil
}

func (e *aesGCMEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
	nonceSize := e.aead.NonceSize()
	
	// Allocate buffer: nonce + ciphertext (with tag)
	ciphertext := mempool.GetBuffer(nonceSize + len(plaintext) + e.aead.Overhead())
	
	// Generate nonce
	nonce := ciphertext[:nonceSize]
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		mempool.PutBuffer(ciphertext)
		return nil, fmt.Errorf("generate nonce: %w", err)
	}
	
	// Encrypt and seal (appends tag)
	sealed := e.aead.Seal(ciphertext[nonceSize:nonceSize], nonce, plaintext, nil)
	
	// Return nonce + sealed data
	return ciphertext[:nonceSize+len(sealed)], nil
}

func (e *aesGCMEncryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	nonceSize := e.aead.NonceSize()
	
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	
	nonce := ciphertext[:nonceSize]
	encrypted := ciphertext[nonceSize:]
	
	// Decrypt
	plaintext, err := e.aead.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	
	// Copy to pooled buffer
	result := mempool.GetBuffer(len(plaintext))
	copy(result, plaintext)
	
	return result, nil
}

func (e *aesGCMEncryptor) Algorithm() Algorithm {
	return e.algorithm
}

func (e *aesGCMEncryptor) Overhead() int {
	return e.aead.NonceSize() + e.aead.Overhead()
}

// chaCha20Encryptor implements ChaCha20-Poly1305 encryption
type chaCha20Encryptor struct {
	aead cipher.AEAD
}

func newChaCha20Encryptor(key []byte) (*chaCha20Encryptor, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("ChaCha20-Poly1305 requires 32-byte key")
	}
	
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("create ChaCha20-Poly1305: %w", err)
	}

	return &chaCha20Encryptor{
		aead: aead,
	}, nil
}

func (e *chaCha20Encryptor) Encrypt(plaintext []byte) ([]byte, error) {
	nonceSize := e.aead.NonceSize()
	
	// Allocate buffer: nonce + ciphertext (with tag)
	ciphertext := mempool.GetBuffer(nonceSize + len(plaintext) + e.aead.Overhead())
	
	// Generate nonce
	nonce := ciphertext[:nonceSize]
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		mempool.PutBuffer(ciphertext)
		return nil, fmt.Errorf("generate nonce: %w", err)
	}
	
	// Encrypt and seal
	sealed := e.aead.Seal(ciphertext[nonceSize:nonceSize], nonce, plaintext, nil)
	
	return ciphertext[:nonceSize+len(sealed)], nil
}

func (e *chaCha20Encryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	nonceSize := e.aead.NonceSize()
	
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	
	nonce := ciphertext[:nonceSize]
	encrypted := ciphertext[nonceSize:]
	
	plaintext, err := e.aead.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	
	result := mempool.GetBuffer(len(plaintext))
	copy(result, plaintext)
	
	return result, nil
}

func (e *chaCha20Encryptor) Algorithm() Algorithm {
	return AlgorithmChaCha20
}

func (e *chaCha20Encryptor) Overhead() int {
	return e.aead.NonceSize() + e.aead.Overhead()
}

// NewEncryptor creates an encryptor based on algorithm and key
func NewEncryptor(algorithm Algorithm, key []byte) (Encryptor, error) {
	switch algorithm {
	case AlgorithmNone:
		return &noopEncryptor{}, nil
		
	case AlgorithmAES256GCM:
		if len(key) != 32 {
			return nil, fmt.Errorf("AES-256-GCM requires 32-byte key, got %d", len(key))
		}
		return newAESGCMEncryptor(key, AlgorithmAES256GCM)
		
	case AlgorithmAES128GCM:
		if len(key) != 16 {
			return nil, fmt.Errorf("AES-128-GCM requires 16-byte key, got %d", len(key))
		}
		return newAESGCMEncryptor(key, AlgorithmAES128GCM)
		
	case AlgorithmChaCha20:
		if len(key) != 32 {
			return nil, fmt.Errorf("ChaCha20-Poly1305 requires 32-byte key, got %d", len(key))
		}
		return newChaCha20Encryptor(key)
		
	default:
		return nil, fmt.Errorf("unknown encryption algorithm: %s", algorithm)
	}
}
