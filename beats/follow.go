package beats

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"

	"naevis/globals"
	"naevis/infra"
	"naevis/models"
	"naevis/userdata"
)

func HandleFollowAction(
	w http.ResponseWriter,
	r *http.Request,
	ps httprouter.Params,
	action string,
	app *infra.Deps,
) {
	ctx := r.Context()

	currentUserID, ok := r.Context().Value(globals.UserIDKey).(string)
	if !ok || currentUserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	targetUserID := ps.ByName("id")
	if targetUserID == "" {
		http.Error(w, "Target user ID required", http.StatusBadRequest)
		return
	}

	if err := UpdateFollowRelationship(ctx, currentUserID, targetUserID, action, app); err != nil {
		log.Printf("Error updating follow relationship: %v", err)
		http.Error(w, "Failed to update follow relationship", http.StatusInternalServerError)
		return
	}

	userdata.SetUserData(action, targetUserID, currentUserID, "profile", targetUserID, app)

	response := map[string]any{
		"isFollowing": action == "follow",
		"ok":          true,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func ToggleFollow(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		HandleFollowAction(w, r, ps, "follow", app)
	}
}

func ToggleUnFollow(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		HandleFollowAction(w, r, ps, "unfollow", app)
	}
}

func UpdateFollowRelationship(
	ctx context.Context,
	currentUserID,
	targetUserID,
	action string,
	app *infra.Deps,
) error {
	if action != "follow" && action != "unfollow" {
		return fmt.Errorf("invalid action: %s", action)
	}

	var currentUserUpdate any
	var targetUserUpdate any

	if action == "follow" {
		currentUserUpdate = bson.M{
			"$addToSet": bson.M{"follows": targetUserID},
		}
		targetUserUpdate = bson.M{
			"$addToSet": bson.M{"followers": currentUserID},
		}
	} else {
		currentUserUpdate = bson.M{
			"$pull": bson.M{"follows": targetUserID},
		}
		targetUserUpdate = bson.M{
			"$pull": bson.M{"followers": currentUserID},
		}
	}

	// Upsert current user follow list
	if err := app.DB.Upsert(
		ctx,
		followingsCollection,
		bson.M{"userid": currentUserID},
		currentUserUpdate,
	); err != nil {
		return fmt.Errorf("failed to update current user's follows: %w", err)
	}

	// Upsert target user followers list
	if err := app.DB.Upsert(
		ctx,
		followingsCollection,
		bson.M{"userid": targetUserID},
		targetUserUpdate,
	); err != nil {
		return fmt.Errorf("failed to update target user's followers: %w", err)
	}

	return nil
}

func CreateFollowEntry(userid string, app *infra.Deps) {
	follow := models.UserFollow{
		UserID:    userid,
		Follows:   []string{},
		Followers: []string{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := app.DB.Insert(
		ctx,
		followingsCollection,
		follow,
	)
	if err != nil {
		log.Printf("Error inserting follow entry for %s: %v", userid, err)
	}
}
