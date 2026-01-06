// Copyright 2025 Takhin Data, Inc.

package sasl

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrAuthenticationFailed = errors.New("authentication failed")
	ErrUnsupportedMechanism = errors.New("unsupported SASL mechanism")
	ErrInvalidCredentials   = errors.New("invalid credentials format")
)

// Mechanism represents a SASL authentication mechanism
type Mechanism string

const (
	PLAIN         Mechanism = "PLAIN"
	SCRAM_SHA_256 Mechanism = "SCRAM-SHA-256"
	SCRAM_SHA_512 Mechanism = "SCRAM-SHA-512"
	GSSAPI        Mechanism = "GSSAPI"
)

// Authenticator defines the interface for SASL authentication
type Authenticator interface {
	// Name returns the mechanism name
	Name() Mechanism
	
	// Authenticate performs authentication and returns principal/error
	Authenticate(ctx context.Context, authBytes []byte) (string, error)
	
	// Step handles multi-step authentication (for SCRAM/GSSAPI)
	Step(ctx context.Context, state *AuthState, challenge []byte) (response []byte, complete bool, err error)
}

// AuthState represents the state of a multi-step authentication
type AuthState struct {
	Mechanism   Mechanism
	Principal   string
	SessionData map[string]interface{}
	StartTime   time.Time
}

// Session represents an authenticated session
type Session struct {
	Principal     string
	Mechanism     Mechanism
	AuthTime      time.Time
	ExpiryTime    time.Time
	SessionID     string
	Attributes    map[string]interface{}
	mu            sync.RWMutex
}

// IsExpired checks if the session has expired
func (s *Session) IsExpired() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Now().After(s.ExpiryTime)
}

// GetAttribute retrieves a session attribute
func (s *Session) GetAttribute(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.Attributes[key]
	return val, ok
}

// SetAttribute sets a session attribute
func (s *Session) SetAttribute(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Attributes == nil {
		s.Attributes = make(map[string]interface{})
	}
	s.Attributes[key] = value
}

// Manager manages SASL authentication mechanisms and sessions
type Manager struct {
	authenticators map[Mechanism]Authenticator
	sessions       map[string]*Session
	userStore      UserStore
	cacheConfig    CacheConfig
	mu             sync.RWMutex
}

// CacheConfig holds authentication cache configuration
type CacheConfig struct {
	Enabled           bool
	TTL               time.Duration
	MaxEntries        int
	CleanupIntervalMs int
}

// UserStore defines interface for user credential storage
type UserStore interface {
	// GetUser retrieves user credentials
	GetUser(username string) (*User, error)
	
	// ValidateUser validates username and password
	ValidateUser(username, password string) (bool, error)
	
	// ListUsers returns all usernames (for admin purposes)
	ListUsers() ([]string, error)
}

// User represents a stored user
type User struct {
	Username      string
	PasswordHash  string
	Salt          []byte
	Iterations    int
	Mechanism     Mechanism
	Roles         []string
	Attributes    map[string]string
	CreatedAt     time.Time
	LastLoginAt   *time.Time
}

// NewManager creates a new SASL manager
func NewManager(userStore UserStore, cacheConfig CacheConfig) *Manager {
	m := &Manager{
		authenticators: make(map[Mechanism]Authenticator),
		sessions:       make(map[string]*Session),
		userStore:      userStore,
		cacheConfig:    cacheConfig,
	}
	
	if cacheConfig.Enabled && cacheConfig.CleanupIntervalMs > 0 {
		go m.cleanupExpiredSessions()
	}
	
	return m
}

// RegisterAuthenticator registers a SASL authenticator
func (m *Manager) RegisterAuthenticator(auth Authenticator) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.authenticators[auth.Name()] = auth
}

// GetAuthenticator retrieves an authenticator by mechanism
func (m *Manager) GetAuthenticator(mechanism Mechanism) (Authenticator, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	auth, ok := m.authenticators[mechanism]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedMechanism, mechanism)
	}
	return auth, nil
}

// SupportedMechanisms returns list of supported mechanisms
func (m *Manager) SupportedMechanisms() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	mechanisms := make([]string, 0, len(m.authenticators))
	for mech := range m.authenticators {
		mechanisms = append(mechanisms, string(mech))
	}
	return mechanisms
}

// Authenticate performs authentication
func (m *Manager) Authenticate(ctx context.Context, mechanism Mechanism, authBytes []byte) (*Session, error) {
	auth, err := m.GetAuthenticator(mechanism)
	if err != nil {
		return nil, err
	}
	
	principal, err := auth.Authenticate(ctx, authBytes)
	if err != nil {
		return nil, err
	}
	
	session := m.createSession(principal, mechanism)
	return session, nil
}

// createSession creates a new authenticated session
func (m *Manager) createSession(principal string, mechanism Mechanism) *Session {
	sessionID := fmt.Sprintf("%s-%d", principal, time.Now().UnixNano())
	
	session := &Session{
		Principal:  principal,
		Mechanism:  mechanism,
		AuthTime:   time.Now(),
		ExpiryTime: time.Now().Add(m.cacheConfig.TTL),
		SessionID:  sessionID,
		Attributes: make(map[string]interface{}),
	}
	
	if m.cacheConfig.Enabled {
		m.mu.Lock()
		m.sessions[sessionID] = session
		m.mu.Unlock()
	}
	
	return session
}

// GetSession retrieves a session by ID
func (m *Manager) GetSession(sessionID string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	session, ok := m.sessions[sessionID]
	if !ok {
		return nil, errors.New("session not found")
	}
	
	if session.IsExpired() {
		return nil, errors.New("session expired")
	}
	
	return session, nil
}

// InvalidateSession removes a session
func (m *Manager) InvalidateSession(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, sessionID)
}

// cleanupExpiredSessions periodically removes expired sessions
func (m *Manager) cleanupExpiredSessions() {
	ticker := time.NewTicker(time.Duration(m.cacheConfig.CleanupIntervalMs) * time.Millisecond)
	defer ticker.Stop()
	
	for range ticker.C {
		m.mu.Lock()
		for sessionID, session := range m.sessions {
			if session.IsExpired() {
				delete(m.sessions, sessionID)
			}
		}
		m.mu.Unlock()
	}
}

// SessionCount returns the number of active sessions
func (m *Manager) SessionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}
