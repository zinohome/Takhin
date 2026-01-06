// Copyright 2025 Takhin Data, Inc.

package console

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/takhin-data/takhin/pkg/config"
	"github.com/takhin-data/takhin/pkg/coordinator"
	"github.com/takhin-data/takhin/pkg/storage/topic"
)

func setupTestServerWithWS(t *testing.T) (*Server, func()) {
	dataDir := t.TempDir()
	topicMgr := topic.NewManager(dataDir, 1024*1024)
	coord := coordinator.NewCoordinator()
	coord.Start()

	authConfig := AuthConfig{
		Enabled: false,
	}

	server := NewServer(":0", topicMgr, coord, nil, authConfig, nil, &config.Config{})

	cleanup := func() {
		server.Shutdown()
		topicMgr.Close()
	}

	return server, cleanup
}

func TestWebSocketHub(t *testing.T) {
	hub := NewWebSocketHub()
	go hub.Run()
	defer hub.Stop()

	assert.NotNil(t, hub)
	assert.Equal(t, 0, hub.GetClientCount())

	err := hub.BroadcastMessage(MessageTypeMetrics, map[string]string{"test": "data"})
	assert.NoError(t, err)
}

func TestWebSocketConnection(t *testing.T) {
	server, cleanup := setupTestServerWithWS(t)
	defer cleanup()

	httpServer := httptest.NewServer(server.router)
	defer httpServer.Close()

	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http") + "/api/monitoring/ws"

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 1, server.wsHub.GetClientCount())

	ws.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, message, err := ws.ReadMessage()
	require.NoError(t, err)

	var wsMsg WebSocketMessage
	err = json.Unmarshal(message, &wsMsg)
	require.NoError(t, err)
	assert.Equal(t, MessageTypeMetrics, wsMsg.Type)
}

func TestWebSocketMetricsStreaming(t *testing.T) {
	server, cleanup := setupTestServerWithWS(t)
	defer cleanup()

	httpServer := httptest.NewServer(server.router)
	defer httpServer.Close()

	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http") + "/api/monitoring/ws"

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	messagesReceived := 0
	ws.SetReadDeadline(time.Now().Add(6 * time.Second))

	for messagesReceived < 2 {
		_, message, err := ws.ReadMessage()
		if err != nil {
			break
		}

		var wsMsg WebSocketMessage
		err = json.Unmarshal(message, &wsMsg)
		require.NoError(t, err)

		if wsMsg.Type == MessageTypeMetrics {
			messagesReceived++
			assert.NotNil(t, wsMsg.Data)
			assert.Greater(t, wsMsg.Timestamp, int64(0))
		}
	}

	assert.GreaterOrEqual(t, messagesReceived, 2, "should receive at least 2 metrics messages")
}

func TestWebSocketBroadcast(t *testing.T) {
	server, cleanup := setupTestServerWithWS(t)
	defer cleanup()

	httpServer := httptest.NewServer(server.router)
	defer httpServer.Close()

	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http") + "/api/monitoring/ws"

	ws1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws1.Close()

	ws2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws2.Close()

	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, 2, server.wsHub.GetClientCount())

	testData := map[string]interface{}{
		"name":       "test-topic",
		"partitions": 3,
	}
	err = server.wsHub.BroadcastMessage(MessageTypeTopicCreated, testData)
	require.NoError(t, err)

	ws1.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, message1, err := ws1.ReadMessage()
	require.NoError(t, err)

	var wsMsg1 WebSocketMessage
	err = json.Unmarshal(message1, &wsMsg1)
	require.NoError(t, err)

	ws2.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, message2, err := ws2.ReadMessage()
	require.NoError(t, err)

	var wsMsg2 WebSocketMessage
	err = json.Unmarshal(message2, &wsMsg2)
	require.NoError(t, err)
}

func TestWebSocketTopicEvents(t *testing.T) {
	server, cleanup := setupTestServerWithWS(t)
	defer cleanup()

	httpServer := httptest.NewServer(server.router)
	defer httpServer.Close()

	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http") + "/api/monitoring/ws"

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	time.Sleep(100 * time.Millisecond)

	server.BroadcastTopicCreated("new-topic", 5)

	receivedEvent := false
	ws.SetReadDeadline(time.Now().Add(3 * time.Second))

	for !receivedEvent {
		_, message, err := ws.ReadMessage()
		if err != nil {
			break
		}

		var wsMsg WebSocketMessage
		err = json.Unmarshal(message, &wsMsg)
		require.NoError(t, err)

		if wsMsg.Type == MessageTypeTopicCreated {
			receivedEvent = true
			data := wsMsg.Data.(map[string]interface{})
			assert.Equal(t, "new-topic", data["name"])
			assert.Equal(t, float64(5), data["partitions"])
		}
	}

	assert.True(t, receivedEvent, "should receive topic created event")
}

