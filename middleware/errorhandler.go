package middleware

import (
	"log"
	"net/http"

	"naevis/utils"
)

// SafeError represents a sanitized error for client responses
type SafeError struct {
	Code       string
	Message    string
	StatusCode int
	LogMessage string // Detailed log message for server
}

// Error implements the error interface
func (se *SafeError) Error() string {
	if se.LogMessage != "" {
		return se.LogMessage
	}
	return se.Message
}

// SafeErrorHandler wraps handlers to provide consistent error handling
func SafeErrorHandler(handler func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := handler(w, r); err != nil {
			// Log detailed error server-side
			log.Printf("[%s] %s %s - Error: %v", r.RemoteAddr, r.Method, r.RequestURI, err)

			// Return generic error to client
			utils.RespondWithJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "An error occurred processing your request",
			})
		}
	}
}

// LogErrorSafely logs full error server-side but returns generic message to client
func LogErrorSafely(w http.ResponseWriter, r *http.Request, statusCode int, logMsg string, clientMsg string) {
	log.Printf("[%s] %s - %s (from %s)", r.RemoteAddr, r.RequestURI, logMsg, r.RemoteAddr)

	if clientMsg == "" {
		clientMsg = "An error occurred. Please try again later."
	}

	utils.RespondWithJSON(w, statusCode, map[string]string{
		"error": clientMsg,
	})
}

// ValidationError represents a validation issue
type ValidationError struct {
	Field   string
	Message string
}

// ValidateRequest performs common validation checks
func ValidateRequest(r *http.Request, maxBodySize int64) error {
	// Check Content-Type for POST/PUT/PATCH
	if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
		ct := r.Header.Get("Content-Type")
		if ct == "" || (ct != "application/json" && ct != "application/x-www-form-urlencoded") {
			return &SafeError{
				Code:       "invalid_content_type",
				Message:    "Invalid Content-Type",
				StatusCode: http.StatusUnsupportedMediaType,
				LogMessage: "Invalid Content-Type: " + ct,
			}
		}
	}

	return nil
}
