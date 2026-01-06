// Copyright 2025 Takhin Data, Inc.

package sasl

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"hash"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlainAuthentication(t *testing.T) {
	userStore := NewMemoryUserStore()
	err := userStore.AddUser("testuser", "testpass", []string{"user"})
	require.NoError(t, err)
	
	auth := NewPlainAuthenticator(userStore)
	
	tests := []struct {
		name      string
		authBytes []byte
		wantUser  string
		wantErr   bool
	}{
		{
			name:      "valid credentials - standard format",
			authBytes: EncodePlainCredentials("testuser", "testpass"),
			wantUser:  "testuser",
			wantErr:   false,
		},
		{
			name:      "invalid password",
			authBytes: EncodePlainCredentials("testuser", "wrongpass"),
			wantUser:  "",
			wantErr:   true,
		},
		{
			name:      "invalid username",
			authBytes: EncodePlainCredentials("wronguser", "testpass"),
			wantUser:  "",
			wantErr:   true,
		},
		{
			name:      "empty credentials",
			authBytes: EncodePlainCredentials("", ""),
			wantUser:  "",
			wantErr:   true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := auth.Authenticate(context.Background(), tt.authBytes)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantUser, user)
			}
		})
	}
}

func TestScramSHA256Authentication(t *testing.T) {
	userStore := NewMemoryUserStore()
	err := userStore.AddUserWithScram("testuser", "testpass", SCRAM_SHA_256, []string{"user"})
	require.NoError(t, err)
	
	auth := NewScramSHA256Authenticator(userStore)
	assert.Equal(t, SCRAM_SHA_256, auth.Name())
	
	// Test basic authentication (would be expanded for full SCRAM flow)
	_, err = auth.Authenticate(context.Background(), []byte("n,,n=testuser,r=clientnonce123"))
	assert.NoError(t, err)
}

func TestScramSHA512Authentication(t *testing.T) {
	userStore := NewMemoryUserStore()
	err := userStore.AddUserWithScram("testuser", "testpass", SCRAM_SHA_512, []string{"user"})
	require.NoError(t, err)
	
	auth := NewScramSHA512Authenticator(userStore)
	assert.Equal(t, SCRAM_SHA_512, auth.Name())
	
	_, err = auth.Authenticate(context.Background(), []byte("n,,n=testuser,r=clientnonce123"))
	assert.NoError(t, err)
}

func TestGSSAPIAuthentication(t *testing.T) {
	auth := NewGSSAPIAuthenticator("kafka", "/etc/kafka.keytab", "EXAMPLE.COM", true)
	assert.Equal(t, GSSAPI, auth.Name())
	
	// GSSAPI should return error for single-step auth
	_, err := auth.Authenticate(context.Background(), []byte("token"))
	assert.Error(t, err)
	
	// Step should also return error (not implemented)
	_, _, err = auth.Step(context.Background(), &AuthState{}, []byte("token"))
	assert.Error(t, err)
}

func TestSaslManager(t *testing.T) {
	userStore := NewMemoryUserStore()
	err := userStore.AddUser("testuser", "testpass", []string{"user"})
	require.NoError(t, err)
	
	cacheConfig := CacheConfig{
		Enabled:           true,
		TTL:               time.Hour,
		MaxEntries:        100,
		CleanupIntervalMs: 60000,
	}
	
	manager := NewManager(userStore, cacheConfig)
	
	// Register authenticators
	manager.RegisterAuthenticator(NewPlainAuthenticator(userStore))
	manager.RegisterAuthenticator(NewScramSHA256Authenticator(userStore))
	manager.RegisterAuthenticator(NewScramSHA512Authenticator(userStore))
	
	// Test supported mechanisms
	mechanisms := manager.SupportedMechanisms()
	assert.Len(t, mechanisms, 3)
	assert.Contains(t, mechanisms, string(PLAIN))
	assert.Contains(t, mechanisms, string(SCRAM_SHA_256))
	assert.Contains(t, mechanisms, string(SCRAM_SHA_512))
	
	// Test authentication
	authBytes := EncodePlainCredentials("testuser", "testpass")
	session, err := manager.Authenticate(context.Background(), PLAIN, authBytes)
	require.NoError(t, err)
	assert.NotNil(t, session)
	assert.Equal(t, "testuser", session.Principal)
	assert.Equal(t, PLAIN, session.Mechanism)
	assert.False(t, session.IsExpired())
	
	// Test session retrieval
	retrievedSession, err := manager.GetSession(session.SessionID)
	require.NoError(t, err)
	assert.Equal(t, session.SessionID, retrievedSession.SessionID)
	
	// Test session count
	assert.Equal(t, 1, manager.SessionCount())
	
	// Test session invalidation
	manager.InvalidateSession(session.SessionID)
	assert.Equal(t, 0, manager.SessionCount())
	
	// Test unsupported mechanism
	_, err = manager.Authenticate(context.Background(), GSSAPI, []byte("token"))
	assert.Error(t, err)
}

