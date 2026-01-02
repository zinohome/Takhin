// Package main demonstrates how to use the Takhin Console REST API
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const (
	baseURL = "http://localhost:8080/api"
	apiKey  = "your-api-key" // Set if authentication is enabled
)

// Client for Takhin Console API
type Client struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

// NewClient creates a new Console API client
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) doRequest(method, path string, body interface{}, result interface{}) error {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
	}

	if result != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}

// Topic operations

type Topic struct {
	Name           string      `json:"name"`
	PartitionCount int         `json:"partitionCount"`
	Partitions     []Partition `json:"partitions"`
}

type Partition struct {
	ID            int   `json:"id"`
	HighWaterMark int64 `json:"highWaterMark"`
}

type CreateTopicRequest struct {
	Name       string `json:"name"`
	Partitions int    `json:"partitions"`
}

func (c *Client) ListTopics() ([]Topic, error) {
	var topics []Topic
	err := c.doRequest("GET", "/topics", nil, &topics)
	return topics, err
}

func (c *Client) GetTopic(name string) (*Topic, error) {
	var topic Topic
	err := c.doRequest("GET", "/topics/"+name, nil, &topic)
	return &topic, err
}

func (c *Client) CreateTopic(name string, partitions int) error {
	req := CreateTopicRequest{
		Name:       name,
		Partitions: partitions,
	}
	return c.doRequest("POST", "/topics", req, nil)
}

func (c *Client) DeleteTopic(name string) error {
	return c.doRequest("DELETE", "/topics/"+name, nil, nil)
}

// Message operations

type Message struct {
	Partition int    `json:"partition"`
	Offset    int64  `json:"offset"`
	Key       string `json:"key"`
	Value     string `json:"value"`
	Timestamp int64  `json:"timestamp"`
}

type ProduceMessageRequest struct {
	Partition int    `json:"partition"`
	Key       string `json:"key"`
	Value     string `json:"value"`
}

type ProduceMessageResponse struct {
	Offset    int64 `json:"offset"`
	Partition int32 `json:"partition"`
}

func (c *Client) GetMessages(topic string, partition int, offset int64, limit int) ([]Message, error) {
	path := fmt.Sprintf("/topics/%s/messages?partition=%d&offset=%d&limit=%d",
		topic, partition, offset, limit)
	var messages []Message
	err := c.doRequest("GET", path, nil, &messages)
	return messages, err
}

func (c *Client) ProduceMessage(topic string, partition int, key, value string) (*ProduceMessageResponse, error) {
	req := ProduceMessageRequest{
		Partition: partition,
		Key:       key,
		Value:     value,
	}
	var resp ProduceMessageResponse
	err := c.doRequest("POST", "/topics/"+topic+"/messages", req, &resp)
	return &resp, err
}

// Consumer Group operations

type ConsumerGroup struct {
	GroupID string `json:"groupId"`
	State   string `json:"state"`
	Members int    `json:"members"`
}

type ConsumerGroupDetails struct {
	GroupID       string              `json:"groupId"`
	State         string              `json:"state"`
	ProtocolType  string              `json:"protocolType"`
	Protocol      string              `json:"protocol"`
	Members       []GroupMember       `json:"members"`
	OffsetCommits []OffsetCommitEntry `json:"offsetCommits"`
}

type GroupMember struct {
	MemberID   string               `json:"memberId"`
	ClientID   string               `json:"clientId"`
	ClientHost string               `json:"clientHost"`
	Partitions []PartitionAssignment `json:"partitions"`
}

type PartitionAssignment struct {
	Topic     string `json:"topic"`
	Partition int    `json:"partition"`
}

type OffsetCommitEntry struct {
	Topic     string `json:"topic"`
	Partition int    `json:"partition"`
	Offset    int64  `json:"offset"`
	Metadata  string `json:"metadata"`
}

func (c *Client) ListConsumerGroups() ([]ConsumerGroup, error) {
	var groups []ConsumerGroup
	err := c.doRequest("GET", "/consumer-groups", nil, &groups)
	return groups, err
}

