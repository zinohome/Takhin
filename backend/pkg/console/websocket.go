// Copyright 2025 Takhin Data, Inc.

package console

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/takhin-data/takhin/pkg/logger"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024
)

// WebSocketMessage represents different types of messages sent over WebSocket
type WebSocketMessage struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
}

// MessageType constants
const (
	MessageTypeMetrics      = "metrics"
	MessageTypeTopicCreated = "topic_created"
	MessageTypeTopicDeleted = "topic_deleted"
	MessageTypeGroupCreated = "group_created"
	MessageTypeGroupUpdated = "group_updated"
	MessageTypeGroupDeleted = "group_deleted"
	MessageTypePing         = "ping"
	MessageTypePong         = "pong"
	MessageTypeSubscribe    = "subscribe"
	MessageTypeUnsubscribe  = "unsubscribe"
	MessageTypeError        = "error"
)

// Client represents a WebSocket client connection
type Client struct {
	id            string
	conn          *websocket.Conn
	send          chan []byte
	hub           *WebSocketHub
	logger        *logger.Logger
	subscriptions map[string]bool
	mu            sync.RWMutex
}

// WebSocketHub manages all active WebSocket connections
type WebSocketHub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
	logger     *logger.Logger
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub() *WebSocketHub {
	ctx, cancel := context.WithCancel(context.Background())
	return &WebSocketHub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		logger:     logger.Default().WithComponent("websocket-hub"),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Run starts the WebSocket hub
func (h *WebSocketHub) Run() {
	h.logger.Info("starting websocket hub")
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			h.logger.Info("client registered", "client_id", client.id, "total_clients", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				h.logger.Info("client unregistered", "client_id", client.id, "total_clients", len(h.clients))
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					h.mu.RUnlock()
					h.mu.Lock()
					close(client.send)
					delete(h.clients, client)
					h.mu.Unlock()
					h.mu.RLock()
				}
			}
			h.mu.RUnlock()

		case <-h.ctx.Done():
			h.logger.Info("shutting down websocket hub")
			return
		}
	}
}

// Stop stops the WebSocket hub
func (h *WebSocketHub) Stop() {
	h.cancel()
}

// BroadcastMessage sends a message to all connected clients
func (h *WebSocketHub) BroadcastMessage(msgType string, data interface{}) error {
	msg := WebSocketMessage{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now().Unix(),
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	h.broadcast <- jsonData
	return nil
}

// GetClientCount returns the number of connected clients
func (h *WebSocketHub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// readPump pumps messages from the websocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	c.conn.SetReadLimit(maxMessageSize)

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Error("unexpected close error", "error", err)
			}
			break
		}

		c.handleClientMessage(message)
	}
}

// writePump pumps messages from the hub to the websocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleClientMessage processes messages from the client
func (c *Client) handleClientMessage(message []byte) {
	var msg WebSocketMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		c.logger.Error("failed to unmarshal client message", "error", err)
		return
	}

	switch msg.Type {
	case MessageTypeSubscribe:
		c.handleSubscribe(msg.Data)
	case MessageTypeUnsubscribe:
		c.handleUnsubscribe(msg.Data)
	case MessageTypePing:
		c.sendPong()
	default:
		c.logger.Debug("unknown message type", "type", msg.Type)
	}
}

// handleSubscribe subscribes the client to a topic
func (c *Client) handleSubscribe(data interface{}) {
	if topic, ok := data.(string); ok {
		c.mu.Lock()
		c.subscriptions[topic] = true
		c.mu.Unlock()
		c.logger.Debug("client subscribed", "client_id", c.id, "topic", topic)
	}
}

// handleUnsubscribe unsubscribes the client from a topic
func (c *Client) handleUnsubscribe(data interface{}) {
	if topic, ok := data.(string); ok {
		c.mu.Lock()
		delete(c.subscriptions, topic)
		c.mu.Unlock()
		c.logger.Debug("client unsubscribed", "client_id", c.id, "topic", topic)
	}
}

// sendPong sends a pong message to the client
func (c *Client) sendPong() {
	msg := WebSocketMessage{
		Type:      MessageTypePong,
		Timestamp: time.Now().Unix(),
	}
	if data, err := json.Marshal(msg); err == nil {
		select {
		case c.send <- data:
		default:
		}
	}
}

// handleMonitoringWebSocket godoc
// @Summary      WebSocket monitoring stream
// @Description  Establishes a WebSocket connection for real-time monitoring metrics updates
// @Tags         Monitoring
// @Produce      json
// @Success      101  {object}  WebSocketMessage  "Switching Protocols - WebSocket established"
// @Security     ApiKeyAuth
// @Router       /monitoring/ws [get]
func (s *Server) handleMonitoringWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("failed to upgrade websocket connection", "error", err)
		return
	}

	clientID := r.RemoteAddr
	client := &Client{
		id:            clientID,
		conn:          conn,
		send:          make(chan []byte, 256),
		hub:           s.wsHub,
		logger:        s.logger.WithComponent("ws-client"),
		subscriptions: make(map[string]bool),
	}

	s.wsHub.register <- client

	go client.writePump()
	go client.readPump()

	go s.streamMetricsToClient(client)
}

// streamMetricsToClient sends periodic metrics updates to a specific client
func (s *Server) streamMetricsToClient(client *Client) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			metrics := MonitoringMetrics{
				Throughput:    s.collectThroughputMetrics(),
				Latency:       s.collectLatencyMetrics(),
				TopicStats:    s.collectTopicStats(),
				ConsumerLags:  s.collectConsumerLags(),
				ClusterHealth: s.collectClusterHealth(),
				Timestamp:     time.Now().Unix(),
			}

			msg := WebSocketMessage{
				Type:      MessageTypeMetrics,
				Data:      metrics,
				Timestamp: time.Now().Unix(),
			}

			data, err := json.Marshal(msg)
			if err != nil {
				s.logger.Error("failed to marshal metrics", "error", err)
				continue
			}

			select {
			case client.send <- data:
			default:
				return
			}

		case <-client.hub.ctx.Done():
			return
		}
	}
}

// BroadcastTopicCreated broadcasts topic creation event
func (s *Server) BroadcastTopicCreated(topicName string, partitions int32) {
	data := map[string]interface{}{
		"name":       topicName,
		"partitions": partitions,
	}
	s.wsHub.BroadcastMessage(MessageTypeTopicCreated, data)
}

// BroadcastTopicDeleted broadcasts topic deletion event
func (s *Server) BroadcastTopicDeleted(topicName string) {
	data := map[string]interface{}{
		"name": topicName,
	}
	s.wsHub.BroadcastMessage(MessageTypeTopicDeleted, data)
}

// BroadcastGroupUpdated broadcasts consumer group update event
func (s *Server) BroadcastGroupUpdated(groupID string, state string, members int) {
	data := map[string]interface{}{
		"groupId": groupID,
		"state":   state,
		"members": members,
	}
	s.wsHub.BroadcastMessage(MessageTypeGroupUpdated, data)
}
