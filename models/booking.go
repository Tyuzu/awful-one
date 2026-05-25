package models

// Tier defines a pricing/capacity tier for bookings
type Tier struct {
	ID         string   `json:"id" bson:"id"`
	EntityType string   `json:"entityType" bson:"entityType"`
	EntityId   string   `json:"entityId" bson:"entityId"`
	Name       string   `json:"name" bson:"name"`
	Price      float64  `json:"price" bson:"price"`
	Capacity   int      `json:"capacity" bson:"capacity"`
	TimeRange  []string `json:"timeRange,omitempty" bson:"timeRange,omitempty"`   // ["09:00", "17:00"]
	DaysOfWeek []int    `json:"daysOfWeek,omitempty" bson:"daysOfWeek,omitempty"` // 0=Sun..6=Sat
	Features   []string `json:"features,omitempty" bson:"features,omitempty"`
	CreatedAt  int64    `json:"createdAt" bson:"createdAt"`
}

// Slot represents an available time slot for booking
type Slot struct {
	ID         string `json:"id" bson:"id"`
	EntityType string `json:"entityType" bson:"entityType"`
	EntityId   string `json:"entityId" bson:"entityId"`
	Date       string `json:"date" bson:"date"`
	Start      string `json:"start" bson:"start"`
	End        string `json:"end,omitempty" bson:"end,omitempty"`
	Capacity   int    `json:"capacity" bson:"capacity"`
	TierId     string `json:"tierId,omitempty" bson:"tierId,omitempty"`
	TierName   string `json:"tierName,omitempty" bson:"tierName,omitempty"`
	CreatedAt  int64  `json:"createdAt" bson:"createdAt"`
}

// Booking represents a user's booking of a slot or tier
type Booking struct {
	ID         string  `json:"id" bson:"id"`
	SlotId     string  `json:"slotId,omitempty" bson:"slotId,omitempty"`
	TierId     string  `json:"tierId,omitempty" bson:"tierId,omitempty"`
	TierName   string  `json:"tierName,omitempty" bson:"tierName,omitempty"`
	PricePaid  float64 `json:"pricePaid,omitempty" bson:"pricePaid,omitempty"`
	EntityType string  `json:"entityType" bson:"entityType"`
	EntityId   string  `json:"entityId" bson:"entityId"`
	UserId     string  `json:"userId" bson:"userId"`
	Date       string  `json:"date" bson:"date"`
	Start      string  `json:"start" bson:"start"`
	End        string  `json:"end,omitempty" bson:"end,omitempty"`
	Status     string  `json:"status" bson:"status"` // pending, confirmed, cancelled
	CreatedAt  int64   `json:"createdAt" bson:"createdAt"`
}

// DateCap represents the capacity limit for a specific date
type DateCap struct {
	EntityType string `json:"entityType" bson:"entityType"`
	EntityId   string `json:"entityId" bson:"entityId"`
	Date       string `json:"date" bson:"date"`
	Capacity   int    `json:"capacity" bson:"capacity"`
}
