package models

import (
	"time"
)

type PlacesResponse struct {
	PlaceID        string   `json:"placeid"`
	Name           string   `json:"name"`
	ShortDesc      string   `json:"short_desc"`
	Address        string   `json:"address,omitempty"`
	Distance       float64  `json:"distance,omitempty"`
	OperatingHours []string `json:"operatinghours,omitempty"`
	Category       string   `json:"category"`
	Tags           []string `json:"tags"`
	Banner         string   `json:"banner"`
}

/* ---------- MODELS ---------- */

type BaitosResponse struct {
	BaitoId      string    `bson:"baitoid,omitempty" json:"baitoid"`
	Title        string    `bson:"title" json:"title"`
	Description  string    `bson:"description" json:"description"`
	Category     string    `bson:"category" json:"category"`
	SubCategory  string    `bson:"subcategory" json:"subcategory"`
	Location     string    `bson:"location" json:"location"`
	Wage         string    `bson:"wage" json:"wage"`
	Requirements string    `bson:"requirements" json:"requirements"`
	BannerURL    string    `bson:"banner,omitempty" json:"banner,omitempty"`
	WorkHours    string    `bson:"workHours" json:"workHours"`
	CreatedAt    time.Time `bson:"createdAt" json:"createdAt"`
	OwnerID      string    `bson:"ownerId" json:"ownerId"`
}

type BaitoWorkersResponse struct {
	UserID      string   `json:"userId" bson:"userId"`
	BaitoUserID string   `json:"baitoUserId" bson:"baitoUserId"`
	Name        string   `json:"name" bson:"name"`
	Age         int      `json:"age" bson:"age"`
	Phone       string   `json:"phone" bson:"phone"`
	Location    string   `json:"location" bson:"location"`
	Preferred   []string `json:"preferredRoles" bson:"preferredRoles"`
	Bio         string   `json:"bio" bson:"bio"`
	ProfilePic  string   `json:"profilePic" bson:"profilePic"`
	CreatedAt   int64    `json:"createdAt" bson:"createdAt"`
}

// --- BlogPostResponse for list view ---

type BlogPostResponse struct {
	PostID      string    `bson:"postid" json:"postid"`
	Title       string    `bson:"title" json:"title"`
	Category    string    `bson:"category" json:"category"`
	Subcategory string    `bson:"subcategory" json:"subcategory"`
	ReferenceID *string   `bson:"referenceId,omitempty" json:"referenceId,omitempty"`
	Thumb       string    `bson:"thumb" json:"thumb"`
	CreatedBy   string    `bson:"createdBy" json:"createdBy"`
	Username    string    `bson:"username" json:"username"`
	CreatedAt   time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time `bson:"updatedAt" json:"updatedAt"`
}
