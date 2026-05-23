package hashtags

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"naevis/utils"

	"github.com/julienschmidt/httprouter"
)

// HashtagPost is the shape we return to the frontend grid
type HashtagPost struct {
	PostID      string      `json:"postid"`
	MediaURL    interface{} `json:"media_url,omitempty"`
	Type        string      `json:"type,omitempty"`
	Title       string      `json:"title,omitempty"`
	Description string      `json:"description,omitempty"`
	Tags        []string    `json:"tags,omitempty"`
	UserID      string      `json:"userid,omitempty"`
	Timestamp   time.Time   `json:"timestamp,omitempty"`
	Resolution  interface{} `json:"resolution,omitempty"`
}

// TrendingHashtag is shape for trending endpoint
type TrendingHashtag struct {
	Tag   string `json:"tag"`
	Count int32  `json:"count"`
}

// Person is a simple shape for hashtag "people" tab
type Person struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

// parse pagination query
func parsePagination(r *http.Request) (page int, limit int) {
	q := r.URL.Query()
	page = 0
	limit = 30

	if p := q.Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v >= 0 {
			page = v
		}
	}
	if l := q.Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}
	return
}

// apply pagination to a slice of any type
func paginate[T any](items []T, page, limit int) []T {
	start := page * limit
	end := start + limit
	if start >= len(items) {
		return []T{}
	}
	if end > len(items) {
		end = len(items)
	}
	return items[start:end]
}

// GetHashtagPosts returns dummy posts for a hashtag (all types)
func GetHashtagPosts(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	tag := ps.ByName("tag")
	if tag == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Missing tag parameter")
		return
	}

	page, limit := parsePagination(r)

	now := time.Now()
	posts := []HashtagPost{
		{
			PostID:      "p1",
			MediaURL:    []string{"/static/dummy1.jpg"},
			Type:        "image",
			Title:       "Sunny day",
			Description: "A beautiful sunny day!",
			Tags:        []string{tag, "nature"},
			UserID:      "u1",
			Timestamp:   now.Add(-time.Hour),
		},
		{
			PostID:      "p2",
			MediaURL:    []string{"/static/dummy2.mp4"},
			Type:        "video",
			Title:       "Skating",
			Description: "Check out my new trick",
			Tags:        []string{tag, "sports"},
			UserID:      "u2",
			Timestamp:   now.Add(-2 * time.Hour),
			Resolution:  "1080p",
		},
		{
			PostID:      "p3",
			MediaURL:    []string{"/static/dummy3.jpg"},
			Type:        "image",
			Title:       "Coffee time",
			Description: "Morning ritual ☕",
			Tags:        []string{tag, "coffee"},
			UserID:      "u3",
			Timestamp:   now.Add(-3 * time.Hour),
		},
	}

	results := paginate(posts, page, limit)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(results); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to encode response")
	}
}

// GetTopHashtagPosts returns dummy "top" posts (sorted by engagement in real app)
func GetTopHashtagPosts(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	GetHashtagPosts(w, r, ps)
}

// GetLatestHashtagPosts returns dummy "latest" posts (sorted by timestamp in real app)
func GetLatestHashtagPosts(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	GetHashtagPosts(w, r, ps)
}

// GetHashtagPeople returns dummy people who used the hashtag
func GetHashtagPeople(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	tag := ps.ByName("tag")
	if tag == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Missing tag parameter")
		return
	}

	page, limit := parsePagination(r)

	people := []Person{
		{Username: "alice", DisplayName: "Alice Smith"},
		{Username: "bob", DisplayName: "Bob Johnson"},
		{Username: "carol", DisplayName: "Carol Doe"},
		{Username: "david", DisplayName: "David Kim"},
		{Username: "eva", DisplayName: "Eva Brown"},
	}

	results := paginate(people, page, limit)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(results); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to encode response")
	}
}

// GetTrendingHashtags returns dummy trending hashtags
func GetTrendingHashtags(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}

	all := []TrendingHashtag{
		{Tag: "nature", Count: 120},
		{Tag: "sports", Count: 95},
		{Tag: "coffee", Count: 80},
		{Tag: "travel", Count: 60},
		{Tag: "music", Count: 50},
	}

	if limit < len(all) {
		all = all[:limit]
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(all); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to encode response")
	}
}
