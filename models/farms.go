package models

import (
	"fmt"
	"time"
)

type ContactInfo struct {
	Phone   string `bson:"phone,omitempty" json:"phone,omitempty"`
	Email   string `bson:"email,omitempty" json:"email,omitempty"`
	Website string `bson:"website,omitempty" json:"website,omitempty"`
}

type Farm struct {
	FarmID             string      `bson:"farmid,omitempty"         json:"farmid"`
	Name               string      `bson:"name"                  json:"name"`
	Location           string      `bson:"location"              json:"location"`
	Latitude           float64     `bson:"latitude,omitempty"    json:"latitude,omitempty"`
	Longitude          float64     `bson:"longitude,omitempty"   json:"longitude,omitempty"`
	Description        string      `bson:"description,omitempty" json:"description,omitempty"`
	Owner              string      `bson:"owner"                 json:"owner"`
	ContactInfo        ContactInfo `bson:"contactInfo,omitempty" json:"contactInfo,omitempty"`
	AvailabilityTiming string      `bson:"availabilityTiming,omitempty" json:"availabilityTiming,omitempty"`
	Tags               []string    `bson:"tags,omitempty"        json:"tags,omitempty"`
	Banner             string      `bson:"banner,omitempty"       json:"photo,omitempty"`
	Crops              []Crop      `bson:"crops" json:"crops,omitempty"` // loaded via lookup or separate query
	Media              []string    `bson:"media,omitempty"       json:"media,omitempty"`
	AvgRating          float64     `bson:"avgRating,omitempty"   json:"avgRating,omitempty"`
	ReviewCount        int         `bson:"reviewCount,omitempty" json:"reviewCount,omitempty"`
	FavoritesCount     int64       `bson:"favoritesCount,omitempty" json:"favoritesCount,omitempty"`
	CreatedBy          string      `bson:"createdBy"             json:"createdBy"`
	CreatedAt          time.Time   `bson:"createdAt"             json:"createdAt"`
	UpdatedAt          time.Time   `bson:"updatedAt"             json:"updatedAt"`
	Contact            string      `json:"contact"`
}

type PricePoint struct {
	Date  time.Time `json:"date" bson:"date"`
	Price float64   `json:"price" bson:"price"`
}

type Crop struct {
	Name         string       `json:"name"`
	CropId       string       `json:"cropid"`
	Price        float64      `json:"price"`
	Quantity     int          `json:"quantity"`
	Unit         string       `json:"unit"`
	Banner       string       `bson:"banner" json:"banner"`
	Notes        string       `json:"notes,omitempty"`
	Category     string       `json:"category,omitempty"`
	CatalogueId  string       `json:"catalogueid,omitempty"`
	Featured     bool         `json:"featured,omitempty"`
	OutOfStock   bool         `json:"outOfStock,omitempty"`
	HarvestDate  *time.Time   `json:"harvestDate,omitempty"`
	ExpiryDate   *time.Time   `json:"expiryDate,omitempty"`
	UpdatedAt    time.Time    `json:"updatedAt"`
	PriceHistory []PricePoint `json:"priceHistory,omitempty"`
	FieldPlot    string       `json:"fieldPlot,omitempty"`
	CreatedAt    time.Time    `json:"createdAt"`
	CreatedBy    string       `json:"createdby"`
	FarmID       string       `bson:"farmid,omitempty" json:"farmid,omitempty"`
	FarmName     string       `json:"farmName,omitempty"` // CRITICAL FIX: Add farm name for cart/display
}

type IncomingOrder struct {
	ID           string `json:"id"`
	Buyer        string `json:"buyer"`
	Contact      string `json:"contact"`
	Crop         string `json:"crop"`
	Qty          int    `json:"qty"`
	Unit         string `json:"unit"`
	OrderDate    string `json:"orderDate"`
	DeliveryDate string `json:"deliveryDate"`
	Address      string `json:"address"`
	Payment      string `json:"payment"`
	Status       string `json:"status"`
}

type FarmOrder struct {
	OrderID         string                `bson:"orderid,omitempty"  json:"orderid"`
	UserID          string                `bson:"userid"         json:"userid"`
	FarmID          string                `bson:"farmid"         json:"farmid"`
	CropID          string                `bson:"cropid"         json:"cropid"`
	Quantity        int                   `bson:"quantity"       json:"quantity"`
	PriceAtPurchase float64               `bson:"priceAtPurchase" json:"priceAtPurchase"`
	CreatedAt       time.Time             `bson:"createdAt"       json:"createdAt"`
	Status          OrderStatus           `bson:"status"       json:"status"`
	ApprovedBy      []string              `bson:"approved"       json:"approved"`
	Items           map[string][]CartItem `json:"items" bson:"items"`
}

