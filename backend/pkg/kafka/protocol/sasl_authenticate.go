// Copyright 2025 Takhin Data, Inc.

package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
)

// SaslAuthenticateRequest represents a SASL authenticate request
type SaslAuthenticateRequest struct {
	AuthBytes []byte
}

// SaslAuthenticateResponse represents a SASL authenticate response
type SaslAuthenticateResponse struct {
	ErrorCode         ErrorCode
	ErrorMessage      *string
	AuthBytes         []byte
	SessionLifetimeMs int64
}

// DecodeSaslAuthenticateRequest decodes a SASL authenticate request
func DecodeSaslAuthenticateRequest(r io.Reader, version int16) (*SaslAuthenticateRequest, error) {
	// Read AuthBytes
	authBytesLen, err := ReadInt32(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read auth_bytes length: %w", err)
	}

	authBytes := make([]byte, authBytesLen)
	if _, err := io.ReadFull(r, authBytes); err != nil {
		return nil, fmt.Errorf("failed to read auth_bytes: %w", err)
	}

	return &SaslAuthenticateRequest{
		AuthBytes: authBytes,
	}, nil
}

// EncodeSaslAuthenticateResponse encodes a SASL authenticate response
func EncodeSaslAuthenticateResponse(resp *SaslAuthenticateResponse, version int16) ([]byte, error) {
	buf := make([]byte, 0, 256)

	// Write ErrorCode
	errCode := make([]byte, 2)
	binary.BigEndian.PutUint16(errCode, uint16(resp.ErrorCode))
	buf = append(buf, errCode...)

	// Write ErrorMessage (nullable string)
	if resp.ErrorMessage == nil {
		// Null string (-1)
		buf = append(buf, 0xFF, 0xFF)
	} else {
		msgLen := make([]byte, 2)
		binary.BigEndian.PutUint16(msgLen, uint16(len(*resp.ErrorMessage)))
		buf = append(buf, msgLen...)
		buf = append(buf, []byte(*resp.ErrorMessage)...)
	}

	// Write AuthBytes
	authBytesLen := make([]byte, 4)
	binary.BigEndian.PutUint32(authBytesLen, uint32(len(resp.AuthBytes)))
	buf = append(buf, authBytesLen...)
	buf = append(buf, resp.AuthBytes...)

	// Write SessionLifetimeMs (version 1+)
	if version >= 1 {
		sessionLifetime := make([]byte, 8)
		binary.BigEndian.PutUint64(sessionLifetime, uint64(resp.SessionLifetimeMs))
		buf = append(buf, sessionLifetime...)
	}

	return buf, nil
}

// DecodeSaslAuthenticateResponse decodes a SASL authenticate response
func DecodeSaslAuthenticateResponse(r io.Reader, version int16) (*SaslAuthenticateResponse, error) {
	// Read ErrorCode
	errorCode, err := ReadInt16(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read error_code: %w", err)
	}

	// Read ErrorMessage (nullable string)
	errorMessage, err := ReadNullableString(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read error_message: %w", err)
	}

	// Read AuthBytes
	authBytesLen, err := ReadInt32(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read auth_bytes length: %w", err)
	}

	authBytes := make([]byte, authBytesLen)
	if _, err := io.ReadFull(r, authBytes); err != nil {
		return nil, fmt.Errorf("failed to read auth_bytes: %w", err)
	}

	// Read SessionLifetimeMs (version 1+)
	var sessionLifetimeMs int64
	if version >= 1 {
		sessionLifetimeMs, err = ReadInt64(r)
		if err != nil {
			return nil, fmt.Errorf("failed to read session_lifetime_ms: %w", err)
		}
	}

	return &SaslAuthenticateResponse{
		ErrorCode:         ErrorCode(errorCode),
		ErrorMessage:      errorMessage,
		AuthBytes:         authBytes,
		SessionLifetimeMs: sessionLifetimeMs,
	}, nil
}

// WriteSaslAuthenticateResponse writes the response to a writer
func WriteSaslAuthenticateResponse(w io.Writer, header *RequestHeader, resp *SaslAuthenticateResponse) error {
	// Encode the response
	respData, err := EncodeSaslAuthenticateResponse(resp, header.APIVersion)
	if err != nil {
		return err
	}

	// Write correlation ID
	if err := binary.Write(w, binary.BigEndian, header.CorrelationID); err != nil {
		return err
	}

	// Write response data
	if _, err := w.Write(respData); err != nil {
		return err
	}

	return nil
}
