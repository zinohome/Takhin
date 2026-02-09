# TLS/SSL Quick Reference

## Quick Start

### Enable Basic TLS
```yaml
# configs/takhin.yaml
server:
  tls:
    enabled: true
    cert:
      file: "/path/to/server.pem"
    key:
      file: "/path/to/server-key.pem"
    min:
      version: "TLS1.2"
```

### Enable Mutual TLS (mTLS)
```yaml
server:
  tls:
    enabled: true
    cert:
      file: "/path/to/server.pem"
    key:
      file: "/path/to/server-key.pem"
    ca:
      file: "/path/to/ca.pem"
    client:
      auth: "require"
    verify:
      client:
        cert: true
```

## Environment Variables

```bash
# Enable TLS
export TAKHIN_SERVER_TLS_ENABLED=true
export TAKHIN_SERVER_TLS_CERT_FILE=/certs/server.pem
export TAKHIN_SERVER_TLS_KEY_FILE=/certs/server-key.pem

# Configure minimum TLS version
export TAKHIN_SERVER_TLS_MIN_VERSION=TLS1.3

# Enable client authentication
export TAKHIN_SERVER_TLS_CLIENT_AUTH=require
export TAKHIN_SERVER_TLS_CA_FILE=/certs/ca.pem
export TAKHIN_SERVER_TLS_VERIFY_CLIENT_CERT=true
```

## Generate Test Certificates

```bash
# Using Go code
go run scripts/generate-certs.go /path/to/output

# Using OpenSSL
openssl genrsa -out server-key.pem 2048
openssl req -new -key server-key.pem -out server.csr
openssl x509 -req -in server.csr -signkey server-key.pem -out server.pem
```

## Test TLS Connection

```bash
# Basic test
openssl s_client -connect localhost:9092

# Test specific TLS version
openssl s_client -connect localhost:9092 -tls1_3

# Test with client certificate (mTLS)
openssl s_client -connect localhost:9092 \
  -cert client.pem -key client-key.pem -CAfile ca.pem
```

## Client Configuration

### Java/Kafka Client
```properties
bootstrap.servers=localhost:9092
security.protocol=SSL
ssl.truststore.location=/path/to/truststore.jks
ssl.truststore.password=password

# For mTLS
ssl.keystore.location=/path/to/keystore.jks
ssl.keystore.password=password
```

### Go/Sarama Client
```go
config := sarama.NewConfig()
config.Net.TLS.Enable = true
config.Net.TLS.Config = &tls.Config{
    RootCAs: caCertPool,
}

// For mTLS
cert, _ := tls.LoadX509KeyPair("client.pem", "client-key.pem")
config.Net.TLS.Config.Certificates = []tls.Certificate{cert}
```

## Common Commands

```bash
# Run all TLS tests
go test ./pkg/tls/... -v

# Run server integration tests
go test ./pkg/kafka/server/... -run TestServerTLS -v

# Run performance benchmarks
go test -bench=BenchmarkTLS ./pkg/tls/...

# Build with TLS support
go build ./cmd/takhin
```

## Configuration Options

| Option | Values | Default | Description |
|--------|--------|---------|-------------|
| `enabled` | true/false | false | Enable TLS |
| `cert.file` | path | - | Server certificate |
| `key.file` | path | - | Server private key |
| `ca.file` | path | - | CA certificate |
| `client.auth` | none/request/require | none | Client auth mode |
| `verify.client.cert` | true/false | false | Verify client certs |
| `min.version` | TLS1.0-1.3 | TLS1.2 | Minimum TLS version |

## Cipher Suites

### TLS 1.2 (Recommended)
- `TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256`
- `TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384`

### TLS 1.3 (Recommended)
- `TLS_AES_128_GCM_SHA256`
- `TLS_AES_256_GCM_SHA384`
- `TLS_CHACHA20_POLY1305_SHA256`

## Troubleshooting

### Certificate verification failed
- Check certificate paths
- Verify CA certificate is correct
- Ensure certificates are not expired

### TLS handshake timeout
- Check network connectivity
- Verify firewall rules
- Check certificate validity

### Client authentication failed
- Verify client certificate is signed by CA
- Check client certificate is not expired
- Ensure CA file is configured

## Security Checklist

- [ ] Use TLS 1.2 or higher
- [ ] Use strong cipher suites (ECDHE)
- [ ] Protect private keys (chmod 600)
- [ ] Use trusted CA in production
- [ ] Monitor certificate expiration
- [ ] Enable mTLS for sensitive environments
- [ ] Disable weak TLS versions (1.0, 1.1)

## Performance Tips

1. Use TLS 1.3 for faster handshakes
2. Enable connection pooling
3. Use hardware acceleration (AES-NI)
4. Monitor CPU usage
5. Consider load balancing for high traffic

## Documentation

- Full docs: `backend/docs/TLS.md`
- Config example: `backend/configs/takhin.yaml`
- Tests: `backend/pkg/tls/*_test.go`

## Support

For issues or questions:
1. Check documentation: `backend/docs/TLS.md`
2. Review test files for examples
3. Check logs with debug level: `logging.level=debug`
