// Copyright 2025 Takhin Data, Inc.

package handler

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/takhin-data/takhin/pkg/acl"
	"github.com/takhin-data/takhin/pkg/config"
	"github.com/takhin-data/takhin/pkg/kafka/protocol"
	"github.com/takhin-data/takhin/pkg/storage/topic"
)

func TestHandleCreateAcls(t *testing.T) {
	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir: t.TempDir(),
		},
		ACL: config.ACLConfig{
			Enabled: true,
		},
	}

	mgr := topic.NewManager(cfg.Storage.DataDir, 1024*1024)
	handler := New(cfg, mgr)
	defer handler.Close()

	// Encode CreateAcls request
	var buf bytes.Buffer
	protocol.WriteInt32(&buf, 1) // 1 creation

	// Write ACL creation
	protocol.WriteInt8(&buf, int8(acl.ResourceTypeTopic))
	protocol.WriteString(&buf, "test-topic")
	protocol.WriteInt8(&buf, int8(acl.PatternTypeLiteral))
	protocol.WriteString(&buf, "User:alice")
	protocol.WriteString(&buf, "*")
	protocol.WriteInt8(&buf, int8(acl.OperationRead))
	protocol.WriteInt8(&buf, int8(acl.PermissionTypeAllow))

	// Handle request
	response, err := handler.HandleCreateAcls(bytes.NewReader(buf.Bytes()), 0)
	require.NoError(t, err)

	// Decode response
	r := bytes.NewReader(response)
	throttleTime, err := protocol.ReadInt32(r)
	require.NoError(t, err)
	assert.Equal(t, int32(0), throttleTime)

	numResults, err := protocol.ReadInt32(r)
	require.NoError(t, err)
	assert.Equal(t, int32(1), numResults)

	errorCode, err := protocol.ReadInt16(r)
	require.NoError(t, err)
	assert.Equal(t, int16(protocol.None), errorCode)

	// Verify ACL was created
	entries := handler.aclStore.GetAll()
	assert.Len(t, entries, 1)
	assert.Equal(t, "User:alice", entries[0].Principal)
	assert.Equal(t, "test-topic", entries[0].ResourceName)
}

func TestHandleDescribeAcls(t *testing.T) {
	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir: t.TempDir(),
		},
		ACL: config.ACLConfig{
			Enabled: true,
		},
	}

	mgr := topic.NewManager(cfg.Storage.DataDir, 1024*1024)
	handler := New(cfg, mgr)
	defer handler.Close()

	// Add some ACLs
	entry1 := acl.Entry{
		Principal:      "User:alice",
		Host:           "*",
		ResourceType:   acl.ResourceTypeTopic,
		ResourceName:   "test-topic",
		PatternType:    acl.PatternTypeLiteral,
		Operation:      acl.OperationRead,
		PermissionType: acl.PermissionTypeAllow,
	}

	entry2 := acl.Entry{
		Principal:      "User:alice",
		Host:           "*",
		ResourceType:   acl.ResourceTypeTopic,
		ResourceName:   "test-topic",
		PatternType:    acl.PatternTypeLiteral,
		Operation:      acl.OperationWrite,
		PermissionType: acl.PermissionTypeAllow,
	}

	require.NoError(t, handler.aclStore.Add(entry1))
	require.NoError(t, handler.aclStore.Add(entry2))

	// Encode DescribeAcls request
	var buf bytes.Buffer
	protocol.WriteInt8(&buf, int8(acl.ResourceTypeTopic))
	
	topicName := "test-topic"
	protocol.WriteInt16(&buf, int16(len(topicName)))
	buf.WriteString(topicName)
	
	protocol.WriteInt8(&buf, int8(acl.PatternTypeLiteral))
	
	principal := "User:alice"
	protocol.WriteInt16(&buf, int16(len(principal)))
	buf.WriteString(principal)
	
	protocol.WriteInt16(&buf, int16(1))
	buf.WriteString("*")
	
	protocol.WriteInt8(&buf, -1) // Any operation
	protocol.WriteInt8(&buf, -1) // Any permission type

	// Handle request
	response, err := handler.HandleDescribeAcls(bytes.NewReader(buf.Bytes()), 0)
	require.NoError(t, err)

	// Decode response
	r := bytes.NewReader(response)
	throttleTime, err := protocol.ReadInt32(r)
	require.NoError(t, err)
	assert.Equal(t, int32(0), throttleTime)

	errorCode, err := protocol.ReadInt16(r)
	require.NoError(t, err)
	assert.Equal(t, int16(protocol.None), errorCode)

	// Skip error message
	_, err = protocol.ReadNullableString(r)
	require.NoError(t, err)

	numResources, err := protocol.ReadInt32(r)
	require.NoError(t, err)
	assert.Equal(t, int32(1), numResources)
}

