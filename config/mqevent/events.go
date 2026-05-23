package mqevent

import "time"

/* ============================================================
   AUTH EVENTS
============================================================ */

const (
	UserRegistered = "auth.user.registered"
	UserLoggedIn   = "auth.user.logged_in"
	UserLoggedOut  = "auth.user.logged_out"

	PasswordResetRequested = "auth.password_reset.requested"
	PasswordResetCompleted = "auth.password_reset.completed"
)

/* ============================================================
   AUTH PAYLOADS
============================================================ */

type UserRegisteredPayload struct {
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type UserLoggedInPayload struct {
	UserID     string    `json:"user_id"`
	Username   string    `json:"username"`
	OccurredAt time.Time `json:"occurred_at"`
	IP         string    `json:"ip"`
}

type UserLoggedOutPayload struct {
	UserID     string    `json:"user_id"`
	OccurredAt time.Time `json:"occurred_at"`
}

/* ============================================================
   CART EVENTS
============================================================ */

const (
	CartItemAdded = "cart.item.added"
)

type CartItemAddedPayload struct {
	UserID     string    `json:"user_id"`
	ProductID  string    `json:"product_id"`
	Quantity   int       `json:"quantity"`
	OccurredAt time.Time `json:"occurred_at"`
}

/* ============================================================
   CHECKOUT EVENTS
============================================================ */

const (
	CheckoutStarted = "checkout.started"
	CheckoutPaid    = "checkout.paid"
	CheckoutFailed  = "checkout.failed"
)

type CheckoutStartedPayload struct {
	CheckoutID string    `json:"checkout_id"`
	UserID     string    `json:"user_id"`
	OccurredAt time.Time `json:"occurred_at"`
}

type CheckoutPaidPayload struct {
	CheckoutID string    `json:"checkout_id"`
	OrderID    string    `json:"order_id"`
	UserID     string    `json:"user_id"`
	Amount     int64     `json:"amount"`
	OccurredAt time.Time `json:"occurred_at"`
}

type CheckoutFailedPayload struct {
	CheckoutID string    `json:"checkout_id"`
	UserID     string    `json:"user_id"`
	Reason     string    `json:"reason"`
	OccurredAt time.Time `json:"occurred_at"`
}

/* ============================================================
   ORDER EVENTS
============================================================ */

const (
	OrderCreated        = "order.created"
	OrderPaid           = "order.paid"
	OrderSellerAccepted = "order.seller.accepted"
	OrderSellerRejected = "order.seller.rejected"
	OrderShipped        = "order.shipped"
	OrderDelivered      = "order.delivered"
	OrderRefunded       = "order.refunded"
)

type OrderCreatedPayload struct {
	OrderID    string    `json:"order_id"`
	UserID     string    `json:"user_id"`
	Amount     int64     `json:"amount"`
	OccurredAt time.Time `json:"occurred_at"`
}

type OrderPaidPayload struct {
	OrderID string    `json:"order_id"`
	UserID  string    `json:"user_id"`
	PaidAt  time.Time `json:"paid_at"`
}

type OrderSellerAcceptedPayload struct {
	OrderID    string    `json:"order_id"`
	SellerID   string    `json:"seller_id"`
	OccurredAt time.Time `json:"occurred_at"`
}

type OrderSellerRejectedPayload struct {
	OrderID    string    `json:"order_id"`
	SellerID   string    `json:"seller_id"`
	Reason     string    `json:"reason"`
	OccurredAt time.Time `json:"occurred_at"`
}

type OrderShippedPayload struct {
	OrderID      string    `json:"order_id"`
	TrackingCode string    `json:"tracking_code"`
	Carrier      string    `json:"carrier"`
	OccurredAt   time.Time `json:"occurred_at"`
}

type OrderDeliveredPayload struct {
	OrderID     string    `json:"order_id"`
	DeliveredAt time.Time `json:"delivered_at"`
}

type OrderRefundedPayload struct {
	OrderID    string    `json:"order_id"`
	Reason     string    `json:"reason"`
	OccurredAt time.Time `json:"occurred_at"`
}

/* ============================================================
   ESCROW EVENTS
============================================================ */

const (
	EscrowHeld     = "escrow.held"
	EscrowReleased = "escrow.released"
	EscrowRefunded = "escrow.refunded"
)

type EscrowHeldPayload struct {
	OrderID    string    `json:"order_id"`
	Amount     int64     `json:"amount"`
	OccurredAt time.Time `json:"occurred_at"`
}

type EscrowReleasedPayload struct {
	OrderID    string    `json:"order_id"`
	Amount     int64     `json:"amount"`
	OccurredAt time.Time `json:"occurred_at"`
}

type EscrowRefundedPayload struct {
	OrderID    string    `json:"order_id"`
	Amount     int64     `json:"amount"`
	OccurredAt time.Time `json:"occurred_at"`
}

/* ============================================================
   LISTING EVENTS
============================================================ */

const (
	ListingCreated  = "listing.created"
	ListingUpdated  = "listing.updated"
	ListingRemoved  = "listing.removed"
	ListingRejected = "listing.rejected"
)

type ListingCreatedPayload struct {
	ListingID  string    `json:"listing_id"`
	SellerID   string    `json:"seller_id"`
	OccurredAt time.Time `json:"occurred_at"`
}

type ListingUpdatedPayload struct {
	ListingID  string    `json:"listing_id"`
	SellerID   string    `json:"seller_id"`
	OccurredAt time.Time `json:"occurred_at"`
}

type ListingRemovedPayload struct {
	ListingID  string    `json:"listing_id"`
	SellerID   string    `json:"seller_id"`
	OccurredAt time.Time `json:"occurred_at"`
}

type ListingRejectedPayload struct {
	ListingID  string    `json:"listing_id"`
	SellerID   string    `json:"seller_id"`
	Reason     string    `json:"reason"`
	OccurredAt time.Time `json:"occurred_at"`
}

/* ============================================================
   WISHLIST EVENTS
============================================================ */

const (
	WishlistAdded   = "wishlist.added"
	WishlistRemoved = "wishlist.removed"
)

type WishlistAddedPayload struct {
	UserID     string    `json:"user_id"`
	ProductID  string    `json:"product_id"`
	OccurredAt time.Time `json:"occurred_at"`
}

type WishlistRemovedPayload struct {
	UserID     string    `json:"user_id"`
	ProductID  string    `json:"product_id"`
	OccurredAt time.Time `json:"occurred_at"`
}

/* ============================================================
   REFUND EVENTS
============================================================ */

const (
	RefundRequested = "refund.requested"
	RefundAccepted  = "refund.accepted"
	RefundRejected  = "refund.rejected"
	RefundForced    = "refund.forced"
	RefundCompleted = "refund.completed"
)

type RefundRequestedPayload struct {
	RefundID   string    `json:"refund_id"`
	OrderID    string    `json:"order_id"`
	UserID     string    `json:"user_id"`
	Reason     string    `json:"reason"`
	OccurredAt time.Time `json:"occurred_at"`
}

type RefundAcceptedPayload struct {
	RefundID   string    `json:"refund_id"`
	OrderID    string    `json:"order_id"`
	OccurredAt time.Time `json:"occurred_at"`
}

type RefundRejectedPayload struct {
	RefundID   string    `json:"refund_id"`
	OrderID    string    `json:"order_id"`
	Reason     string    `json:"reason"`
	OccurredAt time.Time `json:"occurred_at"`
}

type RefundForcedPayload struct {
	RefundID   string    `json:"refund_id"`
	OrderID    string    `json:"order_id"`
	AdminID    string    `json:"admin_id"`
	OccurredAt time.Time `json:"occurred_at"`
}

type RefundCompletedPayload struct {
	RefundID    string    `json:"refund_id"`
	OrderID     string    `json:"order_id"`
	CompletedAt time.Time `json:"completed_at"`
}

/* ============================================================
   POST EVENTS
============================================================ */

const (
	PostCreated = "post.created"
	PostUpdated = "post.updated"
	PostDeleted = "post.deleted"
)

type PostCreatedPayload struct {
	PostID     string    `json:"post_id"`
	UserID     string    `json:"user_id"`
	Username   string    `json:"username"`
	PostType   string    `json:"post_type"`
	OccurredAt time.Time `json:"occurred_at"`
}

type PostUpdatedPayload struct {
	PostID     string    `json:"post_id"`
	UserID     string    `json:"user_id"`
	OccurredAt time.Time `json:"occurred_at"`
}

type PostDeletedPayload struct {
	PostID     string    `json:"post_id"`
	UserID     string    `json:"user_id"`
	OccurredAt time.Time `json:"occurred_at"`
}

/* ============================================================
   FARM EVENTS
============================================================ */

const (
	FarmCreated    = "farm.created"
	FarmUpdated    = "farm.updated"
	CropCreated    = "crop.created"
	CropUpdated    = "crop.updated"
	ProductCreated = "product.created"
	ProductUpdated = "product.updated"
)

type FarmCreatedPayload struct {
	FarmID     string    `json:"farm_id"`
	UserID     string    `json:"user_id"`
	FarmName   string    `json:"farm_name"`
	Location   string    `json:"location"`
	OccurredAt time.Time `json:"occurred_at"`
}

type FarmUpdatedPayload struct {
	FarmID     string    `json:"farm_id"`
	UserID     string    `json:"user_id"`
	OccurredAt time.Time `json:"occurred_at"`
}

type CropCreatedPayload struct {
	CropID     string    `json:"crop_id"`
	FarmID     string    `json:"farm_id"`
	UserID     string    `json:"user_id"`
	CropName   string    `json:"crop_name"`
	OccurredAt time.Time `json:"occurred_at"`
}

type CropUpdatedPayload struct {
	CropID     string    `json:"crop_id"`
	FarmID     string    `json:"farm_id"`
	UserID     string    `json:"user_id"`
	OccurredAt time.Time `json:"occurred_at"`
}

type ProductCreatedPayload struct {
	ProductID   string    `json:"product_id"`
	UserID      string    `json:"user_id"`
	ProductName string    `json:"product_name"`
	OccurredAt  time.Time `json:"occurred_at"`
}

type ProductUpdatedPayload struct {
	ProductID  string    `json:"product_id"`
	UserID     string    `json:"user_id"`
	OccurredAt time.Time `json:"occurred_at"`
}

/* ============================================================
   ARTIST EVENTS
============================================================ */

const (
	ArtistCreated = "artist.created"
	ArtistUpdated = "artist.updated"
	SongCreated   = "song.created"
	SongUpdated   = "song.updated"
)

type ArtistCreatedPayload struct {
	ArtistID   string    `json:"artist_id"`
	UserID     string    `json:"user_id"`
	ArtistName string    `json:"artist_name"`
	OccurredAt time.Time `json:"occurred_at"`
}

type ArtistUpdatedPayload struct {
	ArtistID   string    `json:"artist_id"`
	UserID     string    `json:"user_id"`
	OccurredAt time.Time `json:"occurred_at"`
}

type SongCreatedPayload struct {
	SongID     string    `json:"song_id"`
	ArtistID   string    `json:"artist_id"`
	SongTitle  string    `json:"song_title"`
	OccurredAt time.Time `json:"occurred_at"`
}

type SongUpdatedPayload struct {
	SongID     string    `json:"song_id"`
	ArtistID   string    `json:"artist_id"`
	OccurredAt time.Time `json:"occurred_at"`
}

/* ============================================================
   MEDIA UPLOAD EVENTS
============================================================ */

const (
	MediaUploaded = "media.uploaded"
)

type MediaUploadedPayload struct {
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
	FilePath   string `json:"file_path"`
	Extension  string `json:"extension"`
	FileName   string `json:"file_name"`
	Timestamp  int64  `json:"timestamp"`
}
