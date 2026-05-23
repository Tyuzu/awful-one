package routes

import (
	"naevis/dropify/droping"
	"naevis/dropify/mediaproxy"
	"naevis/infra"
	"naevis/middleware"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func AddStaticRoutes(router *httprouter.Router) {
	// Serve static uploaded files
	router.ServeFiles("/static/uploads/*filepath", http.Dir("static/uploads"))

	// Proxy handler for external media
	router.GET("/static/proxy/*url", mediaproxy.ProxyHandler)
}

func AddFiledropRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	// Apply rate limiter middleware to file upload endpoint
	uploadHandler := func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		// Apply rate limiting
		if !rateLimiter.Allow(r) {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		// Call handler with proper dependency injection
		droping.FiledropHandler(app, w, r, ps)
	}

	// Register routes
	router.POST("/api/v1/filedrop", uploadHandler)
	router.OPTIONS("/api/v1/filedrop", droping.OptionsHandler)
}
