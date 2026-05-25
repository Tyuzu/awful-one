package merch

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"naevis/auditlog"
	"naevis/globals"
	"naevis/infra"
	"naevis/models"
	"naevis/userdata"
	"naevis/utils"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

func respond(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}

func validateEntityType(t string) bool {
	return t == "event" || t == "farm" || t == "artist"
}

// ---------------------- Create Merch ----------------------
func CreateMerch(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		entityType := ps.ByName("entityType")
		eventID := ps.ByName("eventid")

		if !validateEntityType(entityType) {
			respond(w, 400, map[string]any{"success": false, "error": "invalid entity type"})
			return
		}

		// SECURITY: Verify user is authenticated
		userID, ok := r.Context().Value(globals.UserIDKey).(string)
		if !ok || userID == "" {
			respond(w, 401, map[string]any{"success": false, "error": "unauthorized"})
			return
		}

		// SECURITY: Verify user is the owner of the entity
		collection := ""
		idField := ""
		ownerField := ""

		switch entityType {
		case "event":
			collection = "events"
			idField = "eventid"
			ownerField = "creatorid"
		case "farm":
			collection = "farms"
			idField = "farmid"
			ownerField = "createdBy"
		case "artist":
			collection = "artists"
			idField = "artistid"
			ownerField = "creatorid"
		}

		if collection != "" {
			var ownerEntity bson.M
			err := app.DB.FindOne(r.Context(), collection, bson.M{
				idField: eventID,
			}, &ownerEntity)

			if err != nil {
				respond(w, 404, map[string]any{"success": false, "error": "entity not found"})
				return
			}

			// Check ownership based on entity type
			owner, ok := ownerEntity[ownerField].(string)
			if !ok {
				respond(w, 403, map[string]any{"success": false, "error": "cannot verify ownership"})
				return
			}

			if owner != userID {
				respond(w, 403, map[string]any{"success": false, "error": "forbidden: only entity owner can create merch"})
				return
			}
		}

		var body struct {
			Name  string  `json:"name"`
			Price float64 `json:"price"`
			Stock int     `json:"stock"`
			Photo string  `json:"merch_pic"`
		}

		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			respond(w, 400, map[string]any{"success": false, "error": "invalid json"})
			return
		}

		if body.Name == "" || body.Price <= 0 || body.Stock < 0 {
			respond(w, 400, map[string]any{"success": false, "error": "invalid merch data"})
			return
		}

		now := time.Now()
		merch := models.Merch{
			MerchID:    utils.GenerateRandomString(14),
			EntityType: entityType,
			EntityID:   eventID,
			Name:       body.Name,
			Price:      body.Price,
			Stock:      body.Stock,
			MerchPhoto: body.Photo,
			CreatedAt:  now,
			UpdatedAt:  now,
		}

		if err := app.DB.Insert(r.Context(), merchCollection, merch); err != nil {
			respond(w, 500, map[string]any{"success": false, "error": "insert failed"})
			return
		}

		// Log audit trail for merchandise creation
		auditlog.LogAction(
			r.Context(), app, r, userID,
			models.AuditActionMerchCreate,
			"merchandise", merch.MerchID, "success",
			map[string]interface{}{
				"name":        merch.Name,
				"price":       merch.Price,
				"stock":       merch.Stock,
				"entity_type": entityType,
				"entity_id":   eventID,
			},
		)

		respond(w, 201, map[string]any{"success": true, "data": merch})
	}
}

// ---------------------- Edit Merch ----------------------
func EditMerch(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		entityType := ps.ByName("entityType")
		eventID := ps.ByName("eventid")
		merchID := ps.ByName("merchid")

		// SECURITY: Verify user is authenticated
		userID, ok := r.Context().Value(globals.UserIDKey).(string)
		if !ok || userID == "" {
			respond(w, 401, map[string]any{"success": false, "error": "unauthorized"})
			return
		}

		// SECURITY: Verify user is the owner of the entity
		collection := ""
		idField := ""
		ownerField := ""

		switch entityType {
		case "event":
			collection = "events"
			idField = "eventid"
			ownerField = "creatorid"
		case "farm":
			collection = "farms"
			idField = "farmid"
			ownerField = "createdBy"
		case "artist":
			collection = "artists"
			idField = "artistid"
			ownerField = "creatorid"
		}

		if collection != "" {
			var ownerEntity bson.M
			err := app.DB.FindOne(r.Context(), collection, bson.M{
				idField: eventID,
			}, &ownerEntity)

			if err != nil {
				respond(w, 404, map[string]any{"success": false, "error": "entity not found"})
				return
			}

			owner, ok := ownerEntity[ownerField].(string)
			if !ok || owner != userID {
				respond(w, 403, map[string]any{"success": false, "error": "forbidden: only entity owner can edit merch"})
				return
			}
		}

		var body struct {
			Name  *string  `json:"name"`
			Price *float64 `json:"price"`
			Stock *int     `json:"stock"`
		}

		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			respond(w, 400, map[string]any{"success": false, "error": "invalid json"})
			return
		}

		update := bson.M{
			"updatedat": time.Now(),
		}

		if body.Name != nil {
			update["name"] = *body.Name
		}
		if body.Price != nil && *body.Price > 0 {
			update["price"] = *body.Price
		}
		if body.Stock != nil && *body.Stock >= 0 {
			update["stock"] = *body.Stock
		}

		err := app.DB.UpdateOne(
			r.Context(),
			merchCollection,
			bson.M{
				"entity_type": entityType,
				"entity_id":   eventID,
				"merchid":     merchID,
			},
			bson.M{"$set": update},
		)
		if err != nil {
			respond(w, 404, map[string]any{"success": false, "error": "merch not found"})
			return
		}

		// Log audit trail for merchandise update
		auditlog.LogAction(
			r.Context(), app, r, userID,
			models.AuditActionMerchUpdate,
			"merchandise", merchID, "success",
			map[string]interface{}{
				"updates": update,
			},
		)

		respond(w, 200, map[string]any{"success": true})
	}
}

