// Copyright 2025 Takhin Data, Inc.

package console

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/linkedin/goavro/v2"
	"github.com/takhin-data/takhin/pkg/compression"
)

// Producer request/response types

// ProduceRequest represents a REST API produce request
type ProduceRequest struct {
	Records []ProducerRecord `json:"records"`
}

// ProducerRecord represents a single message record
type ProducerRecord struct {
	Key       interface{} `json:"key,omitempty"`
	Value     interface{} `json:"value"`
	Partition *int32      `json:"partition,omitempty"`
	Headers   []Header    `json:"headers,omitempty"`
}

// Header represents a message header
type Header struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// ProduceResponse represents the produce response
type ProduceResponse struct {
	Offsets []ProducedRecordMetadata `json:"offsets"`
}

// ProducedRecordMetadata represents metadata for a produced record
type ProducedRecordMetadata struct {
	Partition int32  `json:"partition"`
	Offset    int64  `json:"offset"`
	Timestamp int64  `json:"timestamp,omitempty"`
	Error     string `json:"error,omitempty"`
}

// AsyncProduceResponse represents an async produce response
type AsyncProduceResponse struct {
	RequestID string `json:"requestId"`
	Status    string `json:"status"`
}

// ProduceStatusResponse represents status of an async produce request
type ProduceStatusResponse struct {
	RequestID string                     `json:"requestId"`
	Status    string                     `json:"status"`
	Offsets   []ProducedRecordMetadata   `json:"offsets,omitempty"`
	Error     string                     `json:"error,omitempty"`
}

// DataFormat represents the serialization format
type DataFormat string

const (
	FormatJSON   DataFormat = "json"
	FormatAvro   DataFormat = "avro"
	FormatBinary DataFormat = "binary"
	FormatString DataFormat = "string"
)

// Async request tracking
type asyncProduceRequest struct {
	id        string
	status    string
	offsets   []ProducedRecordMetadata
	error     error
	createdAt time.Time
}

var (
	asyncRequests   = make(map[string]*asyncProduceRequest)
	asyncRequestsMu sync.RWMutex
)

// handleProduceBatch godoc
// @Summary      Produce messages (batch)
// @Description  Produce a batch of messages to a topic with JSON or Avro encoding
// @Tags         Producer
// @Accept       json
// @Produce      json
// @Param        topic         path      string          true   "Topic name"
// @Param        request       body      ProduceRequest  true   "Produce request"
// @Param        key.format    query     string          false  "Key serialization format (json, avro, binary, string)" default(json)
// @Param        value.format  query     string          false  "Value serialization format (json, avro, binary, string)" default(json)
// @Param        key.schema    query     string          false  "Avro schema subject for key (required if key.format=avro)"
// @Param        value.schema  query     string          false  "Avro schema subject for value (required if value.format=avro)"
// @Param        compression   query     string          false  "Compression codec (none, gzip, snappy, lz4, zstd)" default(none)
// @Param        async         query     boolean         false  "Enable async produce (returns immediately with request ID)" default(false)
// @Success      200           {object}  ProduceResponse
// @Success      202           {object}  AsyncProduceResponse
// @Failure      400           {object}  map[string]string
// @Failure      404           {object}  map[string]string
// @Failure      500           {object}  map[string]string
// @Security     ApiKeyAuth
// @Router       /topics/{topic}/produce [post]
func (s *Server) handleProduceBatch(w http.ResponseWriter, r *http.Request) {
	topicName := chi.URLParam(r, "topic")

	// Check if topic exists
	topic, exists := s.topicManager.GetTopic(topicName)
	if !exists {
		s.respondError(w, http.StatusNotFound, "topic not found")
		return
	}

	// Parse request
	var req ProduceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Records) == 0 {
		s.respondError(w, http.StatusBadRequest, "no records to produce")
		return
	}

	// Parse query parameters
	keyFormat := DataFormat(r.URL.Query().Get("key.format"))
	if keyFormat == "" {
		keyFormat = FormatJSON
	}
	valueFormat := DataFormat(r.URL.Query().Get("value.format"))
	if valueFormat == "" {
		valueFormat = FormatJSON
	}
	keySchema := r.URL.Query().Get("key.schema")
	valueSchema := r.URL.Query().Get("value.schema")
	compressionType := r.URL.Query().Get("compression")
	if compressionType == "" {
		compressionType = "none"
	}
	async := r.URL.Query().Get("async") == "true"

	// Validate schema requirements
	if keyFormat == FormatAvro && keySchema == "" {
		s.respondError(w, http.StatusBadRequest, "key.schema required when key.format=avro")
		return
	}
	if valueFormat == FormatAvro && valueSchema == "" {
		s.respondError(w, http.StatusBadRequest, "value.schema required when value.format=avro")
		return
	}

	// Create producer context
	ctx := &producerContext{
		server:          s,
		topic:           topic,
		topicName:       topicName,
		keyFormat:       keyFormat,
		valueFormat:     valueFormat,
		keySchema:       keySchema,
		valueSchema:     valueSchema,
		compressionType: compressionType,
	}

	// Handle async vs sync
	if async {
		requestID := generateRequestID()
		
		asyncRequestsMu.Lock()
		asyncRequests[requestID] = &asyncProduceRequest{
			id:        requestID,
			status:    "pending",
			createdAt: time.Now(),
		}
		asyncRequestsMu.Unlock()

		// Process asynchronously
		go func() {
			offsets, err := ctx.produceRecords(req.Records)
			
			asyncRequestsMu.Lock()
			if asyncReq, exists := asyncRequests[requestID]; exists {
				if err != nil {
					asyncReq.status = "failed"
					asyncReq.error = err
				} else {
					asyncReq.status = "completed"
					asyncReq.offsets = offsets
				}
			}
			asyncRequestsMu.Unlock()
		}()

		s.respondJSON(w, http.StatusAccepted, AsyncProduceResponse{
			RequestID: requestID,
			Status:    "pending",
		})
		return
	}

	// Synchronous produce
	offsets, err := ctx.produceRecords(req.Records)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, ProduceResponse{
		Offsets: offsets,
	})
}

