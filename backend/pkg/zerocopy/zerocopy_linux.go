// Copyright 2025 Takhin Data, Inc.

//go:build linux
// +build linux

package zerocopy

import (
	"os"
	"syscall"
)

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
	dstFd := int(dst.Fd())
	srcFd := int(src.Fd())

	var written int64
	remaining := count
	off := offset

	for remaining > 0 {
		n, err := syscall.CopyFileRange(srcFd, &off, dstFd, nil, int(remaining), 0)
		written += int64(n)
		remaining -= int64(n)

		if err != nil {
			if err == syscall.EINTR || err == syscall.EAGAIN {
				continue
			}
			// If copy_file_range is not supported, fallback
			if err == syscall.ENOSYS || err == syscall.EXDEV {
				return fallbackCopy(dst, src, offset, count)
			}
			return written, err
		}

		if n == 0 {
			break
		}
	}

	return written, nil
}
