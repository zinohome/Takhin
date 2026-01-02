# TLS/SSL Support for Takhin

## Overview

Takhin supports TLS/SSL encryption for secure communication between clients and the Kafka server. This includes:
- **TLS encryption** for data in transit
- **Certificate management** for server and client authentication
- **Mutual TLS (mTLS)** for client certificate verification
- **Configurable cipher suites** and TLS versions

## Configuration

### Basic TLS Configuration

Enable TLS in `configs/takhin.yaml`:

```yaml
server:
  host: "0.0.0.0"
  port: 9092
  tls:
    enabled: true
    cert:
      file: "/path/to/server.pem"
    key:
      file: "/path/to/server-key.pem"
    min:
      version: "TLS1.2"
```

### Environment Variables

Override configuration with environment variables:

```bash
export TAKHIN_SERVER_TLS_ENABLED=true
export TAKHIN_SERVER_TLS_CERT_FILE=/path/to/server.pem
export TAKHIN_SERVER_TLS_KEY_FILE=/path/to/server-key.pem
export TAKHIN_SERVER_TLS_MIN_VERSION=TLS1.3
```

### Mutual TLS (mTLS) Configuration

For client certificate verification:

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
      auth: "require"              # Options: none, request, require
    verify:
      client:
        cert: true                 # Enable client certificate verification
    min:
      version: "TLS1.2"
```

### Advanced Configuration

Configure cipher suites and other TLS options:

```yaml
server:
  tls:
    enabled: true
    cert:
      file: "/path/to/server.pem"
    key:
      file: "/path/to/server-key.pem"
    min:
      version: "TLS1.3"
    cipher:
      suites:
        - "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"
        - "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"
        - "TLS_AES_128_GCM_SHA256"
        - "TLS_AES_256_GCM_SHA384"
    prefer:
      server:
        cipher: true
```

## Certificate Management

### Generating Test Certificates

For development and testing, generate self-signed certificates:

```go
import tlsutil "github.com/takhin-data/takhin/pkg/tls"

// Generate server certificates
certFile, keyFile, caFile, err := tlsutil.GenerateTestCertificates("/path/to/certs")

