package routes

import (
	"naevis/activity"
	"naevis/ads"
	"naevis/analytics"
	"naevis/artists"
	"naevis/auth"
	"naevis/baito"
	"naevis/beats"
	"naevis/booking"
	"naevis/cart"
	"naevis/comments"
	"naevis/discord"
	"naevis/events"
	"naevis/fanmade"
	"naevis/farms"
	"naevis/feed"
	"naevis/hashtags"
	"naevis/home"
	"naevis/infra"
	"naevis/itinerary"
	"naevis/jobs"
	"naevis/maps"
	"naevis/mechat"
	"naevis/media"
	"naevis/menu"
	"naevis/merch"
	"naevis/metadata"
	"naevis/middleware"
	"naevis/moderator"
	"naevis/musicon"
	"naevis/newchat"
	"naevis/notices"
	"naevis/notifications"
	"naevis/places"
	"naevis/posts"
	"naevis/products"
	"naevis/profile"
	"naevis/recipes"
	"naevis/reports"
	"naevis/reviews"
	"naevis/search"
	"naevis/settings"
	"naevis/stripe"
	"naevis/suggestions"
	"naevis/tickets"
	"naevis/userdata"
	"naevis/utils"
	"naevis/vendors"
	"naevis/vlive"

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

// func AddAdminRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {

// 	// Moderator-only endpoints
// 	router.GET("/api/v1/moderator/reports",
// 		authmidware(
// 			middleware.RequireRoles("moderator")(
// 				moderator.GetReports,
// 			),
// 		),
// 	)

// 	router.POST("/api/v1/moderator/apply",
// 		rateLimiter.Limit(
// 			authmidware(moderator.ApplyModerator),
// 		),
// 	)
// }

func AddAdminRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	authmidware := middleware.Authenticate(app)

	// Moderator-only endpoints
	router.GET("/api/v1/moderator/reports",
		authmidware(
			middleware.RequireRoles("moderator")(
				moderator.GetReports(app),
			),
		),
	)

	router.POST("/api/v1/moderator/apply",
		rateLimiter.Limit(
			authmidware(moderator.ApplyModerator(app)),
		),
	)

	// Moderator-only: soft-delete entities
	router.PUT("/api/v1/moderator/delete/:type/:id",
		authmidware(
			middleware.RequireRoles("moderator")(
				reports.SoftDeleteEntity(app),
			),
		),
	)

	// Moderator-only: appeals management (list + update)
	router.GET("/api/v1/moderator/appeals",
		authmidware(
			middleware.RequireRoles("moderator")(
				reports.GetAppeals(app),
			),
		),
	)
	router.PUT("/api/v1/moderator/appeals/:id",
		authmidware(
			middleware.RequireRoles("moderator")(
				reports.UpdateAppeal(app),
			),
		),
	)
}

func AddJobRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	authmidware := middleware.Authenticate(app)
	router.GET("/api/v1/jobs/:entitytype/:entityid", rateLimiter.Limit(jobs.GetJobsRelatedTOEntity(app)))
	router.POST("/api/v1/jobs/:entitytype/:entityid", rateLimiter.Limit(authmidware(jobs.CreateBaitoForEntity(app))))
}

func AddBaitoRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	authmidware := middleware.Authenticate(app)
	// Create / update jobs → require auth
	router.POST("/api/v1/baitos/baito", rateLimiter.Limit(authmidware(baito.CreateBaito(app))))
	router.PUT("/api/v1/baitos/baito/:baitoid", rateLimiter.Limit(authmidware(baito.UpdateBaito(app))))
	router.DELETE("/api/v1/baitos/baito/:baitoid", rateLimiter.Limit(authmidware(baito.DeleteBaito(app))))

	// Public job browsing
	router.GET("/api/v1/baitos/latest", rateLimiter.Limit(baito.GetLatestBaitos(app)))
	router.GET("/api/v1/baitos/related", rateLimiter.Limit(baito.GetRelatedBaitos(app)))

	router.GET("/api/v1/baitos/baito/:baitoid", rateLimiter.Limit(baito.GetBaitoByID(app)))

	// Owner-specific views → require auth
	router.GET("/api/v1/baitos/mine", authmidware(baito.GetMyBaitos(app)))
	router.GET("/api/v1/baitos/baito/:baitoid/applicants", authmidware(baito.GetBaitoApplicants(app)))

	// Part-timer actions → require auth
	router.POST("/api/v1/baitos/baito/:baitoid/apply", authmidware(baito.ApplyToBaito(app)))
	router.GET("/api/v1/baitos/applications", authmidware(baito.GetMyApplications(app)))

	// Profile creation → require auth
	router.POST("/api/v1/baitos/profile", authmidware(baito.CreateWorkerProfile(app)))
	router.PATCH("/api/v1/baitos/profile/:workerId", authmidware(baito.UpdateWorkerProfile(app)))

	// Worker directory (probably private) → require auth
	router.GET("/api/v1/baitos/workers", rateLimiter.Limit(baito.GetWorkers(app)))

	router.GET("/api/v1/baitos/workers/skills", rateLimiter.Limit(baito.GetWorkerSkills(app)))
	router.GET("/api/v1/baitos/worker/:workerId", rateLimiter.Limit(baito.GetWorkerById(app)))
}

func AddBeatRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	authmidware := middleware.Authenticate(app)
	// User must be logged in to like/unlike
	router.PUT("/api/v1/likes/:entitytype/like/:entityid", rateLimiter.Limit(authmidware(beats.ToggleLike(app))))

	// Get users who liked a post/beat
	router.GET("/api/v1/likes/:entitytype/users/:entityid", rateLimiter.Limit(authmidware(beats.GetLikers(app))))

	// Batch check user likes
	router.POST("/api/v1/likes/:entitytype/batch/users", rateLimiter.Limit(authmidware(beats.BatchUserLikes(app))))

	// Like count is public
	router.GET("/api/v1/likes/:entitytype/count/:entityid", rateLimiter.Limit(beats.GetLikeCount(app)))

	// Follows
	router.PUT("/api/v1/follows/:id", rateLimiter.Limit(authmidware(beats.ToggleFollow(app))))
	router.DELETE("/api/v1/follows/:id", rateLimiter.Limit(authmidware(beats.ToggleUnFollow(app))))
	router.GET("/api/v1/follows/:id/status", rateLimiter.Limit(authmidware(beats.DoesFollow(app))))
	router.GET("/api/v1/followers/:id", rateLimiter.Limit(beats.GetFollowers(app)))
	router.GET("/api/v1/following/:id", rateLimiter.Limit(beats.GetFollowing(app)))

	// Subscribes / Follows
	router.PUT("/api/v1/subscribes/:id", rateLimiter.Limit(authmidware(beats.SubscribeEntity(app))))
	router.DELETE("/api/v1/subscribes/:id", rateLimiter.Limit(authmidware(beats.UnsubscribeEntity(app))))
	router.GET("/api/v1/subscribes/:id", rateLimiter.Limit(authmidware(beats.DoesSubscribeEntity(app))))

	// Get all subscribers of a user/artist
	router.GET("/api/v1/subscribers/:id", rateLimiter.Limit(beats.GetSubscribers(app)))

}

func AddRecipeRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	authmidware := middleware.Authenticate(app)
	router.GET("/api/v1/recipes/tags", rateLimiter.Limit(recipes.GetRecipeTags(app)))         // Public
	router.GET("/api/v1/recipes", middleware.OptionalAuth(recipes.GetRecipes(app)))           // Public/optional
	router.GET("/api/v1/recipes/recipe/:id", middleware.OptionalAuth(recipes.GetRecipe(app))) // Public/optional

	// Modifications require auth
	router.POST("/api/v1/recipes", authmidware(recipes.CreateRecipe(app)))
	router.PUT("/api/v1/recipes/recipe/:id", authmidware(recipes.UpdateRecipe(app)))
	router.DELETE("/api/v1/recipes/recipe/:id", authmidware(recipes.DeleteRecipe(app)))
}

func AddHomeRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	// router.GET("/api/v1/home/:apiRoute", middleware.OptionalAuth(home.GetHomeContent)) // Public/optional
	router.GET("/api/v1/homecards", middleware.OptionalAuth(home.HomeCardsHandler(app))) // Public/optional
}

func AddProductRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	router.GET("/api/v1/products/:entityType/:entityId", middleware.OptionalAuth(products.GetProductDetails(app)))
}

func AddReportRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	authmidware := middleware.Authenticate(app)
	router.PUT("/api/v1/report/:id",
		authmidware(
			middleware.RequireRoles("moderator")(
				reports.UpdateReport(app),
			),
		),
	)

	router.POST("/api/v1/report",
		rateLimiter.Limit(
			authmidware(reports.ReportContent(app)),
		),
	)

	// Public (authenticated) endpoint to create appeals
	router.POST("/api/v1/appeals",
		rateLimiter.Limit(
			authmidware(reports.CreateAppeal(app)),
		),
	)

	// router.POST("/api/v1/moderator/apply", moderator.ApplyModerator)
	router.GET("/api/v1/moderator/applications", moderator.ListModeratorApplications(app))
	router.PUT("/api/v1/moderator/approve/:id", moderator.ApproveModerator(app))
	router.PUT("/api/v1/moderator/reject/:id", moderator.RejectModerator(app))

}

// Routes registration
func AddNoticesRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	authmidware := middleware.Authenticate(app)
	// CREATE
	router.POST("/api/v1/notices/:entitytype/:entityid", rateLimiter.Limit(authmidware(notices.CreateNotice(app))))

	// READ
	router.GET("/api/v1/notices/:entitytype/:entityid", notices.GetNotices(app))
	router.GET("/api/v1/notices/:entitytype/:entityid/:noticeid", notices.GetNotice(app))

	// UPDATE + DELETE
	router.PUT("/api/v1/notices/:entitytype/:entityid/:noticeid", rateLimiter.Limit(authmidware(notices.UpdateNotice(app))))
	router.DELETE("/api/v1/notices/:entitytype/:entityid/:noticeid", rateLimiter.Limit(authmidware(notices.DeleteNotice(app))))
}

// Notifications routes
func AddNotificationsRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	authmidware := middleware.Authenticate(app)

	// Create notification
	router.POST("/api/v1/notifs", rateLimiter.Limit(authmidware(notifications.CreateNotification(app))))

	// Bulk create notifications
	router.POST("/api/v1/notifs/bulk", rateLimiter.Limit(authmidware(notifications.BulkCreateNotifications(app))))

	// Get user notifications
	router.GET("/api/v1/notifs/user/:userid", notifications.GetUserNotifications(app))

	// Get unread count
	router.GET("/api/v1/notifs/user/:userid/unread", notifications.GetUnreadCount(app))

	// Mark notification as read
	router.PUT("/api/v1/notifs/notif/:notificationid/read", rateLimiter.Limit(authmidware(notifications.MarkAsRead(app))))

	// Mark all as read
	router.PUT("/api/v1/notifs/user/:userid/read-all", rateLimiter.Limit(authmidware(notifications.MarkAllAsRead(app))))

	// Delete notification
	router.DELETE("/api/v1/notifs/notif/:notificationid", rateLimiter.Limit(authmidware(notifications.DeleteNotification(app))))

	// Clear all notifications
	router.DELETE("/api/v1/notifs/user/:userid", rateLimiter.Limit(authmidware(notifications.ClearAllNotifications(app))))

	// Notification preferences
	router.GET("/api/v1/notifs/user/:userid/preferences", authmidware(notifications.GetPreferences(app)))
	router.PUT("/api/v1/notifs/user/:userid/preferences", rateLimiter.Limit(authmidware(notifications.UpdatePreferences(app))))
}

func AddCommentsRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	authmidware := middleware.Authenticate(app)
	// Create comment
	router.POST("/api/v1/comments/:entitytype/:entityid", rateLimiter.Limit(authmidware(comments.CreateComment(app))))

	// Get comments for an entity (supports pagination/sorting via query params)
	router.GET("/api/v1/comments/:entitytype/:entityid", comments.GetComments(app)) // Public

	router.GET("/api/v1/comments/:entitytype/:entityid/:commentid", comments.GetComment(app))

	// Update & Delete
	router.PUT("/api/v1/comments/:entitytype/:entityid/:commentid", rateLimiter.Limit(authmidware(comments.UpdateComment(app))))
	router.DELETE("/api/v1/comments/:entitytype/:entityid/:commentid", rateLimiter.Limit(authmidware(comments.DeleteComment(app))))
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

// func AddAuthRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
// 	// Public routes
// 	router.POST("/api/v1/auth/register", rateLimiter.Limit(auth.Register))
// 	router.POST("/api/v1/auth/login", rateLimiter.Limit(auth.Login))
// 	router.POST("/api/v1/auth/refresh", rateLimiter.Limit(auth.RefreshToken))
// 	router.POST("/api/v1/auth/verify-otp", rateLimiter.Limit(auth.VerifyOTPHandler))
// 	router.POST("/api/v1/auth/request-otp", rateLimiter.Limit(auth.RequestOTPHandler))

// 	// Protected routes
// 	router.POST("/api/v1/auth/logout", authmidware(auth.LogoutUser))
// }

func AddBookingRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	authmidware := middleware.Authenticate(app)
	// existing routes
	router.GET("/api/v1/bookings/slots", rateLimiter.Limit(authmidware(booking.ListSlots(app))))
	router.POST("/api/v1/bookings/slots", rateLimiter.Limit(authmidware(booking.CreateSlot(app))))
	router.DELETE("/api/v1/bookings/slots/:id", rateLimiter.Limit(authmidware(booking.DeleteSlot(app))))

	router.GET("/api/v1/bookings/bookings", rateLimiter.Limit(authmidware(booking.ListBookings(app))))
	router.POST("/api/v1/bookings/bookings", rateLimiter.Limit(authmidware(booking.CreateBooking(app))))
	router.PUT("/api/v1/bookings/bookings/:id/status", rateLimiter.Limit(authmidware(booking.UpdateBookingStatus(app))))
	router.DELETE("/api/v1/bookings/bookings/:id", rateLimiter.Limit(authmidware(booking.CancelBooking(app))))

	router.GET("/api/v1/bookings/date-capacity", rateLimiter.Limit(authmidware(booking.GetDateCapacity(app))))
	router.POST("/api/v1/bookings/date-capacity", rateLimiter.Limit(authmidware(booking.SetDateCapacity(app))))

	// NEW: pricing tiers
	router.GET("/api/v1/bookings/tiers", rateLimiter.Limit(authmidware(booking.ListTiers(app))))
	router.POST("/api/v1/bookings/tiers", rateLimiter.Limit(authmidware(booking.CreateTier(app))))
	router.DELETE("/api/v1/bookings/tiers/:id", rateLimiter.Limit(authmidware(booking.DeleteTier(app))))

	// NEW: auto slot generation from tier
	router.POST("/api/v1/bookings/tiers/:id/generate-slots", rateLimiter.Limit(authmidware(booking.GenerateSlotsFromTier(app))))
}

func AddEventsRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	authmidware := middleware.Authenticate(app)
	router.GET("/api/v1/events/events", rateLimiter.Limit(events.GetEvents(app)))            // Public
	router.GET("/api/v1/events/events/count", rateLimiter.Limit(events.GetEventsCount(app))) // Public
	router.POST("/api/v1/events/event", authmidware(events.CreateEvent(app)))
	router.GET("/api/v1/events/event/:eventid", events.GetEvent(app)) // Public
	router.PUT("/api/v1/events/event/:eventid", authmidware(events.EditEvent(app)))
	router.DELETE("/api/v1/events/event/:eventid", authmidware(events.DeleteEvent(app)))

	// Should probably require auth if restricted
	router.POST("/api/v1/events/event/:eventid/faqs", authmidware(events.AddFAQs(app)))
}

func AddCartRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	authmidware := middleware.Authenticate(app)
	// Cart operations
	router.POST("/api/v1/cart", rateLimiter.Limit(authmidware(cart.AddToCart(app))))
	router.GET("/api/v1/cart", authmidware(cart.GetCart(app)))
	router.POST("/api/v1/cart/update", rateLimiter.Limit(authmidware(cart.UpdateCart(app))))
	router.DELETE("/api/v1/cart/item", rateLimiter.Limit(authmidware(cart.RemoveFromCart(app))))
	router.DELETE("/api/v1/cart", rateLimiter.Limit(authmidware(cart.ClearCart(app))))
	router.PATCH("/api/v1/cart/item", rateLimiter.Limit(authmidware(cart.UpdateItemQuantity(app))))
	router.POST("/api/v1/cart/checkout", rateLimiter.Limit(authmidware(cart.InitiateCheckout(app))))

	// Checkout session creation
	router.POST("/api/v1/checkout/session", rateLimiter.Limit(authmidware(cart.CreateCheckoutSession(app))))

	// Order placement
	router.POST("/api/v1/order", rateLimiter.Limit(authmidware(cart.PlaceOrder(app))))
	router.GET("/api/v1/order/mine", authmidware(cart.GetMyOrders(app)))

	router.POST("/api/v1/coupon/validate", rateLimiter.Limit(authmidware(cart.ValidateCouponHandler(app))))

}

func RegisterFarmRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	authmidware := middleware.Authenticate(app)
	// 🌾 Farm CRUD
	router.POST("/api/v1/farms", rateLimiter.Limit(authmidware(farms.CreateFarm(app))))
	router.GET("/api/v1/farms", farms.GetPaginatedFarms(app)) // Public
	router.GET("/api/v1/farms/:id", middleware.OptionalAuth(farms.GetFarm(app)))
	router.PUT("/api/v1/farms/:id", rateLimiter.Limit(authmidware(farms.EditFarm(app))))
	router.DELETE("/api/v1/farms/:id", rateLimiter.Limit(authmidware(farms.DeleteFarm(app))))

	// 🌱 Crops (within farm)
	router.POST("/api/v1/farms/:id/crops", rateLimiter.Limit(authmidware(farms.AddCrop(app))))
	router.PUT("/api/v1/farms/:id/crops/:cropid", rateLimiter.Limit(authmidware(farms.EditCrop(app))))
	router.DELETE("/api/v1/farms/:id/crops/:cropid", rateLimiter.Limit(authmidware(farms.DeleteCrop(app))))
	router.PUT("/api/v1/farms/:id/crops/:cropid/buy", rateLimiter.Limit(authmidware(farms.BuyCrop(app))))

	// 📊 Dashboard
	router.GET("/api/v1/dash/farms", authmidware(farms.GetFarmDash(app)))

	// 📦 Farm Orders
	router.GET("/api/v1/orders/mine", authmidware(farms.GetMyFarmOrders(app)))
	router.GET("/api/v1/orders/incoming", authmidware(farms.GetIncomingFarmOrders(app)))
	router.POST("/api/v1/farmorders/order/:id/accept", rateLimiter.Limit(authmidware(farms.AcceptOrder(app))))
	router.POST("/api/v1/farmorders/order/:id/reject", rateLimiter.Limit(authmidware(farms.RejectOrder(app))))
	router.POST("/api/v1/farmorders/order/:id/deliver", rateLimiter.Limit(authmidware(farms.MarkOrderDelivered(app))))
	router.POST("/api/v1/farmorders/order/:id/markpaid", rateLimiter.Limit(authmidware(farms.MarkOrderPaid(app))))
	router.GET("/api/v1/farmorders/order/:id/receipt", authmidware(farms.DownloadReceipt(app)))
	// Bulk actions
	router.POST("/api/v1/farmorders/bulk/accept", rateLimiter.Limit(authmidware(farms.BulkAcceptOrders(app))))
	router.POST("/api/v1/farmorders/bulk/reject", rateLimiter.Limit(authmidware(farms.BulkRejectOrders(app))))
	router.POST("/api/v1/farmorders/bulk/deliver", rateLimiter.Limit(authmidware(farms.BulkMarkOrdersDelivered(app))))

	// 🌾 Crop catalogue & type browsing
	router.GET("/api/v1/crops", farms.GetFilteredCrops(app))                 // Public
	router.GET("/api/v1/crops/catalogue", farms.GetCropCatalogue(app))       // Public
	router.GET("/api/v1/crops/precatalogue", farms.GetPreCropCatalogue(app)) // Public
	router.GET("/api/v1/crops/types", farms.GetCropTypes(app))               // Public
	router.GET("/api/v1/crops/crop/:cropname", middleware.OptionalAuth(farms.GetCropTypeFarms(app)))

	// 🛒 Items, Products, Tools
	// -- GET
	router.GET("/api/v1/farm/items", farms.GetItems(app))                     // Public
	router.GET("/api/v1/farm/items/categories", farms.GetItemCategories(app)) // Public

	// -- Products (CRUD)
	router.POST("/api/v1/farm/product", rateLimiter.Limit(authmidware(farms.CreateProduct(app))))
	router.PUT("/api/v1/farm/product/:id", rateLimiter.Limit(authmidware(farms.UpdateProduct(app))))
	router.DELETE("/api/v1/farm/product/:id", rateLimiter.Limit(authmidware(farms.DeleteProduct(app))))

	// -- Tools (CRUD)
	router.POST("/api/v1/farm/tool", rateLimiter.Limit(authmidware(farms.CreateTool(app))))
	router.PUT("/api/v1/farm/tool/:id", rateLimiter.Limit(authmidware(farms.UpdateTool(app))))
	router.DELETE("/api/v1/farm/tool/:id", rateLimiter.Limit(authmidware(farms.DeleteTool(app))))

	// 🖼 Upload
	// router.POST("/api/v1/upload/images", rateLimiter.Limit(authmidware(utils.UploadImages)))
}

func AddMerchRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	authmidware := middleware.Authenticate(app)
	// Create merch
	router.POST("/api/v1/merch/:entityType/:eventid", rateLimiter.Limit(authmidware(merch.CreateMerch(app))))

	// Buy merch
	router.POST("/api/v1/merch/:entityType/:eventid/:merchid/buy", rateLimiter.Limit(authmidware(merch.BuyMerch(app))))

	// Public view
	router.GET("/api/v1/merch/:entityType/:eventid", merch.GetMerchs(app))
	router.GET("/api/v1/merch/:entityType/:eventid/:merchid", merch.GetMerch(app))
	router.GET("/api/v1/merch/:entityType", merch.GetMerchPage(app))

	// Edit/Delete
	router.PUT("/api/v1/merch/:entityType/:eventid/:merchid", rateLimiter.Limit(authmidware(merch.EditMerch(app))))
	router.DELETE("/api/v1/merch/:entityType/:eventid/:merchid", rateLimiter.Limit(authmidware(merch.DeleteMerch(app))))

	// Payment flows
	router.POST("/api/v1/merch/:entityType/:eventid/:merchid/payment-session", rateLimiter.Limit(authmidware(merch.CreateMerchPaymentSession(app))))
	router.POST("/api/v1/merch/:entityType/:eventid/:merchid/confirm-purchase", rateLimiter.Limit(authmidware(merch.ConfirmMerchPurchase(app))))
}

func AddTicketRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	authmidware := middleware.Authenticate(app)
	// Ticket CRUD
	router.POST("/api/v1/ticket/event/:eventid", rateLimiter.Limit(authmidware(tickets.CreateTicket(app))))
	router.GET("/api/v1/ticket/event/:eventid", rateLimiter.Limit(tickets.GetTickets(app)))
	router.GET("/api/v1/ticket/event/:eventid/:ticketid", rateLimiter.Limit(tickets.GetTicket(app)))
	router.PUT("/api/v1/ticket/event/:eventid/:ticketid", rateLimiter.Limit(authmidware(tickets.EditTicket(app))))
	router.DELETE("/api/v1/ticket/event/:eventid/:ticketid", rateLimiter.Limit(authmidware(tickets.DeleteTicket(app))))

	// Buying
	router.POST("/api/v1/ticket/event/:eventid/:ticketid/buy", rateLimiter.Limit(authmidware(tickets.BuyTicket(app))))
	router.POST("/api/v1/tickets/book", rateLimiter.Limit(authmidware(tickets.BuysTicket(app))))

	// Payment flows
	router.POST("/api/v1/ticket/event/:eventid/:ticketid/payment-session", rateLimiter.Limit(authmidware(tickets.CreateTicketPaymentSession(app))))
	router.POST("/api/v1/ticket/event/:eventid/:ticketid/confirm-purchase", rateLimiter.Limit(authmidware(tickets.ConfirmTicketPurchase(app))))

	// Verification/printing
	router.GET("/api/v1/ticket/verify/:eventid", rateLimiter.Limit(authmidware(tickets.VerifyTicket(app))))
	router.GET("/api/v1/ticket/print/:eventid", rateLimiter.Limit(authmidware(tickets.PrintTicket(app))))
	router.POST("/api/v1/ticket/transfer/:eventid", rateLimiter.Limit(authmidware(tickets.TransferTicket(app))))
	router.POST("/api/v1/ticket/cancel/:eventid", rateLimiter.Limit(authmidware(tickets.CancelTicket(app))))
	router.GET("/api/v1/ticket/mytickets/:eventid", rateLimiter.Limit(authmidware(tickets.ListMyTickets(app))))

	// Event updates
	router.GET("/api/v1/events/event/:eventid/updates", rateLimiter.Limit(tickets.EventUpdates(app)))

	// Seats
	router.GET("/api/v1/seats/:eventid/available-seats", rateLimiter.Limit(tickets.GetAvailableSeats(app)))
	router.POST("/api/v1/seats/:eventid/lock-seats", rateLimiter.Limit(authmidware(tickets.LockSeats(app))))
	router.POST("/api/v1/seats/:eventid/unlock-seats", rateLimiter.Limit(authmidware(tickets.UnlockSeats(app))))
	router.POST("/api/v1/seats/:eventid/ticket/:ticketid/confirm-purchase", rateLimiter.Limit(authmidware(tickets.ConfirmSeatPurchase(app))))
	router.GET("/api/v1/ticket/event/:eventid/:ticketid/seats", rateLimiter.Limit(tickets.GetTicketSeats(app)))
}

func AddSuggestionsRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	authmidware := middleware.Authenticate(app)
	router.GET("/api/v1/suggestions/places/nearby", rateLimiter.Limit(suggestions.GetNearbyPlaces(app)))
	router.GET("/api/v1/suggestions/follow", rateLimiter.Limit(authmidware(suggestions.SuggestFollowers(app))))
}

func AddReviewsRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	authmidware := middleware.Authenticate(app)
	// Public view, but rate-limited
	router.GET("/api/v1/reviews/:entityType/:entityId", rateLimiter.Limit(reviews.GetReviews(app)))
	router.GET("/api/v1/reviews/:entityType/:entityId/:reviewId", rateLimiter.Limit(reviews.GetReview(app)))

	// Authenticated actions
	router.POST("/api/v1/reviews/:entityType/:entityId", rateLimiter.Limit(authmidware(reviews.AddReview(app))))
	router.PUT("/api/v1/reviews/:entityType/:entityId/:reviewId", rateLimiter.Limit(authmidware(reviews.EditReview(app))))
	router.DELETE("/api/v1/reviews/:entityType/:entityId/:reviewId", rateLimiter.Limit(authmidware(reviews.DeleteReview(app))))
}

func AddMediaRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	authmidware := middleware.Authenticate(app)
	// Public view, but rate-limited
	router.GET("/api/v1/media/:entitytype/:entityid/:id", rateLimiter.Limit(media.GetMedia(app)))
	router.GET("/api/v1/media/:entitytype/:entityid", rateLimiter.Limit(media.GetMedias(app)))

	// Authenticated actions
	router.POST("/api/v1/media/:entitytype/:entityid", rateLimiter.Limit(authmidware(media.AddMedia(app))))
	router.PUT("/api/v1/media/:entitytype/:entityid/:id", rateLimiter.Limit(authmidware(media.EditMedia(app))))
	router.DELETE("/api/v1/media/:entitytype/:entityid/:id", rateLimiter.Limit(authmidware(media.DeleteMedia(app))))
}

func AddFanmadeRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	authmidware := middleware.Authenticate(app)
	router.GET("/api/v1/fanmade/:entitytype/:entityid/:id", rateLimiter.Limit(fanmade.GetMedia(app)))
	router.GET("/api/v1/fanmade/:entitytype/:entityid", rateLimiter.Limit(fanmade.GetMedias(app)))

	router.POST("/api/v1/fanmade/:entitytype/:entityid", rateLimiter.Limit(authmidware(fanmade.AddMedia(app))))
	router.PUT("/api/v1/fanmade/:entitytype/:entityid/:id", rateLimiter.Limit(authmidware(fanmade.EditMedia(app))))
	router.DELETE("/api/v1/fanmade/:entitytype/:entityid/:id", rateLimiter.Limit(authmidware(fanmade.DeleteMedia(app))))
}

func AddPostRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	authmidware := middleware.Authenticate(app)
	// Public read
	router.GET("/api/v1/posts/post/:id", rateLimiter.Limit(posts.GetPost(app)))
	router.GET("/api/v1/posts", rateLimiter.Limit(posts.GetAllPosts(app)))
	// router.POST("/api/v1/posts/upload", rateLimiter.Limit(posts.UploadImage))

	// Authenticated write
	router.POST("/api/v1/posts/post", rateLimiter.Limit(authmidware(posts.CreatePost(app))))
	router.PATCH("/api/v1/posts/post/:id", rateLimiter.Limit(authmidware(posts.UpdatePost(app))))
	router.DELETE("/api/v1/posts/post/:id", rateLimiter.Limit(authmidware(posts.DeletePost(app))))

	router.GET("/api/v1/posts/post/:id/related", rateLimiter.Limit(authmidware(posts.GetRelatedPosts(app))))

}

func AddPlaceRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	authmidware := middleware.Authenticate(app)
	// Public
	router.GET("/api/v1/places/places", rateLimiter.Limit(places.GetPlaces(app)))
	router.GET("/api/v1/places/place/:placeid", rateLimiter.Limit(places.GetPlace(app)))
	router.GET("/api/v1/places/place-details", rateLimiter.Limit(places.GetPlaceQ(app)))

	// Authenticated place management
	router.POST("/api/v1/places/place", rateLimiter.Limit(authmidware(places.CreatePlace(app))))
	router.PUT("/api/v1/places/place/:placeid", rateLimiter.Limit(authmidware(places.EditPlace(app))))
	router.DELETE("/api/v1/places/place/:placeid", rateLimiter.Limit(authmidware(places.DeletePlace(app))))
	router.PUT("/api/v1/places/place/:placeid/info", rateLimiter.Limit(authmidware(places.UpdatePlaceInfo(app))))

	// Menus (public view + auth for changes)
	router.GET("/api/v1/places/menu/:placeid", rateLimiter.Limit(menu.GetMenus(app)))
	router.GET("/api/v1/places/menu/:placeid/:menuid/stock", rateLimiter.Limit(menu.GetStock(app)))
	router.GET("/api/v1/places/menu/:placeid/:menuid", rateLimiter.Limit(menu.GetMenu(app)))

	router.POST("/api/v1/places/menu/:placeid", rateLimiter.Limit(authmidware(menu.CreateMenu(app))))
	router.PUT("/api/v1/places/menu/:placeid/:menuid", rateLimiter.Limit(authmidware(menu.EditMenu(app))))
	router.DELETE("/api/v1/places/menu/:placeid/:menuid", rateLimiter.Limit(authmidware(menu.DeleteMenu(app))))

	// Buying & payment flows
	router.POST("/api/v1/places/menu/:placeid/:menuid/buy", rateLimiter.Limit(authmidware(menu.BuyMenu(app))))
	router.POST("/api/v1/places/menu/:placeid/:menuid/payment-session", rateLimiter.Limit(authmidware(menu.CreateMenuPaymentSession(app))))
	router.POST("/api/v1/places/menu/:placeid/:menuid/confirm-purchase", rateLimiter.Limit(authmidware(menu.ConfirmMenuPurchase(app))))
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

func AddArtistRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	authmidware := middleware.Authenticate(app)
	// Public read
	router.GET("/api/v1/artists", rateLimiter.Limit(artists.GetAllArtists(app)))
	router.GET("/api/v1/artists/:id", rateLimiter.Limit(artists.GetArtistByID(app)))
	router.GET("/api/v1/events/event/:eventid/artists", rateLimiter.Limit(artists.GetArtistsByEvent(app)))
	router.GET("/api/v1/artists/:id/songs", rateLimiter.Limit(artists.GetArtistsSongs(app)))
	router.GET("/api/v1/artists/:id/albums", rateLimiter.Limit(artists.GetArtistsAlbums(app)))
	router.GET("/api/v1/artists/:id/posts", rateLimiter.Limit(artists.GetArtistsPosts(app)))
	router.GET("/api/v1/artists/:id/merch", rateLimiter.Limit(artists.GetArtistsMerch(app)))
	router.GET("/api/v1/artists/:id/events", rateLimiter.Limit(artists.GetArtistEvents(app)))

	// Authenticated write
	router.POST("/api/v1/artists", rateLimiter.Limit(authmidware(artists.CreateArtist(app))))
	router.PUT("/api/v1/artists/:id", rateLimiter.Limit(authmidware(artists.UpdateArtist(app))))
	router.DELETE("/api/v1/artists/:id", rateLimiter.Limit(authmidware(artists.DeleteArtistByID(app))))

	// OLD (bulk update) – optional to keep
	// router.PUT("/api/v1/artists/:id/members", rateLimiter.Limit(authmidware(artists.UpdateArtistMembers)))

	// NEW — per-member endpoints
	router.POST("/api/v1/artists/:id/members",
		rateLimiter.Limit(authmidware(artists.AddArtistMember(app))))

	router.PUT("/api/v1/artists/:id/members/:memberId",
		rateLimiter.Limit(authmidware(artists.UpdateArtistMember(app))))

	router.DELETE("/api/v1/artists/:id/members/:memberId",
		rateLimiter.Limit(authmidware(artists.DeleteArtistMember(app))))

	router.POST("/api/v1/artists/:id/songs", rateLimiter.Limit(authmidware(artists.PostNewSong(app))))
	router.PUT("/api/v1/artists/:id/songs/:songId/edit", rateLimiter.Limit(authmidware(artists.EditSong(app))))
	router.DELETE("/api/v1/artists/:id/songs/:songId", rateLimiter.Limit(authmidware(artists.DeleteSong(app))))

	router.PUT("/api/v1/artists/:id/events/addtoevent", rateLimiter.Limit(authmidware(artists.AddArtistToEvent(app))))
	router.POST("/api/v1/artists/:id/events", rateLimiter.Limit(authmidware(artists.CreateArtistEvent(app))))
	router.PUT("/api/v1/artists/:id/events", rateLimiter.Limit(authmidware(artists.UpdateArtistEvent(app))))
	router.DELETE("/api/v1/artists/:id/events", rateLimiter.Limit(authmidware(artists.DeleteArtistEvent(app))))
}

