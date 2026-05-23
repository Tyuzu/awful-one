package pay

import (
	"encoding/json"
	"naevis/auditlog"
	"naevis/models"
	"naevis/utils"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

// CashOnDelivery handles cash-on-delivery payment requests
func (p *PaymentService) CashOnDelivery(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	ctx := r.Context()
	userID := utils.GetUserIDFromRequest(r)

	var req struct {
		PaymentType string `json:"paymentType"`
		EntityType  string `json:"entityType"`
		EntityID    string `json:"entityId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request")
		return
	}

	// ────────── VALIDATION ──────────
	if req.PaymentType == "" || req.EntityType == "" || req.EntityID == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "missing required fields: paymentType, entityType, or entityId")
		return
	}

	// Validate payment rules
	rule, ok := PaymentRules[req.PaymentType]
	if !ok {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid payment type")
		return
	}

	if !rule.AllowedEntities[req.EntityType] {
		utils.RespondWithError(w, http.StatusBadRequest, "entity not allowed for payment type")
		return
	}

	if !rule.AllowedMethods["cod"] {
		utils.RespondWithError(w, http.StatusBadRequest, "cash on delivery not allowed for this payment type")
		return
	}

	// ────────── PRICE RESOLUTION ──────────
	resolver, err := p.resolver(req.EntityType)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "unsupported entity")
		return
	}

	price, err := resolver(ctx, req.EntityID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "entity not found: "+req.EntityType+" ("+req.EntityID+")")
		return
	}

	// ────────── CREATE TRANSACTION RECORD ──────────
	txnID := utils.GetUUID()
	now := time.Now()

	txn := models.Transaction{
		ID:         txnID,
		UserID:     userID,
		Type:       "payment",
		Method:     "cash_on_delivery",
		EntityType: req.EntityType,
		EntityID:   req.EntityID,
		Amount:     price,
		Currency:   "INR",
		Status:     "pending",
		CreatedAt:  now,
		UpdatedAt:  now,
		Meta:       models.Meta{"payment_type": req.PaymentType},
	}

	if err := p.app.DB.InsertOne(ctx, transactionsCollection, txn); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to create transaction")
		return
	}

	// ────────── UPDATE ORDER STATUS ──────────
	if req.EntityType == "order" {
		filter := map[string]any{"orderId": req.EntityID}
		update := map[string]any{
			"$set": map[string]any{
				"paymentMethod": "cash_on_delivery",
				"status":        "cod_pending",
				"updatedAt":     now,
			},
		}
		if err := p.app.DB.UpdateOne(ctx, "orders", filter, update); err != nil {
			// Log but don't fail - transaction was created
			auditlog.LogAction(
				ctx, p.app, r, userID,
				models.AuditActionPayment,
				"payment_error", "order_update_failed", "warning",
				map[string]interface{}{
					"entity_type": req.EntityType,
					"entity_id":   req.EntityID,
					"error":       err.Error(),
				},
			)
		}
	}

	// ────────── AUDIT LOG ──────────
	auditlog.LogAction(
		ctx, p.app, r, userID,
		models.AuditActionPayment,
		"cash_on_delivery", req.EntityID, "success",
		map[string]interface{}{
			"amount":       price,
			"entity_type":  req.EntityType,
			"payment_type": req.PaymentType,
			"transaction":  txnID,
		},
	)

	utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}
