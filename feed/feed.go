package feed

import (
	"context"
	"encoding/json"
	"errors"
	"naevis/config/mqevent"
	"naevis/infra"
	"naevis/utils"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

// DELETE /api/v1/feed/post/:postid
func DeletePost(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

		ctx := r.Context()

		token := r.Header.Get("Authorization")
		claims, err := utils.ValidateJWT(token)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		postID := ps.ByName("postid")
		if postID == "" {
			http.Error(w, "postid is required", http.StatusBadRequest)
			return
		}

		err = DeletePostFromDB(ctx, claims.UserID, postID, app)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func DeletePostFromDB(ctx context.Context, userID string, postID string, app *infra.Deps) error {

	filter := bson.M{
		"_id":     postID,
		"user_id": userID,
	}

	deleted, err := app.DB.DeleteOne(ctx, "posts", filter)
	if err != nil {
		return err
	}

	if deleted == 0 {
		return errors.New("post not found or unauthorized")
	}

	/* -------- Publish PostDeleted Event -------- */
	deletePayload := mqevent.PostDeletedPayload{
		PostID:     postID,
		UserID:     userID,
		OccurredAt: time.Now(),
	}

	deleteBytes, err := json.Marshal(deletePayload)
	if err == nil {
		publishCtx, cancel := context.WithTimeout(
			context.Background(),
			3*time.Second,
		)
		defer cancel()

		_ = app.MQ.Publish(
			publishCtx,
			mqevent.PostDeleted,
			deleteBytes,
		)
	}

	return nil
}