type OrderStatus string

const (
	OrderActive   OrderStatus = "active"
	OrderRejected OrderStatus = "rejected"
	OrderClosed   OrderStatus = "closed"
)

type CropCatalogueItem struct {
	Name       string `json:"name"`
	Category   string `json:"category"`
	Banner     string `bson:"banner" json:"banner"`
	Stock      int    `json:"stock"`
	Unit       string `json:"unit"`
	Featured   bool   `json:"featured"`
	PriceRange []int  `json:"priceRange,omitempty"`
}

type CropListing struct {
	FarmID         string   `json:"farmid"`
	CropId         string   `json:"cropid"`
	FarmName       string   `json:"farmName"`
	Location       string   `json:"location"`
	Breed          string   `json:"breed"`
	PricePerKg     float64  `json:"pricePerKg"`
	AvailableQtyKg int      `json:"availableQtyKg,omitempty"`
	HarvestDate    string   `json:"harvestDate,omitempty"` // ISO string
	Tags           []string `json:"tags,omitempty"`
	Banner         string   `bson:"banner" json:"banner"`
}

type Product struct {
	ProductID   string   `bson:"productid,omitempty" json:"productid"`
	UserID      string   `bson:"userid" json:"userid"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Price       float64  `json:"price"`
	Unit        string   `json:"unit"`
	Images      []string `json:"images"`
	Category    string   `bson:"category" json:"category"`
	Quantity    float64  `bson:"quantity" json:"quantity"`
	Type        string   `bson:"type" json:"type"`

	// Physical product fields
	Size        string            `json:"size,omitempty"`
	Color       string            `json:"color,omitempty"`
	Ingredients string            `json:"ingredients,omitempty"`
	ExpiryDate  string            `json:"expiryDate,omitempty"`
	Weight      string            `json:"weight,omitempty"`
	Specs       map[string]string `json:"specs,omitempty"`

	// Media fields
	Author     string `json:"author,omitempty"`     // book
	ISBN       string `json:"isbn,omitempty"`       // book
	Platform   string `json:"platform,omitempty"`   // software
	Version    string `json:"version,omitempty"`    // software
	License    string `json:"license,omitempty"`    // software
	Instructor string `json:"instructor,omitempty"` // course
	Duration   string `json:"duration,omitempty"`   // course / subscription

	// Subscription fields
	BillingCycle string `json:"billingCycle,omitempty"`
	TrialPeriod  string `json:"trialPeriod,omitempty"`
	Scope        string `json:"scope,omitempty"`

	// Creative / art
	Artist     string `json:"artist,omitempty"`
	Medium     string `json:"medium,omitempty"`
	Dimensions string `json:"dimensions,omitempty"`

	// Vehicle fields
	Engine   string `json:"engine,omitempty"`
	Mileage  string `json:"mileage,omitempty"`
	FuelType string `json:"fuelType,omitempty"`

	Featured      bool      `json:"featured,omitempty"`
	SKU           string    `json:"sku,omitempty"`
	AvailableFrom *SafeTime `bson:"availableFrom,omitempty" json:"availableFrom,omitempty"`
	AvailableTo   *SafeTime `bson:"availableTo,omitempty" json:"availableTo,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type SafeTime struct {
	time.Time
}

type Tool struct {
	ToolID        string    `bson:"toolid,omitempty" json:"toolid"`
	Name          string    `bson:"name" json:"name"`
	Price         float64   `bson:"price" json:"price"`
	Description   string    `bson:"description" json:"description"`
	Banner        string    `bson:"banner" json:"banner"`
	Category      string    `bson:"category" json:"category"`
	SKU           string    `bson:"sku,omitempty" json:"sku,omitempty"`
	AvailableFrom *SafeTime `bson:"availableFrom,omitempty" json:"availableFrom,omitempty"`
	AvailableTo   *SafeTime `bson:"availableTo,omitempty" json:"availableTo,omitempty"`
	Quantity      float64   `bson:"quantity" json:"quantity"`
	Unit          string    `bson:"unit" json:"unit"`
	Featured      bool      `bson:"featured" json:"featured"`
}

// UnmarshalJSON tries RFC3339, then "2006-01-02"
func (st *SafeTime) UnmarshalJSON(b []byte) error {
	s := string(b)
	// strip quotes
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	if s == "" || s == "null" {
		// leave st.Time zero or nil
		return nil
	}
	// Try full RFC3339 first
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		st.Time = t
		return nil
	}
	// Fallback to date-only
	if t, err := time.Parse("2006-01-02", s); err == nil {
		st.Time = t
		return nil
	}
	return fmt.Errorf("invalid date format: %q", s)
}
