package userdata

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"naevis/infra"
	"naevis/models"
	"naevis/utils"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

// GetUserProfileData fetches user-specific entity data
func GetUserProfileData(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		username := ps.ByName("username")

		// Validate JWT
		tokenString := r.Header.Get("Authorization")
		claims, err := utils.ValidateJWT(tokenString)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if username != claims.UserID {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Parse query parameter
		entityType := r.URL.Query().Get("entity_type")
		if entityType == "" {
			http.Error(w, "Entity type is required", http.StatusBadRequest)
			return
		}
		if !IsValidEntityType(entityType) {
			http.Error(w, "Invalid entity type", http.StatusBadRequest)
			return
		}

		// Fetch data from Database interface
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var results []models.UserData
		filter := map[string]any{
			"entity_type": entityType,
			"userid":      username,
		}

		if err := app.DB.FindMany(ctx, userdataCollection, filter, &results); err != nil {
			http.Error(w, "Failed to fetch user data", http.StatusInternalServerError)
			log.Printf("Error fetching user data: %v", err)
			return
		}

		// Ensure empty slice instead of nil
		if results == nil {
			results = []models.UserData{}
		}

		// Respond with JSON
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(results); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			log.Printf("Error encoding response: %v", err)
			return
		}
	}
}

func GetOtherUserProfileData(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		// username := ps.ByName("username") // from /user/{username}/data
		// entityType := r.URL.Query().Get("entity_type")

		// if entityType != "feedpost" {
		// 	http.Error(w, "Invalid entity type", http.StatusBadRequest)
		// 	return
		// }

		// // Connect to MongoDB collection "posts"
		// collection := db.PostsCollection

		// filter := bson.M{"userid": username}
		// cursor, err := collection.Find(context.Background(), filter)
		// if err != nil {
		// 	http.Error(w, "DB error", http.StatusInternalServerError)
		// 	return
		// }
		// defer cursor.Close(context.Background())

		// var posts []bson.M
		// if err = cursor.All(context.Background(), &posts); err != nil {
		// 	http.Error(w, "Cursor decode error", http.StatusInternalServerError)
		// 	return
		// }

		// // Convert MongoDB docs to a simplified response
		// var response []map[string]interface{}
		// for _, post := range posts {
		// 	response = append(response, map[string]interface{}{
		// 		"id":         post["_id"],
		// 		"image_url":  post["image_url"],
		// 		"caption":    post["caption"],
		// 		"created_at": post["created_at"],
		// 	})
		// }

		// w.Header().Set("Content-Type", "application/json")
		// json.NewEncoder(w).Encode(response)

		const res string = `[
		{
		  "id": "post_001",
		  "image_url": "https://i.pinimg.com/1200x/41/6a/dd/416add5d8b8f0ea89bbcd78ef4471866.jpg",
		  "caption": "Sunset in Bali",
		  "created_at": "2025-05-19T12:34:56Z"
		},
		{
		  "id": "post_002",
		  "image_url": "https://i.pinimg.com/736x/28/f5/b9/28f5b9cb0281e66944d2b5834283122f.jpg",
		  "caption": "Coffee time",
		  "created_at": "2025-05-18T08:12:44Z"
		},
		{
		  "id": "post_003",
		  "image_url": "https://i.pinimg.com/736x/ce/a1/6a/cea16abe291fde62df608ed9460c99a1.jpg",
		  "caption": "City lights",
		  "created_at": "2025-05-17T21:09:13Z"
		}
	  ]
	  `
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, "%s", res)
	}
}
