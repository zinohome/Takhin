// Copyright 2025 Takhin Data, Inc.

package console

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestAuthMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		authConfig     AuthConfig
		requestPath    string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "authentication disabled - should pass",
			authConfig:     AuthConfig{Enabled: false},
			requestPath:    "/api/topics",
			authHeader:     "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "valid API key - should pass",
			authConfig:     AuthConfig{Enabled: true, APIKeys: []string{"test-key-123"}},
			requestPath:    "/api/topics",
			authHeader:     "test-key-123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "valid API key with Bearer prefix - should pass",
			authConfig:     AuthConfig{Enabled: true, APIKeys: []string{"test-key-456"}},
			requestPath:    "/api/topics",
			authHeader:     "Bearer test-key-456",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid API key - should fail",
			authConfig:     AuthConfig{Enabled: true, APIKeys: []string{"valid-key"}},
			requestPath:    "/api/topics",
			authHeader:     "invalid-key",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "missing authorization header - should fail",
			authConfig:     AuthConfig{Enabled: true, APIKeys: []string{"test-key"}},
			requestPath:    "/api/topics",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "health check path - should skip auth",
			authConfig:     AuthConfig{Enabled: true, APIKeys: []string{"test-key"}},
			requestPath:    "/api/health",
			authHeader:     "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "swagger path - should skip auth",
			authConfig:     AuthConfig{Enabled: true, APIKeys: []string{"test-key"}},
			requestPath:    "/swagger/index.html",
			authHeader:     "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "multiple valid keys - first key should pass",
			authConfig:     AuthConfig{Enabled: true, APIKeys: []string{"key1", "key2", "key3"}},
			requestPath:    "/api/topics",
			authHeader:     "key1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "multiple valid keys - last key should pass",
			authConfig:     AuthConfig{Enabled: true, APIKeys: []string{"key1", "key2", "key3"}},
			requestPath:    "/api/topics",
			authHeader:     "Bearer key3",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test router with auth middleware
			router := chi.NewRouter()
			router.Use(AuthMiddleware(tt.authConfig))

			// Add a test handler
			router.Get("/api/*", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			router.Get("/swagger/*", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			// Create test request
			req := httptest.NewRequest("GET", tt.requestPath, nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()

			// Serve request
			router.ServeHTTP(w, req)

			// Assert status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			// If unauthorized, check error response
			if tt.expectedStatus == http.StatusUnauthorized {
				assert.Contains(t, w.Body.String(), "error")
			}
		})
	}
}

func TestIsValidAPIKey(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		validKeys []string
		expected  bool
	}{
		{
			name:      "key exists in list",
			key:       "test-key",
			validKeys: []string{"test-key", "another-key"},
			expected:  true,
		},
		{
			name:      "key does not exist",
			key:       "invalid-key",
			validKeys: []string{"test-key", "another-key"},
			expected:  false,
		},
		{
			name:      "empty valid keys list",
			key:       "test-key",
			validKeys: []string{},
			expected:  false,
		},
		{
			name:      "empty key",
			key:       "",
			validKeys: []string{"test-key"},
			expected:  false,
		},
		{
			name:      "case sensitive match",
			key:       "Test-Key",
			validKeys: []string{"test-key"},
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidAPIKey(tt.key, tt.validKeys)
			assert.Equal(t, tt.expected, result)
		})
	}
}
