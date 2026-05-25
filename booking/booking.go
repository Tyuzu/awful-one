package booking

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"

	"naevis/infra"
)

// ---------- Bookings ----------

func CreateBooking(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		var p Booking
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}

		if p.UserId == "" || p.EntityType == "" || p.EntityId == "" || p.Date == "" || p.Start == "" {
			http.Error(w, "missing fields", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// one booking per user per date (excluding cancelled)
		count, err := app.DB.CountDocuments(ctx, bookingsCollection, bson.M{
			"entityType": p.EntityType,
			"entityId":   p.EntityId,
			"userId":     p.UserId,
			"date":       p.Date,
			"status":     bson.M{"$ne": "cancelled"},
		})
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		if count > 0 {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok":     false,
				"reason": "one-per-day",
			})
			return
		}

		// SLOT-BASED booking
		if p.SlotId != "" {
			var slot Slot
			if err := app.DB.FindOne(ctx, slotsCollection, bson.M{"id": p.SlotId}, &slot); err != nil {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"ok":     false,
					"reason": "slot-missing",
				})
				return
			}

			slotCount, err := app.DB.CountDocuments(ctx, bookingsCollection, bson.M{
				"entityType": p.EntityType,
				"entityId":   p.EntityId,
				"slotId":     p.SlotId,
				"status":     bson.M{"$ne": "cancelled"},
			})
			if err != nil {
				http.Error(w, "db error", http.StatusInternalServerError)
				return
			}

			if int(slotCount) >= slot.Capacity {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"ok":     false,
					"reason": "slot-full",
				})
				return
			}

			if slot.TierId != "" {
				p.TierId = slot.TierId
				p.TierName = slot.TierName
			}
		}

		// TIER-BASED booking
		if p.SlotId == "" && p.TierId != "" {
			var tier Tier
			if err := app.DB.FindOne(ctx, tiersCollection, bson.M{"id": p.TierId}, &tier); err != nil {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"ok":     false,
					"reason": "tier-missing",
				})
				return
			}

			tCount, err := app.DB.CountDocuments(ctx, bookingsCollection, bson.M{
				"entityType": p.EntityType,
				"entityId":   p.EntityId,
				"tierId":     p.TierId,
				"date":       p.Date,
				"status":     bson.M{"$ne": "cancelled"},
			})
			if err != nil {
				http.Error(w, "db error", http.StatusInternalServerError)
				return
			}

			if int(tCount) >= tier.Capacity {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"ok":     false,
					"reason": "tier-full",
				})
				return
			}

			p.TierName = tier.Name
			if p.PricePaid == 0 {
				p.PricePaid = tier.Price
			}
		}

		// DATE capacity (no slot, no tier)
		if p.SlotId == "" && p.TierId == "" {
			var dc DateCap
			if err := app.DB.FindOne(ctx, dateCapsCollection, bson.M{
				"entityType": p.EntityType,
				"entityId":   p.EntityId,
				"date":       p.Date,
			}, &dc); err == nil {

				total, err := app.DB.CountDocuments(ctx, bookingsCollection, bson.M{
					"entityType": p.EntityType,
					"entityId":   p.EntityId,
					"date":       p.Date,
					"status":     bson.M{"$ne": "cancelled"},
				})
				if err != nil {
					http.Error(w, "db error", http.StatusInternalServerError)
					return
				}

				if int(total) >= dc.Capacity {
					_ = json.NewEncoder(w).Encode(map[string]any{
						"ok":     false,
						"reason": "date-full",
					})
					return
				}
			}
		}

		p.ID = genID()
		p.Status = "pending"
		p.CreatedAt = time.Now().Unix()

		if err := app.DB.InsertOne(ctx, bookingsCollection, p); err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":      true,
			"booking": p,
		})
	}
}

// ---------- Booking status ----------

func UpdateBookingStatus(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		bookingID := ps.ByName("id")
		if bookingID == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}

		var body struct {
			Status string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}

		if body.Status != "pending" && body.Status != "confirmed" && body.Status != "cancelled" {
			http.Error(w, "invalid status", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var updated Booking
		err := app.DB.FindOneAndUpdate(
			ctx,
			bookingsCollection,
			bson.M{"id": bookingID},
			bson.M{"$set": bson.M{"status": body.Status}},
			&updated,
		)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":      true,
			"booking": updated,
		})
	}
}

func CancelBooking(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		bookingID := ps.ByName("id")
		if bookingID == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var updated Booking
		err := app.DB.FindOneAndUpdate(
			ctx,
			bookingsCollection,
			bson.M{"id": bookingID},
			bson.M{"$set": bson.M{"status": "cancelled"}},
			&updated,
		)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":      true,
			"booking": updated,
		})
	}
}

// ---------- Date capacity ----------

// ---------- Date capacity ----------

func SetDateCapacity(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		var p DateCap
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}

		if p.EntityType == "" || p.EntityId == "" || p.Date == "" || p.Capacity <= 0 {
			http.Error(w, "missing fields", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := app.DB.UpdateOne(
			ctx,
			dateCapsCollection,
			bson.M{
				"entityType": p.EntityType,
				"entityId":   p.EntityId,
				"date":       p.Date,
			},
			bson.M{"$set": p},
		)
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
		})
	}
}
