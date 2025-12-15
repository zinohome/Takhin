// Copyright 2025 Takhin Data, Inc.

package protocol

import (
	"io"
)

type MetadataRequest struct {
	Header *RequestHeader
	Topics []string
}

type MetadataResponse struct {
	Brokers       []Broker
	ClusterID     *string
	ControllerID  int32
	TopicMetadata []TopicMetadata
}

type Broker struct {
	NodeID int32
	Host   string
	Port   int32
	Rack   *string
}

type TopicMetadata struct {
	ErrorCode         ErrorCode
	TopicName         string
	IsInternal        bool
	PartitionMetadata []PartitionMetadata
}

type PartitionMetadata struct {
	ErrorCode       ErrorCode
	PartitionID     int32
	Leader          int32
	Replicas        []int32
	ISR             []int32
	OfflineReplicas []int32
}

func DecodeMetadataRequest(r io.Reader, header *RequestHeader) (*MetadataRequest, error) {
	req := &MetadataRequest{
		Header: header,
	}

	topicsLen, err := ReadArrayLength(r)
	if err != nil {
		return nil, err
	}

	if topicsLen == -1 {
		req.Topics = nil
		return req, nil
	}

	req.Topics = make([]string, topicsLen)
	for i := int32(0); i < topicsLen; i++ {
		topic, err := ReadString(r)
		if err != nil {
			return nil, err
		}
		req.Topics[i] = topic
	}

	return req, nil
}

func (r *MetadataResponse) Encode(w io.Writer) error {
	if err := WriteArray(w, len(r.Brokers)); err != nil {
		return err
	}
	for _, broker := range r.Brokers {
		if err := WriteInt32(w, broker.NodeID); err != nil {
			return err
		}
		if err := WriteString(w, broker.Host); err != nil {
			return err
		}
		if err := WriteInt32(w, broker.Port); err != nil {
			return err
		}
		if err := WriteNullableString(w, broker.Rack); err != nil {
			return err
		}
	}

	if err := WriteNullableString(w, r.ClusterID); err != nil {
		return err
	}

	if err := WriteInt32(w, r.ControllerID); err != nil {
		return err
	}

	if err := WriteArray(w, len(r.TopicMetadata)); err != nil {
		return err
	}
	for _, topic := range r.TopicMetadata {
		if err := WriteInt16(w, int16(topic.ErrorCode)); err != nil {
			return err
		}
		if err := WriteString(w, topic.TopicName); err != nil {
			return err
		}
		if err := WriteInt8(w, boolToInt8(topic.IsInternal)); err != nil {
			return err
		}

		if err := WriteArray(w, len(topic.PartitionMetadata)); err != nil {
			return err
		}
		for _, partition := range topic.PartitionMetadata {
			if err := WriteInt16(w, int16(partition.ErrorCode)); err != nil {
				return err
			}
			if err := WriteInt32(w, partition.PartitionID); err != nil {
				return err
			}
			if err := WriteInt32(w, partition.Leader); err != nil {
				return err
			}

			if err := WriteArray(w, len(partition.Replicas)); err != nil {
				return err
			}
			for _, replica := range partition.Replicas {
				if err := WriteInt32(w, replica); err != nil {
					return err
				}
			}

			if err := WriteArray(w, len(partition.ISR)); err != nil {
				return err
			}
			for _, isr := range partition.ISR {
				if err := WriteInt32(w, isr); err != nil {
					return err
				}
			}

			if err := WriteArray(w, len(partition.OfflineReplicas)); err != nil {
				return err
			}
			for _, replica := range partition.OfflineReplicas {
				if err := WriteInt32(w, replica); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func boolToInt8(b bool) int8 {
	if b {
		return 1
	}
	return 0
}
