// Copyright 2025 Takhin Data, Inc.

package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
)

// APIKey represents a Kafka API key
type APIKey int16

// Kafka API Keys
const (
	ProduceKey            APIKey = 0
	FetchKey              APIKey = 1
	ListOffsetsKey        APIKey = 2
	MetadataKey           APIKey = 3
	LeaderAndIsrKey       APIKey = 4
	StopReplicaKey        APIKey = 5
	UpdateMetadataKey     APIKey = 6
	ControlledShutdownKey APIKey = 7
	OffsetCommitKey       APIKey = 8
	OffsetFetchKey        APIKey = 9
	FindCoordinatorKey    APIKey = 10
	JoinGroupKey          APIKey = 11
	HeartbeatKey          APIKey = 12
	LeaveGroupKey         APIKey = 13
	SyncGroupKey          APIKey = 14
	DescribeGroupsKey     APIKey = 15
	ListGroupsKey         APIKey = 16
	SaslHandshakeKey      APIKey = 17
	ApiVersionsKey        APIKey = 18
	CreateTopicsKey       APIKey = 19
	DeleteTopicsKey       APIKey = 20
	DeleteRecordsKey      APIKey = 21
	InitProducerIDKey     APIKey = 22
	AddPartitionsToTxnKey APIKey = 24
	AddOffsetsToTxnKey    APIKey = 25
	EndTxnKey             APIKey = 26
	WriteTxnMarkersKey    APIKey = 27
	TxnOffsetCommitKey    APIKey = 28
	DescribeConfigsKey    APIKey = 32
	AlterConfigsKey       APIKey = 33
	DescribeLogDirsKey    APIKey = 35
	SaslAuthenticateKey   APIKey = 36
)

// ResourceType represents Kafka resource types
type ResourceType int8

const (
	ResourceTypeBroker ResourceType = 4
	ResourceTypeTopic  ResourceType = 2
)

// ErrorCode represents a Kafka error code
type ErrorCode int16

// Kafka Error Codes
const (
	None                               ErrorCode = 0
	OffsetOutOfRange                   ErrorCode = 1
	CorruptMessage                     ErrorCode = 2
	UnknownTopicOrPartition            ErrorCode = 3
	InvalidFetchSize                   ErrorCode = 4
	LeaderNotAvailable                 ErrorCode = 5
	NotLeaderForPartition              ErrorCode = 6
	RequestTimedOut                    ErrorCode = 7
	BrokerNotAvailable                 ErrorCode = 8
	ReplicaNotAvailable                ErrorCode = 9
	MessageTooLarge                    ErrorCode = 10
	StaleControllerEpoch               ErrorCode = 11
	OffsetMetadataTooLarge             ErrorCode = 12
	NetworkException                   ErrorCode = 13
	CoordinatorLoadInProgress          ErrorCode = 14
	CoordinatorNotAvailable            ErrorCode = 15
	NotCoordinator                     ErrorCode = 16
	InvalidTopicException              ErrorCode = 17
	RecordListTooLarge                 ErrorCode = 18
	NotEnoughReplicas                  ErrorCode = 19
	NotEnoughReplicasAfterAppend       ErrorCode = 20
	InvalidRequiredAcks                ErrorCode = 21
	IllegalGeneration                  ErrorCode = 22
	InconsistentGroupProtocol          ErrorCode = 23
	InvalidGroupID                     ErrorCode = 24
	UnknownMemberID                    ErrorCode = 25
	InvalidSessionTimeout              ErrorCode = 26
	RebalanceInProgress                ErrorCode = 27
	InvalidCommitOffsetSize            ErrorCode = 28
	TopicAuthorizationFailed           ErrorCode = 29
	GroupAuthorizationFailed           ErrorCode = 30
	ClusterAuthorizationFailed         ErrorCode = 31
	InvalidTimestamp                   ErrorCode = 32
	UnsupportedSaslMechanism           ErrorCode = 33
	IllegalSaslState                   ErrorCode = 34
	UnsupportedVersion                 ErrorCode = 35
	TopicAlreadyExists                 ErrorCode = 36
	InvalidPartitions                  ErrorCode = 37
	InvalidReplicationFactor           ErrorCode = 38
	InvalidReplicaAssignment           ErrorCode = 39
	InvalidConfig                      ErrorCode = 40
	NotController                      ErrorCode = 41
	InvalidRequest                     ErrorCode = 42
	UnsupportedForMessageFormat        ErrorCode = 43
	PolicyViolation                    ErrorCode = 44
	OutOfOrderSequenceNumber           ErrorCode = 45
	DuplicateSequenceNumber            ErrorCode = 46
	InvalidProducerEpoch               ErrorCode = 47
	InvalidTxnState                    ErrorCode = 48
	InvalidProducerIDMapping           ErrorCode = 49
	InvalidTransactionTimeout          ErrorCode = 50
	ConcurrentTransactions             ErrorCode = 51
	TransactionCoordinatorFenced       ErrorCode = 52
	TransactionalIDAuthorizationFailed ErrorCode = 53
	SecurityDisabled                   ErrorCode = 54
	OperationNotAttempted              ErrorCode = 55
	KafkaStorageError                  ErrorCode = 56
	LogDirNotFound                     ErrorCode = 57
	SaslAuthenticationFailed           ErrorCode = 58
	UnknownProducerID                  ErrorCode = 59
	ReassignmentInProgress             ErrorCode = 60
	DelegationTokenAuthDisabled        ErrorCode = 61
	DelegationTokenNotFound            ErrorCode = 62
	DelegationTokenOwnerMismatch       ErrorCode = 63
	DelegationTokenRequestNotAllowed   ErrorCode = 64
	DelegationTokenAuthorizationFailed ErrorCode = 65
	DelegationTokenExpired             ErrorCode = 66
	InvalidPrincipalType               ErrorCode = 67
	NonEmptyGroup                      ErrorCode = 68
	GroupIDNotFound                    ErrorCode = 69
	FetchSessionIDNotFound             ErrorCode = 70
	InvalidFetchSessionEpoch           ErrorCode = 71
	ListenerNotFound                   ErrorCode = 72
	TopicDeletionDisabled              ErrorCode = 73
	FencedLeaderEpoch                  ErrorCode = 74
	UnknownLeaderEpoch                 ErrorCode = 75
	UnsupportedCompressionType         ErrorCode = 76
)

