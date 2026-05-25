package models

import (
	"time"
)

type Artist struct {
	ArtistID  string            `bson:"artistid,omitempty" json:"artistid"`
	Category  string            `bson:"category" json:"category"`
	Name      string            `bson:"name" json:"name"`
	Place     string            `bson:"place" json:"place"`
	Country   string            `bson:"country" json:"country"`
	Bio       string            `bson:"bio" json:"bio"`
	DOB       string            `bson:"dob" json:"dob"`
	Photo     string            `bson:"photo" json:"photo"`
	Banner    string            `bson:"banner" json:"banner"`
	Genres    []string          `bson:"genres" json:"genres"`
	Socials   map[string]string `bson:"socials" json:"socials"`
	EventIDs  []string          `bson:"events" json:"events"`
	Members   []BandMember      `bson:"members,omitempty" json:"members,omitempty"` // ✅ ADD THIS
	CreatedAt time.Time         `json:"createdAt" bson:"createdAt"`
	CreatorID string            `bson:"creatorid" json:"creatorid"`
}

type BandMember struct {
	MemberID        string `bson:"memberid,omitempty" json:"memberid,omitempty"`
	ReferenceArtist string `bson:"ref_artistid,omitempty" json:"ref_artistid,omitempty"`
	Name            string `bson:"name" json:"name"`
	Role            string `bson:"role,omitempty" json:"role,omitempty"`
	DOB             string `bson:"dob,omitempty" json:"dob,omitempty"`
	Image           string `bson:"image,omitempty" json:"image,omitempty"`
}

// ArtistEvent Struct
type ArtistEvent struct {
	EventID   string `bson:"eventid,omitempty" json:"eventid"`
	ArtistID  string `bson:"artistid" json:"artistid"`
	Title     string `bson:"title" json:"title"`
	Date      string `bson:"date" json:"date"`
	Venue     string `bson:"venue" json:"venue"`
	City      string `bson:"city" json:"city"`
	Country   string `bson:"country" json:"country"`
	CreatorID string `bson:"creatorid" json:"creatorid"`
	TicketURL string `bson:"ticket_url,omitempty" json:"ticketUrl,omitempty"`
}
