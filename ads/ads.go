package ads

import (
	"encoding/json"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// Ad represents the structure of an advertisement.
type Ad struct {
	ID          string `json:"id,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Image       string `json:"image,omitempty"`
	Link        string `json:"link,omitempty"`
	Category    string `json:"category,omitempty"`
}

// Dummy ad data
var ads = []Ad{
	{
		ID:          "1",
		Title:       "Tech Gadget Sale",
		Description: "Get the latest gadgets at unbeatable prices!",
		Image:       "https://via.placeholder.com/300x150?text=Tech+Ad",
		Link:        "https://example.com/tech-sale",
		Category:    "tech",
	},
	{
		ID:          "2",
		Title:       "Travel Deals",
		Description: "Explore the world with our exclusive travel packages.",
		Image:       "https://via.placeholder.com/300x150?text=Travel+Ad",
		Link:        "https://example.com/travel-deals",
		Category:    "travel",
	},
	{
		ID:          "3",
		Title:       "Local Restaurant",
		Description: "Taste the best food in town at amazing discounts.",
		Image:       "https://via.placeholder.com/300x150?text=Food+Ad",
		Link:        "https://example.com/restaurant",
		Category:    "food",
	},
}

// GetAds handles the API request to fetch ads.
func GetAds(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	category := r.URL.Query().Get("category")

	var filteredAds []Ad
	if category == "" || category == "default" {
		// no category specified → return all ads
		filteredAds = ads
	} else {
		// filter by category
		for _, ad := range ads {
			if ad.Category == category {
				filteredAds = append(filteredAds, ad)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(filteredAds); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