// RequestHeader represents the request header
type RequestHeader struct {
	APIKey        APIKey
	APIVersion    int16
	CorrelationID int32
	ClientID      string
}

// ResponseHeader represents the response header
type ResponseHeader struct {
	CorrelationID int32
}

// Encoder interface for encoding messages
type Encoder interface {
	Encode(w io.Writer) error
}

// Decoder interface for decoding messages
type Decoder interface {
	Decode(r io.Reader) error
}

// WriteInt8 writes an int8 to the writer
func WriteInt8(w io.Writer, v int8) error {
	return binary.Write(w, binary.BigEndian, v)
}

// WriteInt16 writes an int16 to the writer
func WriteInt16(w io.Writer, v int16) error {
	return binary.Write(w, binary.BigEndian, v)
}

// WriteInt32 writes an int32 to the writer
func WriteInt32(w io.Writer, v int32) error {
	return binary.Write(w, binary.BigEndian, v)
}

// WriteInt64 writes an int64 to the writer
func WriteInt64(w io.Writer, v int64) error {
	return binary.Write(w, binary.BigEndian, v)
}

// WriteString writes a string to the writer
func WriteString(w io.Writer, s string) error {
	if err := WriteInt16(w, int16(len(s))); err != nil {
		return err
	}
	_, err := w.Write([]byte(s))
	return err
}

// WriteNullableString writes a nullable string to the writer
func WriteNullableString(w io.Writer, s *string) error {
	if s == nil {
		return WriteInt16(w, -1)
	}
	return WriteString(w, *s)
}

// WriteBytes writes bytes to the writer
func WriteBytes(w io.Writer, b []byte) error {
	if err := WriteInt32(w, int32(len(b))); err != nil {
		return err
	}
	_, err := w.Write(b)
	return err
}

// WriteArray writes an array length to the writer
func WriteArray(w io.Writer, length int) error {
	return WriteInt32(w, int32(length))
}

// WriteBool writes a boolean to the writer
func WriteBool(w io.Writer, v bool) error {
	var b byte
	if v {
		b = 1
	}
	return WriteInt8(w, int8(b))
}

// ReadInt8 reads an int8 from the reader
func ReadInt8(r io.Reader) (int8, error) {
	var v int8
	err := binary.Read(r, binary.BigEndian, &v)
	return v, err
}

// ReadInt16 reads an int16 from the reader
func ReadInt16(r io.Reader) (int16, error) {
	var v int16
	err := binary.Read(r, binary.BigEndian, &v)
	return v, err
}

// ReadInt32 reads an int32 from the reader
func ReadInt32(r io.Reader) (int32, error) {
	var v int32
	err := binary.Read(r, binary.BigEndian, &v)
	return v, err
}

// ReadInt64 reads an int64 from the reader
func ReadInt64(r io.Reader) (int64, error) {
	var v int64
	err := binary.Read(r, binary.BigEndian, &v)
	return v, err
}

