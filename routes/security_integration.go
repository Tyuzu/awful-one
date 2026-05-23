package routes

import (
	"naevis/config"
	"naevis/middleware"
	"net/http"
	"time"
)

// SecurityMiddlewareConfig holds all security middleware instances
type SecurityMiddlewareConfig struct {
	RateLimiters     *middleware.MultiLimiter
	CSRFManager      *middleware.CSRFManager
	ContentValidator *middleware.ContentTypeValidator
	SizeValidator    *middleware.RequestSizeValidator
}

// InitializeSecurityMiddleware creates and configures all security middleware
func InitializeSecurityMiddleware() *SecurityMiddlewareConfig {
	// Initialize rate limiters
	multiLimiter := middleware.NewMultiLimiter()

	// Configure per-endpoint rate limits from config
	for endpoint, limits := range config.RateLimitEndpoints {
		multiLimiter.AddLimiter(endpoint, limits.Requests, limits.Window)
	}

	// Initialize CSRF manager
	csrfManager := middleware.NewCSRFManager(24 * time.Hour)

	// Initialize content type validator
	contentValidator := middleware.NewContentTypeValidator(true)
	contentValidator.AddAllowedType("application/json")
	contentValidator.AddAllowedType("application/x-www-form-urlencoded")
	contentValidator.AddAllowedType("multipart/form-data")

	// Initialize size validator
	sizeValidator := middleware.NewRequestSizeValidator(config.SecurityConfig.MaxRequestBodySize)

	// Set specific size limits for file upload endpoints
	sizeValidator.SetRouteLimit("POST:/api/v1/upload", config.SecurityConfig.MaxUploadSize)
	sizeValidator.SetRouteLimit("POST:/api/v1/filedrop", config.SecurityConfig.MaxUploadSize)

	return &SecurityMiddlewareConfig{
		RateLimiters:     multiLimiter,
		CSRFManager:      csrfManager,
		ContentValidator: contentValidator,
		SizeValidator:    sizeValidator,
	}
}

// ApplyDefaultSecurityMiddleware applies security middleware to global routes
func ApplyDefaultSecurityMiddleware(handler http.Handler, secConfig *SecurityMiddlewareConfig) http.Handler {
	// Apply in reverse order (last added is executed first)

	// 1. Request size validation
	handler = middleware.RequestSizeMiddleware(secConfig.SizeValidator)(handler)

	// 2. Content-Type validation
	handler = middleware.ContentTypeMiddleware(secConfig.ContentValidator)(handler)

	// 3. CSRF protection (skip webhook paths)
	skipPaths := []string{
		"/api/v1/pay/webhook",
		"/api/v1/webhook",
		"/webhooks/",
	}
	handler = middleware.CSRFProtectionMiddleware(secConfig.CSRFManager, skipPaths)(handler)

	// 4. Include CSRF token in responses
	handler = middleware.IncludeCSRFTokenMiddleware(secConfig.CSRFManager)(handler)

	return handler
}

// ApplyRateLimitingMiddleware applies rate limiting to specific routes
func ApplyRateLimitingMiddleware(endpoint string, handler http.Handler, rateLimiter *middleware.MultiLimiter) http.Handler {
	return rateLimiter.GetMultiMiddleware(endpoint)(handler)
}

// INTEGRATION EXAMPLES FOR MAIN.GO:

/*
Example 1: Apply global security middleware to all routes

In your main.go or route initialization:

	secConfig := routes.InitializeSecurityMiddleware()

	// Initialize indexes
	ctx := context.Background()
	if err := app.DB.InitializeIndexes(ctx); err != nil {
		log.Fatalf("Failed to initialize indexes: %v", err)
	}

	// Apply security middleware to main router
	mainRouter := httprouter.New()
	// ... add your routes to mainRouter ...

	globalHandler := routes.ApplyDefaultSecurityMiddleware(
		mainRouter,
		secConfig,
	)

	// Start server with global security middleware
	http.ListenAndServe(":8080", globalHandler)

---

Example 2: Apply rate limiting to specific endpoints

	secConfig := routes.InitializeSecurityMiddleware()
	mainRouter := httprouter.New()

	// Ticket purchase endpoint with rate limiting
	ticketBuyHandler := http.HandleFunc(func(w http.ResponseWriter, r *http.Request) {
		// ... ticket buy logic ...
	})

	rateLimitedHandler := routes.ApplyRateLimitingMiddleware(
		"POST:/api/v1/ticket/event/:eventid/:ticketid/buy",
		ticketBuyHandler,
		secConfig.RateLimiters,
	)

	mainRouter.Handler("POST", "/api/v1/ticket/event/:eventid/:ticketid/buy", rateLimitedHandler)

---

Example 3: Initialize payment webhooks with signature verification

In your pay module initialization:

	paymentService.InitializeWebhooks(app, secConfig.CSRFManager)
	// Then register webhook receiver route
	mainRouter.POST("/api/v1/pay/webhook", paymentService.HandlePaymentWebhook)

---

Example 4: Database index initialization

In your startup code:

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := app.DB.InitializeIndexes(ctx); err != nil {
		log.Fatalf("Failed to create indexes: %v", err)
	}

	if err := app.DB.VerifyIndexes(ctx); err != nil {
		log.Fatalf("Failed to verify indexes: %v", err)
	}

---

Example 5: Route registration with multiple middleware

	router := httprouter.New()

	// Cart endpoints with validation and rate limiting
	router.POST("/api/v1/cart",
		secConfig.RateLimiters.GetMultiMiddleware("POST:/api/v1/cart")(
			http.HandlerFunc(cartHandler.AddToCart),
		).ServeHTTP,
	)

---

Example 6: CSRF token generation endpoint for SPA

	router.GET("/api/v1/csrf-token", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionID := getSessionID(r)
		if sessionID == "" {
			http.Error(w, "no session", http.StatusForbidden)
			return
		}

		token, err := secConfig.CSRFManager.GenerateToken(sessionID)
		if err != nil {
			http.Error(w, "failed to generate token", http.StatusInternalServerError)
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, map[string]string{
			"csrf_token": token,
		})
	}))
*/
