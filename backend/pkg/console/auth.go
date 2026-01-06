// Copyright 2025 Takhin Data, Inc.

package console

import (
	"net/http"
	"strings"

	"github.com/takhin-data/takhin/pkg/audit"
)

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Enabled     bool     // Enable authentication
	APIKeys     []string // Valid API keys
	AuditLogger *audit.Logger // Audit logger (optional)
}

// AuthMiddleware creates an authentication middleware
func AuthMiddleware(config AuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip authentication if disabled
			if !config.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			// Skip authentication for health check and swagger docs
			if strings.HasPrefix(r.URL.Path, "/swagger") || r.URL.Path == "/api/health" {
				next.ServeHTTP(w, r)
				return
			}

			// Extract API key from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				// Log failed auth attempt
				if config.AuditLogger != nil {
					config.AuditLogger.LogAuth("unknown", r.RemoteAddr, "failure", "", nil)
				}
				respondError(w, http.StatusUnauthorized, "missing authorization header")
				return
			}

			// Support both "Bearer <key>" and direct key formats
			apiKey := authHeader
			if strings.HasPrefix(authHeader, "Bearer ") {
				apiKey = strings.TrimPrefix(authHeader, "Bearer ")
			}

			// Validate API key
			if !isValidAPIKey(apiKey, config.APIKeys) {
				// Log failed auth attempt
				if config.AuditLogger != nil {
					config.AuditLogger.LogAuth("unknown", r.RemoteAddr, "failure", apiKey, nil)
				}
				respondError(w, http.StatusUnauthorized, "invalid API key")
				return
			}

			// Log successful auth
			if config.AuditLogger != nil {
				config.AuditLogger.LogAuth("api-key-user", r.RemoteAddr, "success", apiKey, nil)
			}

			// API key is valid, proceed to next handler
			next.ServeHTTP(w, r)
		})
	}
}

// isValidAPIKey checks if the provided key is valid
func isValidAPIKey(key string, validKeys []string) bool {
	for _, validKey := range validKeys {
		if key == validKey {
			return true
		}
	}
	return false
}

// respondError sends an error response
func respondError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write([]byte(`{"error":"` + message + `"}`))
}
