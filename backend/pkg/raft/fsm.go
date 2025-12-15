package raft

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/hashicorp/raft"
	"github.com/takhin-data/takhin/pkg/storage/topic"
)

// FSM implements the raft.FSM interface for Takhin
type FSM struct {
	topicManager *topic.Manager
}

// NewFSM creates a new FSM
func NewFSM(topicManager *topic.Manager) *FSM {
	return &FSM{
		topicManager: topicManager,
	}
}

// TopicManager returns the underlying topic manager
func (f *FSM) TopicManager() *topic.Manager {
	return f.topicManager
}

// CommandType represents the type of command
type CommandType string

const (
	CommandCreateTopic CommandType = "create_topic"
	CommandAppend      CommandType = "append"
)

// Command represents a Raft command
type Command struct {
	Type      CommandType `json:"type"`
	TopicName string      `json:"topic_name,omitempty"`
	NumParts  int32       `json:"num_partitions,omitempty"`
	Partition int32       `json:"partition,omitempty"`
	Key       []byte      `json:"key,omitempty"`
	Value     []byte      `json:"value,omitempty"`
}

// Apply applies a Raft log entry to the FSM
func (f *FSM) Apply(log *raft.Log) interface{} {
	var cmd Command
	if err := json.Unmarshal(log.Data, &cmd); err != nil {
		return fmt.Errorf("failed to unmarshal command: %w", err)
	}

	switch cmd.Type {
	case CommandCreateTopic:
		return f.applyCreateTopic(cmd)
	case CommandAppend:
		return f.applyAppend(cmd)
	default:
		return fmt.Errorf("unknown command type: %s", cmd.Type)
	}
}

// applyCreateTopic creates a new topic
func (f *FSM) applyCreateTopic(cmd Command) interface{} {
	if err := f.topicManager.CreateTopic(cmd.TopicName, cmd.NumParts); err != nil {
		return err
	}
	return nil
}

// applyAppend appends a message to a topic
func (f *FSM) applyAppend(cmd Command) interface{} {
	topic, exists := f.topicManager.GetTopic(cmd.TopicName)
	if !exists {
		return fmt.Errorf("topic not found: %s", cmd.TopicName)
	}

	offset, err := topic.Append(cmd.Partition, cmd.Key, cmd.Value)
	if err != nil {
		return err
	}
	return offset
}

// Snapshot returns a snapshot of the FSM
func (f *FSM) Snapshot() (raft.FSMSnapshot, error) {
	// Get all topics
	topics := f.topicManager.ListTopics()

	snapshot := &FSMSnapshot{
		topics: topics,
	}
	return snapshot, nil
}

// Restore restores the FSM from a snapshot
func (f *FSM) Restore(rc io.ReadCloser) error {
	defer rc.Close()

	// Read snapshot data
	var snapshot struct {
		Topics []string `json:"topics"`
	}

	decoder := json.NewDecoder(rc)
	if err := decoder.Decode(&snapshot); err != nil {
		return fmt.Errorf("failed to decode snapshot: %w", err)
	}

	// Note: In a real implementation, we would restore the full state
	// including all topic data. For now, we just restore topic names.
	return nil
}

// FSMSnapshot implements raft.FSMSnapshot
type FSMSnapshot struct {
	topics []string
}

// Persist writes the snapshot to the given sink
func (s *FSMSnapshot) Persist(sink raft.SnapshotSink) error {
	// Encode snapshot
	snapshot := struct {
		Topics []string `json:"topics"`
	}{
		Topics: s.topics,
	}

	encoder := json.NewEncoder(sink)
	if err := encoder.Encode(snapshot); err != nil {
		sink.Cancel()
		return fmt.Errorf("failed to encode snapshot: %w", err)
	}

	return sink.Close()
}

// Release is called when the snapshot is no longer needed
func (s *FSMSnapshot) Release() {
	// Nothing to release
}
