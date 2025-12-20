// Copyright 2025 Takhin Data, Inc.

package handler

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"github.com/takhin-data/takhin/pkg/kafka/protocol"
	"github.com/takhin-data/takhin/pkg/logger"
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

	// Parse PLAIN SASL authentication
	// Format: [authzid] \0 username \0 password
	authStr := string(req.AuthBytes)
	parts := strings.Split(authStr, "\x00")

	var username, password string
	if len(parts) == 3 {
		// Standard format: [authzid] \0 username \0 password
		username = parts[1]
		password = parts[2]
	} else if len(parts) == 2 {
		// Alternative format: username \0 password
		username = parts[0]
		password = parts[1]
	} else {
		errMsg := "invalid SASL PLAIN credentials format"
		logger.Warn("sasl authentication failed",
			"component", "kafka-handler",
			"error", errMsg,
		)
		resp := &protocol.SaslAuthenticateResponse{
			ErrorCode:         protocol.SaslAuthenticationFailed,
			ErrorMessage:      &errMsg,
			AuthBytes:         []byte{},
			SessionLifetimeMs: 0,
		}
		return encodeSaslAuthResponse(header, resp)
	}

	// Basic validation (for demo purposes)
	// In production, this should validate against a user store
	authenticated := h.validateCredentials(username, password)

	var errorCode protocol.ErrorCode
	var errorMessage *string
	var sessionLifetime int64

	if authenticated {
		errorCode = protocol.None
		sessionLifetime = 3600000 // 1 hour in milliseconds
		logger.Info("sasl authentication successful",
			"component", "kafka-handler",
			"username", username,
		)
	} else {
		errorCode = protocol.SaslAuthenticationFailed
		errMsg := "authentication failed"
		errorMessage = &errMsg
		logger.Warn("sasl authentication failed",
			"component", "kafka-handler",
			"username", username,
		)
	}

	resp := &protocol.SaslAuthenticateResponse{
		ErrorCode:         errorCode,
		ErrorMessage:      errorMessage,
		AuthBytes:         []byte{},
		SessionLifetimeMs: sessionLifetime,
	}

	return encodeSaslAuthResponse(header, resp)
}

// validateCredentials validates username and password
// This is a placeholder implementation. In production, this should:
// - Check against a secure user store (database, LDAP, etc.)
// - Use proper password hashing (bcrypt, scrypt, etc.)
// - Implement rate limiting and account lockout
func (h *Handler) validateCredentials(username, password string) bool {
	// For demo/testing purposes, accept any non-empty credentials
	// In production, replace this with actual authentication logic
	if username == "" || password == "" {
		return false
	}

	// Example: accept "admin" with any password for testing
	// Remove this in production!
	if username == "admin" {
		return true
	}

	// For now, accept any valid-looking credentials
	return len(username) > 0 && len(password) > 0
}

// encodeSaslAuthResponse encodes the SASL authenticate response
func encodeSaslAuthResponse(header *protocol.RequestHeader, resp *protocol.SaslAuthenticateResponse) ([]byte, error) {
	var buf bytes.Buffer
	if err := protocol.WriteSaslAuthenticateResponse(&buf, header, resp); err != nil {
		return nil, fmt.Errorf("write response: %w", err)
	}
	return buf.Bytes(), nil
}

// Helper function to encode credentials for testing
func EncodePlainSaslCredentials(username, password string) string {
	// Format: \0username\0password
	credentials := fmt.Sprintf("\x00%s\x00%s", username, password)
	return base64.StdEncoding.EncodeToString([]byte(credentials))
}
