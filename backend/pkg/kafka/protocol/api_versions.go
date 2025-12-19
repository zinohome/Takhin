// Copyright 2025 Takhin Data, Inc.

package protocol

import (
	"io"
)

// ApiVersionsRequest represents an ApiVersions request
type ApiVersionsRequest struct {
	Header *RequestHeader
}

// ApiVersionsResponse represents an ApiVersions response
type ApiVersionsResponse struct {
	ErrorCode   ErrorCode
	APIVersions []APIVersion
}

// APIVersion represents a supported API version range
type APIVersion struct {
	APIKey     APIKey
	MinVersion int16
	MaxVersion int16
}

// DecodeApiVersionsRequest decodes an ApiVersions request
func DecodeApiVersionsRequest(r io.Reader, header *RequestHeader) (*ApiVersionsRequest, error) {
	return &ApiVersionsRequest{
		Header: header,
	}, nil
}

// Encode encodes the ApiVersions response
func (r *ApiVersionsResponse) Encode(w io.Writer) error {
	// Write error code
	if err := WriteInt16(w, int16(r.ErrorCode)); err != nil {
		return err
	}

	// Write array length
	if err := WriteArray(w, len(r.APIVersions)); err != nil {
		return err
	}

	// Write each API version
	for _, v := range r.APIVersions {
		if err := WriteInt16(w, int16(v.APIKey)); err != nil {
			return err
		}
		if err := WriteInt16(w, v.MinVersion); err != nil {
			return err
		}
		if err := WriteInt16(w, v.MaxVersion); err != nil {
			return err
		}
	}

	return nil
}

// GetSupportedAPIVersions returns the list of supported API versions
func GetSupportedAPIVersions() []APIVersion {
	return []APIVersion{
		{APIKey: ProduceKey, MinVersion: 0, MaxVersion: 7},
		{APIKey: FetchKey, MinVersion: 0, MaxVersion: 11},
		{APIKey: ListOffsetsKey, MinVersion: 0, MaxVersion: 5},
		{APIKey: MetadataKey, MinVersion: 0, MaxVersion: 9},
		{APIKey: OffsetCommitKey, MinVersion: 0, MaxVersion: 7},
		{APIKey: OffsetFetchKey, MinVersion: 0, MaxVersion: 6},
		{APIKey: FindCoordinatorKey, MinVersion: 0, MaxVersion: 2},
		{APIKey: JoinGroupKey, MinVersion: 0, MaxVersion: 5},
		{APIKey: HeartbeatKey, MinVersion: 0, MaxVersion: 3},
		{APIKey: LeaveGroupKey, MinVersion: 0, MaxVersion: 3},
		{APIKey: SyncGroupKey, MinVersion: 0, MaxVersion: 3},
		{APIKey: DescribeGroupsKey, MinVersion: 0, MaxVersion: 3},
		{APIKey: ListGroupsKey, MinVersion: 0, MaxVersion: 2},
		{APIKey: ApiVersionsKey, MinVersion: 0, MaxVersion: 2},
		{APIKey: CreateTopicsKey, MinVersion: 0, MaxVersion: 4},
		{APIKey: DeleteTopicsKey, MinVersion: 0, MaxVersion: 3},
		{APIKey: DeleteRecordsKey, MinVersion: 0, MaxVersion: 1},
	}
}
