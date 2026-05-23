package pay

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"naevis/infra"
	"naevis/models"
	"naevis/utils"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

const (
	webhookCollection     = "payment_webhooks"
	webhookAttempts       = "webhook_attempts"
	maxWebhookRetries     = 3
	webhookTimeoutSeconds = 30
)

// PaymentWebhookPayload represents incoming payment webhook data
type PaymentWebhookPayload struct {
	EventID          string                 `json:"eventId"`
	TransactionID    string                 `json:"transactionId"`
	OrderID          string                 `json:"orderId"`
	UserID           string                 `json:"userId"`
	Amount           float64                `json:"amount"`
	Currency         string                 `json:"currency"`
	Status           string                 `json:"status"` // "success", "failed", "pending"
	PaymentMethod    string                 `json:"paymentMethod"`
	PaymentTimestamp int64                  `json:"timestamp"`
	Signature        string                 `json:"signature"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// VerifyWebhookSignature validates webhook signature
func VerifyWebhookSignature(payload []byte, signature string, secret string) bool {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	expectedSignature := hex.EncodeToString(h.Sum(nil))
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// HandlePaymentWebhook processes incoming payment webhooks
func (p *PaymentService) HandlePaymentWebhook(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	ctx, cancel := context.WithTimeout(r.Context(), webhookTimeoutSeconds*time.Second)
	defer cancel()

	// Read and validate webhook payload
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read webhook body: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// Verify webhook signature using environment variable
	webhookSecret := os.Getenv("PAYMENT_WEBHOOK_SECRET")
	if webhookSecret == "" {
		log.Printf("PAYMENT_WEBHOOK_SECRET not configured")
		utils.RespondWithError(w, http.StatusInternalServerError, "Webhook not configured")
		return
	}

	signature := r.Header.Get("X-Webhook-Signature")
	if signature == "" {
		log.Printf("Missing webhook signature header from %s", r.RemoteAddr)
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	if !VerifyWebhookSignature(body, signature, webhookSecret) {
		log.Printf("Invalid webhook signature from %s", r.RemoteAddr)
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var payload PaymentWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Printf("Failed to parse webhook payload: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid payload")
		return
	}

	// Validate webhook payload
	if err := validateWebhookPayload(&payload); err != nil {
		log.Printf("Invalid webhook payload: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Check if webhook already processed (idempotency)
	var existingWebhook bson.M
	if err := p.app.DB.FindOne(ctx, webhookCollection, bson.M{
		"transactionId": payload.TransactionID,
	}, &existingWebhook); err == nil {
		// Webhook already processed, return success
		utils.RespondWithJSON(w, http.StatusOK, map[string]string{
			"status": "already_processed",
		})
		return
	}

	// Process webhook based on status
	switch payload.Status {
	case "success":
		if err := p.processSuccessfulPayment(ctx, &payload); err != nil {
			logWebhookAttempt(ctx, p.app, &payload, "failed", err.Error())
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to process payment")
			return
		}

	case "failed":
		if err := p.processFailedPayment(ctx, &payload); err != nil {
			logWebhookAttempt(ctx, p.app, &payload, "failed", err.Error())
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to process failure")
			return
		}

	default:
		utils.RespondWithError(w, http.StatusBadRequest, "Unknown payment status")
		return
	}

	// Record successful webhook processing
	if err := p.app.DB.InsertOne(ctx, webhookCollection, bson.M{
		"transactionId": payload.TransactionID,
		"orderId":       payload.OrderID,
		"userId":        payload.UserID,
		"status":        payload.Status,
		"amount":        payload.Amount,
		"processedAt":   time.Now(),
	}); err != nil {
		log.Printf("Failed to record webhook: %v", err)
	}

	logWebhookAttempt(ctx, p.app, &payload, "processed", "")

	utils.RespondWithJSON(w, http.StatusOK, map[string]string{
		"status": "processed",
	})
}

// processSuccessfulPayment updates transaction status and applies balance changes
func (p *PaymentService) processSuccessfulPayment(ctx context.Context, payload *PaymentWebhookPayload) error {
	// Fetch the original transaction to determine its type
	var txn models.Transaction
	if err := p.app.DB.FindOne(ctx, transactionsCollection, bson.M{
		"_id": payload.TransactionID,
	}, &txn); err != nil {
		return fmt.Errorf("transaction not found: %w", err)
	}

	// For topup transactions, increment the account balance
	// For other transaction types, balance should already be updated
	if txn.Type == "topup" {
		if err := p.app.DB.UpdateOne(ctx, accountsCollection, bson.M{
			"userid": payload.UserID,
		}, bson.M{
			"$inc": bson.M{
				"cached_balance": int64(payload.Amount),
			},
			"$set": bson.M{
				"updated_at": time.Now(),
			},
		}); err != nil {
			return err
		}
	}

	// Update transaction status to success
	if err := p.app.DB.UpdateOne(ctx, transactionsCollection, bson.M{
		"_id": payload.TransactionID,
	}, bson.M{
		"$set": bson.M{
			"status":     "success",
			"updated_at": time.Now(),
		},
	}); err != nil {
		return err
	}

	return nil
}

// processFailedPayment marks transaction as failed
func (p *PaymentService) processFailedPayment(ctx context.Context, payload *PaymentWebhookPayload) error {
	if err := p.app.DB.UpdateOne(ctx, transactionsCollection, bson.M{
		"_id": payload.TransactionID,
	}, bson.M{
		"$set": bson.M{
			"status":     "failed",
			"updated_at": time.Now(),
		},
	}); err != nil {
		return err
	}

	return nil
}

// validateWebhookPayload checks required fields
func validateWebhookPayload(payload *PaymentWebhookPayload) error {
	if payload.TransactionID == "" {
		return fmt.Errorf("transaction_id_required: Transaction ID is required")
	}

	if payload.Amount <= 0 {
		return fmt.Errorf("invalid_amount: Amount must be positive")
	}

	if payload.Status == "" {
		return fmt.Errorf("status_required: Payment status is required")
	}

	// Verify timestamp is not too old (prevent replay attacks)
	if time.Now().Unix()-payload.PaymentTimestamp > 3600 { // 1 hour
		return fmt.Errorf("stale_timestamp: Payment timestamp too old")
	}

	return nil
}

// logWebhookAttempt records webhook processing attempt
func logWebhookAttempt(ctx context.Context, app *infra.Deps, payload *PaymentWebhookPayload, status string, reason string) {
	_, _ = ctx, app
	log.Printf("Webhook: %s - TxnID: %s, Status: %s, Reason: %s", status, payload.TransactionID, payload.Status, reason)
	// TODO: Store in database for monitoring
}