func (c *Client) GetConsumerGroup(groupID string) (*ConsumerGroupDetails, error) {
	var details ConsumerGroupDetails
	err := c.doRequest("GET", "/consumer-groups/"+groupID, nil, &details)
	return &details, err
}

// Health check

type HealthResponse struct {
	Status string `json:"status"`
}

func (c *Client) Health() (*HealthResponse, error) {
	var health HealthResponse
	err := c.doRequest("GET", "/health", nil, &health)
	return &health, err
}

// Example usage
func main() {
	client := NewClient(baseURL, apiKey)

	// Example 1: Health check
	fmt.Println("=== Health Check ===")
	health, err := client.Health()
	if err != nil {
		log.Fatalf("Health check failed: %v", err)
	}
	fmt.Printf("Status: %s\n\n", health.Status)

	// Example 2: Create topic
	fmt.Println("=== Create Topic ===")
	topicName := "orders"
	if err := client.CreateTopic(topicName, 3); err != nil {
		log.Printf("Warning: %v\n", err)
	} else {
		fmt.Printf("Topic '%s' created\n", topicName)
	}
	fmt.Println()

	// Example 3: List topics
	fmt.Println("=== List Topics ===")
	topics, err := client.ListTopics()
	if err != nil {
		log.Fatalf("List topics failed: %v", err)
	}
	for _, topic := range topics {
		fmt.Printf("Topic: %s, Partitions: %d\n", topic.Name, topic.PartitionCount)
		for _, p := range topic.Partitions {
			fmt.Printf("  Partition %d: HWM=%d\n", p.ID, p.HighWaterMark)
		}
	}
	fmt.Println()

	// Example 4: Produce messages
	fmt.Println("=== Produce Messages ===")
	messages := []struct {
		key   string
		value string
	}{
		{"order-1", `{"product":"laptop","price":1299.99}`},
		{"order-2", `{"product":"mouse","price":29.99}`},
		{"order-3", `{"product":"keyboard","price":79.99}`},
	}

	for _, msg := range messages {
		resp, err := client.ProduceMessage(topicName, 0, msg.key, msg.value)
		if err != nil {
			log.Printf("Produce failed: %v", err)
			continue
		}
		fmt.Printf("Produced: key=%s, offset=%d, partition=%d\n",
			msg.key, resp.Offset, resp.Partition)
	}
	fmt.Println()

	// Example 5: Read messages
	fmt.Println("=== Read Messages ===")
	readMessages, err := client.GetMessages(topicName, 0, 0, 10)
	if err != nil {
		log.Fatalf("Read messages failed: %v", err)
	}
	for _, msg := range readMessages {
		fmt.Printf("Offset %d: key=%s, value=%s\n", msg.Offset, msg.Key, msg.Value)
	}
	fmt.Println()

	// Example 6: List consumer groups
	fmt.Println("=== List Consumer Groups ===")
	groups, err := client.ListConsumerGroups()
	if err != nil {
		log.Fatalf("List consumer groups failed: %v", err)
	}
	for _, group := range groups {
		fmt.Printf("Group: %s, State: %s, Members: %d\n",
			group.GroupID, group.State, group.Members)
	}
	fmt.Println()

	// Example 7: Get consumer group details
	if len(groups) > 0 {
		fmt.Println("=== Consumer Group Details ===")
		groupID := groups[0].GroupID
		details, err := client.GetConsumerGroup(groupID)
		if err != nil {
			log.Printf("Get consumer group failed: %v", err)
		} else {
			fmt.Printf("Group: %s\n", details.GroupID)
			fmt.Printf("State: %s\n", details.State)
			fmt.Printf("Protocol: %s\n", details.Protocol)
			fmt.Printf("Members:\n")
			for _, member := range details.Members {
				fmt.Printf("  %s (%s)\n", member.MemberID, member.ClientID)
			}
			fmt.Printf("Offset Commits:\n")
			for _, commit := range details.OffsetCommits {
				fmt.Printf("  %s[%d] = %d\n", commit.Topic, commit.Partition, commit.Offset)
			}
		}
	}

	fmt.Println("\nAll examples completed!")
}
