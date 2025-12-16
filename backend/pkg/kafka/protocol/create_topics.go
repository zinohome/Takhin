// Copyright 2025 Takhin Data, Inc.

package protocol

import (
	"io"
)

// CreateTopicsRequest represents a CreateTopics request (API Key 19)
type CreateTopicsRequest struct {
	Header       *RequestHeader
	Topics       []CreatableTopic
	TimeoutMs    int32
	ValidateOnly bool
}

// CreatableTopic represents a topic to be created
type CreatableTopic struct {
	Name              string
	NumPartitions     int32
	ReplicationFactor int16
	Assignments       []CreatableReplicaAssignment
	Configs           []CreatableTopicConfig
}

// CreatableReplicaAssignment represents partition replica assignment
type CreatableReplicaAssignment struct {
	PartitionIndex int32
	BrokerIDs      []int32
}

// CreatableTopicConfig represents topic configuration
type CreatableTopicConfig struct {
	Name  string
	Value *string
}

// CreateTopicsResponse represents a CreateTopics response
type CreateTopicsResponse struct {
	ThrottleTimeMs int32
	Topics         []CreatableTopicResult
}

// CreatableTopicResult represents the result of creating a topic
type CreatableTopicResult struct {
	Name              string
	ErrorCode         ErrorCode
	ErrorMessage      *string
	NumPartitions     int32
	ReplicationFactor int16
	Configs           []CreatableTopicConfig
}

// DecodeCreateTopicsRequest decodes a CreateTopics request
func DecodeCreateTopicsRequest(r io.Reader, header *RequestHeader) (*CreateTopicsRequest, error) {
	req := &CreateTopicsRequest{
		Header: header,
	}

	// Read topics array
	topicCount, err := ReadArrayLength(r)
	if err != nil {
		return nil, err
	}

	req.Topics = make([]CreatableTopic, topicCount)
	for i := int32(0); i < topicCount; i++ {
		// Read topic name
		name, err := ReadString(r)
		if err != nil {
			return nil, err
		}

		// Read num partitions
		numPartitions, err := ReadInt32(r)
		if err != nil {
			return nil, err
		}

		// Read replication factor
		replicationFactor, err := ReadInt16(r)
		if err != nil {
			return nil, err
		}

		// Read assignments array
		assignmentCount, err := ReadArrayLength(r)
		if err != nil {
			return nil, err
		}

		assignments := make([]CreatableReplicaAssignment, assignmentCount)
		for j := int32(0); j < assignmentCount; j++ {
			partitionIndex, err := ReadInt32(r)
			if err != nil {
				return nil, err
			}

			brokerIDCount, err := ReadArrayLength(r)
			if err != nil {
				return nil, err
			}

			brokerIDs := make([]int32, brokerIDCount)
			for k := int32(0); k < brokerIDCount; k++ {
				brokerID, err := ReadInt32(r)
				if err != nil {
					return nil, err
				}
				brokerIDs[k] = brokerID
			}

			assignments[j] = CreatableReplicaAssignment{
				PartitionIndex: partitionIndex,
				BrokerIDs:      brokerIDs,
			}
		}

		// Read configs array
		configCount, err := ReadArrayLength(r)
		if err != nil {
			return nil, err
		}

		configs := make([]CreatableTopicConfig, configCount)
		for j := int32(0); j < configCount; j++ {
			configName, err := ReadString(r)
			if err != nil {
				return nil, err
			}

			configValue, err := ReadNullableString(r)
			if err != nil {
				return nil, err
			}

			configs[j] = CreatableTopicConfig{
				Name:  configName,
				Value: configValue,
			}
		}

		req.Topics[i] = CreatableTopic{
			Name:              name,
			NumPartitions:     numPartitions,
			ReplicationFactor: replicationFactor,
			Assignments:       assignments,
			Configs:           configs,
		}
	}

	// Read timeout
	timeoutMs, err := ReadInt32(r)
	if err != nil {
		return nil, err
	}
	req.TimeoutMs = timeoutMs

	// Read validate only
	validateOnly, err := ReadBool(r)
	if err != nil {
		return nil, err
	}
	req.ValidateOnly = validateOnly

	return req, nil
}

// Encode encodes the CreateTopics response
func (r *CreateTopicsResponse) Encode(w io.Writer) error {
	// Write throttle time
	if err := WriteInt32(w, r.ThrottleTimeMs); err != nil {
		return err
	}

	// Write topics array
	if err := WriteArray(w, len(r.Topics)); err != nil {
		return err
	}

	for _, topic := range r.Topics {
		// Write topic name
		if err := WriteString(w, topic.Name); err != nil {
			return err
		}

		// Write error code
		if err := WriteInt16(w, int16(topic.ErrorCode)); err != nil {
			return err
		}

		// Write error message
		if err := WriteNullableString(w, topic.ErrorMessage); err != nil {
			return err
		}

		// Write num partitions (version 5+)
		if err := WriteInt32(w, topic.NumPartitions); err != nil {
			return err
		}

		// Write replication factor (version 5+)
		if err := WriteInt16(w, topic.ReplicationFactor); err != nil {
			return err
		}

		// Write configs (version 5+)
		if err := WriteArray(w, len(topic.Configs)); err != nil {
			return err
		}

		for _, config := range topic.Configs {
			if err := WriteString(w, config.Name); err != nil {
				return err
			}
			if err := WriteNullableString(w, config.Value); err != nil {
				return err
			}
		}
	}

	return nil
}
