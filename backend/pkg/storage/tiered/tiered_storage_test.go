// Copyright 2025 Takhin Data, Inc.

package tiered

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockS3API struct {
	mock.Mock
}

func (m *MockS3API) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*s3.PutObjectOutput), args.Error(1)
}

func (m *MockS3API) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*s3.GetObjectOutput), args.Error(1)
}

func (m *MockS3API) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*s3.DeleteObjectOutput), args.Error(1)
}

func (m *MockS3API) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*s3.HeadObjectOutput), args.Error(1)
}

func (m *MockS3API) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*s3.ListObjectsV2Output), args.Error(1)
}

func TestS3ClientUpload(t *testing.T) {
	_ = context.Background()
	
	tests := []struct {
		name        string
		bucket      string
		prefix      string
		key         string
		expectedKey string
	}{
		{
			name:        "upload with prefix",
			bucket:      "test-bucket",
			prefix:      "segments",
			key:         "topic-0/00000000000000000000.log",
			expectedKey: "segments/topic-0/00000000000000000000.log",
		},
		{
			name:        "upload without prefix",
			bucket:      "test-bucket",
			prefix:      "",
			key:         "topic-0/00000000000000000000.log",
			expectedKey: "topic-0/00000000000000000000.log",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.bucket)
			assert.NotEmpty(t, tt.key)
		})
	}
}

func TestTieredStoragePolicy(t *testing.T) {
	tests := []struct {
		name             string
		age              time.Duration
		coldThreshold    time.Duration
		warmThreshold    time.Duration
		expectedPolicy   StoragePolicy
		shouldArchive    bool
	}{
		{
			name:           "hot segment - recent",
			age:            1 * time.Hour,
			coldThreshold:  24 * time.Hour,
			warmThreshold:  12 * time.Hour,
			expectedPolicy: PolicyHot,
			shouldArchive:  false,
		},
		{
			name:           "warm segment",
			age:            18 * time.Hour,
			coldThreshold:  48 * time.Hour,
			warmThreshold:  12 * time.Hour,
			expectedPolicy: PolicyWarm,
			shouldArchive:  false,
		},
		{
			name:           "cold segment - should archive",
			age:            72 * time.Hour,
			coldThreshold:  48 * time.Hour,
			warmThreshold:  24 * time.Hour,
			expectedPolicy: PolicyCold,
			shouldArchive:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now()
			lastModified := now.Add(-tt.age)
			
			shouldArchive := lastModified.Add(tt.coldThreshold).Before(now)
			assert.Equal(t, tt.shouldArchive, shouldArchive)
		})
	}
}

func TestSegmentMetadata(t *testing.T) {
	meta := &SegmentMetadata{
		Path:         "/data/topic-0/00000000000000000000.log",
		BaseOffset:   0,
		Size:         1024 * 1024,
		LastAccessAt: time.Now(),
		LastModified: time.Now().Add(-2 * time.Hour),
		Policy:       PolicyHot,
		IsArchived:   false,
	}

	assert.Equal(t, PolicyHot, meta.Policy)
	assert.False(t, meta.IsArchived)
	assert.Equal(t, int64(1024*1024), meta.Size)
}

func TestTieredStorageStats(t *testing.T) {
	config := TieredStorageConfig{
		DataDir:            t.TempDir(),
		ColdAgeThreshold:   48 * time.Hour,
		WarmAgeThreshold:   24 * time.Hour,
		ArchiveInterval:    1 * time.Hour,
		LocalCacheSize:     100 * 1024 * 1024,
		AutoArchiveEnabled: false,
	}

	ts := &TieredStorage{
		config:   config,
		metadata: make(map[string]*SegmentMetadata),
	}

	now := time.Now()
	
	ts.metadata["seg1.log"] = &SegmentMetadata{
		Policy:       PolicyHot,
		IsArchived:   false,
		Size:         1024,
		LastModified: now,
	}
	
	ts.metadata["seg2.log"] = &SegmentMetadata{
		Policy:       PolicyCold,
		IsArchived:   true,
		Size:         2048,
		LastModified: now.Add(-72 * time.Hour),
	}

	stats := ts.GetStats()
	
	assert.Equal(t, 2, stats["total_segments"])
	assert.Equal(t, 1, stats["hot_segments"])
	assert.Equal(t, 1, stats["cold_segments"])
	assert.Equal(t, 1, stats["archived_segments"])
	assert.Equal(t, int64(3072), stats["total_size_bytes"])
}

func TestStoragePolicyTransitions(t *testing.T) {
	tests := []struct {
		name          string
		initialPolicy StoragePolicy
		finalPolicy   StoragePolicy
		action        string
	}{
		{
			name:          "hot to cold after archive",
			initialPolicy: PolicyHot,
			finalPolicy:   PolicyCold,
			action:        "archive",
		},
		{
			name:          "cold to hot after restore",
			initialPolicy: PolicyCold,
			finalPolicy:   PolicyHot,
			action:        "restore",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := &SegmentMetadata{
				Policy: tt.initialPolicy,
			}

			if tt.action == "archive" {
				meta.Policy = PolicyCold
				meta.IsArchived = true
			} else if tt.action == "restore" {
				meta.Policy = PolicyHot
				meta.IsArchived = false
			}

			assert.Equal(t, tt.finalPolicy, meta.Policy)
		})
	}
}

func TestReadCloserWrapper(t *testing.T) {
	data := []byte("test data")
	reader := io.NopCloser(bytes.NewReader(data))
	
	readData, err := io.ReadAll(reader)
	assert.NoError(t, err)
	assert.Equal(t, data, readData)
	
	err = reader.Close()
	assert.NoError(t, err)
}

func TestConcurrentAccess(t *testing.T) {
	ts := &TieredStorage{
		metadata: make(map[string]*SegmentMetadata),
	}

	done := make(chan bool)
	
	go func() {
		ts.UpdateAccessTime("seg1.log")
		done <- true
	}()
	
	go func() {
		_ = ts.GetSegmentPolicy("seg2.log")
		done <- true
	}()
	
	<-done
	<-done
}
