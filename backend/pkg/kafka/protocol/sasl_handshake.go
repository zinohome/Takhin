// Copyright 2025 Takhin Data, Inc.

package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
)

// SaslHandshakeRequest represents a SASL handshake request
type SaslHandshakeRequest struct {
	Mechanism string
}

// SaslHandshakeResponse represents a SASL handshake response
type SaslHandshakeResponse struct {
	ErrorCode         ErrorCode
	EnabledMechanisms []string
}

// DecodeSaslHandshakeRequest decodes a SASL handshake request
func DecodeSaslHandshakeRequest(r io.Reader, version int16) (*SaslHandshakeRequest, error) {
	// Read Mechanism
	mechanism, err := ReadString(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read mechanism: %w", err)
	}

	return &SaslHandshakeRequest{
		Mechanism: mechanism,
	}, nil
}

// EncodeSaslHandshakeResponse encodes a SASL handshake response
func EncodeSaslHandshakeResponse(resp *SaslHandshakeResponse, version int16) ([]byte, error) {
	buf := make([]byte, 0, 256)

	// Write ErrorCode
	errCode := make([]byte, 2)
	binary.BigEndian.PutUint16(errCode, uint16(resp.ErrorCode))
	buf = append(buf, errCode...)

	// Write EnabledMechanisms array
	mechanismsLen := make([]byte, 4)
	binary.BigEndian.PutUint32(mechanismsLen, uint32(len(resp.EnabledMechanisms)))
	buf = append(buf, mechanismsLen...)

	for _, mechanism := range resp.EnabledMechanisms {
		buf = append(buf, encodeString(mechanism)...)
	}

	return buf, nil
}

// DecodeSaslHandshakeResponse decodes a SASL handshake response
func DecodeSaslHandshakeResponse(r io.Reader, version int16) (*SaslHandshakeResponse, error) {
	// Read ErrorCode
	errorCode, err := ReadInt16(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read error_code: %w", err)
	}

	// Read EnabledMechanisms array
	mechanismsLen, err := ReadInt32(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read enabled_mechanisms length: %w", err)
	}

	mechanisms := make([]string, mechanismsLen)
	for i := int32(0); i < mechanismsLen; i++ {
		mechanism, err := ReadString(r)
		if err != nil {
			return nil, fmt.Errorf("failed to read mechanism: %w", err)
		}
		mechanisms[i] = mechanism
	}

	return &SaslHandshakeResponse{
		ErrorCode:         ErrorCode(errorCode),
		EnabledMechanisms: mechanisms,
	}, nil
}

// WriteSaslHandshakeResponse writes the response to a writer
func WriteSaslHandshakeResponse(w io.Writer, header *RequestHeader, resp *SaslHandshakeResponse) error {
	// Encode the response
	respData, err := EncodeSaslHandshakeResponse(resp, header.APIVersion)
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
