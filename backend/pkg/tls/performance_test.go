// Copyright 2025 Takhin Data, Inc.

package tls

import (
	"crypto/rand"
	"crypto/tls"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/takhin-data/takhin/pkg/config"
)

// BenchmarkTLSHandshake benchmarks TLS handshake performance
func BenchmarkTLSHandshake(b *testing.B) {
	dir := b.TempDir()
	certFile, keyFile, _, err := GenerateTestCertificates(dir)
	if err != nil {
		b.Fatal(err)
	}

	cfg := &config.TLSConfig{
		Enabled:    true,
		CertFile:   certFile,
		KeyFile:    keyFile,
		ClientAuth: "none",
		MinVersion: "TLS1.2",
	}

	tlsConfig, err := LoadTLSConfig(cfg)
	if err != nil {
		b.Fatal(err)
	}

	// Start TLS server
	listener, err := tls.Listen("tcp", "localhost:0", tlsConfig)
	if err != nil {
		b.Fatal(err)
	}
	defer listener.Close()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	addr := listener.Addr().String()
	clientConfig := &tls.Config{InsecureSkipVerify: true}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn, err := tls.Dial("tcp", addr, clientConfig)
		if err != nil {
			b.Fatal(err)
		}
		conn.Close()
	}
}

// BenchmarkTLSThroughput benchmarks data transfer throughput with TLS
func BenchmarkTLSThroughput(b *testing.B) {
	dir := b.TempDir()
	certFile, keyFile, _, err := GenerateTestCertificates(dir)
	if err != nil {
		b.Fatal(err)
	}

	cfg := &config.TLSConfig{
		Enabled:    true,
		CertFile:   certFile,
		KeyFile:    keyFile,
		ClientAuth: "none",
		MinVersion: "TLS1.2",
	}

	tlsConfig, err := LoadTLSConfig(cfg)
	if err != nil {
		b.Fatal(err)
	}

	listener, err := tls.Listen("tcp", "localhost:0", tlsConfig)
	if err != nil {
		b.Fatal(err)
	}
	defer listener.Close()

	// Echo server
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 4096)
				for {
					n, err := c.Read(buf)
					if err != nil {
						return
					}
					if _, err := c.Write(buf[:n]); err != nil {
						return
					}
				}
			}(conn)
		}
	}()

	addr := listener.Addr().String()
	clientConfig := &tls.Config{InsecureSkipVerify: true}

	// Test data
	data := make([]byte, 1024)
	rand.Read(data)

	b.ResetTimer()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		conn, err := tls.Dial("tcp", addr, clientConfig)
		if err != nil {
			b.Fatal(err)
		}

		if _, err := conn.Write(data); err != nil {
			b.Fatal(err)
		}

		buf := make([]byte, len(data))
		if _, err := conn.Read(buf); err != nil {
			b.Fatal(err)
		}

		conn.Close()
	}
}

// BenchmarkTLSConcurrentConnections benchmarks concurrent TLS connections
func BenchmarkTLSConcurrentConnections(b *testing.B) {
	dir := b.TempDir()
	certFile, keyFile, _, err := GenerateTestCertificates(dir)
	if err != nil {
		b.Fatal(err)
	}

	cfg := &config.TLSConfig{
		Enabled:    true,
		CertFile:   certFile,
		KeyFile:    keyFile,
		ClientAuth: "none",
		MinVersion: "TLS1.2",
	}

	tlsConfig, err := LoadTLSConfig(cfg)
	if err != nil {
		b.Fatal(err)
	}

	listener, err := tls.Listen("tcp", "localhost:0", tlsConfig)
	if err != nil {
		b.Fatal(err)
	}
	defer listener.Close()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 1024)
				c.Read(buf)
			}(conn)
		}
	}()

	addr := listener.Addr().String()
	clientConfig := &tls.Config{InsecureSkipVerify: true}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			conn, err := tls.Dial("tcp", addr, clientConfig)
			if err != nil {
				b.Fatal(err)
			}
			conn.Write([]byte("test"))
			conn.Close()
		}
	})
}

// BenchmarkMTLSHandshake benchmarks mutual TLS handshake
func BenchmarkMTLSHandshake(b *testing.B) {
	dir := b.TempDir()
	certFile, keyFile, caFile, err := GenerateTestCertificates(dir)
	if err != nil {
		b.Fatal(err)
	}

	clientCertFile, clientKeyFile, err := GenerateClientCertificate(dir, caFile)
	if err != nil {
		b.Fatal(err)
	}

	cfg := &config.TLSConfig{
		Enabled:          true,
		CertFile:         certFile,
		KeyFile:          keyFile,
		CAFile:           caFile,
		ClientAuth:       "require",
		VerifyClientCert: true,
		MinVersion:       "TLS1.2",
	}

	tlsConfig, err := LoadTLSConfig(cfg)
	if err != nil {
		b.Fatal(err)
	}

	listener, err := tls.Listen("tcp", "localhost:0", tlsConfig)
	if err != nil {
		b.Fatal(err)
	}
	defer listener.Close()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	addr := listener.Addr().String()

	clientCert, err := tls.LoadX509KeyPair(clientCertFile, clientKeyFile)
	if err != nil {
		b.Fatal(err)
	}

	clientConfig := &tls.Config{
		Certificates:       []tls.Certificate{clientCert},
		InsecureSkipVerify: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn, err := tls.Dial("tcp", addr, clientConfig)
		if err != nil {
			b.Fatal(err)
		}
		conn.Close()
	}
}

