package routes

import (
	"naevis/infra"
	"naevis/middleware"

	"github.com/julienschmidt/httprouter"
)

func RoutesWrapper(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	AddActivityRoutes(router, app, rateLimiter)
	AddAnalyticsRoutes(router, app, rateLimiter)
	AddAuthRoutes(router, app, rateLimiter)
	AddHomeRoutes(router, app, rateLimiter)
	AddProfileRoutes(router, app, rateLimiter)
	AddUtilityRoutes(router, app, rateLimiter)
}
