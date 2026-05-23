package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strings"
)

// ValidateContentType checks if the request has JSON content type
func ValidateContentType(contentTypes ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only validate state-changing requests
			if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
				contentType := r.Header.Get("Content-Type")

				allowed := len(contentTypes) == 0 // if empty, allow all
				for _, ct := range contentTypes {
					if strings.Contains(contentType, ct) {
						allowed = true
						break
					}
				}

				if !allowed && len(contentTypes) > 0 {
					log.Printf("Invalid Content-Type: %s for %s %s", contentType, r.Method, r.RequestURI)
					http.Error(w, "Invalid Content-Type", http.StatusUnsupportedMediaType)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ValidateBodySize limits request body size to prevent DoS
func ValidateBodySize(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}

// ValidateEmail checks if email is valid format
func ValidateEmail(email string) bool {
	// Simple email validation
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, email)
	return matched
}

// ValidatePhone checks if phone is valid format (basic international)
func ValidatePhone(phone string) bool {
	// Remove common formatting characters
	clean := strings.NewReplacer(
		" ", "",
		"-", "",
		"(", "",
		")", "",
		"+", "",
	).Replace(phone)

	// Check if it's numeric and between 10-15 digits
	if len(clean) < 10 || len(clean) > 15 {
		return false
	}

	pattern := `^\d{10,15}$`
	matched, _ := regexp.MatchString(pattern, clean)
	return matched
}

// DecodeJSON safely decodes JSON from request body with size limit
func DecodeJSON(w http.ResponseWriter, r *http.Request, v interface{}, maxBytes int64) error {
	// Limit request body size
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		// Log detailed error server-side
		log.Printf("JSON decode error from %s: %v", r.RemoteAddr, err)
		// Return generic error to client
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return err
	}

	return nil
}
