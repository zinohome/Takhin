// Copyright 2025 Takhin Data, Inc.

package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
)

// ApiVersionsRequest represents an ApiVersions request
type ApiVersionsRequest struct {
	ClientSoftwareName    string
	ClientSoftwareVersion string
}

// ApiVersionsResponse represents an ApiVersions response
type ApiVersionsResponse struct {
	ErrorCode      ErrorCode
	APIVersions    []APIVersion
	ThrottleTimeMs int32
}

// APIVersion represents a supported API version range
type APIVersion struct {
	APIKey     int16
	MinVersion int16
	MaxVersion int16
}

// DecodeApiVersionsRequest decodes an ApiVersions request
func DecodeApiVersionsRequest(r io.Reader, version int16) (*ApiVersionsRequest, error) {
	req := &ApiVersionsRequest{}

	// Version 3+ includes client software name and version
	if version >= 3 {
		clientSoftwareName, err := ReadString(r)
		if err != nil {
			return nil, fmt.Errorf("failed to read client_software_name: %w", err)
		}
		req.ClientSoftwareName = clientSoftwareName

		clientSoftwareVersion, err := ReadString(r)
		if err != nil {
			return nil, fmt.Errorf("failed to read client_software_version: %w", err)
		}
		req.ClientSoftwareVersion = clientSoftwareVersion
	}

	return req, nil
}

// EncodeApiVersionsResponse encodes an ApiVersions response
func EncodeApiVersionsResponse(resp *ApiVersionsResponse, version int16) ([]byte, error) {
	buf := make([]byte, 0, 256)

	// Write ErrorCode
	errCode := make([]byte, 2)
	binary.BigEndian.PutUint16(errCode, uint16(resp.ErrorCode))
	buf = append(buf, errCode...)

	// Write APIVersions array
	apiVersionsLen := make([]byte, 4)
	binary.BigEndian.PutUint32(apiVersionsLen, uint32(len(resp.APIVersions)))
	buf = append(buf, apiVersionsLen...)

	for _, apiVersion := range resp.APIVersions {
		// Write APIKey
		apiKey := make([]byte, 2)
		binary.BigEndian.PutUint16(apiKey, uint16(apiVersion.APIKey))
		buf = append(buf, apiKey...)

		// Write MinVersion
		minVersion := make([]byte, 2)
		binary.BigEndian.PutUint16(minVersion, uint16(apiVersion.MinVersion))
		buf = append(buf, minVersion...)

		// Write MaxVersion
		maxVersion := make([]byte, 2)
		binary.BigEndian.PutUint16(maxVersion, uint16(apiVersion.MaxVersion))
		buf = append(buf, maxVersion...)
	}

	// Write ThrottleTimeMs (version 1+)
	if version >= 1 {
		throttle := make([]byte, 4)
		binary.BigEndian.PutUint32(throttle, uint32(resp.ThrottleTimeMs))
		buf = append(buf, throttle...)
	}

	return buf, nil
}

// DecodeApiVersionsResponse decodes an ApiVersions response
func DecodeApiVersionsResponse(r io.Reader, version int16) (*ApiVersionsResponse, error) {
	// Read ErrorCode
	errorCode, err := ReadInt16(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read error_code: %w", err)
	}

	// Read APIVersions array
	apiVersionsLen, err := ReadInt32(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read api_versions length: %w", err)
	}

	apiVersions := make([]APIVersion, apiVersionsLen)
	for i := int32(0); i < apiVersionsLen; i++ {
		apiKey, err := ReadInt16(r)
		if err != nil {
			return nil, fmt.Errorf("failed to read api_key: %w", err)
		}

		minVersion, err := ReadInt16(r)
		if err != nil {
			return nil, fmt.Errorf("failed to read min_version: %w", err)
		}

		maxVersion, err := ReadInt16(r)
		if err != nil {
			return nil, fmt.Errorf("failed to read max_version: %w", err)
		}

		apiVersions[i] = APIVersion{
			APIKey:     apiKey,
			MinVersion: minVersion,
			MaxVersion: maxVersion,
		}
	}

	// Read ThrottleTimeMs (version 1+)
	var throttleTimeMs int32
	if version >= 1 {
		throttleTimeMs, err = ReadInt32(r)
		if err != nil {
			return nil, fmt.Errorf("failed to read throttle_time_ms: %w", err)
		}
	}

	return &ApiVersionsResponse{
		ErrorCode:      ErrorCode(errorCode),
		APIVersions:    apiVersions,
		ThrottleTimeMs: throttleTimeMs,
	}, nil
}

// WriteApiVersionsResponse writes the response to a writer
func WriteApiVersionsResponse(w io.Writer, header *RequestHeader, resp *ApiVersionsResponse) error {
	// Encode the response
	respData, err := EncodeApiVersionsResponse(resp, header.APIVersion)
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
