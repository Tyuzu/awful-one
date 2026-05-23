package dels

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"slices"
	"time"

	"naevis/globals"
	"naevis/infra"
	"naevis/infra/cache"
	"naevis/models"
	"naevis/userdata"
	"naevis/utils"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type permissionFn func(ctx context.Context, r *http.Request, entityID string) error
type afterDeleteFn func(ctx context.Context, entityID, userID string)

/* ---------------------------------------------------- */
/* Core helpers using db.Database                       */
/* ---------------------------------------------------- */

func deleteByField(
	w http.ResponseWriter,
	r *http.Request,
	ps httprouter.Params,
	app *infra.Deps,
	collection string,
	paramKey string,
	fieldKey string,
	perm permissionFn,
	after afterDeleteFn,
) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	entityID := ps.ByName(paramKey)
	if entityID == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	userID, _ := ctx.Value(globals.UserIDKey).(string)

	if perm != nil {
		if err := perm(ctx, r, entityID); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
	}

	if _, err := app.DB.DeleteOne(ctx, collection, bson.M{fieldKey: entityID}); err != nil {
		http.Error(w, "delete failed", http.StatusInternalServerError)
		return
	}

	if after != nil {
		after(ctx, entityID, userID)
	}

	utils.RespondWithJSON(w, http.StatusOK, utils.M{"success": true})
}

func softDeleteByField(
	w http.ResponseWriter,
	r *http.Request,
	ps httprouter.Params,
	app *infra.Deps,
	collection string,
	paramKey string,
	fieldKey string,
	update bson.M,
	perm permissionFn,
	after afterDeleteFn,
) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	entityID := ps.ByName(paramKey)
	if entityID == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	userID, _ := ctx.Value(globals.UserIDKey).(string)

	if perm != nil {
		if err := perm(ctx, r, entityID); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
	}

	if err := app.DB.UpdateOne(ctx, collection, bson.M{fieldKey: entityID}, update); err != nil {
		http.Error(w, "delete failed", http.StatusInternalServerError)
		return
	}

	if after != nil {
		after(ctx, entityID, userID)
	}

	utils.RespondWithJSON(w, http.StatusOK, utils.M{"success": true})
}

/* ---------------------------------------------------- */
/* Handlers                                             */
/* ---------------------------------------------------- */

func DeleteRecipe(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		deleteByField(w, r, ps, app, "recipes", "id", "recipeid", nil, nil)
	}
}

func DeleteMessage(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		softDeleteByField(
			w, r, ps, app,
			"messages",
			"messageId",
			"messageid",
			bson.M{"$set": bson.M{"deleted": true}},
			nil,
			nil,
		)
	}
}

func DeletesMessage(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		deleteByField(
			w, r, ps, app,
			"messages",
			"msgid",
			"messageid",
			func(ctx context.Context, r *http.Request, entityID string) error {
				objID, err := primitive.ObjectIDFromHex(entityID)
				if err != nil {
					return fmt.Errorf("invalid id")
				}

				var msg models.Message
				if err := app.DB.FindOne(ctx, messagesCollection, bson.M{"messageid": objID}, &msg); err != nil {
					return fmt.Errorf("not found")
				}

				if msg.UserID != utils.GetUserIDFromRequest(r) {
					return fmt.Errorf("forbidden")
				}

				_ = app.DB.UpdateOne(
					ctx,
					chatsCollection,
					bson.M{"chatid": msg.ChatID},
					bson.M{"$set": bson.M{"updatedAt": time.Now()}},
				)
				return nil
			},
			nil,
		)
	}
}

func DeleteComment(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		deleteByField(
			w, r, ps, app,
			"comments",
			"commentid",
			"commentid",
			func(ctx context.Context, r *http.Request, entityID string) error {
				objID, err := primitive.ObjectIDFromHex(entityID)
				if err != nil {
					return fmt.Errorf("invalid id")
				}

				var c models.Comment
				if err := app.DB.FindOne(ctx, commentsCollection, bson.M{"commentid": objID}, &c); err != nil {
					return fmt.Errorf("not found")
				}
				if c.CreatedBy != utils.GetUserIDFromRequest(r) {
					return fmt.Errorf("forbidden")
				}
				return nil
			},
			nil,
		)
	}
}

