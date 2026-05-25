package routes

import (
	"naevis/activity"
	"naevis/analytics"
	"naevis/auth"
	"naevis/home"
	"naevis/infra"
	"naevis/middleware"
	"naevis/profile"
	"naevis/userdata"
	"naevis/utils"

	"github.com/julienschmidt/httprouter"
)

// func AddStaticRoutes(router *httprouter.Router) {
// 	router.ServeFiles("/static/uploads/*filepath", http.Dir("static/uploads"))
// }

func AddActivityRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	// If activity log/feed is user-specific, keep auth
	authmidware := middleware.Authenticate(app)
	router.POST("/api/v1/activity/log", rateLimiter.Limit(authmidware(activity.LogActivities(app))))
	router.GET("/api/v1/activity/get", authmidware(activity.GetActivityFeed(app)))

	// Public analytics/telemetry ingestion
	router.POST("/api/v1/scitylana/event", activity.HandleAnalyticsEvent(app))
}

func AddHomeRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	// router.GET("/api/v1/home/:apiRoute", middleware.OptionalAuth(home.GetHomeContent)) // Public/optional
	router.GET("/api/v1/homecards", middleware.OptionalAuth(home.HomeCardsHandler(app))) // Public/optional
}

func AddAuthRoutes(router *httprouter.Router, app *infra.Deps, limiter *middleware.RateLimiter) {
	authmid := middleware.Authenticate(app)
	router.POST("/api/v1/auth/register", limiter.Limit(auth.Register(app)))
	router.POST("/api/v1/auth/login", limiter.Limit(auth.Login(app)))

	// Refresh should NOT use aggressive limiter
	router.POST("/api/v1/auth/refresh", auth.RefreshToken(app))

	// Logout does NOT need Authenticate middleware
	router.POST("/api/v1/auth/logout", auth.LogoutUser(app))
	router.POST("/api/v1/auth/logout-all", authmid(auth.LogoutAllSessions(app)))

	router.POST("/api/v1/auth/verify-otp", limiter.Limit(auth.VerifyOTPHandler(app)))
	router.POST("/api/v1/auth/request-otp", limiter.Limit(auth.RequestOTPHandler(app)))
}

func AddProfileRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	authmidware := middleware.Authenticate(app)
	// Own profile
	router.GET("/api/v1/profile/profile", rateLimiter.Limit(authmidware(profile.GetProfile(app))))
	router.PUT("/api/v1/profile/edit", rateLimiter.Limit(authmidware(profile.EditProfile(app))))
	router.DELETE("/api/v1/profile/delete", rateLimiter.Limit(authmidware(profile.DeleteProfile(app))))

	// Public profile viewing
	router.GET("/api/v1/user/:username", rateLimiter.Limit(profile.GetUserProfile(app)))

	// Other user data (requires auth to see private info)
	router.GET("/api/v1/user/:username/data", rateLimiter.Limit(authmidware(userdata.GetUserProfileData(app))))
	router.GET("/api/v1/user/:username/udata", rateLimiter.Limit(authmidware(userdata.GetOtherUserProfileData(app))))

}

func AddUtilityRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	authmidware := middleware.Authenticate(app)
	router.GET("/api/v1/csrf", rateLimiter.Limit(authmidware(utils.CSRF)))
}

func AddAnalyticsRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	// Example: /api/v1/antics/events/123 or /api/v1/analytics/places/456
	router.GET("/api/v1/antics/:entityType/:entityId", rateLimiter.Limit(analytics.GetEntityAnalytics))
}