// ReadString reads a string from the reader
func ReadString(r io.Reader) (string, error) {
	length, err := ReadInt16(r)
	if err != nil {
		return "", err
	}
	if length < 0 {
		return "", fmt.Errorf("invalid string length: %d", length)
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

// ReadNullableString reads a nullable string from the reader
func ReadNullableString(r io.Reader) (*string, error) {
	length, err := ReadInt16(r)
	if err != nil {
		return nil, err
	}
	if length < 0 {
		return nil, nil
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	s := string(buf)
	return &s, nil
}

// ReadBytes reads bytes from the reader
func ReadBytes(r io.Reader) ([]byte, error) {
	length, err := ReadInt32(r)
	if err != nil {
		return nil, err
	}
	if length < 0 {
		return nil, fmt.Errorf("invalid bytes length: %d", length)
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

// ReadArrayLength reads an array length from the reader
func ReadArrayLength(r io.Reader) (int32, error) {
	return ReadInt32(r)
}

// ReadBool reads a boolean from the reader
func ReadBool(r io.Reader) (bool, error) {
	b, err := ReadInt8(r)
	if err != nil {
		return false, err
	}
	return b != 0, nil
}

// DecodeRequestHeader decodes a request header
func DecodeRequestHeader(r io.Reader) (*RequestHeader, error) {
	apiKey, err := ReadInt16(r)
	if err != nil {
		return nil, fmt.Errorf("read api key: %w", err)
	}

	apiVersion, err := ReadInt16(r)
	if err != nil {
		return nil, fmt.Errorf("read api version: %w", err)
	}

	correlationID, err := ReadInt32(r)
	if err != nil {
		return nil, fmt.Errorf("read correlation id: %w", err)
	}

	clientID, err := ReadString(r)
	if err != nil {
		return nil, fmt.Errorf("read client id: %w", err)
	}

	return &RequestHeader{
		APIKey:        APIKey(apiKey),
		APIVersion:    apiVersion,
		CorrelationID: correlationID,
		ClientID:      clientID,
	}, nil
}

// Encode encodes the request header
func (h *RequestHeader) Encode(w io.Writer) error {
	if err := WriteInt16(w, int16(h.APIKey)); err != nil {
		return err
	}
	if err := WriteInt16(w, h.APIVersion); err != nil {
		return err
	}
	if err := WriteInt32(w, h.CorrelationID); err != nil {
		return err
	}
	return WriteString(w, h.ClientID)
}

// Encode encodes the response header
func (h *ResponseHeader) Encode(w io.Writer) error {
	return WriteInt32(w, h.CorrelationID)
}

// Byte-level encoding/decoding helpers for new protocols

// encodeInt16 encodes an int16 to bytes
func encodeInt16(v int16) []byte {
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, uint16(v))
	return buf
}

// encodeInt32 encodes an int32 to bytes
func encodeInt32(v int32) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(v))
	return buf
}

// encodeInt64 encodes an int64 to bytes
func encodeInt64(v int64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(v))
	return buf
}

// encodeString encodes a string to bytes
func encodeString(s string) []byte {
	buf := encodeInt16(int16(len(s)))
	buf = append(buf, []byte(s)...)
	return buf
}

// encodeNullableString encodes a nullable string to bytes
func encodeNullableString(s string) []byte {
	if s == "" {
		return encodeInt16(-1)
	}
	return encodeString(s)
}

// encodeBytes encodes bytes to bytes with length prefix
func encodeBytes(data []byte) []byte {
	if data == nil {
		return encodeInt32(-1)
	}
	buf := encodeInt32(int32(len(data)))
	buf = append(buf, data...)
	return buf
}

// decodeInt16 decodes an int16 from bytes
func decodeInt16(data []byte) int16 {
	return int16(binary.BigEndian.Uint16(data))
}

// decodeInt32 decodes an int32 from bytes
func decodeInt32(data []byte) int32 {
	return int32(binary.BigEndian.Uint32(data))
}

// decodeInt64 decodes an int64 from bytes
func decodeInt64(data []byte) int64 {
	return int64(binary.BigEndian.Uint64(data))
}

// decodeString decodes a string from bytes
func decodeString(data []byte) (string, int) {
	length := decodeInt16(data)
	if length < 0 {
		return "", 2
	}
	return string(data[2 : 2+length]), 2 + int(length)
}

// decodeNullableString decodes a nullable string from bytes
func decodeNullableString(data []byte) (string, int) {
	length := decodeInt16(data)
	if length < 0 {
		return "", 2
	}
	return string(data[2 : 2+length]), 2 + int(length)
}

// decodeBytes decodes bytes from bytes
func decodeBytes(data []byte) ([]byte, int) {
	length := decodeInt32(data)
	if length < 0 {
		return nil, 4
	}
	return data[4 : 4+length], 4 + int(length)
}
