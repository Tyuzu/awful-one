package stripe

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"naevis/infra"
	"naevis/utils"

	"github.com/joho/godotenv"
	"github.com/julienschmidt/httprouter"
	"github.com/stripe/stripe-go/v83"
	"github.com/stripe/stripe-go/v83/paymentintent"
	"github.com/stripe/stripe-go/v83/webhook"
	"go.mongodb.org/mongo-driver/bson"
)

func init() {
	_ = godotenv.Load()
	key := os.Getenv("STRIPE_SECRET_KEY")
	if key == "" {
		log.Fatal("STRIPE_SECRET_KEY not set")
	}
	stripe.Key = key
}

/* ----------------------------------------
   Helpers
---------------------------------------- */

func updatePaymentStatus(
	ctx context.Context,
	entityType string,
	entityId string,
	amount int64,
	paymentIntentId string,
	app *infra.Deps,
) error {

	var collection string
	var idField string

	switch entityType {
	case "funding":
		collection = fundingCollection
		idField = "fundingid"
	case "order":
		collection = stripeOrdersCollection
		idField = "orderid"
	default:
		return errors.New("invalid entityType")
	}

	update := bson.M{
		"paid":            true,
		"amount":          amount,
		"paymentIntentId": paymentIntentId,
		"paidAt":          time.Now().UTC(),
	}

	return app.DB.Update(
		ctx,
		collection,
		bson.M{idField: entityId},
		update,
	)
}

/* ----------------------------------------
   Create Payment Intent
---------------------------------------- */

type CreatePaymentIntentRequest struct {
	Amount      int64  `json:"amount"`
	Currency    string `json:"currency"`
	EntityType  string `json:"entityType"`
	EntityId    string `json:"entityId"`
	Description string `json:"description"`
}

func CreatePaymentIntent(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		var req CreatePaymentIntentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid JSON")
			return
		}

		if req.Amount <= 0 {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid amount")
			return
		}
		if req.Currency == "" {
			req.Currency = "usd"
		}
		if req.EntityType == "" || req.EntityId == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "Missing entity info")
			return
		}

		params := &stripe.PaymentIntentParams{
			Amount:   stripe.Int64(req.Amount),
			Currency: stripe.String(req.Currency),
			Metadata: map[string]string{
				"entityType":  req.EntityType,
				"entityId":    req.EntityId,
				"description": req.Description,
			},
		}

		pi, err := paymentintent.New(params)
		if err != nil {
			log.Println("Stripe PaymentIntent error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create payment intent")
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, map[string]string{
			"clientSecret": pi.ClientSecret,
		})
	}
}

/* ----------------------------------------
   Manual Payment Success Callback
---------------------------------------- */

type PaymentSuccessRequest struct {
	EntityType      string `json:"entityType"`
	EntityId        string `json:"entityId"`
	PaymentIntentId string `json:"paymentIntentId"`
	Amount          int64  `json:"amount"`
}

func PaymentSuccess(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		var req PaymentSuccessRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid JSON")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		if err := updatePaymentStatus(
			ctx,
			req.EntityType,
			req.EntityId,
			req.Amount,
			req.PaymentIntentId,
			app,
		); err != nil {
			log.Println("Payment status update failed:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Database update failed")
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

/* ----------------------------------------
   Stripe Webhook
---------------------------------------- */

func StripeWebhook(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		const MaxBodyBytes = int64(65536)
		r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)

		payload, err := io.ReadAll(r.Body)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid payload")
			return
		}

		sigHeader := r.Header.Get("Stripe-Signature")
		secret := os.Getenv("STRIPE_WEBHOOK_SECRET")
		if secret == "" {
			log.Println("STRIPE_WEBHOOK_SECRET missing")
			utils.RespondWithError(w, http.StatusInternalServerError, "Webhook not configured")
			return
		}

		event, err := webhook.ConstructEvent(payload, sigHeader, secret)
		if err != nil {
			log.Println("Webhook signature verification failed:", err)
			utils.RespondWithError(w, http.StatusBadRequest, "Signature verification failed")
			return
		}

		switch event.Type {

		case "payment_intent.succeeded":
			var pi stripe.PaymentIntent
			if err := json.Unmarshal(event.Data.Raw, &pi); err == nil {
				entityType := pi.Metadata["entityType"]
				entityId := pi.Metadata["entityId"]

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				err := updatePaymentStatus(
					ctx,
					entityType,
					entityId,
					pi.Amount,
					pi.ID,
					app,
				)
				cancel()

				if err != nil {
					log.Println("PaymentIntent success DB update failed:", err)
				}
			}

		case "payment_intent.payment_failed":
			var pi stripe.PaymentIntent
			if err := json.Unmarshal(event.Data.Raw, &pi); err == nil {
				log.Printf("PaymentIntent failed: %s error=%v\n", pi.ID, pi.LastPaymentError)
			}

		default:
			log.Println("Unhandled Stripe event:", event.Type)
		}

		w.WriteHeader(http.StatusOK)
	}
}
