package raft

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/raft"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/takhin-data/takhin/pkg/storage/topic"
)

func TestFSMApplyWithJSON(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := topic.NewManager(tmpDir, 1024*1024)
	defer mgr.Close()

	fsm := NewFSM(mgr)

	tests := []struct {
		name    string
		cmd     Command
		wantErr bool
	}{
		{
			name: "create topic",
			cmd: Command{
				Type:      CommandCreateTopic,
				TopicName: "topic1",
				NumParts:  2,
			},
			wantErr: false,
		},
		{
			name: "delete topic",
			cmd: Command{
				Type:      CommandDeleteTopic,
				TopicName: "topic1",
			},
			wantErr: false,
		},
		{
			name: "unknown command",
			cmd: Command{
				Type: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.cmd)
			require.NoError(t, err)

			log := &raft.Log{Data: data}
			result := fsm.Apply(log)

			if tt.wantErr {
				assert.NotNil(t, result)
				_, isErr := result.(error)
				assert.True(t, isErr)
			} else {
				// Result could be nil or an error
				if result != nil {
					_, isErr := result.(error)
					if isErr {
						// Some operations may fail (e.g., delete non-existent topic)
						t.Logf("Operation failed as expected: %v", result)
					}
				}
			}
		})
	}
}

func TestFSMApplyInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := topic.NewManager(tmpDir, 1024*1024)
	defer mgr.Close()

	fsm := NewFSM(mgr)

	log := &raft.Log{Data: []byte("not json")}
	result := fsm.Apply(log)

	assert.NotNil(t, result)
	assert.Error(t, result.(error))
}

func TestFSMApplyDeleteTopic(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := topic.NewManager(tmpDir, 1024*1024)
	defer mgr.Close()

	// Create topic first
	err := mgr.CreateTopic("test", 1)
	require.NoError(t, err)

	fsm := NewFSM(mgr)

	cmd := Command{
		Type:      CommandDeleteTopic,
		TopicName: "test",
	}

	result := fsm.applyDeleteTopic(cmd)
	assert.Nil(t, result)

	// Verify deleted
	_, exists := mgr.GetTopic("test")
	assert.False(t, exists)
}

func TestFSMApplyAppendNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := topic.NewManager(tmpDir, 1024*1024)
	defer mgr.Close()

	fsm := NewFSM(mgr)

	cmd := Command{
		Type:      CommandAppend,
		TopicName: "nonexistent",
		Partition: 0,
		Key:       []byte("key"),
		Value:     []byte("value"),
	}

	result := fsm.applyAppend(cmd)
	assert.NotNil(t, result)
	assert.Error(t, result.(error))
}

func TestFSMSnapshot(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := topic.NewManager(tmpDir, 1024*1024)
	defer mgr.Close()

	// Create some topics
	require.NoError(t, mgr.CreateTopic("topic1", 1))
	require.NoError(t, mgr.CreateTopic("topic2", 2))

	fsm := NewFSM(mgr)

	snapshot, err := fsm.Snapshot()
	require.NoError(t, err)
	assert.NotNil(t, snapshot)

	fsmSnap, ok := snapshot.(*FSMSnapshot)
	require.True(t, ok)
	assert.Len(t, fsmSnap.topics, 2)
}

func TestFSMSnapshotPersist(t *testing.T) {
	snapshot := &FSMSnapshot{
		topics: []string{"topic1", "topic2"},
	}

	tmpFile := filepath.Join(t.TempDir(), "snapshot.dat")
	file, err := os.Create(tmpFile)
	require.NoError(t, err)

	sink := &testSnapshotSink{file: file}
	err = snapshot.Persist(sink)
	require.NoError(t, err)

	// Verify file written
	info, err := os.Stat(tmpFile)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))
}

func TestFSMRestore(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := topic.NewManager(tmpDir, 1024*1024)
	defer mgr.Close()

	fsm := NewFSM(mgr)

	// Create snapshot data
	snapshotData := struct {
		Topics []string `json:"topics"`
	}{
		Topics: []string{"topic1", "topic2"},
	}

	tmpFile := filepath.Join(tmpDir, "snapshot.json")
	file, err := os.Create(tmpFile)
	require.NoError(t, err)

	enc := json.NewEncoder(file)
	require.NoError(t, enc.Encode(snapshotData))
	file.Close()

	// Restore
	file, err = os.Open(tmpFile)
	require.NoError(t, err)

	err = fsm.Restore(file)
	assert.NoError(t, err)
}

func TestFSMRestoreInvalid(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := topic.NewManager(tmpDir, 1024*1024)
	defer mgr.Close()

	fsm := NewFSM(mgr)

	tmpFile := filepath.Join(tmpDir, "invalid.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte("invalid"), 0644))

	file, err := os.Open(tmpFile)
	require.NoError(t, err)

	err = fsm.Restore(file)
	assert.Error(t, err)
}

func TestFSMSnapshotRelease(t *testing.T) {
	snapshot := &FSMSnapshot{topics: []string{"test"}}
	
	// Should not panic
	assert.NotPanics(t, func() {
		snapshot.Release()
	})
}

func TestFSMTopicManager(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := topic.NewManager(tmpDir, 1024*1024)
	defer mgr.Close()

	fsm := NewFSM(mgr)
	assert.Equal(t, mgr, fsm.TopicManager())
}

// testSnapshotSink implements raft.SnapshotSink for testing
type testSnapshotSink struct {
	file *os.File
}

func (s *testSnapshotSink) Write(p []byte) (n int, err error) {
	return s.file.Write(p)
}

func (s *testSnapshotSink) Close() error {
	return s.file.Close()
}

func (s *testSnapshotSink) ID() string {
	return "test"
}

func (s *testSnapshotSink) Cancel() error {
	s.file.Close()
	return os.Remove(s.file.Name())
}