func TestHandleDeleteAcls(t *testing.T) {
	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir: t.TempDir(),
		},
		ACL: config.ACLConfig{
			Enabled: true,
		},
	}

	mgr := topic.NewManager(cfg.Storage.DataDir, 1024*1024)
	handler := New(cfg, mgr)
	defer handler.Close()

	// Add ACL
	entry := acl.Entry{
		Principal:      "User:alice",
		Host:           "*",
		ResourceType:   acl.ResourceTypeTopic,
		ResourceName:   "test-topic",
		PatternType:    acl.PatternTypeLiteral,
		Operation:      acl.OperationRead,
		PermissionType: acl.PermissionTypeAllow,
	}

	require.NoError(t, handler.aclStore.Add(entry))

	// Encode DeleteAcls request
	var buf bytes.Buffer
	protocol.WriteInt32(&buf, 1) // 1 filter

	// Write filter
	protocol.WriteInt8(&buf, int8(acl.ResourceTypeTopic))
	
	topicName := "test-topic"
	protocol.WriteInt16(&buf, int16(len(topicName)))
	buf.WriteString(topicName)
	
	protocol.WriteInt8(&buf, int8(acl.PatternTypeLiteral))
	
	principal := "User:alice"
	protocol.WriteInt16(&buf, int16(len(principal)))
	buf.WriteString(principal)
	
	protocol.WriteInt16(&buf, int16(1))
	buf.WriteString("*")
	
	protocol.WriteInt8(&buf, -1) // Any operation
	protocol.WriteInt8(&buf, -1) // Any permission type

	// Handle request
	response, err := handler.HandleDeleteAcls(bytes.NewReader(buf.Bytes()), 0)
	require.NoError(t, err)

	// Decode response
	r := bytes.NewReader(response)
	throttleTime, err := protocol.ReadInt32(r)
	require.NoError(t, err)
	assert.Equal(t, int32(0), throttleTime)

	numResults, err := protocol.ReadInt32(r)
	require.NoError(t, err)
	assert.Equal(t, int32(1), numResults)

	errorCode, err := protocol.ReadInt16(r)
	require.NoError(t, err)
	assert.Equal(t, int16(protocol.None), errorCode)

	// Verify ACL was deleted
	entries := handler.aclStore.GetAll()
	assert.Len(t, entries, 0)
}

func TestHandleCreateAclsDuplicate(t *testing.T) {
	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir: t.TempDir(),
		},
		ACL: config.ACLConfig{
			Enabled: true,
		},
	}

	mgr := topic.NewManager(cfg.Storage.DataDir, 1024*1024)
	handler := New(cfg, mgr)
	defer handler.Close()

	// Add ACL first
	entry := acl.Entry{
		Principal:      "User:alice",
		Host:           "*",
		ResourceType:   acl.ResourceTypeTopic,
		ResourceName:   "test-topic",
		PatternType:    acl.PatternTypeLiteral,
		Operation:      acl.OperationRead,
		PermissionType: acl.PermissionTypeAllow,
	}
	require.NoError(t, handler.aclStore.Add(entry))

	// Try to create duplicate
	var buf bytes.Buffer
	protocol.WriteInt32(&buf, 1)
	protocol.WriteInt8(&buf, int8(acl.ResourceTypeTopic))
	protocol.WriteString(&buf, "test-topic")
	protocol.WriteInt8(&buf, int8(acl.PatternTypeLiteral))
	protocol.WriteString(&buf, "User:alice")
	protocol.WriteString(&buf, "*")
	protocol.WriteInt8(&buf, int8(acl.OperationRead))
	protocol.WriteInt8(&buf, int8(acl.PermissionTypeAllow))

	response, err := handler.HandleCreateAcls(bytes.NewReader(buf.Bytes()), 0)
	require.NoError(t, err)

	// Decode response
	r := bytes.NewReader(response)
	_, err = protocol.ReadInt32(r)
	require.NoError(t, err)

	_, err = protocol.ReadInt32(r)
	require.NoError(t, err)

	errorCode, err := protocol.ReadInt16(r)
	require.NoError(t, err)
	assert.NotEqual(t, int16(protocol.None), errorCode)
}
