// Copyright 2025 Takhin Data, Inc.
// Example: SASL Authentication Usage

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/takhin-data/takhin/pkg/sasl"
)

func main() {
	// Example 1: Simple PLAIN authentication
	fmt.Println("=== Example 1: PLAIN Authentication ===")
	plainExample()

	// Example 2: SCRAM-SHA-256 authentication
	fmt.Println("\n=== Example 2: SCRAM-SHA-256 Authentication ===")
	scramExample()

	// Example 3: Full SASL manager with multiple mechanisms
	fmt.Println("\n=== Example 3: Full SASL Manager ===")
	managerExample()

	// Example 4: Session management
	fmt.Println("\n=== Example 4: Session Management ===")
	sessionExample()
}

func plainExample() {
	// Create user store
	userStore := sasl.NewMemoryUserStore()

	// Add users
	if err := userStore.AddUser("alice", "alice-password", []string{"user"}); err != nil {
		log.Fatal(err)
	}

	// Create PLAIN authenticator
	plainAuth := sasl.NewPlainAuthenticator(userStore)

	// Encode credentials
	credentials := sasl.EncodePlainCredentials("alice", "alice-password")

	// Authenticate
	principal, err := plainAuth.Authenticate(context.Background(), credentials)
	if err != nil {
		log.Fatalf("Authentication failed: %v", err)
	}

	fmt.Printf("✓ Authenticated as: %s\n", principal)
}

func scramExample() {
	// Create user store
	userStore := sasl.NewMemoryUserStore()

	// Add SCRAM user
	if err := userStore.AddUserWithScram("bob", "bob-secret", sasl.SCRAM_SHA_256, []string{"admin"}); err != nil {
		log.Fatal(err)
	}

	// Create SCRAM-SHA-256 authenticator
	scramAuth := sasl.NewScramSHA256Authenticator(userStore)

	fmt.Printf("✓ Created SCRAM-SHA-256 authenticator: %s\n", scramAuth.Name())
	fmt.Printf("✓ User 'bob' added with SCRAM credentials\n")
	fmt.Printf("Note: Full SCRAM requires multi-step client-server exchange\n")
}

func managerExample() {
	// Create user store
	userStore := sasl.NewMemoryUserStore()

	// Add users for different mechanisms
	userStore.AddUser("plain-user", "plain-pass", []string{"user"})
	userStore.AddUserWithScram("scram-user", "scram-pass", sasl.SCRAM_SHA_256, []string{"user"})

	// Configure cache
	cacheConfig := sasl.CacheConfig{
		Enabled:           true,
		TTL:               time.Hour,
		MaxEntries:        1000,
		CleanupIntervalMs: 60000,
	}

	// Create SASL manager
	manager := sasl.NewManager(userStore, cacheConfig)

	// Register authenticators
	manager.RegisterAuthenticator(sasl.NewPlainAuthenticator(userStore))
	manager.RegisterAuthenticator(sasl.NewScramSHA256Authenticator(userStore))
	manager.RegisterAuthenticator(sasl.NewScramSHA512Authenticator(userStore))

	// List supported mechanisms
	mechanisms := manager.SupportedMechanisms()
	fmt.Printf("✓ Supported mechanisms: %v\n", mechanisms)

	// Authenticate with PLAIN
	credentials := sasl.EncodePlainCredentials("plain-user", "plain-pass")
	session, err := manager.Authenticate(context.Background(), sasl.PLAIN, credentials)
	if err != nil {
		log.Fatalf("Authentication failed: %v", err)
	}

	fmt.Printf("✓ Authenticated: %s\n", session.Principal)
	fmt.Printf("✓ Session ID: %s\n", session.SessionID)
	fmt.Printf("✓ Mechanism: %s\n", session.Mechanism)
	fmt.Printf("✓ Expires at: %s\n", session.ExpiryTime.Format(time.RFC3339))
	fmt.Printf("✓ Active sessions: %d\n", manager.SessionCount())
}

func sessionExample() {
	// Create setup
	userStore := sasl.NewMemoryUserStore()
	userStore.AddUser("charlie", "password", []string{"user"})

	cacheConfig := sasl.CacheConfig{
		Enabled:           true,
		TTL:               5 * time.Second, // Short TTL for demo
		MaxEntries:        100,
		CleanupIntervalMs: 1000,
	}

	manager := sasl.NewManager(userStore, cacheConfig)
	manager.RegisterAuthenticator(sasl.NewPlainAuthenticator(userStore))

	// Create session
	credentials := sasl.EncodePlainCredentials("charlie", "password")
	session, err := manager.Authenticate(context.Background(), sasl.PLAIN, credentials)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("✓ Session created: %s\n", session.SessionID)

	// Add custom attributes
	session.SetAttribute("client-id", "example-client")
	session.SetAttribute("ip-address", "192.168.1.100")

	// Retrieve attributes
	if clientID, ok := session.GetAttribute("client-id"); ok {
		fmt.Printf("✓ Client ID: %v\n", clientID)
	}

	// Check session validity
	fmt.Printf("✓ Session expired? %v\n", session.IsExpired())

	// Retrieve session from manager
	retrieved, err := manager.GetSession(session.SessionID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✓ Retrieved session for: %s\n", retrieved.Principal)

	// Invalidate session
	manager.InvalidateSession(session.SessionID)
	fmt.Printf("✓ Session invalidated\n")

	// Try to retrieve invalidated session
	_, err = manager.GetSession(session.SessionID)
	if err != nil {
		fmt.Printf("✓ Session not found (as expected): %v\n", err)
	}
}

// Advanced example: Custom user store
type DatabaseUserStore struct {
	// Database connection, etc.
}

func (s *DatabaseUserStore) GetUser(username string) (*sasl.User, error) {
	// Query database for user
	// Return user with hashed password
	return nil, fmt.Errorf("not implemented")
}

func (s *DatabaseUserStore) ValidateUser(username, password string) (bool, error) {
	// Validate against database
	return false, fmt.Errorf("not implemented")
}

func (s *DatabaseUserStore) ListUsers() ([]string, error) {
	// List all users from database
	return nil, fmt.Errorf("not implemented")
}

// Advanced example: Integration with Kafka handler
func kafkaHandlerIntegration() {
	// This is typically done in handler initialization
	
	// 1. Load configuration
	// cfg, _ := config.Load("configs/takhin.yaml")
	
	// 2. Create user store
	userStore := sasl.NewMemoryUserStore()
	
	// 3. Configure cache
	cacheConfig := sasl.CacheConfig{
		Enabled:           true,
		TTL:               time.Hour,
		MaxEntries:        1000,
		CleanupIntervalMs: 60000,
	}
	
	// 4. Create SASL manager
	manager := sasl.NewManager(userStore, cacheConfig)
	
	// 5. Register authenticators based on config
	mechanisms := []string{"PLAIN", "SCRAM-SHA-256"}
	for _, mechanism := range mechanisms {
		switch sasl.Mechanism(mechanism) {
		case sasl.PLAIN:
			manager.RegisterAuthenticator(sasl.NewPlainAuthenticator(userStore))
		case sasl.SCRAM_SHA_256:
			manager.RegisterAuthenticator(sasl.NewScramSHA256Authenticator(userStore))
		}
	}
	
	fmt.Printf("✓ SASL manager initialized with mechanisms: %v\n", mechanisms)
}
