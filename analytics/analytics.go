package analytics

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

// Analytics represents extended analytics data for any entity
type Analytics struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type"` // "event", "place", "product", etc.
	Metrics      map[string]int         `json:"metrics"`
	Trend        []int                  `json:"trend"`        // last 7 days of activity
	TopLocations []string               `json:"topLocations"` // relevant for places
	Engagement   map[string]interface{} `json:"engagement"`   // e.g., avg time, bounce rate
	Insights     map[string]string      `json:"insights"`     // textual summaries
	LastUpdated  time.Time              `json:"lastUpdated"`
}

// --- Event Analytics ---
func getEventAnalytics(entityID string) Analytics {
	return Analytics{
		ID:   entityID,
		Name: "Sample Event " + entityID,
		Type: "event",
		Metrics: map[string]int{
			"totalTickets":   120,
			"views":          350,
			"attended":       95,
			"rsvps":          70,
			"shares":         40,
			"favorites":      25,
			"refunds":        3,
			"checkIns":       90,
			"uniqueVisitors": 280,
		},
		Trend: []int{15, 20, 25, 30, 28, 34, 40},
		Engagement: map[string]interface{}{
			"avgViewTimeSec": 45,
			"conversionRate": 27.3,
			"bounceRate":     14.5,
		},
		Insights: map[string]string{
			"growth":         "Ticket sales up 15% from last week",
			"conversion":     "Strong conversion from social media referrals",
			"attendance":     "85% of RSVPs attended the event",
			"recommendation": "Increase ad spend on Wednesday to capture peak interest",
		},
		LastUpdated: time.Now(),
	}
}

// --- Place Analytics ---
func getPlaceAnalytics(entityID string) Analytics {
	return Analytics{
		ID:   entityID,
		Name: "Sample Place " + entityID,
		Type: "place",
		Metrics: map[string]int{
			"visits":       420,
			"checkIns":     180,
			"favorites":    65,
			"reviews":      50,
			"avgRating":    4, // out of 5
			"photoUploads": 18,
		},
		Trend:        []int{40, 55, 60, 75, 80, 90, 85},
		TopLocations: []string{"Downtown", "Uptown", "Central Park"},
		Engagement: map[string]interface{}{
			"avgSessionSec":  95,
			"reviewResponse": 82.5,
			"returnRate":     41.7,
		},
		Insights: map[string]string{
			"traffic":        "Most visits occur between 4–7 PM",
			"popularity":     "Top location among users aged 20–35",
			"loyalty":        "41% of users return within a week",
			"recommendation": "Add photo spots or perks to boost engagement",
		},
		LastUpdated: time.Now(),
	}
}

// --- Product Analytics ---
func getProductAnalytics(entityID string) Analytics {
	return Analytics{
		ID:   entityID,
		Name: "Sample Product " + entityID,
		Type: "product",
		Metrics: map[string]int{
			"totalSold":    580,
			"views":        1400,
			"favorites":    320,
			"reviews":      48,
			"availableQty": 100,
			"cartAdds":     190,
			"cartAbandons": 35,
		},
		Trend: []int{60, 75, 70, 85, 95, 110, 105},
		Engagement: map[string]interface{}{
			"avgViewTimeSec":  30,
			"conversionRate":  12.5,
			"returnCustomers": 22.4,
		},
		Insights: map[string]string{
			"sales":          "Weekly sales up 18%",
			"visibility":     "Product performs best via organic search",
			"conversion":     "High drop-off after cart — consider improving checkout speed",
			"recommendation": "Highlight limited stock to encourage faster purchase decisions",
		},
		LastUpdated: time.Now(),
	}
}

// --- Delegator Handler ---
func GetEntityAnalytics(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	entityType := ps.ByName("entityType")
	entityID := ps.ByName("entityId")

	var analytics Analytics

	switch entityType {
	case "events":
		analytics = getEventAnalytics(entityID)
	case "places":
		analytics = getPlaceAnalytics(entityID)
	case "products":
		analytics = getProductAnalytics(entityID)
	default:
		http.Error(w, "Invalid entity type", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(analytics)
}