func TestWebSocketGroupEvents(t *testing.T) {
	server, cleanup := setupTestServerWithWS(t)
	defer cleanup()

	httpServer := httptest.NewServer(server.router)
	defer httpServer.Close()

	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http") + "/api/monitoring/ws"

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	time.Sleep(100 * time.Millisecond)

	server.BroadcastGroupUpdated("test-group", "Stable", 3)

	receivedEvent := false
	ws.SetReadDeadline(time.Now().Add(3 * time.Second))

	for !receivedEvent {
		_, message, err := ws.ReadMessage()
		if err != nil {
			break
		}

		var wsMsg WebSocketMessage
		err = json.Unmarshal(message, &wsMsg)
		require.NoError(t, err)

		if wsMsg.Type == MessageTypeGroupUpdated {
			receivedEvent = true
			data := wsMsg.Data.(map[string]interface{})
			assert.Equal(t, "test-group", data["groupId"])
			assert.Equal(t, "Stable", data["state"])
			assert.Equal(t, float64(3), data["members"])
		}
	}

	assert.True(t, receivedEvent, "should receive group updated event")
}

func TestWebSocketClientSubscription(t *testing.T) {
	server, cleanup := setupTestServerWithWS(t)
	defer cleanup()

	httpServer := httptest.NewServer(server.router)
	defer httpServer.Close()

	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http") + "/api/monitoring/ws"

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	time.Sleep(100 * time.Millisecond)

	subscribeMsg := WebSocketMessage{
		Type:      MessageTypeSubscribe,
		Data:      "topic:test-topic",
		Timestamp: time.Now().Unix(),
	}
	data, err := json.Marshal(subscribeMsg)
	require.NoError(t, err)

	err = ws.WriteMessage(websocket.TextMessage, data)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
}

func TestWebSocketPingPong(t *testing.T) {
	server, cleanup := setupTestServerWithWS(t)
	defer cleanup()

	httpServer := httptest.NewServer(server.router)
	defer httpServer.Close()

	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http") + "/api/monitoring/ws"

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	time.Sleep(100 * time.Millisecond)

	pingMsg := WebSocketMessage{
		Type:      MessageTypePing,
		Timestamp: time.Now().Unix(),
	}
	data, err := json.Marshal(pingMsg)
	require.NoError(t, err)

	err = ws.WriteMessage(websocket.TextMessage, data)
	require.NoError(t, err)

	receivedPong := false
	ws.SetReadDeadline(time.Now().Add(2 * time.Second))

	for !receivedPong {
		_, message, err := ws.ReadMessage()
		if err != nil {
			break
		}

		var wsMsg WebSocketMessage
		err = json.Unmarshal(message, &wsMsg)
		require.NoError(t, err)

		if wsMsg.Type == MessageTypePong {
			receivedPong = true
		}
	}

	assert.True(t, receivedPong, "should receive pong response")
}

func TestWebSocketMultipleClients(t *testing.T) {
	server, cleanup := setupTestServerWithWS(t)
	defer cleanup()

	httpServer := httptest.NewServer(server.router)
	defer httpServer.Close()

	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http") + "/api/monitoring/ws"

	numClients := 5
	clients := make([]*websocket.Conn, numClients)

	for i := 0; i < numClients; i++ {
		ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		clients[i] = ws
		defer ws.Close()
	}

	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, numClients, server.wsHub.GetClientCount())

	for i := 0; i < numClients; i++ {
		clients[i].Close()
	}

	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, 0, server.wsHub.GetClientCount())
}

func TestWebSocketConnectionLimit(t *testing.T) {
	server, cleanup := setupTestServerWithWS(t)
	defer cleanup()

	httpServer := httptest.NewServer(server.router)
	defer httpServer.Close()

	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http") + "/api/monitoring/ws"

	numClients := 10
	clients := make([]*websocket.Conn, numClients)

	for i := 0; i < numClients; i++ {
		ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		clients[i] = ws
		defer ws.Close()
	}

	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, numClients, server.wsHub.GetClientCount())
}