func DeleteFarm(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		deleteByField(
			w, r, ps, app,
			"farms",
			"id",
			"farmid",
			nil,
			func(ctx context.Context, entityID, _ string) {
				var farm models.Farm
				if err := app.DB.FindOne(ctx, farmsCollection, bson.M{"farmid": entityID}, &farm); err == nil {
					if farm.Banner != "" {
						_ = os.Remove("." + farm.Banner)
					}
				}
			},
		)
	}
}

func DeletePost(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		deleteByField(
			w, r, ps, app,
			"posts",
			"postid",
			"postid",
			nil,
			func(ctx context.Context, postID, userID string) {
				var file models.FileMetadata
				_ = app.DB.FindOne(ctx, filesCollection, bson.M{"postid": postID}, &file)
				RemoveUserFile(ctx, userID, postID, file.Hash, app)
				userdata.DelUserData("feedpost", postID, userID, app)
			},
		)
	}
}

/* ---------------------------------------------------- */
/* Helpers                                              */
/* ---------------------------------------------------- */

func InvalidateCachedProfile(ctx context.Context, username string, c cache.Cache) error {
	return c.Del(ctx, "profile:"+username)
}

func RemoveUserFile(ctx context.Context, userID, postID, hash string, app *infra.Deps) {
	err := app.DB.UpdateOne(
		ctx,
		filesCollection,
		bson.M{"hash": hash},
		bson.M{"$pull": bson.M{"userPosts." + userID: postID}},
	)
	if err != nil {
		return
	}

	var file models.FileMetadata
	if err := app.DB.FindOne(ctx, filesCollection, bson.M{"hash": hash}, &file); err != nil {
		return
	}

	isAssociated := false
	for _, posts := range file.UserPosts {
		if slices.Contains(posts, postID) {
			isAssociated = true
			break
		}
	}

	if !isAssociated {
		_ = app.DB.UpdateOne(
			ctx,
			filesCollection,
			bson.M{"hash": hash},
			bson.M{"$unset": bson.M{"postUrls." + postID: ""}},
		)
	}

	if len(file.UserPosts) == 0 {
		_ = os.Remove(file.PostURLs[postID])
		_, _ = app.DB.DeleteOne(ctx, filesCollection, bson.M{"hash": hash})
	}
}

func DeleteEvent(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		deleteByField(
			w, r, ps, app,
			"events",
			"eventid",
			"eventid",
			func(ctx context.Context, r *http.Request, entityID string) error {
				userID, _ := ctx.Value(globals.UserIDKey).(string)

				var ev models.Event
				if err := app.DB.FindOne(ctx, eventsCollection, bson.M{"eventid": entityID}, &ev); err != nil {
					return fmt.Errorf("not found")
				}
				if ev.CreatorID != userID {
					return fmt.Errorf("forbidden")
				}
				return nil
			},
			func(ctx context.Context, entityID, userID string) {
				_, _ = deleteRelatedData(ctx, entityID, app)
				userdata.DelUserData("event", entityID, userID, app)
			},
		)
	}
}

func DeleteCrop(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		deleteByField(w, r, ps, app, "crops", "cropid", "cropid", nil, nil)
	}
}

func DeleteProduct(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		deleteByField(w, r, ps, app, "products", "id", "productid", nil, nil)
	}
}

func DeleteTool(app *infra.Deps) httprouter.Handle {
	return DeleteProduct(app)
}

func DeleteMerch(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		deleteByField(
			w, r, ps, app,
			"merch",
			"merchid",
			"merchid",
			nil,
			func(ctx context.Context, entityID, _ string) {
				_ = app.Cache.Del(ctx, "merch:"+entityID)
			},
		)
	}
}

func DeleteTicket(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		deleteByField(w, r, ps, app, "tickets", "ticketid", "ticketid", nil, nil)
	}
}

