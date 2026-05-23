package middleware

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ContentTypeValidator validates request content types
type ContentTypeValidator struct {
	allowedTypes map[string]bool
	strictMode   bool // If true, reject requests without Content-Type header
}

// NewContentTypeValidator creates a new content type validator
func NewContentTypeValidator(strictMode bool) *ContentTypeValidator {
	return &ContentTypeValidator{
		allowedTypes: map[string]bool{
			"application/json":                  true,
			"application/x-www-form-urlencoded": true,
			"multipart/form-data":               true,
			"text/plain":                        true,
		},
		strictMode: strictMode,
	}
}

// AddAllowedType adds a content type to the allowed list
func (ctv *ContentTypeValidator) AddAllowedType(contentType string) {
	ctv.allowedTypes[contentType] = true
}

// ValidateContentType checks if a content type is allowed
func (ctv *ContentTypeValidator) ValidateContentType(contentType string) bool {
	if contentType == "" {
		return !ctv.strictMode
	}

	// Extract media type without charset
	mediaType := strings.Split(contentType, ";")[0]
	mediaType = strings.TrimSpace(mediaType)

	return ctv.allowedTypes[mediaType]
}

// ContentTypeMiddleware enforces content type validation on state-changing requests
func ContentTypeMiddleware(validator *ContentTypeValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only validate state-changing methods
			if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
				contentType := r.Header.Get("Content-Type")

				if !validator.ValidateContentType(contentType) {
					http.Error(w, fmt.Sprintf("unsupported content type: %s", contentType), http.StatusUnsupportedMediaType)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequestSizeValidator validates request body size
type RequestSizeValidator struct {
	defaultMaxSize int64
	perRouteLimit  map[string]int64
}

// NewRequestSizeValidator creates a new size validator
func NewRequestSizeValidator(defaultMaxSize int64) *RequestSizeValidator {
	return &RequestSizeValidator{
		defaultMaxSize: defaultMaxSize,
		perRouteLimit:  make(map[string]int64),
	}
}

// SetRouteLimit sets a specific size limit for a route
func (rsv *RequestSizeValidator) SetRouteLimit(route string, maxSize int64) {
	rsv.perRouteLimit[route] = maxSize
}

// GetMaxSize returns the maximum size for a request
func (rsv *RequestSizeValidator) GetMaxSize(path string) int64 {
	// Check if there's a specific limit for this path
	if limit, ok := rsv.perRouteLimit[path]; ok {
		return limit
	}
	return rsv.defaultMaxSize
}

// RequestSizeMiddleware enforces request body size limits
func RequestSizeMiddleware(validator *RequestSizeValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			maxSize := validator.GetMaxSize(r.URL.Path)

			// Limit request body size
			r.Body = http.MaxBytesReader(w, r.Body, maxSize)

			// Add a wrapper to catch size limit errors
			next.ServeHTTP(w, r)
		})
	}
}

// LimitedReader wraps io.Reader to enforce size limits and return detailed errors
type LimitedReader struct {
	reader    io.Reader
	maxSize   int64
	bytesRead int64
}

// NewLimitedReader creates a limited reader
func NewLimitedReader(reader io.Reader, maxSize int64) *LimitedReader {
	return &LimitedReader{
		reader:  reader,
		maxSize: maxSize,
	}
}

// Read implements io.Reader with size enforcement
func (lr *LimitedReader) Read(p []byte) (int, error) {
	if lr.bytesRead >= lr.maxSize {
		return 0, fmt.Errorf("request body exceeds maximum size of %d bytes", lr.maxSize)
	}

	// Limit the read to not exceed maxSize
	remaining := lr.maxSize - lr.bytesRead
	if int64(len(p)) > remaining {
		p = p[:remaining]
	}

	n, err := lr.reader.Read(p)
	lr.bytesRead += int64(n)

	if lr.bytesRead >= lr.maxSize && err == nil {
		err = io.EOF
	}

	return n, err
}

// RequestValidationOptions contains all validation options
type RequestValidationOptions struct {
	ValidateContentType bool
	ValidateSize        bool
	ValidateCSRF        bool
	MaxBodySize         int64
	AllowedContentTypes []string
}

// DefaultValidationOptions returns sensible validation options
var DefaultValidationOptions = RequestValidationOptions{
	ValidateContentType: true,
	ValidateSize:        true,
	ValidateCSRF:        true,
	MaxBodySize:         10 << 20, // 10 MB
	AllowedContentTypes: []string{
		"application/json",
		"application/x-www-form-urlencoded",
		"multipart/form-data",
	},
}

// ComprehensiveValidationMiddleware combines multiple validation checks
func ComprehensiveValidationMiddleware(options RequestValidationOptions) func(http.Handler) http.Handler {
	contentTypeValidator := NewContentTypeValidator(options.ValidateContentType)
	for _, ct := range options.AllowedContentTypes {
		contentTypeValidator.AddAllowedType(ct)
	}

	sizeValidator := NewRequestSizeValidator(options.MaxBodySize)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Content-Type validation
			if options.ValidateContentType {
				if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
					contentType := r.Header.Get("Content-Type")
					if !contentTypeValidator.ValidateContentType(contentType) {
						http.Error(w, fmt.Sprintf("unsupported content type: %s", contentType), http.StatusUnsupportedMediaType)
						return
					}
				}
			}

			// Size validation
			if options.ValidateSize {
				maxSize := sizeValidator.GetMaxSize(r.URL.Path)
				r.Body = http.MaxBytesReader(w, r.Body, maxSize)
			}

			next.ServeHTTP(w, r)
		})
	}
}
