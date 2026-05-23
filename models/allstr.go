package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Response struct {
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

type Setting struct {
	Type        string `json:"type"`
	Value       any    `json:"value"`
	Description string `json:"description"`
}

//	type UserSettings struct {
//		UserID   string    `bson:"userID" json:"userID"`
//		Settings []Setting `bson:"settings" json:"settings"`
//	}

type FeedPost struct {
	Username    string `bson:"username" json:"username"`
	PostID      string `bson:"postid,omitempty" json:"postid"`
	UserID      string `bson:"userid" json:"userid"`
	Type        string `bson:"type" json:"type"`
	Text        string `bson:"text,omitempty" json:"text,omitempty"`
	Title       string `bson:"title,omitempty" json:"title,omitempty"`
	Description string `bson:"description,omitempty" json:"description,omitempty"`
	Caption     string `bson:"caption,omitempty" json:"caption,omitempty"`

	Media       []string          `bson:"media,omitempty" json:"media,omitempty"`             // full file paths (key/filename.extn)
	MediaURL    []string          `bson:"media_url,omitempty" json:"media_url,omitempty"`     // clean filenames
	Thumbnail   string            `bson:"thumbnail,omitempty" json:"thumbnail,omitempty"`     // video thumbnail
	Resolutions []int             `bson:"resolutions,omitempty" json:"resolutions,omitempty"` // optional resolutions
	Subtitles   map[string]string `bson:"subtitles,omitempty" json:"subtitles,omitempty"`     // lang → file path
	Tags        []string          `bson:"tags,omitempty" json:"tags,omitempty"`               // hashtags or topics

	Timestamp string    `bson:"timestamp" json:"timestamp"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	Likes     int64     `bson:"likes" json:"likes"`
	Likers    []string  `bson:"likers,omitempty" json:"likers,omitempty"`

	// Legacy / optional field kept for backward compatibility
	Content string `bson:"content,omitempty" json:"content,omitempty"`
}

type Activity struct {
	// Username     string              `json:"username,omitempty" bson:"username,omitempty"`
	PlaceID      string    `json:"placeId,omitempty" bson:"placeId,omitempty"`
	Action       string    `json:"action,omitempty" bson:"action,omitempty"`
	PerformedBy  string    `json:"performedBy,omitempty" bson:"performedBy,omitempty"`
	Timestamp    time.Time `json:"timestamp,omitempty" bson:"timestamp,omitempty"`
	Details      string    `json:"details,omitempty" bson:"details,omitempty"`
	IPAddress    string    `json:"ipAddress,omitempty" bson:"ipAddress,omitempty"`
	DeviceInfo   string    `json:"deviceInfo,omitempty" bson:"deviceInfo,omitempty"`
	ActivityID   string    `json:"activityid" bson:"activityid,omitempty"`
	UserID       string    `json:"user_id" bson:"user_id"`
	ActivityType string    `json:"activity_type" bson:"activity_type"` // e.g., "follow", "review", "buy"
	EntityID     string    `json:"entity_id,omitempty" bson:"entity_id,omitempty"`
	EntityType   *string   `json:"entity_type,omitempty" bson:"entity_type,omitempty"` // "event", "place", or null
}
type Merch struct {
	MerchID string `json:"merchid" bson:"merchid"`
	// EventID     string             `json:"eventid" bson:"eventid"` // Reference to Event ID
	Name        string     `json:"name" bson:"name"`
	Slug        string     `json:"slug,omitempty" bson:"slug,omitempty"`         // URL-friendly name (e.g. "concert-tshirt")
	SKU         string     `json:"sku,omitempty" bson:"sku,omitempty"`           // Stock Keeping Unit, unique per product
	Category    string     `json:"category,omitempty" bson:"category,omitempty"` // e.g. “T-Shirts”, “Accessories”
	Price       float64    `json:"price" bson:"price"`
	Discount    float64    `json:"discount,omitempty" bson:"discount,omitempty"`         // e.g. 0.10 for 10% off
	Stock       int        `json:"stock" bson:"stock"`                                   // Number of items available
	StockStatus string     `json:"stock_status,omitempty" bson:"stock_status,omitempty"` // e.g. “In Stock”, “Out of Stock”, “Preorder”
	MerchPhoto  string     `json:"merch_pic" bson:"merch_pic"`
	Gallery     []string   `json:"gallery,omitempty" bson:"gallery,omitempty"` // Additional image filenames
	EntityID    string     `json:"entity_id" bson:"entity_id"`
	EntityType  string     `json:"entity_type" bson:"entity_type"` // “event” or “place”
	Description string     `json:"description,omitempty" bson:"description,omitempty"`
	ShortDesc   string     `json:"short_desc,omitempty" bson:"short_desc,omitempty"` // One-line summary
	Rating      float64    `json:"rating,omitempty" bson:"rating,omitempty"`         // Average rating (0.0–5.0)
	ReviewCount int        `json:"review_count,omitempty" bson:"review_count,omitempty"`
	Weight      float64    `json:"weight,omitempty" bson:"weight,omitempty"`         // In kilograms/pounds
	Dimensions  string     `json:"dimensions,omitempty" bson:"dimensions,omitempty"` // e.g. “30×20×2 cm”
	Tags        []string   `json:"tags,omitempty" bson:"tags,omitempty"`             // e.g. ["rock", "tshirt"]
	CreatedAt   time.Time  `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" bson:"updatedAt"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty" bson:"deletedAt,omitempty"` // Soft delete timestamp
	UserID      string     `bson:"userid" json:"userid"`
}

type Menu struct {
	MenuID      string    `json:"menuid" bson:"menuid"`
	PlaceID     string    `json:"placeid" bson:"placeid"` // Reference to Place ID
	Name        string    `json:"name" bson:"name"`
	Price       float64   `json:"price" bson:"price"`
	Stock       int       `json:"stock" bson:"stock"` // Number of items available
	MenuPhoto   string    `json:"menu_pic" bson:"menu_pic"`
	Description string    `json:"description,omitempty" bson:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at" bson:"created_at"`
	UserID      string    `bson:"userid" json:"userid"`
	UpdatedAt   time.Time `bson:"updated_at" json:"updatedAt"`
}

// UserProfileResponse defines the structure for the user profile response
type UserSuggest struct {
	Username    string `json:"username" bson:"username"`
	UserID      string `json:"userid" bson:"userid"`
	IsFollowing bool
	Bio         string `json:"bio,omitempty" bson:"bio,omitempty"`
}

type Suggestion struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Type        string             `json:"type" bson:"type"` // e.g., "place" or "event"
	Title       string             `json:"title" bson:"title"`
	Description string             `json:"description,omitempty" bson:"description,omitempty"`
	Name        string             `json:"name"`
}

