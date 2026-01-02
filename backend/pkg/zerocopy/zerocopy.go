// Copyright 2025 Takhin Data, Inc.

package zerocopy

import (
	"io"
	"os"
)

// SendFile transfers data from src file to dst writer using zero-copy I/O when possible.
// Returns the number of bytes written and any error encountered.
func SendFile(dst io.Writer, src *os.File, offset int64, count int64) (int64, error) {
	return sendFile(dst, src, offset, count)
}

// CopyFileRange copies data between two files using zero-copy I/O when possible.
// Returns the number of bytes copied and any error encountered.
func CopyFileRange(dst *os.File, src *os.File, offset int64, count int64) (int64, error) {
	return copyFileRange(dst, src, offset, count)
}

// Reader wraps an os.File and provides a zero-copy ReadAt interface.
type Reader struct {
	file *os.File
}

// NewReader creates a new zero-copy Reader.
func NewReader(file *os.File) *Reader {
	return &Reader{file: file}
}

// SendTo sends data from the file to the writer using zero-copy I/O.
func (r *Reader) SendTo(dst io.Writer, offset int64, count int64) (int64, error) {
	return SendFile(dst, r.file, offset, count)
}

// File returns the underlying file.
func (r *Reader) File() *os.File {
	return r.file
}
