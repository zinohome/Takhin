// Copyright 2025 Takhin Data, Inc.

//go:build darwin
// +build darwin

package zerocopy

import (
	"os"
	"syscall"
)

// sendFileImpl performs the actual sendfile syscall on macOS/Darwin.
func sendFileImpl(dstFd int, src *os.File, offset int64, count int64) (int64, error) {
	srcFd := int(src.Fd())
	
	// macOS sendfile signature: sendfile(int fd, int s, off_t offset, off_t *len, struct sf_hdtr *hdtr, int flags)
	// But Go's syscall.Sendfile has signature: func Sendfile(outfd int, infd int, offset *int64, count int) (written int64, err error)
	var written int64
	off := offset
	
	w, err := syscall.Sendfile(dstFd, srcFd, &off, int(count))
	written = int64(w)
	
	if err != nil {
		if err == syscall.EINTR || err == syscall.EAGAIN {
			if written > 0 {
				return written, nil
			}
		}
		return 0, err
	}
	
	return written, nil
}
