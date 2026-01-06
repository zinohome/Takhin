// Copyright 2025 Takhin Data, Inc.

//go:build linux
// +build linux

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
		return sendFileToConn(w, src, offset, count)
	case interface{ File() (*os.File, error) }:
		if file, err := w.File(); err == nil {
			defer file.Close()
			return sendFileToFile(file, src, offset, count)
		}
	}
	return fallbackCopy(dst, src, offset, count)
}

// sendFileToConn uses sendfile syscall to send data to a TCP connection.
func sendFileToConn(conn *net.TCPConn, src *os.File, offset int64, count int64) (int64, error) {
	rawConn, err := conn.SyscallConn()
	if err != nil {
		return fallbackCopy(conn, src, offset, count)
	}

	var written int64
	var sendErr error

	err = rawConn.Write(func(dstFd uintptr) bool {
		written, sendErr = sendFileImpl(int(dstFd), src, offset, count)
		return sendErr == nil
	})

	if err != nil {
		return fallbackCopy(conn, src, offset, count)
	}

	if sendErr != nil {
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

// sendFileImpl performs the actual sendfile syscall on Linux.
func sendFileImpl(dstFd int, src *os.File, offset int64, count int64) (int64, error) {
	srcFd := int(src.Fd())
	var written int64

	remaining := count
	off := offset

	for remaining > 0 {
		// Linux sendfile can send up to 2GB at a time
		n, err := syscall.Sendfile(dstFd, srcFd, &off, int(remaining))
		written += int64(n)
		remaining -= int64(n)

		if err != nil {
			if err == syscall.EINTR || err == syscall.EAGAIN {
				continue
			}
			return written, err
		}

		if n == 0 {
			break
		}
	}

	return written, nil
}

// copyFileRange uses the copy_file_range syscall (Linux kernel 4.5+).
func copyFileRange(dst *os.File, src *os.File, offset int64, count int64) (int64, error) {
	// copy_file_range syscall is not available in standard Go syscall package
	// Fallback to regular copy for portability across Go versions
	return fallbackCopy(dst, src, offset, count)
}

// fallbackCopy performs a regular buffer-based copy.
func fallbackCopy(dst io.Writer, src *os.File, offset int64, count int64) (int64, error) {
	_, err := src.Seek(offset, io.SeekStart)
	if err != nil {
		return 0, err
	}
	return io.CopyN(dst, src, count)
}
