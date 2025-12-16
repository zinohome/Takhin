// Copyright 2025 Takhin Data, Inc.

package protocol

import (
"io"
)

// DeleteTopicsRequest represents a DeleteTopics request (API Key 20)
type DeleteTopicsRequest struct {
	Header     *RequestHeader
	TopicNames []string
	TimeoutMs  int32
}

// DeleteTopicsResponse represents a DeleteTopics response
type DeleteTopicsResponse struct {
	ThrottleTimeMs int32
	Responses      []DeletableTopicResult
}

// DeletableTopicResult represents the result of deleting a topic
type DeletableTopicResult struct {
	Name         string
	ErrorCode    ErrorCode
	ErrorMessage *string
}

// DecodeDeleteTopicsRequest decodes a DeleteTopics request
func DecodeDeleteTopicsRequest(r io.Reader, header *RequestHeader) (*DeleteTopicsRequest, error) {
	req := &DeleteTopicsRequest{
		Header: header,
	}

	// Read topic names array
	topicCount, err := ReadArrayLength(r)
	if err != nil {
		return nil, err
	}

	req.TopicNames = make([]string, topicCount)
	for i := int32(0); i < topicCount; i++ {
		topicName, err := ReadString(r)
		if err != nil {
			return nil, err
		}
		req.TopicNames[i] = topicName
	}

	// Read timeout
	timeoutMs, err := ReadInt32(r)
	if err != nil {
		return nil, err
	}
	req.TimeoutMs = timeoutMs

	return req, nil
}

// Encode encodes the DeleteTopics response
func (r *DeleteTopicsResponse) Encode(w io.Writer) error {
	// Write throttle time
	if err := WriteInt32(w, r.ThrottleTimeMs); err != nil {
		return err
	}

	// Write responses array
	if err := WriteArray(w, len(r.Responses)); err != nil {
		return err
	}

	for _, response := range r.Responses {
		// Write topic name
		if err := WriteString(w, response.Name); err != nil {
			return err
		}

		// Write error code
		if err := WriteInt16(w, int16(response.ErrorCode)); err != nil {
			return err
		}

		// Write error message (nullable)
		if err := WriteNullableString(w, response.ErrorMessage); err != nil {
			return err
		}
	}

	return nil
}