// handleProduceStatus godoc
// @Summary      Get async produce status
// @Description  Get the status of an async produce request
// @Tags         Producer
// @Produce      json
// @Param        requestId  path      string  true  "Request ID"
// @Success      200        {object}  ProduceStatusResponse
// @Failure      404        {object}  map[string]string
// @Security     ApiKeyAuth
// @Router       /produce/status/{requestId} [get]
func (s *Server) handleProduceStatus(w http.ResponseWriter, r *http.Request) {
	requestID := chi.URLParam(r, "requestId")

	asyncRequestsMu.RLock()
	asyncReq, exists := asyncRequests[requestID]
	asyncRequestsMu.RUnlock()

	if !exists {
		s.respondError(w, http.StatusNotFound, "request not found")
		return
	}

	response := ProduceStatusResponse{
		RequestID: asyncReq.id,
		Status:    asyncReq.status,
		Offsets:   asyncReq.offsets,
	}

	if asyncReq.error != nil {
		response.Error = asyncReq.error.Error()
	}

	s.respondJSON(w, http.StatusOK, response)
}

// Producer context for batch operations
type producerContext struct {
	server          *Server
	topic           interface{}
	topicName       string
	keyFormat       DataFormat
	valueFormat     DataFormat
	keySchema       string
	valueSchema     string
	compressionType string
	keyCodec        *goavro.Codec
	valueCodec      *goavro.Codec
}

// produceRecords produces a batch of records
func (ctx *producerContext) produceRecords(records []ProducerRecord) ([]ProducedRecordMetadata, error) {
	// Initialize Avro codecs if needed
	if err := ctx.initializeCodecs(); err != nil {
		return nil, err
	}

	offsets := make([]ProducedRecordMetadata, 0, len(records))
	
	for i, record := range records {
		metadata, err := ctx.produceRecord(record)
		if err != nil {
			metadata = ProducedRecordMetadata{
				Error: fmt.Sprintf("record %d: %v", i, err),
			}
		}
		offsets = append(offsets, metadata)
	}

	return offsets, nil
}

