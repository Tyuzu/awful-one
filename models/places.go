package models

import "time"

type Place struct {
	PlaceID           string            `json:"placeid" bson:"placeid"`
	Name              string            `json:"name" bson:"name"`
	ShortDesc         string            `json:"short_desc" bson:"short_desc"`
	Description       string            `json:"description" bson:"description"`
	Place             string            `json:"place" bson:"place"`
	Capacity          int               `json:"capacity" bson:"capacity"`
	Date              time.Time         `json:"date" bson:"date"`
	Address           string            `json:"address" bson:"address"`
	CreatedBy         string            `json:"createdBy,omitempty" bson:"createdBy,omitempty"`
	OrganizerName     string            `json:"organizer_name" bson:"organizer_name"`
	OrganizerContact  string            `json:"organizer_contact" bson:"organizer_contact"`
	Category          string            `json:"category" bson:"category"`
	Banner            string            `json:"banner" bson:"banner"`
	WebsiteURL        string            `json:"website_url" bson:"website_url"`
	Status            string            `json:"status" bson:"status"`
	AccessibilityInfo string            `json:"accessibility_info" bson:"accessibility_info"`
	SocialMediaLinks  []string          `json:"social_links" bson:"social_links"`
	Tags              []string          `json:"tags" bson:"tags"`
	CustomFields      map[string]any    `json:"custom_fields" bson:"custom_fields"`
	CreatedAt         time.Time         `json:"created_at" bson:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at" bson:"updated_at"`
	City              string            `json:"city,omitempty" bson:"city,omitempty"`
	Country           string            `json:"country,omitempty" bson:"country,omitempty"`
	ZipCode           string            `json:"zipCode,omitempty" bson:"zipCode,omitempty"`
	Jobs              string            `json:"jobs,omitempty" bson:"jobs,omitempty"`
	Location          Coordinates       `json:"location" bson:"location,omitempty"`
	Phone             string            `json:"phone,omitempty" bson:"phone,omitempty"`
	Website           string            `json:"website,omitempty" bson:"website,omitempty"`
	IsOpen            bool              `json:"isopen,omitempty" bson:"isopen,omitempty"`
	Distance          float64           `json:"distance,omitempty" bson:"distance,omitempty"`
	Views             int               `json:"views,omitempty" bson:"views,omitempty"`
	ReviewCount       int               `json:"reviewcount,omitempty" bson:"reviewcount,omitempty"`
	SocialLinks       map[string]string `json:"socialLinks,omitempty" bson:"socialLinks,omitempty"`
	UpdatedBy         string            `json:"updatedBy,omitempty" bson:"updatedBy,omitempty"`
	DeletedAt         *time.Time        `json:"deletedAt,omitempty" bson:"deletedAt,omitempty"`
	Amenities         []string          `json:"amenities,omitempty" bson:"amenities,omitempty"`
	Events            []string          `json:"events,omitempty" bson:"events,omitempty"`
	OperatingHours    []string          `json:"operatinghours,omitempty" bson:"operatinghours,omitempty"`
	Keywords          []string          `json:"keywords,omitempty" bson:"keywords,omitempty"`
}

type PlaceStatus string

const (
	PlaceActive   PlaceStatus = "active"
	PlaceInactive PlaceStatus = "inactive"
	PlaceClosed   PlaceStatus = "closed"
)

type Coordinates struct {
	Latitude  float64 `json:"latitude,omitempty" bson:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty" bson:"longitude,omitempty"`
}

type CheckIn struct {
	UserID    string    `json:"userId,omitempty" bson:"userId,omitempty"`
	PlaceID   string    `json:"placeId,omitempty" bson:"placeId,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty" bson:"timestamp,omitempty"`
	Comment   string    `json:"comment,omitempty" bson:"comment,omitempty"`
	Rating    float64   `json:"rating,omitempty" bson:"rating,omitempty"` // Optional
	Medias    []Media   `json:"images,omitempty" bson:"images,omitempty"` // Optional
}

type PlaceVersion struct {
	PlaceID   string            `json:"placeId,omitempty" bson:"placeId,omitempty"`
	Version   int               `json:"version,omitempty" bson:"version,omitempty"`
	Data      Place             `json:"data,omitempty" bson:"data,omitempty"`
	UpdatedAt time.Time         `json:"updatedAt,omitempty" bson:"updatedAt,omitempty"`
	UpdatedBy string            `json:"updatedBy,omitempty" bson:"updatedBy,omitempty"`
	Changes   map[string]string `json:"changes,omitempty" bson:"changes,omitempty"`
}

type OperatingHours struct {
	Day          []string `json:"day,omitempty" bson:"day,omitempty"`
	OpeningHours []string `json:"opening,omitempty" bson:"opening,omitempty"`
	ClosingHours []string `json:"closing,omitempty" bson:"closing,omitempty"`
	TimeZone     string   `json:"timeZone,omitempty" bson:"timeZone,omitempty"`
}

type Tag struct {
	ID     string   `json:"id,omitempty" bson:"_id,omitempty"`
	Name   string   `json:"name,omitempty" bson:"name,omitempty"`
	Places []string `json:"places,omitempty" bson:"places,omitempty"` // List of Place IDs tagged with this keyword
}

const (
	PlaceStatusActive     = "active"
	PlaceStatusClosed     = "closed"
	PlaceStatusRenovation = "under renovation"
)