// AddMapRoutes binds routes to the provided router and rate limiter
func AddMapRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	// entity-specific endpoints
	router.GET("/api/v1/maps/config/:entity", rateLimiter.Limit(maps.GetMapConfig))
	router.GET("/api/v1/maps/markers/:entity", rateLimiter.Limit(maps.GetMapMarkers))

	// player progression endpoints
	router.POST("/api/v1/player/progress", rateLimiter.Limit(maps.UpdatePlayerProgress))
	router.GET("/api/v1/player/progress", rateLimiter.Limit(maps.GetPlayerProgress))
}

func AddItineraryRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	authmidware := middleware.Authenticate(app)
	// Public
	router.GET("/api/v1/itineraries", rateLimiter.Limit(itinerary.GetItineraries(app)))
	router.GET("/api/v1/itineraries/all/:id", rateLimiter.Limit(itinerary.GetItinerary(app)))
	router.GET("/api/v1/itineraries/search", rateLimiter.Limit(itinerary.SearchItineraries(app)))

	// Authenticated write
	router.POST("/api/v1/itineraries", rateLimiter.Limit(authmidware(itinerary.CreateItinerary(app))))
	router.PUT("/api/v1/itineraries/:id", rateLimiter.Limit(authmidware(itinerary.UpdateItinerary(app))))
	router.DELETE("/api/v1/itineraries/:id", rateLimiter.Limit(authmidware(itinerary.DeleteItinerary(app))))
	router.POST("/api/v1/itineraries/:id/fork", rateLimiter.Limit(authmidware(itinerary.ForkItinerary(app))))
	router.PUT("/api/v1/itineraries/:id/publish", rateLimiter.Limit(authmidware(itinerary.PublishItinerary(app))))
}

func AddUtilityRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	authmidware := middleware.Authenticate(app)
	router.GET("/api/v1/csrf", rateLimiter.Limit(authmidware(utils.CSRF)))
}

func AddFeedRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	authmidware := middleware.Authenticate(app)
	// Public viewing
	router.GET("/api/v1/feed/post/:postid", rateLimiter.Limit(feed.GetPost(app)))
	router.POST("/api/v1/feed/feed/metadata", rateLimiter.Limit(feed.GetPostsMetadata(app)))

	// Authenticated feed actions
	router.GET("/api/v1/feed/feed", rateLimiter.Limit(authmidware(feed.GetPosts(app))))
	router.GET("/api/v1/feed/media/:entityType/:entityId", rateLimiter.Limit(authmidware(feed.GetPosts(app))))

	router.POST("/api/v1/feed/post", rateLimiter.Limit(authmidware(feed.CreateFeedPost(app))))
	router.DELETE("/api/v1/feed/post/:postid", rateLimiter.Limit(authmidware(feed.DeletePost(app))))

	// NEW
	router.PATCH("/api/v1/feed/post/:postid", rateLimiter.Limit(authmidware(feed.EditPost(app))))
	// router.POST("/api/v1/feed/post/:postid/subtitles/:lang", rateLimiter.Limit(authmidware(filedrop.UploadSubtitle)))
}

//	func AddSettingsRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
//		router.GET("/api/v1/settings/init/:userid", rateLimiter.Limit(authmidware(settings.InitUserSettings(app))))
//		router.GET("/api/v1/settings/all", rateLimiter.Limit(authmidware(settings.GetUserSettings(app))))
//		router.PUT("/api/v1/settings/setting/:type", rateLimiter.Limit(authmidware(settings.UpdateUserSetting(app))))
//	}
func AddSettingsRoutes(
	router *httprouter.Router,
	app *infra.Deps,
	rateLimiter *middleware.RateLimiter,
) {
	authmidware := middleware.Authenticate(app)

	router.GET(
		"/api/v1/settings/init/:userid",
		rateLimiter.Limit(authmidware(settings.InitUserSettings(app))),
	)

	router.GET(
		"/api/v1/settings/all",
		rateLimiter.Limit(authmidware(settings.GetUserSettings(app))),
	)

	router.PUT(
		"/api/v1/settings/setting/:type",
		rateLimiter.Limit(authmidware(settings.UpdateUserSetting(app))),
	)
}

func AddAdsRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	router.GET("/api/v1/sda/sda", rateLimiter.Limit(middleware.OptionalAuth(ads.GetAds)))
}

func AddHashtagRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	router.GET("/api/v1/hashtags/hashtag/:tag", hashtags.GetHashtagPosts)
	router.GET("/api/v1/hashtags/hashtag/:tag/top", hashtags.GetTopHashtagPosts)
	router.GET("/api/v1/hashtags/hashtag/:tag/latest", hashtags.GetLatestHashtagPosts)
	router.GET("/api/v1/hashtags/hashtag/:tag/people", hashtags.GetHashtagPeople)
	router.GET("/api/v1/hashtags/hashtags/trending", hashtags.GetTrendingHashtags)

	// router.GET("/api/v1/hashtags/hashtag/:tag", hashtags.GetHashtagPosts)
	// router.GET("/api/v1/hashtags/hashtags/trending", hashtags.GetTrendingHashtags)
}

func AddAnalyticsRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	// Example: /api/v1/antics/events/123 or /api/v1/analytics/places/456
	router.GET("/api/v1/antics/:entityType/:entityId", rateLimiter.Limit(analytics.GetEntityAnalytics))
}

func AddMiscRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	router.GET("/api/v1/users/meta", rateLimiter.Limit(metadata.GetUsersMeta(app)))

	// router.POST("/api/v1/check-file", rateLimiter.Limit(filecheck.CheckFileExists))
	// router.POST("/api/v1/upload", rateLimiter.Limit(filecheck.UploadFile))
	// router.POST("/api/v1/feed/remhash", rateLimiter.Limit(filecheck.RemoveUserFile))
	// router.GET("/resize/:folder/*filename", cdn.ServeStatic)

}

func AddStripeRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	authmidware := middleware.Authenticate(app)
	router.POST("/api/v1/stripe/create-payment-intent", rateLimiter.Limit(authmidware(stripe.CreatePaymentIntent(app))))
	router.POST("/api/v1/stripe/payment-success", rateLimiter.Limit(authmidware(stripe.PaymentSuccess(app))))
	router.POST("/api/v1/stripe/webhook", rateLimiter.Limit(authmidware(stripe.StripeWebhook(app))))
}

func AddMusicRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	authmidware := middleware.Authenticate(app)

	// --------------------------- PLAYLISTS ---------------------------
	router.GET(
		"/api/v1/musicon/user/playlists",
		rateLimiter.Limit(authmidware(musicon.GetUserPlaylists(app))),
	)

	router.GET(
		"/api/v1/musicon/user/liked",
		rateLimiter.Limit(authmidware(musicon.GetUserLikes(app))),
	)

	router.POST(
		"/api/v1/musicon/playlists",
		rateLimiter.Limit(authmidware(musicon.CreatePlaylist(app))),
	)

	router.DELETE(
		"/api/v1/musicon/playlists/:playlistid",
		rateLimiter.Limit(authmidware(musicon.DeletePlaylist(app))),
	)

	// Add / Remove songs to playlist
	router.POST(
		"/api/v1/musicon/playlists/:playlistid/songs",
		rateLimiter.Limit(authmidware(musicon.AddSongToPlaylist(app))),
	)

	router.DELETE(
		"/api/v1/musicon/playlists/:playlistid/songs/:songid",
		rateLimiter.Limit(authmidware(musicon.RemoveSongFromPlaylist(app))),
	)

	// Playlist details
	router.GET(
		"/api/v1/musicon/playlists/:playlistid/songs",
		rateLimiter.Limit(authmidware(musicon.GetPlaylistSongs(app))),
	)

	// Rename / Update playlist info
	router.PATCH(
		"/api/v1/musicon/playlists/:playlistid",
		rateLimiter.Limit(authmidware(musicon.UpdatePlaylistInfo(app))),
	)

	// --------------------------- LIKES ---------------------------

	// Like song (idempotent)
	router.POST(
		"/api/v1/musicon/user/liked/:songid",
		rateLimiter.Limit(authmidware(musicon.LikeSong(app))),
	)

	// Unlike song (idempotent)
	router.DELETE(
		"/api/v1/musicon/user/liked/:songid",
		rateLimiter.Limit(authmidware(musicon.UnlikeSong(app))),
	)

	// --------------------------- ARTISTS ---------------------------
	router.GET(
		"/api/v1/musicon/artists/:artistid/songs",
		rateLimiter.Limit(middleware.OptionalAuth(musicon.GetArtistsSongs(app))),
	)

	// --------------------------- ALBUMS ---------------------------
	router.GET(
		"/api/v1/musicon/albums",
		rateLimiter.Limit(middleware.OptionalAuth(musicon.GetAlbums(app))),
	)

	router.GET(
		"/api/v1/musicon/albums/:albumid/songs",
		rateLimiter.Limit(middleware.OptionalAuth(musicon.GetAlbumSongs(app))),
	)

	router.GET(
		"/api/v1/musicon/recommended/albums",
		rateLimiter.Limit(middleware.OptionalAuth(musicon.GetRecommendedAlbums(app))),
	)

	// --------------------------- SONGS & RECOMMENDATIONS ---------------------------
	router.GET(
		"/api/v1/musicon/recommended",
		rateLimiter.Limit(middleware.OptionalAuth(musicon.GetRecommendedSongs(app))),
	)

	router.GET(
		"/api/v1/musicon/recommendations",
		rateLimiter.Limit(middleware.OptionalAuth(musicon.GetRecommendations(app))),
	)
}

/* func AddMusicRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	authmidware := middleware.Authenticate(app)
	--------------------------- PLAYLISTS ---------------------------
	router.GET("/api/v1/musicon/user/playlists", rateLimiter.Limit(middleware.OptionalAuth(musicon.GetUserPlaylists(app))))
	router.GET("/api/v1/musicon/user/liked", rateLimiter.Limit(middleware.OptionalAuth(musicon.GetUserLikes(app))))
	router.POST("/api/v1/musicon/playlists", rateLimiter.Limit(authmidware(musicon.CreatePlaylist(app))))
	router.DELETE("/api/v1/musicon/playlists/:playlistid", rateLimiter.Limit(authmidware(musicon.DeletePlaylist(app))))

	Add / Remove songs to playlist
	router.POST("/api/v1/musicon/playlists/:playlistid/songs/:songid", rateLimiter.Limit(authmidware(musicon.AddSongToPlaylist)))
	router.POST("/api/v1/musicon/playlists/:playlistid/songs", rateLimiter.Limit(authmidware(musicon.AddSongToPlaylist(app))))
	router.POST("/api/v1/musicon/user/liked/:songid", rateLimiter.Limit(middleware.OptionalAuth(musicon.SetUserLikes(app))))
	router.DELETE("/api/v1/musicon/playlists/:playlistid/songs/:songid", rateLimiter.Limit(authmidware(musicon.RemoveSongFromPlaylist(app))))

	Playlist details
	router.GET("/api/v1/musicon/playlists/:playlistid/songs", rateLimiter.Limit(middleware.OptionalAuth(musicon.GetPlaylistSongs(app))))

	Rename / Update playlist info
	router.PATCH("/api/v1/musicon/playlists/:playlistid", rateLimiter.Limit(authmidware(musicon.UpdatePlaylistInfo(app))))

	--------------------------- ARTISTS ---------------------------
	router.GET("/api/v1/musicon/artists/:artistid/songs", rateLimiter.Limit(middleware.OptionalAuth(musicon.GetArtistsSongs(app))))

	--------------------------- ALBUMS ---------------------------
	router.GET("/api/v1/musicon/albums", rateLimiter.Limit(middleware.OptionalAuth(musicon.GetAlbums(app))))
	router.GET("/api/v1/musicon/albums/:albumid/songs", rateLimiter.Limit(middleware.OptionalAuth(musicon.GetAlbumSongs(app))))
	router.GET("/api/v1/musicon/recommended/albums", rateLimiter.Limit(middleware.OptionalAuth(musicon.GetRecommendedAlbums(app))))

	--------------------------- SONGS & RECOMMENDATIONS ---------------------------
	router.GET("/api/v1/musicon/recommended", rateLimiter.Limit(middleware.OptionalAuth(musicon.GetRecommendedSongs(app))))

	Dynamic personalized recommendations
	router.GET("/api/v1/musicon/recommendations", rateLimiter.Limit(middleware.OptionalAuth(musicon.GetRecommendations(app))))
} */

// func AddDiscordRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
// 	// authmidware := middleware.Authenticate(app)

// }

