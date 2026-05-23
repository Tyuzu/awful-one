package home

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"

	"naevis/infra"
	"naevis/infra/db"
	"naevis/utils"
)

// HomeCard response type
type HomeCard struct {
	Banner      string `json:"banner,omitempty" bson:"-"`
	Title       string `json:"title" bson:"title"`
	Description string `json:"description" bson:"description"`
	Href        string `json:"href" bson:"-"`
}

// categoryProjection returns collection name and projection function
func categoryProjection(category string) (string, func(bson.M) HomeCard) {
	switch category {
	case "Places":
		return "places", func(doc bson.M) HomeCard {
			id, _ := doc["placeid"].(string)
			banner, _ := doc["banner"].(string)
			title, _ := doc["name"].(string)
			desc, _ := doc["description"].(string)

			return HomeCard{
				Banner:      banner,
				Title:       title,
				Description: desc,
				Href:        "/place/" + id,
			}
		}

	case "Events":
		return "events", func(doc bson.M) HomeCard {
			id, _ := doc["eventid"].(string)
			banner, _ := doc["banner"].(string)
			title, _ := doc["title"].(string)
			desc, _ := doc["description"].(string)

			return HomeCard{
				Banner:      banner,
				Title:       title,
				Description: desc,
				Href:        "/event/" + id,
			}
		}

	case "Baitos":
		return "baitos", func(doc bson.M) HomeCard {
			id, _ := doc["baitoid"].(string)
			banner, _ := doc["banner"].(string)
			title, _ := doc["title"].(string)
			desc, _ := doc["description"].(string)

			return HomeCard{
				Banner:      banner,
				Title:       title,
				Description: desc,
				Href:        "/baito/" + id,
			}
		}

	case "Products":
		return "products", func(doc bson.M) HomeCard {
			id, _ := doc["productid"].(string)

			banner := ""
			if arr, ok := doc["imageUrls"].(bson.A); ok && len(arr) > 0 {
				if s, ok := arr[0].(string); ok {
					banner = s
				}
			}

			title, _ := doc["name"].(string)
			desc, _ := doc["description"].(string)

			return HomeCard{
				Banner:      banner,
				Title:       title,
				Description: desc,
				Href:        "/product/" + id,
			}
		}

	case "Posts":
		return "posts", func(doc bson.M) HomeCard {
			id, _ := doc["postid"].(string)
			banner, _ := doc["thumb"].(string)
			title, _ := doc["title"].(string)
			desc, _ := doc["content"].(string)

			return HomeCard{
				Banner:      banner,
				Title:       title,
				Description: desc,
				Href:        "/post/" + id,
			}
		}

	default:
		return "", nil
	}
}

func HomeCardsHandler(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		reqID := r.Header.Get("X-Request-Id")
		if reqID == "" {
			reqID = strconv.FormatInt(time.Now().UnixNano(), 36) + "-" + strconv.FormatInt(rand.Int63(), 36)
		}

		w.Header().Set("X-Request-Id", reqID)
		w.Header().Set("Content-Type", "application/json")

		category := r.URL.Query().Get("category")
		collection, projector := categoryProjection(category)
		if collection == "" || projector == nil {
			_ = json.NewEncoder(w).Encode([]HomeCard{})
			return
		}

		skip, limit := utils.ParsePagination(r, 0, 20)

		opts := db.FindManyOptions{
			Skip:  skip,
			Limit: limit,
			Sort:  bson.D{{Key: "createdAt", Value: -1}},
		}

		var docs []bson.M
		if err := app.DB.FindManyWithOptions(ctx, collection, bson.M{}, opts, &docs); err != nil {
			log.Println("Find error:", err, "req_id:", reqID)
			w.Header().Set("X-Error-Request-Id", reqID)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch home cards")
			return
		}

		cards := make([]HomeCard, 0, len(docs))
		for _, doc := range docs {
			cards = append(cards, projector(doc))
		}

		utils.RespondWithJSON(w, http.StatusOK, cards)
	}
}
