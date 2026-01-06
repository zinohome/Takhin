// Copyright 2025 Takhin Data, Inc.

package handler

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/takhin-data/takhin/pkg/kafka/protocol"
	"github.com/takhin-data/takhin/pkg/logger"
	"github.com/takhin-data/takhin/pkg/sasl"
)

// handleSaslAuthenticate handles SASL authenticate requests
func (h *Handler) handleSaslAuthenticate(reader io.Reader, header *protocol.RequestHeader) ([]byte, error) {
	req, err := protocol.DecodeSaslAuthenticateRequest(reader, header.APIVersion)
	if err != nil {
		return nil, fmt.Errorf("decode request: %w", err)
	}

	logger.Info("sasl authenticate request",
		"component", "kafka-handler",
		"auth_bytes_len", len(req.AuthBytes),
	)

	// Get SASL manager
	if h.saslManager == nil {
		errMsg := "SASL not configured"
		logger.Error("sasl authentication failed", "component", "kafka-handler", "error", errMsg)
		resp := &protocol.SaslAuthenticateResponse{
			ErrorCode:         protocol.SaslAuthenticationFailed,
			ErrorMessage:      &errMsg,
			AuthBytes:         []byte{},
			SessionLifetimeMs: 0,
		}
		return encodeSaslAuthResponse(header, resp)
	}

	// Get the negotiated mechanism from connection state
	// For now, we default to PLAIN if not set
	mechanism := h.currentSaslMechanism
	if mechanism == "" {
		mechanism = string(sasl.PLAIN)
	}

	// Authenticate based on mechanism
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session, err := h.saslManager.Authenticate(ctx, sasl.Mechanism(mechanism), req.AuthBytes)
	
	var errorCode protocol.ErrorCode
	var errorMessage *string
	var sessionLifetime int64

	if err != nil {
		errorCode = protocol.SaslAuthenticationFailed
		errMsg := fmt.Sprintf("authentication failed: %v", err)
		errorMessage = &errMsg
		logger.Warn("sasl authentication failed",
			"component", "kafka-handler",
			"mechanism", mechanism,
			"error", err,
		)
	} else {
		errorCode = protocol.None
		sessionLifetime = int64(session.ExpiryTime.Sub(session.AuthTime).Milliseconds())
		logger.Info("sasl authentication successful",
			"component", "kafka-handler",
			"mechanism", mechanism,
			"principal", session.Principal,
		)
		
		// Store session info for authorization
		h.currentPrincipal = session.Principal
	}

	resp := &protocol.SaslAuthenticateResponse{
		ErrorCode:         errorCode,
		ErrorMessage:      errorMessage,
		AuthBytes:         []byte{},
		SessionLifetimeMs: sessionLifetime,
	}

	return encodeSaslAuthResponse(header, resp)
}

// encodeSaslAuthResponse encodes the SASL authenticate response
func encodeSaslAuthResponse(header *protocol.RequestHeader, resp *protocol.SaslAuthenticateResponse) ([]byte, error) {
	var buf bytes.Buffer
	if err := protocol.WriteSaslAuthenticateResponse(&buf, header, resp); err != nil {
		return nil, fmt.Errorf("write response: %w", err)
	}
	return buf.Bytes(), nil
}
