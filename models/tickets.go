package models

import (
	"time"
)

type RefundRequest struct {
	EventID     string     `bson:"eventid" json:"eventID"`
	TicketID    string     `bson:"ticketid" json:"ticketID"`
	UserID      string     `bson:"userid" json:"userID"`
	UniqueCode  string     `bson:"uniquecode" json:"uniqueCode"`
	RequestDate time.Time  `bson:"requestdate" json:"requestDate"`
	Status      string     `bson:"status" json:"status"` // pending, approved, rejected, refunded
	Amount      int        `bson:"amount" json:"amount,omitempty"`
	ProcessedAt *time.Time `bson:"processedat,omitempty" json:"processedAt,omitempty"`
	RefundedAt  *time.Time `bson:"refundedat,omitempty" json:"refundedAt,omitempty"`
}

type Ticket struct {
	TicketID    string    `json:"ticketid" bson:"ticketid"`
	EventID     string    `json:"eventid" bson:"eventid"`
	Name        string    `json:"name" bson:"name"`
	Price       int64     `json:"price" bson:"price"` // CRITICAL FIX: Changed from float64 to int64 (stored in paise)
	Currency    string    `json:"currency" bson:"currency"`
	Color       string    `json:"color" bson:"color"`
	Quantity    int       `json:"quantity" bson:"quantity"`
	EntityID    string    `json:"entity_id" bson:"entity_id"`
	EntityType  string    `json:"entity_type" bson:"entity_type"` // "event" or "place"
	Available   int       `json:"available" bson:"available"`
	Total       int       `json:"total" bson:"total"`
	CreatedAt   time.Time `json:"created_at" bson:"created_at"`
	Description string    `bson:"description,omitempty" json:"description"`
	Sold        int       `bson:"sold" json:"sold"`
	SeatStart   int       `bson:"seatstart" json:"seatstart"`
	SeatEnd     int       `bson:"seatend" json:"seatend"`
	Seats       []Seat    `bson:"seats" json:"seats"` // 👈 new field
	UpdatedAt   time.Time `bson:"updated_at" json:"updatedAt"`
}

type Seat struct {
	SeatID     string `json:"id" bson:"_id,omitempty"`
	EntityID   string `json:"entity_id" bson:"entity_id"`
	EntityType string `json:"entity_type" bson:"entity_type"` // e.g., "event" or "place"
	SeatNumber string `json:"seat_number" bson:"seat_number"`
	UserID     string `json:"user_id" bson:"user_id,omitempty"`
	Status     string `json:"status" bson:"status"` // e.g., "booked", "available"
}
