package beats

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"naevis/infra"
	"naevis/userdata"
	"naevis/utils"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

// Generic subscribe/follow/unfollow handler
func HandleEntitySubscription(
	w http.ResponseWriter,
	r *http.Request,
	ps httprouter.Params,
	entityType,
	action string,
	app *infra.Deps,
) {
	ctx := r.Context()

	currentUserID := utils.GetUserIDFromRequest(r)
	if currentUserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	entityID := ps.ByName("id")
	if entityID == "" {
		http.Error(w, "Entity ID required", http.StatusBadRequest)
		return
	}

	if err := UpdateEntitySubscription(ctx, currentUserID, entityType, entityID, action, app); err != nil {
		log.Printf("Failed to update %s subscription: %v", entityType, err)
		http.Error(w, "Failed to update subscription", http.StatusInternalServerError)
		return
	}

	// Optional UI/userdata hook
	userdata.SetUserData(action, entityID, currentUserID, entityType, entityID, app)

	resp := map[string]any{
		"hasSubscribed": action == "subscribe",
		"ok":            true,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// PUT /api/v1/subscribes/:id
func SubscribeEntity(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		HandleEntitySubscription(w, r, ps, ps.ByName("type"), "subscribe", app)
	}
}

// DELETE /api/v1/subscribes/:id
func UnsubscribeEntity(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		HandleEntitySubscription(w, r, ps, ps.ByName("type"), "unsubscribe", app)
	}
}

// Core logic for subscribe / unsubscribe
func UpdateEntitySubscription(
	ctx context.Context,
	userID,
	entityType,
	entityID,
	action string,
	app *infra.Deps,
) error {
	if action != "subscribe" && action != "unsubscribe" {
		return fmt.Errorf("invalid action: %s", action)
	}

	// Ensure both documents exist
	EnsureSubscriptionEntry(ctx, userID, app)
	EnsureSubscriptionEntry(ctx, entityID, app)

	var userUpdate any
	var entityUpdate any

	if action == "subscribe" {
		userUpdate = bson.M{
			"$addToSet": bson.M{
				"subscribed": entityID,
			},
		}
		entityUpdate = bson.M{
			"$addToSet": bson.M{
				"subscribers": userID,
			},
		}
	} else {
		userUpdate = bson.M{
			"$pull": bson.M{
				"subscribed": entityID,
			},
		}
		entityUpdate = bson.M{
			"$pull": bson.M{
				"subscribers": userID,
			},
		}
	}

	if err := app.DB.UpdateOne(
		ctx,
		subscribersCollection,
		bson.M{"userid": userID},
		userUpdate,
	); err != nil {
		return fmt.Errorf("failed to update user subscriptions: %w", err)
	}

	if err := app.DB.UpdateOne(
		ctx,
		subscribersCollection,
		bson.M{"userid": entityID},
		entityUpdate,
	); err != nil {
		return fmt.Errorf("failed to update entity subscribers: %w", err)
	}

	return nil
}

// Ensure a subscription document exists
func EnsureSubscriptionEntry(
	ctx context.Context,
	userID string,
	app *infra.Deps,
) {
	doc := bson.M{
		"userid":      userID,
		"subscribed":  []string{},
		"subscribers": []string{},
	}

	err := app.DB.Upsert(
		ctx,
		subscribersCollection,
		bson.M{"userid": userID},
		bson.M{
			"$setOnInsert": doc,
		},
	)

	if err != nil {
		log.Printf("Failed to ensure subscription entry for %s: %v", userID, err)
	}
}