func AddMeChatRoutes(router *httprouter.Router, hub *mechat.Hub, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	authmidware := middleware.Authenticate(app)
	router.GET("/api/v1/merechats/all", authmidware(mechat.GetUserChats(app)))
	router.POST("/api/v1/merechats/start", authmidware(mechat.StartNewChat(app)))
	router.GET("/api/v1/merechats/chat/:chatid", authmidware(mechat.GetChatByID(app)))
	router.GET("/api/v1/merechats/chat/:chatid/messages", authmidware(mechat.GetChatMessages(app)))
	router.POST("/api/v1/merechats/chat/:chatid/message", authmidware(mechat.SendMessageREST(app)))
	router.PATCH("/api/v1/merechats/messages/:messageid", authmidware(mechat.EditMessage(app)))
	router.DELETE("/api/v1/merechats/messages/:messageid", authmidware(mechat.DeleteMessage(app)))

	router.GET("/ws/merechat", authmidware(
		mechat.HandleWebSocket(app, hub),
	))

	// // WebSocket also needs auth to ensure only valid users connect
	// router.GET("/ws/merechat", authmidware(func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// 	mechat.HandleWebSocket(app, hub)
	// }))

	router.POST("/api/v1/merechats/chat/:chatid/upload", authmidware(mechat.UploadAttachment(app)))
	router.GET("/api/v1/merechats/chat/:chatid/search", authmidware(mechat.SearchMessages(app)))
	router.GET("/api/v1/merechats/messages/unread-count", authmidware(mechat.GetUnreadCount(app)))
	router.POST("/api/v1/merechats/messages/:messageid/read", authmidware(mechat.MarkAsRead(app)))
}

func AddNewChatRoutes(router *httprouter.Router, hub *newchat.Hub, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	authmidware := middleware.Authenticate(app)
	router.GET("/api/v1/newchats/all", authmidware(newchat.GetUserChats(app)))
	router.POST("/api/v1/newchats/init", authmidware(newchat.InitChat(app)))

	// This should likely be protected; token could be in query or header
	router.GET("/ws/newchat/:room", authmidware(newchat.WebSocketHandler(hub, app)))

	router.POST("/newchat/edit", authmidware(newchat.EditMessageHandler(hub, app)))
	router.POST("/newchat/delete", authmidware(newchat.DeleteMessageHandler(hub, app)))

	// router.GET("/newchat/:room/poll", authmidware(newchat.PollMessagesHandler))

	router.POST("/newchat/upload", authmidware(newchat.UploadHandler(hub, app)))

	router.GET("/api/v1/newchat/:room", authmidware(newchat.GetChat(app)))
	router.POST("/api/v1/newchat/:room/message", authmidware(newchat.CreateMessage(app)))
	router.DELETE("/api/v1/newchat/:room/message/:msgid", authmidware(newchat.DeletesMessage(app)))

	/**/

	router.PUT("/api/v1/newchat/:room/message/:msgid", authmidware(newchat.UpdateMessage(app)))

}

// ----------------------- ROUTES -----------------------

func AddVliveRoutes(r *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	auth := middleware.Authenticate(app)

	// Streams collection
	r.POST("/api/v1/vlive", auth(vlive.CreateStream(app)))
	r.GET("/api/v1/vlive", vlive.ListStreams(app))

	// Streams by ID
	r.GET("/api/v1/vlive/id/:liveid", vlive.GetStream(app))
	r.POST("/api/v1/vlive/id/:liveid/ready", auth(vlive.MarkReady(app)))
	r.POST("/api/v1/vlive/id/:liveid/start", auth(vlive.StartStream(app)))
	r.POST("/api/v1/vlive/id/:liveid/end", auth(vlive.StopStream(app)))
	r.POST("/api/v1/vlive/id/:liveid/privacy", auth(vlive.SetPrivacy(app)))
	r.PUT("/api/v1/vlive/id/:liveid", auth(vlive.UpdateStreamMetadata(app)))

	// VOD
	r.GET("/api/v1/vlive/id/:liveid/vod", auth(vlive.GetVOD(app)))
	r.POST("/api/v1/vlive/id/:liveid/vod/publish", auth(vlive.PublishVOD(app)))
	r.DELETE("/api/v1/vlive/id/:liveid/vod", auth(vlive.DeleteVOD(app)))

	// Ingest / Recording
	r.POST("/api/v1/vlive/ingest/start", vlive.StartStream(app))
	r.POST("/api/v1/vlive/ingest/recording/complete", vlive.RecordingComplete(app))

	// Chat
	r.GET("/ws/v1/vlive/id/:liveid/chat", auth(vlive.ChatWebSocket(app)))
	r.POST("/api/v1/vlive/id/:liveid/chat/enable", auth(vlive.ChatEnable(app, true)))
	r.POST("/api/v1/vlive/id/:liveid/chat/disable", auth(vlive.ChatEnable(app, false)))
	r.POST("/api/v1/vlive/id/:liveid/chat/slow", auth(vlive.ChatSlowMode(app)))

	// WebRTC
	r.GET("/api/v1/vlive/webrtc/turn", auth(vlive.GetTURNServers(app)))
}

func AddDiscordRoutes(router *httprouter.Router, hubs *discord.HubManager, app *infra.Deps) {
	auth := middleware.Authenticate(app)

	router.GET("/ws/discord/:room", auth(discord.WebSocketHandler(hubs)))
	router.POST(
		"/api/v1/discord/:room/message",
		auth(discord.CreateMessageHTTP(app)),
	)

}

// Vendor Routes
func AddVendorRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	authmidware := middleware.Authenticate(app)

	// Vendor management - registration and profile
	router.POST("/api/v1/vendors", rateLimiter.Limit(authmidware(vendors.RegisterVendorHandler(app))))
	router.GET("/api/v1/vendors", rateLimiter.Limit(vendors.GetVendorsHandler(app)))
	router.GET("/api/v1/vendors/vendor/:vendorID", rateLimiter.Limit(vendors.GetVendorHandler(app)))
	router.PATCH("/api/v1/vendors/vendor/:vendorID", rateLimiter.Limit(authmidware(vendors.UpdateVendorHandler(app))))

	// Event vendor hiring
	router.POST("/api/v1/vendors/events/:eventID/hire", rateLimiter.Limit(authmidware(vendors.HireVendorHandler(app))))
	router.GET("/api/v1/vendors/events/:eventID", rateLimiter.Limit(vendors.GetEventVendorsHandler(app)))
	router.DELETE("/api/v1/vendors/events/:eventID/vendor/:vendorID", rateLimiter.Limit(authmidware(vendors.RemoveVendorHandler(app))))
}

// Search Routes - Public endpoints for search functionality
func AddSearchRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	// Autocomplete suggestions - public, rate-limited
	router.GET("/api/v1/ac", rateLimiter.Limit(search.SearchAutocomplete(app)))

	// Search by entity type - public, rate-limited
	router.GET("/api/v1/search/:tabId", rateLimiter.Limit(search.SearchByType(app)))
}