// ---------------------- Delete Merch ----------------------
func DeleteMerch(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		entityType := ps.ByName("entityType")
		eventID := ps.ByName("eventid")
		merchID := ps.ByName("merchid")

		// SECURITY: Verify user is authenticated
		userID, ok := r.Context().Value(globals.UserIDKey).(string)
		if !ok || userID == "" {
			respond(w, 401, map[string]any{"success": false, "error": "unauthorized"})
			return
		}

		// SECURITY: Verify user is the owner of the entity
		collection := ""
		idField := ""
		ownerField := ""

		switch entityType {
		case "event":
			collection = "events"
			idField = "eventid"
			ownerField = "creatorid"
		case "farm":
			collection = "farms"
			idField = "farmid"
			ownerField = "createdBy"
		case "artist":
			collection = "artists"
			idField = "artistid"
			ownerField = "creatorid"
		}

		if collection != "" {
			var ownerEntity bson.M
			err := app.DB.FindOne(r.Context(), collection, bson.M{
				idField: eventID,
			}, &ownerEntity)

			if err != nil {
				respond(w, 404, map[string]any{"success": false, "error": "entity not found"})
				return
			}

			owner, ok := ownerEntity[ownerField].(string)
			if !ok || owner != userID {
				respond(w, 403, map[string]any{"success": false, "error": "forbidden: only entity owner can delete merch"})
				return
			}
		}

		// SECURITY: Use soft delete instead of hard delete
		now := time.Now()
		err := app.DB.UpdateOne(
			r.Context(),
			merchCollection,
			bson.M{
				"entity_type": entityType,
				"entity_id":   eventID,
				"merchid":     merchID,
				"deletedAt":   bson.M{"$exists": false}, // Only soft-delete if not already deleted
			},
			bson.M{"$set": bson.M{
				"deletedAt": now,
				"updatedat": now,
			}},
		)
		if err != nil {
			respond(w, 404, map[string]any{"success": false, "error": "merch not found"})
			return
		}

		// Log audit trail for merchandise deletion
		auditlog.LogAction(
			r.Context(), app, r, userID,
			models.AuditActionMerchDelete,
			"merchandise", merchID, "success",
			map[string]interface{}{
				"deleted_at": now,
			},
		)

		respond(w, 200, map[string]any{"success": true})
	}
}

// ---------------------- Buy Merch ----------------------
func BuyMerch(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		userID, ok := r.Context().Value(globals.UserIDKey).(string)
		if !ok || userID == "" {
			respond(w, 401, map[string]any{"success": false, "error": "unauthorized"})
			return
		}

		var body struct {
			Quantity int `json:"quantity"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Quantity < 1 {
			respond(w, 400, map[string]any{"success": false, "error": "invalid quantity"})
			return
		}

		err := app.DB.WithDB(r.Context(), func(ctx context.Context) error {
			var merch models.Merch
			err := app.DB.FindOne(ctx, merchCollection, bson.M{
				"entity_type": ps.ByName("entityType"),
				"entity_id":   ps.ByName("eventid"),
				"merchid":     ps.ByName("merchid"),
			}, &merch)
			if err != nil {
				return errors.New("merch not found")
			}

			if merch.Stock < body.Quantity {
				return errors.New("insufficient stock")
			}

			err = app.DB.UpdateOne(
				ctx,
				merchCollection,
				bson.M{"merchid": merch.MerchID},
				bson.M{"$inc": bson.M{"stock": -body.Quantity}},
			)
			if err != nil {
				return err
			}

			userdata.SetUserData(
				"merch",
				merch.MerchID,
				userID,
				merch.EntityType,
				merch.EntityID,
				app,
			)

			return nil
		})

		if err != nil {
			respond(w, 400, map[string]any{"success": false, "error": err.Error()})
			return
		}

		respond(w, 200, map[string]any{
			"success": true,
			"message": "purchase successful",
		})
	}
}