// Generate client certificate for mTLS
clientCertFile, clientKeyFile, err := tlsutil.GenerateClientCertificate("/path/to/certs", caFile)
```

### Production Certificates

For production, use certificates from a trusted Certificate Authority (CA):

1. **Generate a private key:**
   ```bash
   openssl genrsa -out server-key.pem 2048
   ```

2. **Create a certificate signing request (CSR):**
   ```bash
   openssl req -new -key server-key.pem -out server.csr
   ```

3. **Get the certificate signed by your CA:**
   ```bash
   # Submit server.csr to your CA and receive server.pem
   ```

4. **Configure Takhin with the certificates:**
   ```yaml
   server:
     tls:
       enabled: true
       cert:
         file: "/path/to/server.pem"
       key:
         file: "/path/to/server-key.pem"
   ```

### Certificate Rotation

To rotate certificates without downtime:

1. Generate new certificates
2. Update configuration to point to new certificate files
3. Restart Takhin service

## Client Configuration

### Kafka Client with TLS

Configure Kafka clients to use TLS:

**Java Client:**
```java
Properties props = new Properties();
props.put("bootstrap.servers", "localhost:9092");
props.put("security.protocol", "SSL");
props.put("ssl.truststore.location", "/path/to/truststore.jks");
props.put("ssl.truststore.password", "password");
```

**Go Client (sarama):**
```go
config := sarama.NewConfig()
config.Net.TLS.Enable = true
config.Net.TLS.Config = &tls.Config{
    RootCAs: caCertPool,
}
```

### Client with mTLS

For mutual TLS, also configure client certificates:

**Java Client:**
```java
props.put("security.protocol", "SSL");
props.put("ssl.keystore.location", "/path/to/keystore.jks");
props.put("ssl.keystore.password", "password");
props.put("ssl.key.password", "password");
props.put("ssl.truststore.location", "/path/to/truststore.jks");
props.put("ssl.truststore.password", "password");
```

**Go Client:**
```go
cert, _ := tls.LoadX509KeyPair("client.pem", "client-key.pem")
config.Net.TLS.Config = &tls.Config{
    Certificates: []tls.Certificate{cert},
    RootCAs:      caCertPool,
}
```

## Supported TLS Versions

- **TLS 1.0** (deprecated, not recommended)
- **TLS 1.1** (deprecated, not recommended)
- **TLS 1.2** (default, recommended minimum)
- **TLS 1.3** (recommended)

## Supported Cipher Suites

### TLS 1.2 Cipher Suites:
- `TLS_RSA_WITH_AES_128_CBC_SHA`
- `TLS_RSA_WITH_AES_256_CBC_SHA`
- `TLS_RSA_WITH_AES_128_GCM_SHA256`
- `TLS_RSA_WITH_AES_256_GCM_SHA384`
- `TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA`
- `TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA`
- `TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA`
- `TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA`
- `TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256`
- `TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384`
- `TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256`
- `TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384`

### TLS 1.3 Cipher Suites:
- `TLS_AES_128_GCM_SHA256`
- `TLS_AES_256_GCM_SHA384`
- `TLS_CHACHA20_POLY1305_SHA256`

## Client Authentication Modes

| Mode      | Description                                      |
|-----------|--------------------------------------------------|
| `none`    | No client certificate required                   |
| `request` | Client certificate requested but not required    |
| `require` | Client certificate required and verified         |

## Performance Considerations

### TLS Overhead

TLS introduces computational overhead:
- **Handshake**: 1-2ms additional latency per connection
- **Throughput**: ~10-20% reduction compared to plain TCP
- **CPU**: Increased CPU usage for encryption/decryption

### Optimization Tips

1. **Use TLS 1.3**: Faster handshake (1-RTT vs 2-RTT in TLS 1.2)
2. **Enable session resumption**: Reduces handshake overhead
3. **Use hardware acceleration**: AES-NI for faster encryption
4. **Connection pooling**: Reuse TLS connections
5. **Prefer ECDHE cipher suites**: Better performance than RSA

### Performance Benchmarks

Run performance tests:

```bash
cd backend
go test -bench=. -benchmem ./pkg/tls/...
```

Example results:
```
BenchmarkTLSHandshake-8              1000    1234567 ns/op
BenchmarkTLSThroughput-8            10000     123456 ns/op   8192 B/op
BenchmarkMTLSHandshake-8             500    2345678 ns/op
```

## Security Best Practices

1. **Use strong TLS versions**: Minimum TLS 1.2, prefer TLS 1.3
2. **Disable weak ciphers**: Avoid CBC mode ciphers if possible
3. **Enable perfect forward secrecy**: Use ECDHE cipher suites
4. **Keep certificates updated**: Monitor expiration dates
5. **Use strong key sizes**: Minimum 2048-bit RSA or 256-bit ECDSA
6. **Protect private keys**: Restrict file permissions (chmod 600)
7. **Use trusted CAs**: Avoid self-signed certificates in production
8. **Enable certificate revocation**: Use OCSP or CRL

## Troubleshooting

### Common Issues

**Certificate verification failed:**
```
Error: failed to load certificate: x509: certificate signed by unknown authority
```
Solution: Ensure the CA certificate is correctly configured and trusted.

**TLS handshake timeout:**
```
Error: TLS handshake timeout
```
Solution: Check network connectivity and firewall rules. Verify certificate validity.

**Client authentication failed:**
```
Error: tls: bad certificate
```
Solution: Verify client certificate is signed by the configured CA and not expired.

**Cipher suite mismatch:**
```
Error: tls: no cipher suite supported by both client and server
```
Solution: Check cipher suite configuration on both client and server.

### Debug Logging

Enable debug logging to troubleshoot TLS issues:

```yaml
logging:
  level: "debug"
```

### Verify TLS Configuration

Test TLS connection with OpenSSL:

```bash
# Test basic TLS
openssl s_client -connect localhost:9092

# Test with specific TLS version
openssl s_client -connect localhost:9092 -tls1_3

# Test with client certificate (mTLS)
openssl s_client -connect localhost:9092 \
  -cert client.pem -key client-key.pem -CAfile ca.pem
```

## Testing

### Unit Tests

Run TLS unit tests:

```bash
cd backend
go test ./pkg/tls/...
```

### Integration Tests

Run server integration tests with TLS:

```bash
go test ./pkg/kafka/server/... -run TestServerTLS
```

### Performance Tests

Run TLS performance benchmarks:

```bash
go test -bench=BenchmarkTLS ./pkg/tls/... -benchtime=10s
```

## References

- [Go crypto/tls Documentation](https://pkg.go.dev/crypto/tls)
- [Kafka Security Documentation](https://kafka.apache.org/documentation/#security)
- [TLS Best Practices](https://www.ssllabs.com/projects/best-practices/)
- [OWASP TLS Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Transport_Layer_Protection_Cheat_Sheet.html)
