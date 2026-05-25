package config

import "time"

// SecurityConfig defines application-wide security settings
var SecurityConfig = struct {
	// Request validation
	MaxRequestBodySize int64
	MaxFormBodySize    int64
	RequestTimeout     time.Duration

	// Rate limiting
	RateLimitRequests int
	RateLimitWindow   time.Duration
	RateLimitEnabled  bool

	// Payment settings
	MaxTopupAmount    float64
	MaxCustomAmount   float64
	RefundMinValidity time.Duration // Minimum time before refund can be requested

	// Audit logging
	AuditLoggingEnabled bool
	LogSensitiveData    bool

	// Session
	SessionTimeout time.Duration

	// File uploads
	MaxUploadSize int64

	// Passwords
	PasswordMinLength      int
	PasswordRequireSpecial bool
}{
	// Request validation
	MaxRequestBodySize: 10 << 20, // 10 MB
	MaxFormBodySize:    50 << 20, // 50 MB
	RequestTimeout:     30 * time.Second,

	// Rate limiting
	RateLimitRequests: 100, // Requests per window
	RateLimitWindow:   1 * time.Minute,
	RateLimitEnabled:  true,

	// Payment settings
	MaxTopupAmount:    100000,           // ₹1,00,000
	MaxCustomAmount:   1000000,          // ₹10,00,000 for donations
	RefundMinValidity: 15 * time.Minute, // Can't refund within 15 mins of purchase

	// Audit logging
	AuditLoggingEnabled: true,
	LogSensitiveData:    false,

	// Session
	SessionTimeout: 24 * time.Hour,

	// File uploads
	MaxUploadSize: 100 << 20, // 100 MB

	// Passwords
	PasswordMinLength:      8,
	PasswordRequireSpecial: true,
}

// TransactionConfig for database transactions
var TransactionConfig = struct {
	MaxRetries int
	RetryDelay time.Duration
	Timeout    time.Duration
}{
	MaxRetries: 3,
	RetryDelay: 100 * time.Millisecond,
	Timeout:    30 * time.Second,
}

// PaymentMethodLimits restricts payment amounts by method
var PaymentMethodLimits = map[string]float64{
	"wallet":      SecurityConfig.MaxTopupAmount,
	"upi":         50000,  // ₹50,000
	"card":        100000, // ₹1,00,000
	"net_banking": 500000, // ₹5,00,000
	"donation":    SecurityConfig.MaxCustomAmount,
}

// RateLimitEndpoints defines per-endpoint rate limits
var RateLimitEndpoints = map[string]struct {
	Requests int
	Window   time.Duration
}{
	"POST:/api/v1/ticket/event/:eventid/:ticketid/buy": {
		Requests: 10,
		Window:   1 * time.Minute,
	},
	"POST:/api/v1/merch/:eventid/:merchid/confirm-purchase": {
		Requests: 10,
		Window:   1 * time.Minute,
	},
	"POST:/api/v1/pay/topup": {
		Requests: 5,
		Window:   1 * time.Minute,
	},
	"POST:/api/v1/order": {
		Requests: 5,
		Window:   1 * time.Minute,
	},
	"POST:/farms/:id/crops/buy": {
		Requests: 5,
		Window:   1 * time.Minute,
	},
}
