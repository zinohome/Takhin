// Copyright 2025 Takhin Data, Inc.

package sasl

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"hash"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

// ScramAuthenticator implements SCRAM-SHA-256 and SCRAM-SHA-512
type ScramAuthenticator struct {
	mechanism   Mechanism
	hashFunc    func() hash.Hash
	hashSize    int
	userStore   UserStore
	activeAuths map[string]*scramState
}

type scramState struct {
	username      string
	clientNonce   string
	serverNonce   string
	salt          []byte
	iterations    int
	clientFirstMessageBare string
	serverFirstMessage     string
}

// NewScramSHA256Authenticator creates SCRAM-SHA-256 authenticator
func NewScramSHA256Authenticator(userStore UserStore) *ScramAuthenticator {
	return &ScramAuthenticator{
		mechanism:   SCRAM_SHA_256,
		hashFunc:    sha256.New,
		hashSize:    sha256.Size,
		userStore:   userStore,
		activeAuths: make(map[string]*scramState),
	}
}

// NewScramSHA512Authenticator creates SCRAM-SHA-512 authenticator
func NewScramSHA512Authenticator(userStore UserStore) *ScramAuthenticator {
	return &ScramAuthenticator{
		mechanism:   SCRAM_SHA_512,
		hashFunc:    sha512.New,
		hashSize:    sha512.Size,
		userStore:   userStore,
		activeAuths: make(map[string]*scramState),
	}
}

// Name returns the mechanism name
func (a *ScramAuthenticator) Name() Mechanism {
	return a.mechanism
}

// Authenticate starts SCRAM authentication (returns server first message)
func (a *ScramAuthenticator) Authenticate(ctx context.Context, authBytes []byte) (string, error) {
	// Parse client-first-message: n,,n=username,r=clientNonce
	clientFirstMessage := string(authBytes)
	
	// GS2 header: n,, (no channel binding)
	if !strings.HasPrefix(clientFirstMessage, "n,,") {
		return "", fmt.Errorf("%w: invalid GS2 header", ErrInvalidCredentials)
	}
	
	clientFirstMessageBare := clientFirstMessage[3:] // Remove "n,,"
	
	// Parse attributes
	attrs := parseScramAttributes(clientFirstMessageBare)
	username, ok := attrs["n"]
	if !ok || username == "" {
		return "", fmt.Errorf("%w: missing username", ErrInvalidCredentials)
	}
	
	clientNonce, ok := attrs["r"]
	if !ok || clientNonce == "" {
		return "", fmt.Errorf("%w: missing nonce", ErrInvalidCredentials)
	}
	
	// Get user from store
	user, err := a.userStore.GetUser(username)
	if err != nil {
		return "", ErrAuthenticationFailed
	}
	
	// Generate server nonce
	serverNonceBytes := make([]byte, 32)
	if _, err := rand.Read(serverNonceBytes); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}
	serverNonce := base64.StdEncoding.EncodeToString(serverNonceBytes)
	
	// Store state for continuation
	state := &scramState{
		username:               username,
		clientNonce:            clientNonce,
		serverNonce:            serverNonce,
		salt:                   user.Salt,
		iterations:             user.Iterations,
		clientFirstMessageBare: clientFirstMessageBare,
	}
	
	// Generate server-first-message
	serverFirstMessage := fmt.Sprintf("r=%s%s,s=%s,i=%d",
		clientNonce,
		serverNonce,
		base64.StdEncoding.EncodeToString(user.Salt),
		user.Iterations,
	)
	state.serverFirstMessage = serverFirstMessage
	
	// Store state keyed by combined nonce
	a.activeAuths[clientNonce+serverNonce] = state
	
	return username, nil
}

