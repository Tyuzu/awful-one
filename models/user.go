package models

import "time"

type User struct {
	// ID          string    `json:"-" bson:"_id,omitempty"`
	UserID       string    `json:"userid" bson:"userid"`
	Username     string    `json:"username" bson:"username"`
	Email        string    `json:"email" bson:"email"`
	Password     string    `json:"-" bson:"password"`
	PasswordHash string    `json:"password_hash" bson:"password_hash"`
	Role         []string  `json:"role" bson:"role"`
	Name         string    `json:"name,omitempty" bson:"name,omitempty"`
	CreatedAt    time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" bson:"updated_at"`
	Bio          string    `json:"bio,omitempty" bson:"bio,omitempty"`
	Online       bool      `json:"online"`
	LastLogin    time.Time `json:"last_login" bson:"last_login"`
	Avatar       string    `json:"avatar" bson:"avatar"`
	Banner       string    `json:"banner" bson:"banner"`
	ProfileViews int       `json:"profile_views,omitempty" bson:"profile_views,omitempty"`
	PhoneNumber  string    `json:"phone_number,omitempty" bson:"phone_number,omitempty"`
	Address      string    `json:"address,omitempty" bson:"address,omitempty"`
	// DateOfBirth    time.Time         `json:"dob" bson:"dob"`
	SocialLinks    map[string]string `json:"social_links,omitempty" bson:"social_links,omitempty"`
	IsVerified     bool              `json:"is_verified" bson:"is_verified"`
	EmailVerified  bool              `json:"email_verified" bson:"email_verified"`
	FollowersCount int               `json:"followerscount" bson:"followerscount"`
	FollowingCount int               `json:"followscount" bson:"followscount"`
	WalletBalance  float64           `bson:"wallet_balance" json:"wallet_balance"`
	RefreshToken   string            `json:"-" bson:"refresh_token,omitempty"`
	RefreshExpiry  time.Time         `json:"-" bson:"refresh_expiry,omitempty"`
	RefreshUA      string            `bson:"refresh_ua,omitempty"`
	RefreshIP      string            `bson:"refresh_ip,omitempty"`
	RefreshPrev    string            `bson:"refresh_prev,omitempty"`
	// Vendor fields
	IsVendor      bool           `json:"is_vendor" bson:"is_vendor"`
	VendorProfile *VendorProfile `json:"vendor_profile,omitempty" bson:"vendor_profile,omitempty"`
}

// VendorProfile contains vendor-specific information for a user
type VendorProfile struct {
	VendorID    string   `json:"vendorid" bson:"vendorid"`
	Category    string   `json:"category" bson:"category"`
	Description string   `json:"description,omitempty" bson:"description,omitempty"`
	Rating      float64  `json:"rating" bson:"rating"`
	RatingCount int      `json:"rating_count" bson:"rating_count"`
	Portfolio   []string `json:"portfolio,omitempty" bson:"portfolio,omitempty"`
	Verified    bool     `json:"verified" bson:"verified"`
}

// UserProfileResponse defines the structure for the user profile response
type UserProfileResponse struct {
	UserID         string            `json:"userid" bson:"userid"`
	Username       string            `json:"username" bson:"username"`
	Name           string            `json:"name" bson:"name"`
	Email          string            `json:"email" bson:"email"`
	Bio            string            `json:"bio,omitempty" bson:"bio,omitempty"`
	PhoneNumber    string            `json:"phone_number,omitempty" bson:"phone_number,omitempty"`
	Avatar         string            `json:"avatar" bson:"avatar"`
	Banner         string            `json:"banner" bson:"banner"`
	IsFollowing    bool              `json:"is_following" bson:"is_following"` // Added here
	FollowersCount int               `json:"followerscount" bson:"followerscount"`
	FollowingCount int               `json:"followscount" bson:"followscount"`
	SocialLinks    map[string]string `json:"social_links,omitempty" bson:"social_links,omitempty"`
	Online         bool              `json:"online,omitempty"`
	LastLogin      time.Time         `json:"last_login" bson:"last_login"`
}

type UserFollow struct {
	UserID    string   `json:"userid" bson:"userid"`
	Follows   []string `json:"follows,omitempty" bson:"follows,omitempty"`
	Followers []string `json:"followers,omitempty" bson:"followers,omitempty"`
}

type UserSubscribe struct {
	UserID      string   `json:"userid" bson:"userid"`
	Subscribed  []string `json:"subscribed,omitempty" bson:"subscribed,omitempty"`   // users this user is subscribed to
	Subscribers []string `json:"subscribers,omitempty" bson:"subscribers,omitempty"` // users who subscribed to this user
}

type UserData struct {
	UserID     string `json:"userid" bson:"userid"`
	EntityID   string `json:"entity_id" bson:"entity_id"`
	EntityType string `json:"entity_type" bson:"entity_type"`
	ItemID     string `json:"item_id" bson:"item_id"`
	ItemType   string `json:"item_type" bson:"item_type"`
	CreatedAt  string `json:"created_at" bson:"created_at"`
}