type Review struct {
	ReviewID string `json:"reviewid" bson:"reviewid"`
	UserID   string `json:"userid" bson:"userid"`

	EntityType string `json:"entityType" bson:"entityType"`
	EntityID   string `json:"entityId" bson:"entityId"`

	Rating  int    `json:"rating" bson:"rating"`
	Comment string `json:"comment" bson:"comment"`

	Likes    int `json:"likes,omitempty" bson:"likes,omitempty"`
	Dislikes int `json:"dislikes,omitempty" bson:"dislikes,omitempty"`

	CreatedAt time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`
}

type Media struct {
	MediaID       string    `json:"mediaid" bson:"mediaid"`
	MediaGroupID  string    `json:"mediaGroupId" bson:"mediaGroupId"` // new field to group multiple files
	Type          string    `json:"type" bson:"type"`                 // "image", "video", "text"
	URL           string    `json:"url,omitempty" bson:"url,omitempty"`
	ThumbnailURL  string    `json:"thumbnailUrl,omitempty" bson:"thumbnailUrl,omitempty"`
	Caption       string    `json:"caption,omitempty" bson:"caption,omitempty"`
	Description   string    `json:"description,omitempty" bson:"description,omitempty"`
	CreatorID     string    `json:"creatorid" bson:"creatorid"`
	LikesCount    int       `json:"likesCount" bson:"likesCount"`
	CommentsCount int       `json:"commentsCount" bson:"commentsCount"`
	Visibility    string    `json:"visibility,omitempty" bson:"visibility,omitempty"`
	Tags          []string  `json:"tags,omitempty" bson:"tags,omitempty"` // e.g., song:123, event:456
	Duration      float64   `json:"duration,omitempty" bson:"duration,omitempty"`
	FileSize      int64     `json:"fileSize,omitempty" bson:"fileSize,omitempty"`
	MimeType      string    `json:"mimeType,omitempty" bson:"mimeType,omitempty"`
	IsFeatured    bool      `json:"isFeatured,omitempty" bson:"isFeatured,omitempty"`
	EntityID      string    `json:"entityid" bson:"entityid"`
	EntityType    string    `json:"entitytype" bson:"entitytype"` // "event", "place", etc.
	CreatedAt     time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt" bson:"updatedAt"`
	UserID        string    `json:"userid" bson:"userid"`
	Extn          string    `json:"extn" bson:"extn"`
	CaptionLang   string    `json:"captionlang" bson:"captionlang"`
}

const (
	MediaTypeImage    = "image"
	MediaTypeVideo    = "video"
	MediaTypePhoto360 = "photo360"
)

type Promotion struct {
	ID          primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	Title       string             `json:"title" bson:"title"`
	Description string             `json:"description" bson:"description"`
	ExpiryDate  time.Time          `json:"expiry_date" bson:"expiry_date"`
}

// Owner Management Handlers
type Owner struct {
	ID       primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	Name     string             `json:"name" bson:"name"`
	Email    string             `json:"email" bson:"email"`
	Password string             `json:"password" bson:"password"`
}
