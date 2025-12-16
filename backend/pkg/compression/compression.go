// Copyright 2025 Takhin Data, Inc.

package compression

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"

	"github.com/golang/snappy"
	"github.com/klauspost/compress/zstd"
	"github.com/pierrec/lz4/v4"
)

// Type represents compression type
type Type int8

const (
	None   Type = 0
	GZIP   Type = 1
	Snappy Type = 2
	LZ4    Type = 3
	ZSTD   Type = 4
)

// Compress compresses data using the specified type
func Compress(t Type, data []byte) ([]byte, error) {
	switch t {
	case None:
		return data, nil
	case GZIP:
		return compressGZIP(data)
	case Snappy:
		return compressSnappy(data)
	case LZ4:
		return compressLZ4(data)
	case ZSTD:
		return compressZSTD(data)
	default:
		return nil, fmt.Errorf("unsupported compression type: %d", t)
	}
}

// Decompress decompresses data using the specified type
func Decompress(t Type, data []byte) ([]byte, error) {
	switch t {
	case None:
		return data, nil
	case GZIP:
		return decompressGZIP(data)
	case Snappy:
		return decompressSnappy(data)
	case LZ4:
		return decompressLZ4(data)
	case ZSTD:
		return decompressZSTD(data)
	default:
		return nil, fmt.Errorf("unsupported compression type: %d", t)
	}
}

func compressGZIP(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)

	if _, err := w.Write(data); err != nil {
		return nil, err
	}

	if err := w.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func decompressGZIP(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()

	return io.ReadAll(r)
}

func compressSnappy(data []byte) ([]byte, error) {
	return snappy.Encode(nil, data), nil
}

func decompressSnappy(data []byte) ([]byte, error) {
	return snappy.Decode(nil, data)
}

func compressLZ4(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := lz4.NewWriter(&buf)

	if _, err := w.Write(data); err != nil {
		return nil, err
	}

	if err := w.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func decompressLZ4(data []byte) ([]byte, error) {
	r := lz4.NewReader(bytes.NewReader(data))
	return io.ReadAll(r)
}

func compressZSTD(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w, err := zstd.NewWriter(&buf)
	if err != nil {
		return nil, err
	}

	if _, err := w.Write(data); err != nil {
		return nil, err
	}

	if err := w.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func decompressZSTD(data []byte) ([]byte, error) {
	r, err := zstd.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()

	return io.ReadAll(r)
}
