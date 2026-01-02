// Copyright 2025 Takhin Data, Inc.

package console

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// handleMonitoringWebSocket godoc
// @Summary      WebSocket monitoring stream
// @Description  Establishes a WebSocket connection for real-time monitoring metrics updates
// @Tags         Monitoring
// @Produce      json
// @Success      101  {object}  MonitoringMetrics  "Switching Protocols - WebSocket established"
// @Security     ApiKeyAuth
// @Router       /monitoring/ws [get]
func (s *Server) handleMonitoringWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("failed to upgrade websocket connection", "error", err)
		return
	}
	defer conn.Close()

	s.logger.Info("websocket client connected", "remote_addr", r.RemoteAddr)

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

			if err := conn.WriteJSON(metrics); err != nil {
				s.logger.Debug("websocket write error", "error", err)
				return
			}

		case <-r.Context().Done():
			s.logger.Info("websocket client disconnected", "remote_addr", r.RemoteAddr)
			return
		}

		if _, _, err := conn.ReadMessage(); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.logger.Debug("websocket read error", "error", err)
			}
			return
		}
	}
}

func (s *Server) broadcastMetrics(metrics MonitoringMetrics) {
	data, err := json.Marshal(metrics)
	if err != nil {
		s.logger.Error("failed to marshal metrics", "error", err)
		return
	}

	s.logger.Debug("broadcasting metrics", "size", len(data))
}