func DeleteReview(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		deleteByField(
			w, r, ps, app,
			"reviews",
			"reviewId",
			"reviewid",
			func(ctx context.Context, r *http.Request, entityID string) error {
				userID, _ := ctx.Value(globals.UserIDKey).(string)

				var rev models.Review
				if err := app.DB.FindOne(ctx, reviewsCollection, bson.M{"reviewid": entityID}, &rev); err != nil {
					return fmt.Errorf("not found")
				}
				if rev.UserID != userID && !isAdmin(ctx) {
					return fmt.Errorf("forbidden")
				}
				return nil
			},
			func(ctx context.Context, entityID, userID string) {
				userdata.DelUserData("review", entityID, userID, app)
			},
		)
	}
}

func DeleteMedia(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		deleteByField(
			w, r, ps, app,
			"media",
			"id",
			"mediaid",
			nil,
			func(ctx context.Context, entityID, userID string) {
				userdata.DelUserData("media", entityID, userID, app)
			},
		)
	}
}

func DeletePlace(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		deleteByField(
			w, r, ps, app,
			"places",
			"placeid",
			"placeid",
			func(ctx context.Context, r *http.Request, entityID string) error {
				userID, _ := ctx.Value(globals.UserIDKey).(string)

				var place models.Place
				if err := app.DB.FindOne(ctx, placesCollection, bson.M{"placeid": entityID}, &place); err != nil {
					return fmt.Errorf("not found")
				}
				if place.CreatedBy != userID {
					return fmt.Errorf("forbidden")
				}
				return nil
			},
			func(ctx context.Context, entityID, userID string) {
				_ = app.Cache.Del(ctx, "place:"+entityID)
				userdata.DelUserData("place", entityID, userID, app)
			},
		)
	}
}

func DeleteMenu(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		deleteByField(
			w, r, ps, app,
			"menus",
			"menuid",
			"menuid",
			nil,
			func(ctx context.Context, entityID, _ string) {
				_ = app.Cache.Del(ctx, "menu:"+entityID)
			},
		)
	}
}

func DeleteProfile(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		deleteByField(
			w, r, ps, app,
			"users",
			"id",
			"userid",
			func(ctx context.Context, r *http.Request, entityID string) error {
				if entityID != utils.GetUserIDFromRequest(r) {
					return fmt.Errorf("forbidden")
				}
				return nil
			},
			func(ctx context.Context, _, userID string) {
				_ = app.Cache.Del(ctx, "profile:"+userID)
			},
		)
	}
}

func DeleteArtistByID(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		softDeleteByField(
			w, r, ps, app,
			"artists",
			"id",
			"artistid",
			bson.M{"$set": bson.M{"deleted": true}},
			nil,
			nil,
		)
	}
}

func DeleteArtistEvent(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		deleteByField(w, r, ps, app, "artist_events", "id", "eventid", nil, nil)
	}
}

func DeleteItinerary(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		softDeleteByField(
			w, r, ps, app,
			"itineraries",
			"id",
			"itineraryid",
			bson.M{"$set": bson.M{"deleted": true}},
			func(ctx context.Context, r *http.Request, entityID string) error {
				userID := utils.GetUserIDFromRequest(r)

				var itin models.Itinerary
				if err := app.DB.FindOne(ctx, itineraryCollection, bson.M{"itineraryid": entityID}, &itin); err != nil {
					return fmt.Errorf("not found")
				}
				if itin.UserID != userID {
					return fmt.Errorf("forbidden")
				}
				return nil
			},
			nil,
		)
	}
}

/* ---------------------------------------------------- */
/* Cross-collection cleanup                             */
/* ---------------------------------------------------- */

func deleteRelatedData(ctx context.Context, eventID string, app *infra.Deps) (int64, error) {
	if err := app.DB.DeleteMany(ctx, ticketsCollection, bson.M{"eventid": eventID}); err != nil {
		return 0, err
	}
	if err := app.DB.DeleteMany(ctx, mediaCollection, bson.M{"eventid": eventID}); err != nil {
		return 0, err
	}
	if err := app.DB.DeleteMany(ctx, merchCollection, bson.M{"eventid": eventID}); err != nil {
		return 0, err
	}
	return app.DB.DeleteOne(ctx, artistEventsCollection, bson.M{"eventid": eventID})
}

func isAdmin(ctx context.Context) bool {
	role, ok := ctx.Value(roleKey).(string)
	return ok && role == "admin"
}

const roleKey = "role"
