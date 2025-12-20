// Copyright 2025 Takhin Data, Inc.

package handler

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/takhin-data/takhin/pkg/config"
	"github.com/takhin-data/takhin/pkg/kafka/protocol"
	"github.com/takhin-data/takhin/pkg/storage/topic"
)

func TestHandleWriteTxnMarkers_Commit(t *testing.T) {
	cfg := &config.Config{
		Server:  config.ServerConfig{Host: "localhost", Port: 9092},
		Storage: config.StorageConfig{DataDir: t.TempDir(), LogSegmentSize: 1024 * 1024},
		Kafka:   config.KafkaConfig{BrokerID: 1},
	}

	topicMgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)
	h := New(cfg, topicMgr)

	err := topicMgr.CreateTopic("test-topic", 2)
	require.NoError(t, err)

	header := &protocol.RequestHeader{
		APIKey:        protocol.WriteTxnMarkersKey,
		APIVersion:    0,
		CorrelationID: 123,
		ClientID:      "test-client",
	}

	reqBuf := encodeWriteTxnMarkersRequest([]protocol.TxnMarkerEntry{
		{
			ProducerID:        1001,
			ProducerEpoch:     0,
			TransactionResult: true,
			Topics: []protocol.TxnMarkerTopic{
				{Topic: "test-topic", Partitions: []int32{0, 1}},
			},
			CoordinatorEpoch: 0,
		},
	})

	reader := bytes.NewReader(reqBuf)
	responseBytes, err := h.handleWriteTxnMarkers(reader, header)
	require.NoError(t, err)
	require.NotNil(t, responseBytes)

	respReader := bytes.NewReader(responseBytes)
	var corrID int32
	binary.Read(respReader, binary.BigEndian, &corrID)
	assert.Equal(t, header.CorrelationID, corrID)

	var markersLen int32
	binary.Read(respReader, binary.BigEndian, &markersLen)
	assert.Equal(t, int32(1), markersLen)

	var producerID int64
	binary.Read(respReader, binary.BigEndian, &producerID)
	assert.Equal(t, int64(1001), producerID)
}

func TestHandleWriteTxnMarkers_Abort(t *testing.T) {
	cfg := &config.Config{
		Server:  config.ServerConfig{Host: "localhost", Port: 9092},
		Storage: config.StorageConfig{DataDir: t.TempDir(), LogSegmentSize: 1024 * 1024},
		Kafka:   config.KafkaConfig{BrokerID: 1},
	}

	topicMgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)
	h := New(cfg, topicMgr)

	err := topicMgr.CreateTopic("test-topic", 3)
	require.NoError(t, err)

	header := &protocol.RequestHeader{
		APIKey:        protocol.WriteTxnMarkersKey,
		APIVersion:    0,
		CorrelationID: 456,
		ClientID:      "test-client",
	}

	reqBuf := encodeWriteTxnMarkersRequest([]protocol.TxnMarkerEntry{
		{
			ProducerID:        1002,
			ProducerEpoch:     1,
			TransactionResult: false,
			Topics: []protocol.TxnMarkerTopic{
				{Topic: "test-topic", Partitions: []int32{0, 1, 2}},
			},
			CoordinatorEpoch: 0,
		},
	})

	reader := bytes.NewReader(reqBuf)
	responseBytes, err := h.handleWriteTxnMarkers(reader, header)
	require.NoError(t, err)
	require.NotNil(t, responseBytes)
}

