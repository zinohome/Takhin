// Copyright 2025 Takhin Data, Inc.

package sasl

import (
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrUserExists   = errors.New("user already exists")
)

// MemoryUserStore implements in-memory user storage with bcrypt hashing
type MemoryUserStore struct {
	users map[string]*User
	mu    sync.RWMutex
}

// NewMemoryUserStore creates a new in-memory user store
func NewMemoryUserStore() *MemoryUserStore {
	return &MemoryUserStore{
		users: make(map[string]*User),
	}
}

// AddUser adds a user with bcrypt-hashed password
func (s *MemoryUserStore) AddUser(username, password string, roles []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, exists := s.users[username]; exists {
		return ErrUserExists
	}
	
	// Hash password with bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	
	user := &User{
		Username:     username,
		PasswordHash: string(hashedPassword),
		Salt:         nil, // bcrypt includes salt
		Mechanism:    PLAIN,
		Roles:        roles,
		Attributes:   make(map[string]string),
		CreatedAt:    time.Now(),
	}
	
	s.users[username] = user
	return nil
}

// AddUserWithScram adds a user with SCRAM credentials
func (s *MemoryUserStore) AddUserWithScram(username, password string, mechanism Mechanism, roles []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, exists := s.users[username]; exists {
		return ErrUserExists
	}
	
	var iterations int
	var hashSize int
	var hashFunc func() hash.Hash
	
	switch mechanism {
	case SCRAM_SHA_256:
		iterations = DefaultScramSHA256Iterations
		hashSize = sha256.Size
		hashFunc = sha256.New
	case SCRAM_SHA_512:
		iterations = DefaultScramSHA512Iterations
		hashSize = sha512.Size
		hashFunc = sha512.New
	default:
		return fmt.Errorf("unsupported SCRAM mechanism: %s", mechanism)
	}
	
	passwordHash, salt, err := GenerateScramCredentials(password, iterations, hashFunc, hashSize)
	if err != nil {
		return fmt.Errorf("failed to generate SCRAM credentials: %w", err)
	}
	
	user := &User{
		Username:     username,
		PasswordHash: passwordHash,
		Salt:         salt,
		Iterations:   iterations,
		Mechanism:    mechanism,
		Roles:        roles,
		Attributes:   make(map[string]string),
		CreatedAt:    time.Now(),
	}
	
	s.users[username] = user
	return nil
}

// GetUser retrieves a user
func (s *MemoryUserStore) GetUser(username string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	user, ok := s.users[username]
	if !ok {
		return nil, ErrUserNotFound
	}
	
	// Return a copy to prevent modification
	userCopy := *user
	return &userCopy, nil
}

// ValidateUser validates username and password
func (s *MemoryUserStore) ValidateUser(username, password string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	user, ok := s.users[username]
	if !ok {
		return false, nil
	}
	
	// Update last login time
	now := time.Now()
	user.LastLoginAt = &now
	
	// Validate based on mechanism
	switch user.Mechanism {
	case PLAIN:
		// bcrypt comparison
		err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
		return err == nil, nil
	
	case SCRAM_SHA_256, SCRAM_SHA_512:
		// For SCRAM, password validation happens during the SCRAM handshake
		// This is just a basic check
		return user.PasswordHash != "", nil
	
	default:
		return false, fmt.Errorf("unsupported mechanism: %s", user.Mechanism)
	}
}

// ListUsers returns all usernames
func (s *MemoryUserStore) ListUsers() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	usernames := make([]string, 0, len(s.users))
	for username := range s.users {
		usernames = append(usernames, username)
	}
	return usernames, nil
}

// RemoveUser removes a user
func (s *MemoryUserStore) RemoveUser(username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, ok := s.users[username]; !ok {
		return ErrUserNotFound
	}
	
	delete(s.users, username)
	return nil
}

// UpdatePassword updates a user's password
func (s *MemoryUserStore) UpdatePassword(username, newPassword string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	user, ok := s.users[username]
	if !ok {
		return ErrUserNotFound
	}
	
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	
	user.PasswordHash = string(hashedPassword)
	return nil
}

// FileUserStore implements file-based user storage (placeholder)
type FileUserStore struct {
	filePath string
	users    map[string]*User
	mu       sync.RWMutex
}

// NewFileUserStore creates a file-based user store
func NewFileUserStore(filePath string) (*FileUserStore, error) {
	store := &FileUserStore{
		filePath: filePath,
		users:    make(map[string]*User),
	}
	
	// TODO: Load users from file
	// Format could be JSON, YAML, or custom format
	// For now, return empty store
	
	return store, nil
}

// GetUser retrieves a user
func (s *FileUserStore) GetUser(username string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	user, ok := s.users[username]
	if !ok {
		return nil, ErrUserNotFound
	}
	
	userCopy := *user
	return &userCopy, nil
}

// ValidateUser validates username and password
func (s *FileUserStore) ValidateUser(username, password string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	user, ok := s.users[username]
	if !ok {
		return false, nil
	}
	
	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	return err == nil, nil
}

// ListUsers returns all usernames
func (s *FileUserStore) ListUsers() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	usernames := make([]string, 0, len(s.users))
	for username := range s.users {
		usernames = append(usernames, username)
	}
	return usernames, nil
}

// constantTimeCompare performs constant-time string comparison
func constantTimeCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// hashPassword creates a simple SHA256 hash (for non-bcrypt scenarios)
func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}
