package booking

import (
	"context"
	"encoding/json"
	"naevis/infra"
	"naevis/utils"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

// ---------- Utility ----------
func genID() string {
	return utils.GenerateRandomDigitString(22)
}

// ---------- Models ----------
type Slot struct {
	ID         string `json:"id" bson:"id"`
	EntityType string `json:"entityType" bson:"entityType"`
	EntityId   string `json:"entityId" bson:"entityId"`
	Date       string `json:"date" bson:"date"`
	Start      string `json:"start" bson:"start"`
	End        string `json:"end,omitempty" bson:"end,omitempty"`
	Capacity   int    `json:"capacity" bson:"capacity"`
	TierId     string `json:"tierId,omitempty" bson:"tierId,omitempty"`
	TierName   string `json:"tierName,omitempty" bson:"tierName,omitempty"`
	CreatedAt  int64  `json:"createdAt" bson:"createdAt"`
}

type Booking struct {
	ID         string  `json:"id" bson:"id"`
	SlotId     string  `json:"slotId,omitempty" bson:"slotId,omitempty"`
	TierId     string  `json:"tierId,omitempty" bson:"tierId,omitempty"`
	TierName   string  `json:"tierName,omitempty" bson:"tierName,omitempty"`
	PricePaid  float64 `json:"pricePaid,omitempty" bson:"pricePaid,omitempty"`
	EntityType string  `json:"entityType" bson:"entityType"`
	EntityId   string  `json:"entityId" bson:"entityId"`
	UserId     string  `json:"userId" bson:"userId"`
	Date       string  `json:"date" bson:"date"`
	Start      string  `json:"start" bson:"start"`
	End        string  `json:"end,omitempty" bson:"end,omitempty"`
	Status     string  `json:"status" bson:"status"` // pending, confirmed, cancelled
	CreatedAt  int64   `json:"createdAt" bson:"createdAt"`
}

type DateCap struct {
	EntityType string `json:"entityType" bson:"entityType"`
	EntityId   string `json:"entityId" bson:"entityId"`
	Date       string `json:"date" bson:"date"`
	Capacity   int    `json:"capacity" bson:"capacity"`
}

type Tier struct {
	ID         string   `json:"id" bson:"id"`
	EntityType string   `json:"entityType" bson:"entityType"`
	EntityId   string   `json:"entityId" bson:"entityId"`
	Name       string   `json:"name" bson:"name"`
	Price      float64  `json:"price" bson:"price"`
	Capacity   int      `json:"capacity" bson:"capacity"`
	TimeRange  []string `json:"timeRange,omitempty" bson:"timeRange,omitempty"`   // ["09:00", "17:00"]
	DaysOfWeek []int    `json:"daysOfWeek,omitempty" bson:"daysOfWeek,omitempty"` // 0=Sun..6=Sat
	Features   []string `json:"features,omitempty" bson:"features,omitempty"`
	CreatedAt  int64    `json:"createdAt" bson:"createdAt"`
}

// ---------- Tier handlers ----------
func CreateTier(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		var tier Tier
		if err := json.NewDecoder(r.Body).Decode(&tier); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		// basic validation
		if tier.ID == "" || tier.EntityType == "" || tier.EntityId == "" || tier.Name == "" {
			http.Error(w, "missing required fields", http.StatusBadRequest)
			return
		}

		tier.CreatedAt = time.Now().Unix()

		if err := app.DB.InsertOne(r.Context(), tiersCollection, tier); err != nil {
			http.Error(w, "db insert failed", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]any{"tier": tier})
	}
}

func DeleteTier(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		tierId := ps.ByName("id")
		if tierId == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if _, err := app.DB.DeleteOne(ctx, tiersCollection, map[string]any{"id": tierId}); err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// ---------- Slots ----------
func CreateSlot(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		var s Slot
		if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}
		if s.EntityType == "" || s.EntityId == "" || s.Date == "" || s.Start == "" || s.Capacity <= 0 {
			http.Error(w, "missing required fields", http.StatusBadRequest)
			return
		}

		if s.TierId != "" {
			ctxTmp, cancelTmp := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancelTmp()
			var t Tier
			if err := app.DB.FindOne(ctxTmp, tiersCollection, map[string]any{"id": s.TierId}, &t); err == nil {
				s.TierName = t.Name
			}
		}

		s.ID = genID()
		s.CreatedAt = time.Now().Unix()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := app.DB.InsertOne(ctx, slotsCollection, s); err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"slot": s})
	}
}

func DeleteSlot(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		slotId := ps.ByName("id")
		if slotId == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if _, err := app.DB.DeleteOne(ctx, slotsCollection, map[string]any{"id": slotId}); err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		_ = app.DB.DeleteMany(ctx, bookingsCollection, map[string]any{"slotId": slotId})

		w.WriteHeader(http.StatusNoContent)
	}
}

// ---------- GenerateSlotsFromTier ----------
func GenerateSlotsFromTier(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		tierId := ps.ByName("id")
		if tierId == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}

		var body struct {
			StartDate string `json:"startDate"`
			EndDate   string `json:"endDate"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}
		if body.StartDate == "" || body.EndDate == "" {
			http.Error(w, "missing date range", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var tier Tier
		if err := app.DB.FindOne(ctx, tiersCollection, map[string]any{"id": tierId}, &tier); err != nil {
			http.Error(w, "tier not found", http.StatusNotFound)
			return
		}

		startDate, err1 := time.Parse("2006-01-02", body.StartDate)
		endDate, err2 := time.Parse("2006-01-02", body.EndDate)
		if err1 != nil || err2 != nil || startDate.After(endDate) {
			http.Error(w, "invalid date range", http.StatusBadRequest)
			return
		}

		var slots []Slot
		for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
			dow := int(d.Weekday())
			if len(tier.DaysOfWeek) > 0 {
				allowed := false
				for _, allowedDay := range tier.DaysOfWeek {
					if allowedDay == dow {
						allowed = true
						break
					}
				}
				if !allowed {
					continue
				}
			}

			start, end := "09:00", "17:00"
			if len(tier.TimeRange) == 2 {
				start, end = tier.TimeRange[0], tier.TimeRange[1]
			}

			slots = append(slots, Slot{
				ID:         genID(),
				EntityType: tier.EntityType,
				EntityId:   tier.EntityId,
				Date:       d.Format("2006-01-02"),
				Start:      start,
				End:        end,
				Capacity:   tier.Capacity,
				TierId:     tier.ID,
				TierName:   tier.Name,
				CreatedAt:  time.Now().Unix(),
			})
		}

		if len(slots) > 0 {
			docs := make([]any, len(slots))
			for i, s := range slots {
				docs[i] = s
			}
			if err := app.DB.InsertMany(ctx, slotsCollection, docs); err != nil {
				http.Error(w, "db error", http.StatusInternalServerError)
				return
			}
		}

		json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "slots": slots})
	}
}
