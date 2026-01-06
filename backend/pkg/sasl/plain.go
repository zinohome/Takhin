// Copyright 2025 Takhin Data, Inc.

package sasl

import (
	"context"
	"fmt"
	"strings"
)

// PlainAuthenticator implements SASL PLAIN mechanism
type PlainAuthenticator struct {
	userStore UserStore
}

// NewPlainAuthenticator creates a new PLAIN authenticator
func NewPlainAuthenticator(userStore UserStore) *PlainAuthenticator {
	return &PlainAuthenticator{
		userStore: userStore,
	}
}

// Name returns the mechanism name
func (a *PlainAuthenticator) Name() Mechanism {
	return PLAIN
}

// Authenticate performs PLAIN authentication
// Format: [authzid] \0 username \0 password
func (a *PlainAuthenticator) Authenticate(ctx context.Context, authBytes []byte) (string, error) {
	authStr := string(authBytes)
	parts := strings.Split(authStr, "\x00")
	
	var username, password string
	if len(parts) == 3 {
		// Standard format: [authzid] \0 username \0 password
		username = parts[1]
		password = parts[2]
	} else if len(parts) == 2 {
		// Alternative format: username \0 password
		username = parts[0]
		password = parts[1]
	} else {
		return "", fmt.Errorf("%w: expected 2 or 3 parts, got %d", ErrInvalidCredentials, len(parts))
	}
	
	if username == "" || password == "" {
		return "", fmt.Errorf("%w: empty username or password", ErrInvalidCredentials)
	}
	
	// Validate against user store
	valid, err := a.userStore.ValidateUser(username, password)
	if err != nil {
		return "", fmt.Errorf("user store error: %w", err)
	}
	
	if !valid {
		return "", ErrAuthenticationFailed
	}
	
	return username, nil
}

// Step is not used for PLAIN (single-step authentication)
func (a *PlainAuthenticator) Step(ctx context.Context, state *AuthState, challenge []byte) ([]byte, bool, error) {
	return nil, false, fmt.Errorf("PLAIN does not support multi-step authentication")
}

// EncodePlainCredentials encodes username/password for PLAIN SASL
func EncodePlainCredentials(username, password string) []byte {
	// Format: \0username\0password
	return []byte(fmt.Sprintf("\x00%s\x00%s", username, password))
}
