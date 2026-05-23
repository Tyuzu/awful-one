package routes

import (
	"naevis/infra"
	"naevis/middleware"
	places "naevis/places/tabs"

	"github.com/julienschmidt/httprouter"
)

// 🍽️ Restaurant / Café → Menu
func DisplayPlaceMenu(router *httprouter.Router, app *infra.Deps) {
	router.GET("/api/v1/place/:placeid/menu", places.GetMenuTab)
	router.POST("/api/v1/place/:placeid/menu", places.PostMenuTab)
	router.PUT("/api/v1/place/:placeid/menu/:itemId", places.PutMenuTab)
	router.DELETE("/api/v1/place/:placeid/menu/:itemId", places.DeleteMenuTab)
	router.POST("/api/v1/place/:placeid/menu/:itemId/order", places.PostMenuOrder)
}

// 🏨 Hotel → Rooms
func DisplayPlaceRooms(router *httprouter.Router, app *infra.Deps) {
	router.GET("/api/v1/place/:placeid/rooms", places.GetRooms)
	router.GET("/api/v1/place/:placeid/rooms/:roomId", places.GetRoom)
	router.POST("/api/v1/place/:placeid/rooms", places.PostRoom)
	router.PUT("/api/v1/place/:placeid/rooms/:roomId", places.PutRoom)
	router.DELETE("/api/v1/place/:placeid/rooms/:roomId", places.DeleteRoom)
}

// 🌳 Park → Facilities
func DisplayPlaceFacilities(router *httprouter.Router, app *infra.Deps) {
	router.GET("/api/v1/place/:placeid/facilities", places.GetFacilities)
	router.POST("/api/v1/place/:placeid/facilities", places.PostFacility)
	router.PUT("/api/v1/place/:placeid/facilities/:facilityId", places.PutFacility)
	router.GET("/api/v1/place/:placeid/facilities/:facilityId", places.GetFacility)
	router.DELETE("/api/v1/place/:placeid/facilities/:facilityId", places.DeleteFacility)
}

// 🏢 Business → Services
func DisplayPlaceServices(router *httprouter.Router, app *infra.Deps) {
	router.GET("/api/v1/place/:placeid/services", places.GetServices)
	router.POST("/api/v1/place/:placeid/services", places.PostService)
	router.PUT("/api/v1/place/:placeid/services/:serviceId", places.PutService)
	router.GET("/api/v1/place/:placeid/services/:serviceId", places.GetService)
	router.DELETE("/api/v1/place/:placeid/services/:serviceId", places.DeleteService)
}

// 🛍️ Shop → Products
func DisplayPlaceProducts(router *httprouter.Router, app *infra.Deps) {
	router.GET("/api/v1/place/:placeid/products", places.GetProducts(app))
	router.POST("/api/v1/place/:placeid/products", places.PostProduct(app))
	router.PUT("/api/v1/place/:placeid/products/:productId", places.PutProduct(app))
	router.GET("/api/v1/place/:placeid/products/:productId", places.GetProduct(app))
	router.DELETE("/api/v1/place/:placeid/products/:productId", places.DeleteProduct(app))
	router.POST("/api/v1/place/:placeid/products/:productId/buy", places.PostProductPurchase(app))
}

// 🖼️ Museum → Exhibits
func DisplayPlaceExhibits(router *httprouter.Router, app *infra.Deps) {
	router.GET("/api/v1/place/:placeid/exhibits", places.GetExhibits)
	router.POST("/api/v1/place/:placeid/exhibits", places.PostExhibit)
	router.PUT("/api/v1/place/:placeid/exhibits/:exhibitId", places.PutExhibit)
	router.GET("/api/v1/place/:placeid/exhibits/:exhibitId", places.GetExhibit)
	router.DELETE("/api/v1/place/:placeid/exhibits/:exhibitId", places.DeleteExhibit)
}

// 🏋️ Gym → Membership
func DisplayPlaceMembership(router *httprouter.Router, app *infra.Deps) {
	router.GET("/api/v1/place/:placeid/membership", places.GetMemberships)
	router.POST("/api/v1/place/:placeid/membership", places.PostMembership)
	router.PUT("/api/v1/place/:placeid/membership/:membershipId", places.PutMembership)
	router.GET("/api/v1/place/:placeid/membership/:membershipId", places.GetMembership)
	router.DELETE("/api/v1/place/:placeid/membership/:membershipId", places.DeleteMembership)
	router.POST("/api/v1/place/:placeid/membership/:membershipId/join", places.PostJoinMembership)
}

// 🎭 Theater → Shows
func DisplayPlaceShows(router *httprouter.Router, app *infra.Deps) {
	router.GET("/api/v1/place/:placeid/shows", places.GetShows)
	router.POST("/api/v1/place/:placeid/shows", places.PostShow)
	router.PUT("/api/v1/place/:placeid/shows/:showId", places.PutShow)
	router.GET("/api/v1/place/:placeid/shows/:showId", places.GetShow)
	router.DELETE("/api/v1/place/:placeid/shows/:showId", places.DeleteShow)
	router.POST("/api/v1/place/:placeid/shows/:showId/book", places.PostBookShow)
}

// 🏟️ Arena → Events
func DisplayPlaceEvents(router *httprouter.Router, app *infra.Deps) {
	router.GET("/api/v1/place/:placeid/events", places.GetEvents(app))
	router.POST("/api/v1/place/:placeid/events", places.PostEvent(app))
	router.PUT("/api/v1/place/:placeid/events/:eventId", places.PutEvent(app))
	router.GET("/api/v1/place/:placeid/events/:eventId", places.GetEvent(app))
	router.DELETE("/api/v1/place/:placeid/events/:eventId", places.DeleteEvent(app))
	router.POST("/api/v1/place/:placeid/events/:eventId/view", places.PostViewEventDetails(app))
}

// 💈 Saloon → Slots (if applicable)
func DisplaySaloonSlots(router *httprouter.Router, app *infra.Deps) {
	router.GET("/api/v1/place/:placeid/saloon/slots", places.GetSaloonSlots)
	router.POST("/api/v1/place/:placeid/saloon/slots", places.PostSaloonSlot)
	router.PUT("/api/v1/place/:placeid/saloon/slots/:slotId", places.PutSaloonSlot)
	router.DELETE("/api/v1/place/:placeid/saloon/slots/:slotId", places.DeleteSaloonSlot)
	router.POST("/api/v1/place/:placeid/saloon/slots/:slotId/book", places.BookSaloonSlot)
}

// ❓ Fallback → Generic Place Info
func DisplayPlaceDetailsFallback(router *httprouter.Router, app *infra.Deps) {
	router.GET("/api/v1/place/:placeid/details", places.GetDetailsFallback)
}

func AddPlaceTabRoutes(router *httprouter.Router, app *infra.Deps, rateLimiter *middleware.RateLimiter) {
	DisplayPlaceMenu(router, app)
	DisplayPlaceRooms(router, app)
	DisplayPlaceFacilities(router, app)
	DisplayPlaceServices(router, app)
	DisplayPlaceProducts(router, app)
	DisplayPlaceExhibits(router, app)
	DisplayPlaceMembership(router, app)
	DisplayPlaceShows(router, app)
	DisplayPlaceEvents(router, app)
	DisplaySaloonSlots(router, app)
	DisplayPlaceDetailsFallback(router, app)
}