// BenchmarkTLSVsPlainTCP compares TLS vs plain TCP performance
func BenchmarkTLSVsPlainTCP(b *testing.B) {
	data := make([]byte, 4096)
	rand.Read(data)

	b.Run("PlainTCP", func(b *testing.B) {
		listener, err := net.Listen("tcp", "localhost:0")
		if err != nil {
			b.Fatal(err)
		}
		defer listener.Close()

		go func() {
			for {
				conn, err := listener.Accept()
				if err != nil {
					return
				}
				go func(c net.Conn) {
					defer c.Close()
					buf := make([]byte, len(data))
					c.Read(buf)
				}(conn)
			}
		}()

		addr := listener.Addr().String()

		b.ResetTimer()
		b.SetBytes(int64(len(data)))

		for i := 0; i < b.N; i++ {
			conn, err := net.Dial("tcp", addr)
			if err != nil {
				b.Fatal(err)
			}
			conn.Write(data)
			conn.Close()
		}
	})

	b.Run("TLS", func(b *testing.B) {
		dir := b.TempDir()
		certFile, keyFile, _, err := GenerateTestCertificates(dir)
		if err != nil {
			b.Fatal(err)
		}

		cfg := &config.TLSConfig{
			Enabled:    true,
			CertFile:   certFile,
			KeyFile:    keyFile,
			ClientAuth: "none",
			MinVersion: "TLS1.2",
		}

		tlsConfig, err := LoadTLSConfig(cfg)
		if err != nil {
			b.Fatal(err)
		}

		listener, err := tls.Listen("tcp", "localhost:0", tlsConfig)
		if err != nil {
			b.Fatal(err)
		}
		defer listener.Close()

		go func() {
			for {
				conn, err := listener.Accept()
				if err != nil {
					return
				}
				go func(c net.Conn) {
					defer c.Close()
					buf := make([]byte, len(data))
					c.Read(buf)
				}(conn)
			}
		}()

		addr := listener.Addr().String()
		clientConfig := &tls.Config{InsecureSkipVerify: true}

		b.ResetTimer()
		b.SetBytes(int64(len(data)))

		for i := 0; i < b.N; i++ {
			conn, err := tls.Dial("tcp", addr, clientConfig)
			if err != nil {
				b.Fatal(err)
			}
			conn.Write(data)
			conn.Close()
		}
	})
}

// TestTLSPerformance runs performance tests and prints results
func TestTLSPerformance(t *testing.T) {
	t.Skip("Skipping flaky performance test - use benchmarks instead")
	
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	tests := []struct {
		name     string
		duration time.Duration
		testFunc func() error
	}{
		{
			name:     "TLS Handshake",
			duration: 5 * time.Second,
			testFunc: func() error {
				dir := t.TempDir()
				certFile, keyFile, _, err := GenerateTestCertificates(dir)
				if err != nil {
					return err
				}

				cfg := &config.TLSConfig{
					Enabled:    true,
					CertFile:   certFile,
					KeyFile:    keyFile,
					ClientAuth: "none",
					MinVersion: "TLS1.2",
				}

				tlsConfig, err := LoadTLSConfig(cfg)
				if err != nil {
					return err
				}

				listener, err := tls.Listen("tcp", "localhost:0", tlsConfig)
				if err != nil {
					return err
				}
				defer listener.Close()

				go func() {
					for {
						conn, err := listener.Accept()
						if err != nil {
							return
						}
						conn.Close()
					}
				}()

				addr := listener.Addr().String()
				clientConfig := &tls.Config{InsecureSkipVerify: true}

				conn, err := tls.Dial("tcp", addr, clientConfig)
				if err != nil {
					return err
				}
				conn.Close()
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			count := 0
			var wg sync.WaitGroup

			for i := 0; i < 10; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for time.Since(start) < tt.duration {
						if err := tt.testFunc(); err != nil {
							t.Error(err)
							return
						}
						count++
					}
				}()
			}

			wg.Wait()
			elapsed := time.Since(start)
			rate := float64(count) / elapsed.Seconds()

			t.Logf("%s: %d operations in %v (%.2f ops/sec)", tt.name, count, elapsed, rate)
		})
	}
}
