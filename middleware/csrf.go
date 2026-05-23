package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// CSRFManager manages CSRF token generation and validation
type CSRFManager struct {
	mu         sync.RWMutex
	tokens     map[string]*CSRFToken
	expiration time.Duration
	cleanup    *time.Ticker
}

// CSRFToken represents a single CSRF token
type CSRFToken struct {
	Token     string
	SessionID string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// NewCSRFManager creates a new CSRF token manager
func NewCSRFManager(expiration time.Duration) *CSRFManager {
	manager := &CSRFManager{
		tokens:     make(map[string]*CSRFToken),
		expiration: expiration,
		cleanup:    time.NewTicker(5 * time.Minute),
	}

	// Cleanup expired tokens periodically
	go func() {
		for range manager.cleanup.C {
			manager.cleanupExpiredTokens()
		}
	}()

	return manager
}

// GenerateToken creates a new CSRF token
func (cm *CSRFManager) GenerateToken(sessionID string) (string, error) {
	token := make([]byte, 32)
	_, err := rand.Read(token)
	if err != nil {
		return "", fmt.Errorf("failed to generate CSRF token: %w", err)
	}

	tokenStr := base64.URLEncoding.EncodeToString(token)
	now := time.Now()
	expiresAt := now.Add(cm.expiration)

	csrfToken := &CSRFToken{
		Token:     tokenStr,
		SessionID: sessionID,
		CreatedAt: now,
		ExpiresAt: expiresAt,
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.tokens[tokenStr] = csrfToken

	return tokenStr, nil
}

// ValidateToken checks if a CSRF token is valid
func (cm *CSRFManager) ValidateToken(token, sessionID string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	csrfToken, exists := cm.tokens[token]
	if !exists {
		return false
	}

	// Check if token has expired
	if time.Now().After(csrfToken.ExpiresAt) {
		return false
	}

	// Check if session ID matches
	if csrfToken.SessionID != sessionID {
		return false
	}

	return true
}

// RevokeToken removes a token (use after successful validation)
func (cm *CSRFManager) RevokeToken(token string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.tokens, token)
}

// cleanupExpiredTokens removes expired tokens
func (cm *CSRFManager) cleanupExpiredTokens() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	now := time.Now()
	for token, csrfToken := range cm.tokens {
		if now.After(csrfToken.ExpiresAt) {
			delete(cm.tokens, token)
		}
	}
}

// Stop stops the cleanup ticker
func (cm *CSRFManager) Stop() {
	cm.cleanup.Stop()
}

// CSRFProtectionMiddleware protects against CSRF attacks for state-changing operations
// Accepts CSRF token via X-CSRF-Token header or csrf_token form parameter
func CSRFProtectionMiddleware(csrfManager *CSRFManager, skipPaths []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only protect state-changing methods
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			// Check if path should be skipped (e.g., webhooks)
			for _, path := range skipPaths {
				if strings.HasPrefix(r.URL.Path, path) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Get CSRF token from header or form parameter
			token := r.Header.Get("X-CSRF-Token")
			if token == "" {
				token = r.FormValue("csrf_token")
			}

			if token == "" {
				http.Error(w, "CSRF token missing", http.StatusForbidden)
				return
			}

			// Get session ID from context or cookie
			sessionID := getSessionID(r)
			if sessionID == "" {
				http.Error(w, "session not found", http.StatusForbidden)
				return
			}

			// Validate token
			if !csrfManager.ValidateToken(token, sessionID) {
				http.Error(w, "CSRF token invalid or expired", http.StatusForbidden)
				return
			}

			// Revoke token after successful validation (one-time use or reuse)
			// Uncomment the line below if you want one-time-use tokens
			// csrfManager.RevokeToken(token)

			next.ServeHTTP(w, r)
		})
	}
}

// IncludeCSRFTokenMiddleware adds CSRF token to response for GET requests
// This allows clients to get a new token for subsequent requests
func IncludeCSRFTokenMiddleware(csrfManager *CSRFManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Generate token for GET requests
			if r.Method == http.MethodGet || r.Method == http.MethodHead {
				sessionID := getSessionID(r)
				if sessionID != "" {
					if token, err := csrfManager.GenerateToken(sessionID); err == nil {
						w.Header().Set("X-CSRF-Token", token)
					}
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// getSessionID retrieves session ID from request
// Can be from cookie, context, or authorization header
func getSessionID(r *http.Request) string {
	// Try to get from context first
	if sessionID, ok := r.Context().Value("sessionID").(string); ok {
		return sessionID
	}

	// Try to get from cookie
	if cookie, err := r.Cookie("session"); err == nil {
		return cookie.Value
	}

	// Try to get from header (useful for API clients)
	if auth := r.Header.Get("Authorization"); auth != "" {
		// Extract from Bearer token or other header format
		parts := strings.Split(auth, " ")
		if len(parts) >= 2 {
			return parts[1]
		}
	}

	return ""
}
