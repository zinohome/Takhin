// Copyright 2025 Takhin Data, Inc.

package handler

import (
	"bytes"
	"fmt"
	"io"
	"net"

	"github.com/takhin-data/takhin/pkg/kafka/protocol"
	"github.com/takhin-data/takhin/pkg/storage/log"
	"github.com/takhin-data/takhin/pkg/zerocopy"
)

// ZeroCopyFetchResponse represents a Fetch response that can be sent using zero-copy I/O.
type ZeroCopyFetchResponse struct {
	HeaderBytes []byte          // Pre-encoded response header and metadata
	Segments    []FetchSegment  // File segments to send with zero-copy
}

// FetchSegment represents a segment of data to be sent with zero-copy.
type FetchSegment struct {
	Segment  *log.Segment
	Position int64
	Size     int64
}

// HandleFetchZeroCopy processes a Fetch request and writes the response directly to the connection
// using zero-copy I/O when possible.
func (h *Handler) HandleFetchZeroCopy(reqData []byte, conn net.Conn) error {
	r := bytes.NewReader(reqData)

	// Decode request header
	header, err := protocol.DecodeRequestHeader(r)
	if err != nil {
		return fmt.Errorf("decode request header: %w", err)
	}

	// Decode Fetch request
	req, err := protocol.DecodeFetchRequest(r, header)
	if err != nil {
		return fmt.Errorf("decode fetch request: %w", err)
	}

	// Check if this is a replica fetch
	isReplicaFetch := req.ReplicaID >= 0

	h.logger.Debug("fetch request (zero-copy)",
		"correlation_id", header.CorrelationID,
		"topics", len(req.Topics),
		"max_wait_ms", req.MaxWaitMs,
		"replica_id", req.ReplicaID,
		"is_replica_fetch", isReplicaFetch,
	)

	// Build response header
	var headerBuf bytes.Buffer
	respHeader := &protocol.ResponseHeader{
		CorrelationID: header.CorrelationID,
	}
	if err := respHeader.Encode(&headerBuf); err != nil {
		return fmt.Errorf("encode response header: %w", err)
	}

	// Build response metadata (without record data)
	resp := &protocol.FetchResponse{
		ThrottleTimeMs: 0,
		ErrorCode:      protocol.None,
		SessionID:      0,
		Responses:      make([]protocol.FetchTopicResponse, 0),
	}

	// Track segments for zero-copy transfer
	segments := make([]FetchSegment, 0)
	totalDataSize := int64(0)

	for _, topicReq := range req.Topics {
		topic, exists := h.backend.GetTopic(topicReq.TopicName)
		if !exists {
			continue
		}

		topicResp := protocol.FetchTopicResponse{
			TopicName:          topicReq.TopicName,
			PartitionResponses: make([]protocol.FetchPartitionResponse, 0),
		}

		for _, partReq := range topicReq.Partitions {
			hwm, _ := topic.HighWaterMark(partReq.PartitionIndex)

			partResp := protocol.FetchPartitionResponse{
				PartitionIndex:   partReq.PartitionIndex,
				ErrorCode:        protocol.None,
				HighWatermark:    hwm,
				LastStableOffset: hwm,
				LogStartOffset:   0,
				Records:          []byte{}, // Will be sent via zero-copy
			}

			// Check if we can use zero-copy for this partition
			if partReq.FetchOffset < hwm && req.MaxBytes > 0 {
				segment, position, size, err := topic.ReadRange(
					partReq.PartitionIndex,
					partReq.FetchOffset,
					int64(req.MaxBytes),
				)
				if err == nil && segment != nil && size > 0 {
					segments = append(segments, FetchSegment{
						Segment:  segment,
						Position: position,
						Size:     size,
					})
					totalDataSize += size
				}
			}

			// If this is a replica fetch, update follower LEO
			if isReplicaFetch && req.ReplicaID != int32(h.config.Kafka.BrokerID) {
				followerLEO := partReq.FetchOffset
				topic.UpdateFollowerLEO(partReq.PartitionIndex, req.ReplicaID, followerLEO)
				leaderLEO := hwm
				newISR := topic.UpdateISR(partReq.PartitionIndex, leaderLEO)

				h.logger.Debug("updated follower state (zero-copy)",
					"topic", topicReq.TopicName,
					"partition", partReq.PartitionIndex,
					"follower_id", req.ReplicaID,
					"follower_leo", followerLEO,
					"leader_leo", leaderLEO,
					"isr", newISR,
				)

				currentHWM, _ := topic.HighWaterMark(partReq.PartitionIndex)
				h.produceWaiter.NotifyHWMAdvanced(topicReq.TopicName, partReq.PartitionIndex, currentHWM)
			}

			topicResp.PartitionResponses = append(topicResp.PartitionResponses, partResp)
		}

		resp.Responses = append(resp.Responses, topicResp)
	}

	// Encode response metadata
	var metaBuf bytes.Buffer
	if err := resp.Encode(&metaBuf); err != nil {
		return fmt.Errorf("encode fetch response: %w", err)
	}

	// Calculate total size: header + metadata + data
	headerBytes := headerBuf.Bytes()
	metaBytes := metaBuf.Bytes()
	totalSize := int64(len(headerBytes) + len(metaBytes)) + totalDataSize

	// Write response size (4 bytes)
	sizeBuf := []byte{
		byte(totalSize >> 24),
		byte(totalSize >> 16),
		byte(totalSize >> 8),
		byte(totalSize),
	}
	if _, err := conn.Write(sizeBuf); err != nil {
		return fmt.Errorf("write size: %w", err)
	}

	// Write header
	if _, err := conn.Write(headerBytes); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	// Write metadata
	if _, err := conn.Write(metaBytes); err != nil {
		return fmt.Errorf("write metadata: %w", err)
	}

	// Use zero-copy to transfer segments
	tcpConn, ok := conn.(*net.TCPConn)
	totalWritten := int64(0)
	
	if ok && len(segments) > 0 {
		// Zero-copy path for TCP connections
		for _, seg := range segments {
			dataFile := seg.Segment.DataFile()
			written, err := zerocopy.SendFile(tcpConn, dataFile, seg.Position, seg.Size)
			if err != nil {
				h.logger.Warn("zero-copy transfer failed, using fallback",
					"error", err,
					"written", written,
				)
				// Error already handled with fallback in SendFile
			}
			totalWritten += written
		}
	} else if len(segments) > 0 {
		// Fallback to regular copy for non-TCP connections
		h.logger.Debug("non-TCP connection, using regular copy")
		for _, seg := range segments {
			dataFile := seg.Segment.DataFile()
			if _, err := dataFile.Seek(seg.Position, io.SeekStart); err != nil {
				return fmt.Errorf("seek segment: %w", err)
			}
			written, err := io.CopyN(conn, dataFile, seg.Size)
			if err != nil {
				return fmt.Errorf("copy segment: %w", err)
			}
			totalWritten += written
		}
	}

	h.logger.Debug("fetch response sent (zero-copy)",
		"correlation_id", header.CorrelationID,
		"total_bytes", totalSize,
		"data_bytes", totalWritten,
		"segments", len(segments),
		"zero_copy", ok && len(segments) > 0,
	)

	return nil
}
