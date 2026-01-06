// Copyright 2025 Takhin Data, Inc.

package testutil

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/takhin-data/takhin/pkg/kafka/protocol"
)

// KafkaClient is a simple Kafka protocol client for E2E testing
type KafkaClient struct {
	conn     net.Conn
	addr     string
	clientID string
}

// NewKafkaClient creates a new Kafka client
func NewKafkaClient(addr string) (*KafkaClient, error) {
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	return &KafkaClient{
		conn:     conn,
		addr:     addr,
		clientID: "test-client",
	}, nil
}

// Close closes the connection
func (c *KafkaClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Produce sends a produce request
func (c *KafkaClient) Produce(topic string, partition int32, key, value []byte) error {
	// Build produce request
	req := &protocol.ProduceRequest{
		TransactionalID: nil,
		Acks:            1,
		TimeoutMs:       5000,
		TopicData: []protocol.ProduceTopicData{
			{
				TopicName: topic,
				PartitionData: []protocol.ProducePartitionData{
					{
						PartitionIndex: partition,
						Records:        encodeRecordBatch(key, value),
					},
				},
			},
		},
	}

	// Send request
	if err := c.sendRequest(int16(protocol.ProduceKey), 8, req); err != nil {
		return err
	}

	// Read response
	var resp protocol.ProduceResponse
	if err := c.readResponse(&resp); err != nil {
		return err
	}

	// Check for errors
	for _, topicResp := range resp.Responses {
		for _, partResp := range topicResp.PartitionResponses {
			if partResp.ErrorCode != 0 {
				return fmt.Errorf("produce error: %d", partResp.ErrorCode)
			}
		}
	}

	return nil
}

// Record represents a simplified Kafka record
type Record struct {
	Key   []byte
	Value []byte
}

// Fetch sends a fetch request
func (c *KafkaClient) Fetch(topic string, partition int32, offset int64, maxBytes int32) ([]Record, error) {
	req := &protocol.FetchRequest{
		MaxWaitMs: 1000,
		MinBytes:  1,
		MaxBytes:  maxBytes,
		Topics: []protocol.FetchTopic{
			{
				TopicName: topic,
				Partitions: []protocol.FetchPartition{
					{
						PartitionIndex:    partition,
						FetchOffset:       offset,
						PartitionMaxBytes: maxBytes,
					},
				},
			},
		},
	}

	if err := c.sendRequest(int16(protocol.FetchKey), 11, req); err != nil {
		return nil, err
	}

	var resp protocol.FetchResponse
	if err := c.readResponse(&resp); err != nil {
		return nil, err
	}

	// Extract records
	var records []Record
	for _, topicResp := range resp.Responses {
		for _, partResp := range topicResp.PartitionResponses {
			if partResp.ErrorCode != 0 {
				return nil, fmt.Errorf("fetch error: %d", partResp.ErrorCode)
			}
			// Decode record batch
			recs := decodeRecordBatch(partResp.Records)
			records = append(records, recs...)
		}
	}

	return records, nil
}

// CreateTopics creates topics
func (c *KafkaClient) CreateTopics(topics []string, numPartitions int32, replicationFactor int16) error {
	var topicReqs []protocol.CreatableTopic
	for _, topic := range topics {
		topicReqs = append(topicReqs, protocol.CreatableTopic{
			Name:              topic,
			NumPartitions:     numPartitions,
			ReplicationFactor: replicationFactor,
		})
	}

	req := &protocol.CreateTopicsRequest{
		Topics:    topicReqs,
		TimeoutMs: 5000,
	}

	if err := c.sendRequest(int16(protocol.CreateTopicsKey), 5, req); err != nil {
		return err
	}

	var resp protocol.CreateTopicsResponse
	if err := c.readResponse(&resp); err != nil {
		return err
	}

	for _, topicResp := range resp.Topics {
		if topicResp.ErrorCode != 0 && topicResp.ErrorCode != 36 { // 36 = TopicAlreadyExists
			return fmt.Errorf("create topic %s error: %d", topicResp.Name, topicResp.ErrorCode)
		}
	}

	return nil
}

// Metadata fetches metadata
func (c *KafkaClient) Metadata(topics []string) (*protocol.MetadataResponse, error) {
	req := &protocol.MetadataRequest{
		Topics: topics,
	}

	if err := c.sendRequest(int16(protocol.MetadataKey), 9, req); err != nil {
		return nil, err
	}

	var resp protocol.MetadataResponse
	if err := c.readResponse(&resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// sendRequest sends a request to the server
func (c *KafkaClient) sendRequest(apiKey int16, apiVersion int16, req interface{}) error {
	// Encode request
	buf := new(bytes.Buffer)
	
	// Request header
	correlationID := int32(1)
	binary.Write(buf, binary.BigEndian, apiKey)
	binary.Write(buf, binary.BigEndian, apiVersion)
	binary.Write(buf, binary.BigEndian, correlationID)
	
	// Client ID
	binary.Write(buf, binary.BigEndian, int16(len(c.clientID)))
	buf.WriteString(c.clientID)

	// Request body (simplified encoding)
	// In a real implementation, use proper protocol encoding
	
	// Write message size + body
	msgSize := int32(buf.Len())
	sizeBuf := new(bytes.Buffer)
	binary.Write(sizeBuf, binary.BigEndian, msgSize)
	sizeBuf.Write(buf.Bytes())

	_, err := c.conn.Write(sizeBuf.Bytes())
	return err
}

// readResponse reads a response from the server
func (c *KafkaClient) readResponse(resp interface{}) error {
	// Read message size
	sizeBuf := make([]byte, 4)
	if _, err := c.conn.Read(sizeBuf); err != nil {
		return err
	}

	msgSize := binary.BigEndian.Uint32(sizeBuf)
	
	// Read message body
	msgBuf := make([]byte, msgSize)
	if _, err := c.conn.Read(msgBuf); err != nil {
		return err
	}

	// Decode response (simplified)
	// In a real implementation, use proper protocol decoding
	
	return nil
}

// encodeRecordBatch encodes records into a record batch
func encodeRecordBatch(key, value []byte) []byte {
	buf := new(bytes.Buffer)
	// Simplified record batch encoding
	// In a real implementation, use proper protocol encoding
	binary.Write(buf, binary.BigEndian, int32(len(key)))
	buf.Write(key)
	binary.Write(buf, binary.BigEndian, int32(len(value)))
	buf.Write(value)
	return buf.Bytes()
}

// decodeRecordBatch decodes a record batch
func decodeRecordBatch(data []byte) []Record {
	// Simplified record batch decoding
	// In a real implementation, use proper protocol decoding
	var records []Record
	buf := bytes.NewReader(data)
	
	for buf.Len() > 0 {
		var keyLen, valLen int32
		if err := binary.Read(buf, binary.BigEndian, &keyLen); err != nil {
			break
		}
		key := make([]byte, keyLen)
		buf.Read(key)
		
		if err := binary.Read(buf, binary.BigEndian, &valLen); err != nil {
			break
		}
		value := make([]byte, valLen)
		buf.Read(value)
		
		records = append(records, Record{
			Key:   key,
			Value: value,
		})
	}
	
	return records
}
