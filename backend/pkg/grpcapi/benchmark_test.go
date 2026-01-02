// Copyright 2025 Takhin Data, Inc.

package grpcapi

import (
"context"
"fmt"
"testing"

"github.com/takhin-data/takhin/pkg/coordinator"
"github.com/takhin-data/takhin/pkg/storage/topic"
)

func BenchmarkProduceMessage(b *testing.B) {
dataDir := b.TempDir()
topicManager := topic.NewManager(dataDir, 1024*1024)
coord := coordinator.NewCoordinator()
coord.Start()

server := NewServer(topicManager, coord, "bench-1.0.0")
server.CreateTopic(context.Background(), &CreateTopicRequest{
Name:          "bench-topic",
NumPartitions: 4,
})

ctx := context.Background()
value := make([]byte, 1024)
b.ResetTimer()
b.ReportAllocs()

for i := 0; i < b.N; i++ {
server.ProduceMessage(ctx, &ProduceMessageRequest{
Topic:     "bench-topic",
Partition: int32(i % 4),
Record: &Record{
Key:   []byte(fmt.Sprintf("key-%d", i)),
Value: value,
},
})
}
}

func BenchmarkListTopics(b *testing.B) {
dataDir := b.TempDir()
topicManager := topic.NewManager(dataDir, 1024*1024)
coord := coordinator.NewCoordinator()
coord.Start()

server := NewServer(topicManager, coord, "bench-1.0.0")
for i := 0; i < 50; i++ {
server.CreateTopic(context.Background(), &CreateTopicRequest{
Name:          fmt.Sprintf("topic-%d", i),
NumPartitions: 1,
})
}

ctx := context.Background()
b.ResetTimer()
b.ReportAllocs()
for i := 0; i < b.N; i++ {
server.ListTopics(ctx, &ListTopicsRequest{})
}
}