func TestSession(t *testing.T) {
	session := &Session{
		Principal:  "testuser",
		Mechanism:  PLAIN,
		AuthTime:   time.Now(),
		ExpiryTime: time.Now().Add(time.Hour),
		SessionID:  "session-123",
		Attributes: make(map[string]interface{}),
	}
	
	// Test not expired
	assert.False(t, session.IsExpired())
	
	// Test attributes
	session.SetAttribute("key1", "value1")
	val, ok := session.GetAttribute("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", val)
	
	_, ok = session.GetAttribute("nonexistent")
	assert.False(t, ok)
	
	// Test expired session
	expiredSession := &Session{
		Principal:  "testuser",
		Mechanism:  PLAIN,
		AuthTime:   time.Now().Add(-2 * time.Hour),
		ExpiryTime: time.Now().Add(-time.Hour),
		SessionID:  "expired-123",
	}
	assert.True(t, expiredSession.IsExpired())
}

func TestMemoryUserStore(t *testing.T) {
	store := NewMemoryUserStore()
	
	// Test add user
	err := store.AddUser("user1", "pass1", []string{"admin"})
	require.NoError(t, err)
	
	// Test duplicate user
	err = store.AddUser("user1", "pass2", []string{"user"})
	assert.ErrorIs(t, err, ErrUserExists)
	
	// Test get user
	user, err := store.GetUser("user1")
	require.NoError(t, err)
	assert.Equal(t, "user1", user.Username)
	assert.Contains(t, user.Roles, "admin")
	
	// Test user not found
	_, err = store.GetUser("nonexistent")
	assert.ErrorIs(t, err, ErrUserNotFound)
	
	// Test validate user
	valid, err := store.ValidateUser("user1", "pass1")
	require.NoError(t, err)
	assert.True(t, valid)
	
	valid, err = store.ValidateUser("user1", "wrongpass")
	require.NoError(t, err)
	assert.False(t, valid)
	
	// Test list users
	err = store.AddUser("user2", "pass2", []string{"user"})
	require.NoError(t, err)
	
	users, err := store.ListUsers()
	require.NoError(t, err)
	assert.Len(t, users, 2)
	assert.Contains(t, users, "user1")
	assert.Contains(t, users, "user2")
	
	// Test remove user
	err = store.RemoveUser("user2")
	require.NoError(t, err)
	
	users, err = store.ListUsers()
	require.NoError(t, err)
	assert.Len(t, users, 1)
	
	// Test update password
	err = store.UpdatePassword("user1", "newpass")
	require.NoError(t, err)
	
	valid, err = store.ValidateUser("user1", "newpass")
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestScramCredentialGeneration(t *testing.T) {
	tests := []struct {
		name       string
		mechanism  Mechanism
		iterations int
		hashFunc   func() hash.Hash
		hashSize   int
	}{
		{
			name:       "SCRAM-SHA-256",
			mechanism:  SCRAM_SHA_256,
			iterations: DefaultScramSHA256Iterations,
			hashFunc:   sha256.New,
			hashSize:   sha256.Size,
		},
		{
			name:       "SCRAM-SHA-512",
			mechanism:  SCRAM_SHA_512,
			iterations: DefaultScramSHA512Iterations,
			hashFunc:   sha512.New,
			hashSize:   sha512.Size,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passwordHash, salt, err := GenerateScramCredentials("password123", tt.iterations, tt.hashFunc, tt.hashSize)
			require.NoError(t, err)
			assert.NotEmpty(t, passwordHash)
			assert.NotEmpty(t, salt)
			assert.Len(t, salt, 32)
		})
	}
}

func TestScramAttributeParsing(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected map[string]string
	}{
		{
			name:    "client first message",
			message: "n=user,r=nonce123",
			expected: map[string]string{
				"n": "user",
				"r": "nonce123",
			},
		},
		{
			name:    "server first message",
			message: "r=nonce123server456,s=salt==,i=4096",
			expected: map[string]string{
				"r": "nonce123server456",
				"s": "salt==",
				"i": "4096",
			},
		},
		{
			name:    "client final message",
			message: "c=biws,r=nonce123server456,p=proof==",
			expected: map[string]string{
				"c": "biws",
				"r": "nonce123server456",
				"p": "proof==",
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs := parseScramAttributes(tt.message)
			assert.Equal(t, tt.expected, attrs)
		})
	}
}

func BenchmarkPlainAuthentication(b *testing.B) {
	userStore := NewMemoryUserStore()
	userStore.AddUser("benchuser", "benchpass", []string{"user"})
	
	auth := NewPlainAuthenticator(userStore)
	authBytes := EncodePlainCredentials("benchuser", "benchpass")
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = auth.Authenticate(ctx, authBytes)
	}
}

func BenchmarkScramCredentialGeneration(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = GenerateScramCredentials("password", DefaultScramSHA256Iterations, sha256.New, sha256.Size)
	}
}