func TestHandleWriteTxnMarkers_MultipleMarkers(t *testing.T) {
	cfg := &config.Config{
		Server:  config.ServerConfig{Host: "localhost", Port: 9092},
		Storage: config.StorageConfig{DataDir: t.TempDir(), LogSegmentSize: 1024 * 1024},
		Kafka:   config.KafkaConfig{BrokerID: 1},
	}

	topicMgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)
	h := New(cfg, topicMgr)

	require.NoError(t, topicMgr.CreateTopic("topic1", 2))
	require.NoError(t, topicMgr.CreateTopic("topic2", 2))

	header := &protocol.RequestHeader{
		APIKey:        protocol.WriteTxnMarkersKey,
		APIVersion:    0,
		CorrelationID: 789,
		ClientID:      "test-client",
	}

	reqBuf := encodeWriteTxnMarkersRequest([]protocol.TxnMarkerEntry{
		{
			ProducerID: 1001, ProducerEpoch: 0, TransactionResult: true,
			Topics:           []protocol.TxnMarkerTopic{{Topic: "topic1", Partitions: []int32{0, 1}}},
			CoordinatorEpoch: 0,
		},
		{
			ProducerID: 1002, ProducerEpoch: 0, TransactionResult: false,
			Topics:           []protocol.TxnMarkerTopic{{Topic: "topic2", Partitions: []int32{0, 1}}},
			CoordinatorEpoch: 0,
		},
	})

	reader := bytes.NewReader(reqBuf)
	responseBytes, err := h.handleWriteTxnMarkers(reader, header)
	require.NoError(t, err)
	require.NotNil(t, responseBytes)

	respReader := bytes.NewReader(responseBytes)
	var corrID int32
	binary.Read(respReader, binary.BigEndian, &corrID)

	var markersLen int32
	binary.Read(respReader, binary.BigEndian, &markersLen)
	assert.Equal(t, int32(2), markersLen)
}

func TestHandleWriteTxnMarkers_UnknownTopic(t *testing.T) {
	cfg := &config.Config{
		Server:  config.ServerConfig{Host: "localhost", Port: 9092},
		Storage: config.StorageConfig{DataDir: t.TempDir(), LogSegmentSize: 1024 * 1024},
		Kafka:   config.KafkaConfig{BrokerID: 1},
	}

	topicMgr := topic.NewManager(cfg.Storage.DataDir, cfg.Storage.LogSegmentSize)
	h := New(cfg, topicMgr)

	header := &protocol.RequestHeader{
		APIKey:        protocol.WriteTxnMarkersKey,
		APIVersion:    0,
		CorrelationID: 999,
		ClientID:      "test-client",
	}

	reqBuf := encodeWriteTxnMarkersRequest([]protocol.TxnMarkerEntry{
		{
			ProducerID: 1003, ProducerEpoch: 0, TransactionResult: true,
			Topics:           []protocol.TxnMarkerTopic{{Topic: "non-existent", Partitions: []int32{0}}},
			CoordinatorEpoch: 0,
		},
	})

	reader := bytes.NewReader(reqBuf)
	responseBytes, err := h.handleWriteTxnMarkers(reader, header)
	require.NoError(t, err)
	require.NotNil(t, responseBytes)
}

func encodeWriteTxnMarkersRequest(markers []protocol.TxnMarkerEntry) []byte {
	buf := make([]byte, 0, 512)

	markersLen := make([]byte, 4)
	binary.BigEndian.PutUint32(markersLen, uint32(len(markers)))
	buf = append(buf, markersLen...)

	for _, marker := range markers {
		producerID := make([]byte, 8)
		binary.BigEndian.PutUint64(producerID, uint64(marker.ProducerID))
		buf = append(buf, producerID...)

		producerEpoch := make([]byte, 2)
		binary.BigEndian.PutUint16(producerEpoch, uint16(marker.ProducerEpoch))
		buf = append(buf, producerEpoch...)

		if marker.TransactionResult {
			buf = append(buf, 1)
		} else {
			buf = append(buf, 0)
		}

		topicsLen := make([]byte, 4)
		binary.BigEndian.PutUint32(topicsLen, uint32(len(marker.Topics)))
		buf = append(buf, topicsLen...)

		for _, topic := range marker.Topics {
			topicNameLen := make([]byte, 2)
			binary.BigEndian.PutUint16(topicNameLen, uint16(len(topic.Topic)))
			buf = append(buf, topicNameLen...)
			buf = append(buf, []byte(topic.Topic)...)

			partitionsLen := make([]byte, 4)
			binary.BigEndian.PutUint32(partitionsLen, uint32(len(topic.Partitions)))
			buf = append(buf, partitionsLen...)

			for _, partition := range topic.Partitions {
				partitionBytes := make([]byte, 4)
				binary.BigEndian.PutUint32(partitionBytes, uint32(partition))
				buf = append(buf, partitionBytes...)
			}
		}

		coordinatorEpoch := make([]byte, 4)
		binary.BigEndian.PutUint32(coordinatorEpoch, uint32(marker.CoordinatorEpoch))
		buf = append(buf, coordinatorEpoch...)
	}

	return buf
}
