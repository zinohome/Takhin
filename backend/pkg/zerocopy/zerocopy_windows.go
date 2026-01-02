// Copyright 2025 Takhin Data, Inc.

//go:build windows
// +build windows

package zerocopy

import (
	"io"
	"os"
)

// sendFile fallback to regular copy on Windows (TransmitFile is complex to implement).
func sendFile(dst io.Writer, src *os.File, offset int64, count int64) (int64, error) {
	return fallbackCopy(dst, src, offset, count)
}

// copyFileRange fallback to regular copy on Windows.
func copyFileRange(dst *os.File, src *os.File, offset int64, count int64) (int64, error) {
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
