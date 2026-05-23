package vlive

import (
	"context"
	"encoding/json"
	"errors"
	"naevis/infra"
	"naevis/models"
	"net/http"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ----------------------- Helpers -----------------------

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, msg string, code int) {
	http.Error(w, msg, code)
}

// ----------------------- DB helpers -----------------------

func getStreamByID(ctx context.Context, app *infra.Deps, id primitive.ObjectID) (models.LiveStream, error) {
	var s models.LiveStream
	err := app.DB.FindOne(ctx, vlivesCollection, bson.M{"_id": id}, &s)
	return s, err
}

func getStreamByKey(ctx context.Context, app *infra.Deps, key string) (models.LiveStream, error) {
	var s models.LiveStream
	err := app.DB.FindOne(ctx, vlivesCollection, bson.M{"stream_key": key}, &s)
	return s, err
}

func isOwner(userID string, stream models.LiveStream) bool {
	return stream.CreatorID == userID
}

func conditionalStateUpdate(
	ctx context.Context,
	app *infra.Deps,
	id string,
	expectedFrom string,
	set bson.M,
) (bool, error) {

	filter := bson.M{"_id": id}
	if expectedFrom != "" {
		filter["state"] = expectedFrom
	}

	update := bson.M{"$set": set}

	err := app.DB.UpdateOne(ctx, vlivesCollection, filter, update)
	if err != nil {
		return false, err
	}

	// No ModifiedCount in interface → re-read to verify
	var updated models.LiveStream
	err = app.DB.FindOne(ctx, vlivesCollection, bson.M{"_id": id}, &updated)
	if err != nil {
		return false, err
	}

	if expectedFrom != "" && updated.State != set["state"] {
		return false, nil
	}

	return true, nil
}

func disallowWhileLive(stream models.LiveStream) error {
	if stream.State == models.LiveLive {
		return errors.New("operation not allowed while live")
	}
	return nil
}

func validateEntityType(t string) bool {
	switch strings.ToUpper(t) {
	case "USER", "CHANNEL", "GROUP", "EVENT", "ARTIST":
		return true
	default:
		return false
	}
}

// ----------------------- Viewer count cache helpers -----------------------

func viewerCountKey(liveID string) string {
	return "vlive:viewers:" + liveID
}

func getViewerCountCache(ctx context.Context, app *infra.Deps, liveID string) int {
	b, err := app.Cache.Get(ctx, viewerCountKey(liveID))
	if err != nil || len(b) == 0 {
		return 0
	}

	n, err := strconv.Atoi(string(b))
	if err != nil {
		return 0
	}
	return n
}

func incViewerCount(ctx context.Context, app *infra.Deps, liveID string) int {
	n, err := app.Cache.Incr(ctx, viewerCountKey(liveID))
	if err != nil {
		return 0
	}
	return int(n)
}

func decViewerCount(ctx context.Context, app *infra.Deps, liveID string) int {
	key := viewerCountKey(liveID)

	n, err := app.Cache.Incr(ctx, key)
	if err != nil {
		return 0
	}

	n = n - 2 // compensate since we only have Incr
	if n < 0 {
		_ = app.Cache.Set(ctx, key, []byte("0"), 0)
		return 0
	}

	_ = app.Cache.Set(ctx, key, []byte(strconv.FormatInt(n, 10)), 0)
	return int(n)
}
