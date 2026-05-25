package cart

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"

	"naevis/infra"
)

/* ───────────────────────── Coupon Models ───────────────────────── */

type Coupon struct {
	Code       string    `bson:"code" json:"code"`
	Discount   float64   `bson:"discount" json:"discount"` // % value
	ExpiresAt  time.Time `bson:"expiresAt" json:"expiresAt"`
	Active     bool      `bson:"active" json:"active"`
	EntityID   string    `bson:"entityId" json:"entityId"`
	EntityType string    `bson:"entityType" json:"entityType"`
}

type CouponRequest struct {
	Code       string  `json:"code"`
	Cart       float64 `json:"cart"`
	EntityID   string  `json:"entityId"`
	EntityType string  `json:"entityType"`
}

type CouponResponse struct {
	Valid    bool    `json:"valid"`
	Discount float64 `json:"discount"`
	Message  string  `json:"message"`
}

/* ───────────────────────── Validate Coupon ───────────────────────── */

func ValidateCouponHandler(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var req CouponRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		code := strings.TrimSpace(strings.ToLower(req.Code))
		if code == "" {
			writeJSON(w, http.StatusBadRequest, CouponResponse{
				Valid:   false,
				Message: "Coupon code missing",
			})
			return
		}

		if req.EntityID == "" || req.EntityType == "" {
			writeJSON(w, http.StatusBadRequest, CouponResponse{
				Valid:   false,
				Message: "Entity details required",
			})
			return
		}

		filter := bson.M{
			"code":       code,
			"entityId":   req.EntityID,
			"entityType": strings.ToLower(req.EntityType),
			"active":     true,
		}

		var coupon Coupon
		if err := app.DB.FindOne(ctx, couponCollection, filter, &coupon); err != nil {
			writeJSON(w, http.StatusNotFound, CouponResponse{
				Valid:   false,
				Message: "Coupon not valid for this entity",
			})
			return
		}

		if time.Now().After(coupon.ExpiresAt) {
			writeJSON(w, http.StatusGone, CouponResponse{
				Valid:   false,
				Message: "Coupon expired",
			})
			return
		}

		discount := 0.0
		if req.Cart > 0 {
			discount = (req.Cart * coupon.Discount) / 100
		}

		writeJSON(w, http.StatusOK, CouponResponse{
			Valid:    true,
			Discount: discount,
			Message:  "Coupon applied",
		})
	}
}

/* ───────────────────────── Helpers ───────────────────────── */

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
