package models

import "time"

// Vendor represents a vendor who can be hired for events
type Vendor struct {
	VendorID     string    `json:"vendorid" bson:"vendorid"`
	UserID       string    `json:"userid" bson:"userid"`
	Name         string    `json:"name" bson:"name"`
	Category     string    `json:"category" bson:"category"`
	Description  string    `json:"description,omitempty" bson:"description,omitempty"`
	Email        string    `json:"email,omitempty" bson:"email,omitempty"`
	Phone        string    `json:"phone,omitempty" bson:"phone,omitempty"`
	Location     string    `json:"location,omitempty" bson:"location,omitempty"`
	Rating       float64   `json:"rating,omitempty" bson:"rating,omitempty"`
	RatingCount  int       `json:"rating_count,omitempty" bson:"rating_count,omitempty"`
	ProfileImage string    `json:"profile_image,omitempty" bson:"profile_image,omitempty"`
	Portfolio    []string  `json:"portfolio,omitempty" bson:"portfolio,omitempty"`
	Verified     bool      `json:"verified" bson:"verified"`
	Available    bool      `json:"available" bson:"available"`
	CreatedAt    time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time `json:"updated_at,omitempty" bson:"updated_at,omitempty"`
}

// VendorHiring represents the relationship between an event and hired vendors
type VendorHiring struct {
	HiringID       string    `json:"hiringid" bson:"hiringid"`
	EventID        string    `json:"eventid" bson:"eventid"`
	VendorID       string    `json:"vendorid" bson:"vendorid"`
	VendorName     string    `json:"vendor_name" bson:"vendor_name"`
	VendorCategory string    `json:"vendor_category" bson:"vendor_category"`
	HiredAt        time.Time `json:"hired_at" bson:"hired_at"`
	HiredBy        string    `json:"hired_by" bson:"hired_by"` // UserID of event creator/organizer
	Status         string    `json:"status" bson:"status"`     // "hired", "accepted", "rejected", "completed"
	Notes          string    `json:"notes,omitempty" bson:"notes,omitempty"`
	CreatedAt      time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt      time.Time `json:"updated_at,omitempty" bson:"updated_at,omitempty"`
}

// VendorResponse is the response structure for vendor data
type VendorResponse struct {
	VendorID     string   `json:"vendorid"`
	Name         string   `json:"name"`
	Category     string   `json:"category"`
	Description  string   `json:"description,omitempty"`
	Email        string   `json:"email,omitempty"`
	Phone        string   `json:"phone,omitempty"`
	Location     string   `json:"location,omitempty"`
	Rating       float64  `json:"rating,omitempty"`
	RatingCount  int      `json:"rating_count,omitempty"`
	ProfileImage string   `json:"profile_image,omitempty"`
	Portfolio    []string `json:"portfolio,omitempty"`
	Verified     bool     `json:"verified"`
}