// produceRecord produces a single record
func (ctx *producerContext) produceRecord(record ProducerRecord) (ProducedRecordMetadata, error) {
	// Serialize key
	keyBytes, err := ctx.serializeData(record.Key, ctx.keyFormat, ctx.keyCodec)
	if err != nil {
		return ProducedRecordMetadata{}, fmt.Errorf("serialize key: %w", err)
	}

	// Serialize value
	valueBytes, err := ctx.serializeData(record.Value, ctx.valueFormat, ctx.valueCodec)
	if err != nil {
		return ProducedRecordMetadata{}, fmt.Errorf("serialize value: %w", err)
	}

	// Apply compression
	if ctx.compressionType != "none" {
		valueBytes, err = ctx.compressData(valueBytes)
		if err != nil {
			return ProducedRecordMetadata{}, fmt.Errorf("compress data: %w", err)
		}
	}

	// Determine partition
	partition := int32(0)
	if record.Partition != nil {
		partition = *record.Partition
	}

	// Use type assertion to access topic methods
	type topicAppender interface {
		Append(partition int32, key, value []byte) (int64, error)
	}

	topicWithAppend, ok := ctx.topic.(topicAppender)
	if !ok {
		return ProducedRecordMetadata{}, fmt.Errorf("topic does not support Append")
	}

	// Append to topic
	offset, err := topicWithAppend.Append(partition, keyBytes, valueBytes)
	if err != nil {
		return ProducedRecordMetadata{}, fmt.Errorf("append to topic: %w", err)
	}

	return ProducedRecordMetadata{
		Partition: partition,
		Offset:    offset,
		Timestamp: time.Now().UnixMilli(),
	}, nil
}

// initializeCodecs initializes Avro codecs
func (ctx *producerContext) initializeCodecs() error {
	if ctx.keyFormat == FormatAvro && ctx.keyCodec == nil {
		codec, err := ctx.loadAvroCodec(ctx.keySchema)
		if err != nil {
			return fmt.Errorf("load key schema: %w", err)
		}
		ctx.keyCodec = codec
	}

	if ctx.valueFormat == FormatAvro && ctx.valueCodec == nil {
		codec, err := ctx.loadAvroCodec(ctx.valueSchema)
		if err != nil {
			return fmt.Errorf("load value schema: %w", err)
		}
		ctx.valueCodec = codec
	}

	return nil
}

// loadAvroCodec loads an Avro codec from schema registry
func (ctx *producerContext) loadAvroCodec(subject string) (*goavro.Codec, error) {
	// For now, return an error - full integration requires schema registry client
	// This will be implemented when schema registry is available
	return nil, fmt.Errorf("schema registry integration not yet implemented")
}

// serializeData serializes data based on format
func (ctx *producerContext) serializeData(data interface{}, format DataFormat, codec *goavro.Codec) ([]byte, error) {
	if data == nil {
		return nil, nil
	}

	switch format {
	case FormatJSON:
		return json.Marshal(data)
	
	case FormatAvro:
		if codec == nil {
			return nil, fmt.Errorf("avro codec not initialized")
		}
		return codec.BinaryFromNative(nil, data)
	
	case FormatBinary:
		// Expect base64-encoded string
		str, ok := data.(string)
		if !ok {
			return nil, fmt.Errorf("binary format expects base64-encoded string")
		}
		return base64.StdEncoding.DecodeString(str)
	
	case FormatString:
		// Convert to string
		str := fmt.Sprintf("%v", data)
		return []byte(str), nil
	
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// compressData compresses data using specified codec
func (ctx *producerContext) compressData(data []byte) ([]byte, error) {
	var compressionType compression.Type
	
	switch ctx.compressionType {
	case "gzip":
		compressionType = compression.GZIP
	case "snappy":
		compressionType = compression.Snappy
	case "lz4":
		compressionType = compression.LZ4
	case "zstd":
		compressionType = compression.ZSTD
	case "none":
		compressionType = compression.None
	default:
		return nil, fmt.Errorf("unsupported compression: %s", ctx.compressionType)
	}
	
	return compression.Compress(compressionType, data)
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}

// Cleanup old async requests (should be called periodically)
func cleanupAsyncRequests(maxAge time.Duration) {
	asyncRequestsMu.Lock()
	defer asyncRequestsMu.Unlock()

	now := time.Now()
	for id, req := range asyncRequests {
		if now.Sub(req.createdAt) > maxAge {
			delete(asyncRequests, id)
		}
	}
}

// StartAsyncCleanup starts the async request cleanup goroutine
func StartAsyncCleanup(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cleanupAsyncRequests(30 * time.Minute)
		case <-ctx.Done():
			return
		}
	}
}
