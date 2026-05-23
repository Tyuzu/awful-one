package search

import (
	"context"
	"naevis/config"
	"naevis/infra"
	"naevis/infra/db"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

// SearchResult represents a single search result item
type SearchResult struct {
	ID          string    `json:"id,omitempty"`
	EntityID    string    `json:"entityid,omitempty"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Image       string    `json:"image,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

// AllSearchResults represents results grouped by entity type
type AllSearchResults struct {
	Events       []SearchResult `json:"events,omitempty"`
	Places       []SearchResult `json:"places,omitempty"`
	Feedposts    []SearchResult `json:"feedposts,omitempty"`
	Merch        []SearchResult `json:"merch,omitempty"`
	Blogposts    []SearchResult `json:"blogposts,omitempty"`
	Farms        []SearchResult `json:"farms,omitempty"`
	Songs        []SearchResult `json:"songs,omitempty"`
	Users        []SearchResult `json:"users,omitempty"`
	Recipes      []SearchResult `json:"recipes,omitempty"`
	Products     []SearchResult `json:"products,omitempty"`
	Menu         []SearchResult `json:"menu,omitempty"`
	Media        []SearchResult `json:"media,omitempty"`
	Crops        []SearchResult `json:"crops,omitempty"`
	Baitoworkers []SearchResult `json:"baitoworkers,omitempty"`
	Baitos       []SearchResult `json:"baitos,omitempty"`
	Artists      []SearchResult `json:"artists,omitempty"`
}

// GetAutocompleteSuggestions returns autocomplete suggestions for a prefix
func GetAutocompleteSuggestions(ctx context.Context, app *infra.Deps, prefix string) ([]string, error) {
	prefix = strings.ToLower(prefix)
	regex := bson.M{"$regex": prefix, "$options": "i"}

	suggestions := make(map[string]bool) // Using map to avoid duplicates

	// Define collections to search and their fields
	type collectionInfo struct {
		name  string
		field string
	}

	collections := []collectionInfo{
		{config.Collections.EventsCollection, "title"},
		{config.Collections.PlacesCollection, "name"},
		{config.Collections.FeedPostsCollection, "title"},
		{config.Collections.MerchCollection, "name"},
		{config.Collections.BlogPostsCollection, "title"},
		{config.Collections.FarmsCollection, "name"},
		{config.Collections.SongsCollection, "title"},
		{config.Collections.UserCollection, "username"},
		{config.Collections.RecipeCollection, "title"},
		{config.Collections.ProductCollection, "name"},
		{config.Collections.MenuCollection, "name"},
		{config.Collections.MediaCollection, "title"},
		{config.Collections.CropsCollection, "name"},
		{config.Collections.BaitoWorkerCollection, "name"},
		{config.Collections.BaitoCollection, "title"},
		{config.Collections.ArtistsCollection, "name"},
	}

	for _, coll := range collections {
		var results []bson.M
		filter := bson.M{coll.field: regex}

		err := app.DB.FindMany(ctx, coll.name, filter, &results)
		if err != nil {
			continue // Skip errors in individual collections
		}

		for _, result := range results {
			if val, ok := result[coll.field]; ok {
				if str, ok := val.(string); ok && str != "" {
					suggestions[str] = true
				}
			}
		}
	}

	// Convert map to slice - initialize as empty slice to avoid nil JSON response
	output := make([]string, 0)
	for k := range suggestions {
		output = append(output, k)
	}

	return output, nil
}

// SearchByEntity searches in a specific entity type
func SearchByEntity(ctx context.Context, app *infra.Deps, entityType, query string) ([]SearchResult, error) {
	var results []SearchResult
	regex := bson.M{"$regex": query, "$options": "i"}

	// Map entity types to collections and search fields
	entityInfo := map[string]struct {
		collection string
		fields     []string
		titleField string
	}{
		"events": {
			collection: config.Collections.EventsCollection,
			fields:     []string{"title", "description", "location"},
			titleField: "title",
		},
		"places": {
			collection: config.Collections.PlacesCollection,
			fields:     []string{"name", "description", "address"},
			titleField: "name",
		},
		"feedposts": {
			collection: config.Collections.FeedPostsCollection,
			fields:     []string{"title", "description", "caption"},
			titleField: "title",
		},
		"merch": {
			collection: config.Collections.MerchCollection,
			fields:     []string{"name", "description"},
			titleField: "name",
		},
		"blogposts": {
			collection: config.Collections.BlogPostsCollection,
			fields:     []string{"title", "description", "content"},
			titleField: "title",
		},
		"farms": {
			collection: config.Collections.FarmsCollection,
			fields:     []string{"name", "description"},
			titleField: "name",
		},
		"songs": {
			collection: config.Collections.SongsCollection,
			fields:     []string{"title", "description", "artist"},
			titleField: "title",
		},
		"users": {
			collection: config.Collections.UserCollection,
			fields:     []string{"username", "displayname", "bio"},
			titleField: "username",
		},
		"recipes": {
			collection: config.Collections.RecipeCollection,
			fields:     []string{"title", "description", "ingredients"},
			titleField: "title",
		},
		"products": {
			collection: config.Collections.ProductCollection,
			fields:     []string{"name", "description"},
			titleField: "name",
		},
		"menu": {
			collection: config.Collections.MenuCollection,
			fields:     []string{"name", "description"},
			titleField: "name",
		},
		"media": {
			collection: config.Collections.MediaCollection,
			fields:     []string{"title", "description"},
			titleField: "title",
		},
		"crops": {
			collection: config.Collections.CropsCollection,
			fields:     []string{"name", "description"},
			titleField: "name",
		},
		"baitoworkers": {
			collection: config.Collections.BaitoWorkerCollection,
			fields:     []string{"name", "description"},
			titleField: "name",
		},
		"baitos": {
			collection: config.Collections.BaitoCollection,
			fields:     []string{"title", "description"},
			titleField: "title",
		},
		"artists": {
			collection: config.Collections.ArtistsCollection,
			fields:     []string{"name", "description", "bio"},
			titleField: "name",
		},
	}

	info, exists := entityInfo[entityType]
	if !exists {
		return results, nil // Return empty results for unknown types
	}

	// Build OR query for multiple fields
	orQuery := bson.A{}
	for _, field := range info.fields {
		orQuery = append(orQuery, bson.M{field: regex})
	}

	filter := bson.M{"$or": orQuery}

	var docs []bson.M
	opts := db.FindManyOptions{
		Limit: 50,
	}

	err := app.DB.FindManyWithOptions(ctx, info.collection, filter, opts, &docs)
	if err != nil {
		return results, err
	}

	// Convert BSON documents to SearchResult
	for _, doc := range docs {
		result := SearchResult{
			Title: getStringField(doc, info.titleField),
		}

		// Try to get description from various fields
		if desc, ok := doc["description"].(string); ok {
			result.Description = desc
		} else if desc, ok := doc["bio"].(string); ok {
			result.Description = desc
		} else if desc, ok := doc["caption"].(string); ok {
			result.Description = desc
		} else if desc, ok := doc["content"].(string); ok && len(desc) > 200 {
			result.Description = desc[:200] + "..."
		}

		// Get ID fields - try multiple common ID field names
		if id, ok := doc["_id"].(string); ok {
			result.ID = id
		}
		if id, ok := doc["id"].(string); ok {
			result.EntityID = id
		}
		if id, ok := doc["eventid"].(string); ok {
			result.EntityID = id
		}
		if id, ok := doc["placeid"].(string); ok {
			result.EntityID = id
		}
		if id, ok := doc["userid"].(string); ok {
			result.EntityID = id
		}

		// Get image/banner field
		if img, ok := doc["image"].(string); ok {
			result.Image = img
		} else if img, ok := doc["banner"].(string); ok {
			result.Image = img
		} else if img, ok := doc["banner_image"].(string); ok {
			result.Image = img
		}

		// Get creation timestamp
		if t, ok := doc["created_at"].(time.Time); ok {
			result.CreatedAt = t
		} else if t, ok := doc["createdAt"].(time.Time); ok {
			result.CreatedAt = t
		}

		results = append(results, result)
	}

	return results, nil
}

// SearchAll searches across all entity types and returns grouped results
func SearchAll(ctx context.Context, app *infra.Deps, query string) (AllSearchResults, error) {
	allResults := AllSearchResults{}

	entityTypes := []string{
		"events", "places", "feedposts", "merch", "blogposts",
		"farms", "songs", "users", "recipes", "products",
		"menu", "media", "crops", "baitoworkers", "baitos", "artists",
	}

	// Search each entity type in parallel
	resultsChan := make(chan struct {
		entityType string
		results    []SearchResult
	}, len(entityTypes))

	for _, entityType := range entityTypes {
		go func(et string) {
			results, _ := SearchByEntity(ctx, app, et, query)
			// Limit results per type to 10 for "all" tab
			if len(results) > 10 {
				results = results[:10]
			}
			resultsChan <- struct {
				entityType string
				results    []SearchResult
			}{et, results}
		}(entityType)
	}

	// Collect results
	for range entityTypes {
		res := <-resultsChan
		switch res.entityType {
		case "events":
			allResults.Events = res.results
		case "places":
			allResults.Places = res.results
		case "feedposts":
			allResults.Feedposts = res.results
		case "merch":
			allResults.Merch = res.results
		case "blogposts":
			allResults.Blogposts = res.results
		case "farms":
			allResults.Farms = res.results
		case "songs":
			allResults.Songs = res.results
		case "users":
			allResults.Users = res.results
		case "recipes":
			allResults.Recipes = res.results
		case "products":
			allResults.Products = res.results
		case "menu":
			allResults.Menu = res.results
		case "media":
			allResults.Media = res.results
		case "crops":
			allResults.Crops = res.results
		case "baitoworkers":
			allResults.Baitoworkers = res.results
		case "baitos":
			allResults.Baitos = res.results
		case "artists":
			allResults.Artists = res.results
		}
	}

	return allResults, nil
}

// Helper function to safely get string field from document
func getStringField(doc bson.M, fieldName string) string {
	if val, ok := doc[fieldName]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}
