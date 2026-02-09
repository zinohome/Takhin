# SASL Authentication - Quick Reference

## Configuration

### Enable SASL
```yaml
# configs/takhin.yaml
sasl:
  enabled: true
  mechanisms: [PLAIN, SCRAM-SHA-256, SCRAM-SHA-512]
```

### Environment Variables
```bash
export TAKHIN_SASL_ENABLED=true
export TAKHIN_SASL_MECHANISMS="PLAIN,SCRAM-SHA-256"
export TAKHIN_SASL_CACHE_TTL_SECONDS=3600
```

## Supported Mechanisms

### 1. PLAIN
- Simple username/password
- Format: `\0username\0password`
- Uses bcrypt for password hashing

### 2. SCRAM-SHA-256
- Challenge-response authentication
- PBKDF2 with 4096 iterations
- No password transmission
- SHA-256 hash function

### 3. SCRAM-SHA-512
- Same as SCRAM-SHA-256
- Uses SHA-512 hash function
- Higher security margin

### 4. GSSAPI (Kerberos)
- Placeholder interface
- Requires gokrb5 library
- For enterprise environments

## Code Examples

### Add Users Programmatically
```go
import "github.com/takhin-data/takhin/pkg/sasl"

// Create user store
store := sasl.NewMemoryUserStore()

// Add PLAIN user
store.AddUser("alice", "password123", []string{"user"})

// Add SCRAM-SHA-256 user
store.AddUserWithScram("bob", "secret456", sasl.SCRAM_SHA_256, []string{"admin"})
```

### Custom SASL Manager
```go
import (
    "time"
    "github.com/takhin-data/takhin/pkg/sasl"
)

// Create cache config
cacheConfig := sasl.CacheConfig{
    Enabled:           true,
    TTL:               time.Hour,
    MaxEntries:        1000,
    CleanupIntervalMs: 60000,
}

// Create manager
manager := sasl.NewManager(userStore, cacheConfig)

// Register authenticators
manager.RegisterAuthenticator(sasl.NewPlainAuthenticator(userStore))
manager.RegisterAuthenticator(sasl.NewScramSHA256Authenticator(userStore))
```

### Authenticate
```go
import "context"

// Authenticate with PLAIN
credentials := sasl.EncodePlainCredentials("alice", "password123")
session, err := manager.Authenticate(context.Background(), sasl.PLAIN, credentials)

if err != nil {
    // Authentication failed
}

// Use session
principal := session.Principal
sessionID := session.SessionID
```

## API Reference

### SASL Manager
```go
type Manager struct {
    // Methods
    RegisterAuthenticator(auth Authenticator)
    GetAuthenticator(mechanism Mechanism) (Authenticator, error)
    SupportedMechanisms() []string
    Authenticate(ctx context.Context, mechanism Mechanism, authBytes []byte) (*Session, error)
    GetSession(sessionID string) (*Session, error)
    InvalidateSession(sessionID string)
    SessionCount() int
}
```

### Session
```go
type Session struct {
    Principal     string
    Mechanism     Mechanism
    AuthTime      time.Time
    ExpiryTime    time.Time
    SessionID     string
    Attributes    map[string]interface{}
    
    // Methods
    IsExpired() bool
    GetAttribute(key string) (interface{}, bool)
    SetAttribute(key string, value interface{})
}
```

### UserStore Interface
```go
type UserStore interface {
    GetUser(username string) (*User, error)
    ValidateUser(username, password string) (bool, error)
    ListUsers() ([]string, error)
}
```

## Configuration Options

### Full SASL Config
```yaml
sasl:
  enabled: true
  mechanisms:
    - PLAIN
    - SCRAM-SHA-256
    - SCRAM-SHA-512
    - GSSAPI
  
  # User file (optional)
  plain:
    users: "/etc/takhin/users.txt"
  
  # Cache settings
  cache:
    enabled: true
    ttl:
      seconds: 3600        # 1 hour
    max:
      entries: 1000
    cleanup:
      ms: 60000            # 1 minute
  
  # Kerberos settings
  gssapi:
    service:
      name: "kafka"
    keytab:
      path: "/etc/kafka.keytab"
    realm: "EXAMPLE.COM"
    validate:
      kdc: true
```

## Testing

### Run SASL Tests
```bash
cd backend
go test -v ./pkg/sasl/...
```

### Test with Kafka Client
```bash
# Using kafka-console-producer
kafka-console-producer \
  --bootstrap-server localhost:9092 \
  --topic test \
  --producer-property security.protocol=SASL_PLAINTEXT \
  --producer-property sasl.mechanism=PLAIN \
  --producer-property sasl.jaas.config='org.apache.kafka.common.security.plain.PlainLoginModule required username="alice" password="password123";'
```

## Security Best Practices

1. **Use TLS**: Always enable TLS with SASL
   ```yaml
   server:
     tls:
       enabled: true
   sasl:
     enabled: true
   ```

2. **Strong Passwords**: Enforce minimum length and complexity
   - Minimum 12 characters
   - Mix of uppercase, lowercase, numbers, symbols

3. **Session TTL**: Balance security and convenience
   - Short-lived: 1 hour (high security)
   - Long-lived: 24 hours (convenience)

4. **Mechanism Choice**:
   - SCRAM > PLAIN (no password transmission)
   - GSSAPI for enterprise (Kerberos)

5. **Remove Default Users**: Delete test/admin users in production

## Troubleshooting

### Common Issues

**Authentication fails with "SASL not configured"**
```yaml
# Ensure SASL is enabled
sasl:
  enabled: true
```

**"Unsupported mechanism" error**
```yaml
# Add mechanism to list
sasl:
  mechanisms:
    - PLAIN
    - SCRAM-SHA-256
```

**Session expires too quickly**
```yaml
# Increase TTL
sasl:
  cache:
    ttl:
      seconds: 7200  # 2 hours
```

**High memory usage**
```yaml
# Reduce max entries
sasl:
  cache:
    max:
      entries: 500
```

## Performance Tuning

### Cache Settings
```yaml
# High-throughput
sasl:
  cache:
    enabled: true
    ttl:
      seconds: 7200      # 2 hours
    max:
      entries: 10000     # More cached sessions
    cleanup:
      ms: 300000         # 5 minutes (less frequent cleanup)
```

### Low-Memory
```yaml
# Memory-constrained
sasl:
  cache:
    enabled: true
    ttl:
      seconds: 1800      # 30 minutes
    max:
      entries: 100       # Fewer cached sessions
    cleanup:
      ms: 30000          # 30 seconds (aggressive cleanup)
```

## Metrics

Monitor SASL performance:
- `sasl_auth_total`: Total authentication attempts
- `sasl_auth_success`: Successful authentications
- `sasl_auth_failed`: Failed authentications
- `sasl_session_count`: Active session count
- `sasl_cache_hit_rate`: Cache hit percentage

## Files

### Package Structure
```
backend/pkg/sasl/
├── sasl.go          # Manager and core types
├── plain.go         # PLAIN authenticator
├── scram.go         # SCRAM authenticators
├── gssapi.go        # GSSAPI authenticator
├── userstore.go     # User storage
└── sasl_test.go     # Tests
```

### Configuration
- `backend/configs/takhin.yaml` - Main config
- `backend/pkg/config/config.go` - Config structure

### Handler Integration
- `backend/pkg/kafka/handler/handler.go` - Manager integration
- `backend/pkg/kafka/handler/sasl_handshake.go` - Handshake
- `backend/pkg/kafka/handler/sasl_authenticate.go` - Authentication
