// Copyright 2025 Takhin Data, Inc.

package handler

import (
	"bytes"
	"fmt"
	"io"

	"github.com/takhin-data/takhin/pkg/kafka/protocol"
	"github.com/takhin-data/takhin/pkg/logger"
)

// getSupportedSaslMechanisms returns all supported SASL mechanisms
func (h *Handler) getSupportedSaslMechanisms() []string {
	// For now, return PLAIN as the basic supported mechanism
	// Future implementations can add SCRAM-SHA-256, SCRAM-SHA-512, etc.
	return []string{
		"PLAIN",
		"SCRAM-SHA-256",
		"SCRAM-SHA-512",
	}
}

// handleSaslHandshake handles SASL handshake requests
func (h *Handler) handleSaslHandshake(reader io.Reader, header *protocol.RequestHeader) ([]byte, error) {
	req, err := protocol.DecodeSaslHandshakeRequest(reader, header.APIVersion)
	if err != nil {
		return nil, fmt.Errorf("decode request: %w", err)
	}

	logger.Info("sasl handshake request",
		"component", "kafka-handler",
		"mechanism", req.Mechanism,
	)

	// Get supported mechanisms
	supportedMechanisms := h.getSupportedSaslMechanisms()

	// Check if requested mechanism is supported
	mechanismSupported := false
	for _, mechanism := range supportedMechanisms {
		if mechanism == req.Mechanism {
			mechanismSupported = true
			break
		}
	}

	var errorCode protocol.ErrorCode
	if mechanismSupported {
		errorCode = protocol.None
		logger.Info("sasl mechanism supported",
			"component", "kafka-handler",
			"mechanism", req.Mechanism,
		)
	} else {
		errorCode = protocol.UnsupportedSaslMechanism
		logger.Warn("sasl mechanism not supported",
			"component", "kafka-handler",
			"mechanism", req.Mechanism,
			"supported_mechanisms", supportedMechanisms,
		)
	}

	resp := &protocol.SaslHandshakeResponse{
		ErrorCode:         errorCode,
		EnabledMechanisms: supportedMechanisms,
	}

	// Encode response
	var buf bytes.Buffer
	if err := protocol.WriteSaslHandshakeResponse(&buf, header, resp); err != nil {
		return nil, fmt.Errorf("write response: %w", err)
	}

	return buf.Bytes(), nil
}
