// Copyright 2025 Takhin Data, Inc.

package handler

import (
	"bytes"
	"fmt"
	"io"

	"github.com/takhin-data/takhin/pkg/kafka/protocol"
	"github.com/takhin-data/takhin/pkg/logger"
)

// getSupportedAPIVersions returns all supported API versions
func (h *Handler) getSupportedAPIVersions() []protocol.APIVersion {
	return []protocol.APIVersion{
		{APIKey: int16(protocol.ProduceKey), MinVersion: 0, MaxVersion: 9},
		{APIKey: int16(protocol.FetchKey), MinVersion: 0, MaxVersion: 12},
		{APIKey: int16(protocol.ListOffsetsKey), MinVersion: 0, MaxVersion: 7},
		{APIKey: int16(protocol.MetadataKey), MinVersion: 0, MaxVersion: 12},
		{APIKey: int16(protocol.OffsetCommitKey), MinVersion: 0, MaxVersion: 8},
		{APIKey: int16(protocol.OffsetFetchKey), MinVersion: 0, MaxVersion: 8},
		{APIKey: int16(protocol.FindCoordinatorKey), MinVersion: 0, MaxVersion: 4},
		{APIKey: int16(protocol.JoinGroupKey), MinVersion: 0, MaxVersion: 9},
		{APIKey: int16(protocol.HeartbeatKey), MinVersion: 0, MaxVersion: 4},
		{APIKey: int16(protocol.LeaveGroupKey), MinVersion: 0, MaxVersion: 5},
		{APIKey: int16(protocol.SyncGroupKey), MinVersion: 0, MaxVersion: 5},
		{APIKey: int16(protocol.DescribeGroupsKey), MinVersion: 0, MaxVersion: 5},
		{APIKey: int16(protocol.ListGroupsKey), MinVersion: 0, MaxVersion: 4},
		{APIKey: int16(protocol.SaslHandshakeKey), MinVersion: 0, MaxVersion: 1},
		{APIKey: int16(protocol.ApiVersionsKey), MinVersion: 0, MaxVersion: 3},
		{APIKey: int16(protocol.CreateTopicsKey), MinVersion: 0, MaxVersion: 7},
		{APIKey: int16(protocol.DeleteTopicsKey), MinVersion: 0, MaxVersion: 6},
		{APIKey: int16(protocol.DeleteRecordsKey), MinVersion: 0, MaxVersion: 2},
		{APIKey: int16(protocol.InitProducerIDKey), MinVersion: 0, MaxVersion: 4},
		{APIKey: int16(protocol.AddPartitionsToTxnKey), MinVersion: 0, MaxVersion: 3},
		{APIKey: int16(protocol.AddOffsetsToTxnKey), MinVersion: 0, MaxVersion: 3},
		{APIKey: int16(protocol.EndTxnKey), MinVersion: 0, MaxVersion: 3},
		{APIKey: int16(protocol.WriteTxnMarkersKey), MinVersion: 0, MaxVersion: 1},
		{APIKey: int16(protocol.TxnOffsetCommitKey), MinVersion: 0, MaxVersion: 3},
		{APIKey: int16(protocol.DescribeConfigsKey), MinVersion: 0, MaxVersion: 4},
		{APIKey: int16(protocol.AlterConfigsKey), MinVersion: 0, MaxVersion: 2},
		{APIKey: int16(protocol.DescribeLogDirsKey), MinVersion: 0, MaxVersion: 4},
		{APIKey: int16(protocol.SaslAuthenticateKey), MinVersion: 0, MaxVersion: 2},
	}
}

// handleApiVersions handles ApiVersions requests
func (h *Handler) handleApiVersions(reader io.Reader, header *protocol.RequestHeader) ([]byte, error) {
	req, err := protocol.DecodeApiVersionsRequest(reader, header.APIVersion)
	if err != nil {
		return nil, fmt.Errorf("decode request: %w", err)
	}

	if req.ClientSoftwareName != "" {
		logger.Info("api versions request",
			"component", "kafka-handler",
			"client_software_name", req.ClientSoftwareName,
			"client_software_version", req.ClientSoftwareVersion,
		)
	} else {
		logger.Info("api versions request",
			"component", "kafka-handler",
		)
	}

	// Get supported API versions
	apiVersions := h.getSupportedAPIVersions()

	resp := &protocol.ApiVersionsResponse{
		ErrorCode:      protocol.None,
		APIVersions:    apiVersions,
		ThrottleTimeMs: 0,
	}

	logger.Info("returning api versions",
		"component", "kafka-handler",
		"num_apis", len(apiVersions),
	)

	// Encode response
	var buf bytes.Buffer
	if err := protocol.WriteApiVersionsResponse(&buf, header, resp); err != nil {
		return nil, fmt.Errorf("write response: %w", err)
	}

	return buf.Bytes(), nil
}
