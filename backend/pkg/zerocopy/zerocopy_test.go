// Copyright 2025 Takhin Data, Inc.

package zerocopy

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"testing"
)

func TestSendFile(t *testing.T) {
	// Create a temporary file with test data
	tmpFile, err := os.CreateTemp("", "zerocopy_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Write test data
	testData := []byte("Hello, zero-copy world! This is test data for sendfile.")
	if _, err := tmpFile.Write(testData); err != nil {
		t.Fatal(err)
	}

	// Sync to ensure data is written
	if err := tmpFile.Sync(); err != nil {
		t.Fatal(err)
	}

	// Test with buffer writer (fallback path)
	t.Run("BufferWriter", func(t *testing.T) {
		var buf bytes.Buffer
		n, err := SendFile(&buf, tmpFile, 0, int64(len(testData)))
		if err != nil {
			t.Fatalf("SendFile failed: %v", err)
		}
		if n != int64(len(testData)) {
			t.Errorf("Expected to write %d bytes, got %d", len(testData), n)
		}
		if !bytes.Equal(buf.Bytes(), testData) {
			t.Errorf("Data mismatch: expected %q, got %q", testData, buf.Bytes())
		}
	})

	// Test with TCP connection (zero-copy path on supported platforms)
	t.Run("TCPConnection", func(t *testing.T) {
		// Create a TCP server
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer listener.Close()

		// Channel to receive data
		dataChan := make(chan []byte, 1)
		errChan := make(chan error, 1)

		// Start server goroutine
		go func() {
			conn, err := listener.Accept()
			if err != nil {
				errChan <- err
				return
			}
			defer conn.Close()

			data, err := io.ReadAll(conn)
			if err != nil {
				errChan <- err
				return
			}
			dataChan <- data
		}()

		// Connect to server
		conn, err := net.Dial("tcp", listener.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()

		// Send file data
		tcpConn := conn.(*net.TCPConn)
		n, err := SendFile(tcpConn, tmpFile, 0, int64(len(testData)))
		if err != nil {
			t.Fatalf("SendFile failed: %v", err)
		}
		if n != int64(len(testData)) {
			t.Errorf("Expected to write %d bytes, got %d", len(testData), n)
		}

		// Close write side to signal EOF
		conn.(*net.TCPConn).CloseWrite()

		// Wait for data
		select {
		case data := <-dataChan:
			if !bytes.Equal(data, testData) {
				t.Errorf("Data mismatch: expected %q, got %q", testData, data)
			}
		case err := <-errChan:
			t.Fatalf("Server error: %v", err)
		}
	})
}

func TestSendFilePartial(t *testing.T) {
	// Create a temporary file with test data
	tmpFile, err := os.CreateTemp("", "zerocopy_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Write test data
	testData := []byte("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	if _, err := tmpFile.Write(testData); err != nil {
		t.Fatal(err)
	}
	if err := tmpFile.Sync(); err != nil {
		t.Fatal(err)
	}

	// Test partial read
	var buf bytes.Buffer
	n, err := SendFile(&buf, tmpFile, 10, 10)
	if err != nil {
		t.Fatalf("SendFile failed: %v", err)
	}
	if n != 10 {
		t.Errorf("Expected to write 10 bytes, got %d", n)
	}
	expected := testData[10:20]
	if !bytes.Equal(buf.Bytes(), expected) {
		t.Errorf("Data mismatch: expected %q, got %q", expected, buf.Bytes())
	}
}

func BenchmarkSendFile(b *testing.B) {
	sizes := []int64{
		1024,        // 1KB
		64 * 1024,   // 64KB
		1024 * 1024, // 1MB
	}

	for _, size := range sizes {
		b.Run(formatSize(size), func(b *testing.B) {
			benchmarkSendFile(b, size)
		})
	}
}

func benchmarkSendFile(b *testing.B, size int64) {
	// Create temporary file with test data
	tmpFile, err := os.CreateTemp("", "benchmark")
	if err != nil {
		b.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Write test data
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 256)
	}
	if _, err := tmpFile.Write(data); err != nil {
		b.Fatal(err)
	}
	if err := tmpFile.Sync(); err != nil {
		b.Fatal(err)
	}

	// Create a buffer writer for benchmarking (to avoid network overhead)
	buf := make([]byte, size)
	writer := bytes.NewBuffer(buf[:0])

	b.ResetTimer()
	b.SetBytes(size)

	for i := 0; i < b.N; i++ {
		writer.Reset()
		_, err := SendFile(writer, tmpFile, 0, size)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRegularCopy(b *testing.B) {
	sizes := []int64{
		1024,        // 1KB
		64 * 1024,   // 64KB
		1024 * 1024, // 1MB
	}

	for _, size := range sizes {
		b.Run(formatSize(size), func(b *testing.B) {
			benchmarkRegularCopy(b, size)
		})
	}
}

func benchmarkRegularCopy(b *testing.B, size int64) {
	// Create temporary file with test data
	tmpFile, err := os.CreateTemp("", "benchmark")
	if err != nil {
		b.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Write test data
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 256)
	}
	if _, err := tmpFile.Write(data); err != nil {
		b.Fatal(err)
	}
	if err := tmpFile.Sync(); err != nil {
		b.Fatal(err)
	}

	// Create a buffer writer for benchmarking (to match SendFile benchmark)
	buf := make([]byte, size)
	writer := bytes.NewBuffer(buf[:0])

	b.ResetTimer()
	b.SetBytes(size)

	for i := 0; i < b.N; i++ {
		writer.Reset()
		if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
			b.Fatal(err)
		}
		_, err := io.CopyN(writer, tmpFile, size)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func formatSize(size int64) string {
	switch {
	case size < 1024:
		return fmt.Sprintf("%dB", size)
	case size < 1024*1024:
		return fmt.Sprintf("%dKB", size/1024)
	default:
		return fmt.Sprintf("%dMB", size/(1024*1024))
	}
}