// Step handles SCRAM authentication steps
func (a *ScramAuthenticator) Step(ctx context.Context, authState *AuthState, challenge []byte) ([]byte, bool, error) {
	// Client sends client-final-message: c=biws,r=clientNonce+serverNonce,p=clientProof
	clientFinalMessage := string(challenge)
	attrs := parseScramAttributes(clientFinalMessage)
	
	nonce, ok := attrs["r"]
	if !ok {
		return nil, false, fmt.Errorf("%w: missing nonce in final message", ErrInvalidCredentials)
	}
	
	state, ok := a.activeAuths[nonce]
	if !ok {
		return nil, false, fmt.Errorf("%w: invalid authentication state", ErrAuthenticationFailed)
	}
	defer delete(a.activeAuths, nonce)
	
	clientProof, ok := attrs["p"]
	if !ok {
		return nil, false, fmt.Errorf("%w: missing client proof", ErrInvalidCredentials)
	}
	
	// Get user
	user, err := a.userStore.GetUser(state.username)
	if err != nil {
		return nil, false, ErrAuthenticationFailed
	}
	
	// Compute salted password
	saltedPassword := pbkdf2.Key([]byte(user.PasswordHash), state.salt, state.iterations, a.hashSize, a.hashFunc)
	
	// Compute client key and server key
	clientKey := hmacHash(a.hashFunc, saltedPassword, []byte("Client Key"))
	serverKey := hmacHash(a.hashFunc, saltedPassword, []byte("Server Key"))
	
	// Compute stored key
	h := a.hashFunc()
	h.Write(clientKey)
	storedKey := h.Sum(nil)
	
	// Build auth message
	clientFinalMessageWithoutProof := clientFinalMessage[:strings.LastIndex(clientFinalMessage, ",p=")]
	authMessage := fmt.Sprintf("%s,%s,%s",
		state.clientFirstMessageBare,
		state.serverFirstMessage,
		clientFinalMessageWithoutProof,
	)
	
	// Compute client signature
	clientSignature := hmacHash(a.hashFunc, storedKey, []byte(authMessage))
	
	// Verify client proof
	proofBytes, err := base64.StdEncoding.DecodeString(clientProof)
	if err != nil {
		return nil, false, fmt.Errorf("%w: invalid proof encoding", ErrInvalidCredentials)
	}
	
	// XOR to get client key
	computedClientKey := make([]byte, len(clientSignature))
	for i := range clientSignature {
		computedClientKey[i] = clientSignature[i] ^ proofBytes[i]
	}
	
	// Verify
	if !hmac.Equal(computedClientKey, clientKey) {
		return nil, false, ErrAuthenticationFailed
	}
	
	// Compute server signature
	serverSignature := hmacHash(a.hashFunc, serverKey, []byte(authMessage))
	
	// Build server-final-message
	serverFinalMessage := fmt.Sprintf("v=%s", base64.StdEncoding.EncodeToString(serverSignature))
	
	return []byte(serverFinalMessage), true, nil
}

// parseScramAttributes parses SCRAM attribute-value pairs
func parseScramAttributes(message string) map[string]string {
	attrs := make(map[string]string)
	parts := strings.Split(message, ",")
	
	for _, part := range parts {
		if len(part) < 2 || part[1] != '=' {
			continue
		}
		key := part[0:1]
		value := part[2:]
		attrs[key] = value
	}
	
	return attrs
}

// hmacHash computes HMAC
func hmacHash(hashFunc func() hash.Hash, key, data []byte) []byte {
	h := hmac.New(hashFunc, key)
	h.Write(data)
	return h.Sum(nil)
}

// GenerateScramCredentials generates SCRAM credentials for a user
func GenerateScramCredentials(password string, iterations int, hashFunc func() hash.Hash, hashSize int) (passwordHash string, salt []byte, err error) {
	// Generate random salt
	salt = make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return "", nil, fmt.Errorf("failed to generate salt: %w", err)
	}
	
	// Generate salted password using PBKDF2
	saltedPassword := pbkdf2.Key([]byte(password), salt, iterations, hashSize, hashFunc)
	
	// For storage, we need the base password that will be used with PBKDF2
	// Store the salted password hash
	passwordHash = base64.StdEncoding.EncodeToString(saltedPassword)
	
	return passwordHash, salt, nil
}

// DefaultScramIterations for SCRAM
const (
	DefaultScramSHA256Iterations = 4096
	DefaultScramSHA512Iterations = 4096
)
