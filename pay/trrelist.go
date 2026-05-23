package pay

import (
	"encoding/json"
	"naevis/models"
	"naevis/utils"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

func (p *PaymentService) Transfer(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	ctx := r.Context()
	senderID := utils.GetUserIDFromRequest(r)

	var req struct {
		Recipient string `json:"recipient"`
		Amount    int64  `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Amount <= 0 || req.Recipient == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request")
		return
	}

	senderAcc, err := p.getOrCreateAccount(ctx, senderID)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "account error")
		return
	}
	recipientAcc, err := p.getOrCreateAccount(ctx, req.Recipient)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "recipient error")
		return
	}

	// deterministic lock ordering
	lockA, lockB := senderAcc, recipientAcc
	if lockB < lockA {
		lockA, lockB = lockB, lockA
	}

	ok, _ := p.lock(ctx, lockA)
	if !ok {
		utils.RespondWithError(w, http.StatusTooManyRequests, "retry")
		return
	}
	defer p.unlock(ctx, lockA)

	ok, _ = p.lock(ctx, lockB)
	if !ok {
		utils.RespondWithError(w, http.StatusTooManyRequests, "retry")
		return
	}
	defer p.unlock(ctx, lockB)

	var sender models.Account
	if err := p.app.DB.FindOne(ctx, accountsCollection, map[string]any{"_id": senderAcc}, &sender); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "account error")
		return
	}

	if sender.CachedBalance < req.Amount {
		utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"message": "insufficient balance",
		})
		return
	}

	txnID := utils.GetUUID()
	now := time.Now()

	master := models.Transaction{
		ID:          txnID,
		Type:        "transfer",
		Method:      "wallet",
		FromAccount: senderAcc,
		ToAccount:   recipientAcc,
		Amount:      req.Amount,
		Currency:    "INR",
		Status:      "initiated",
		CreatedAt:   now,
		UpdatedAt:   now,
		Meta:        models.Meta{"note": "user transfer"},
	}

	if err := p.app.DB.InsertOne(ctx, transactionsCollection, master); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed")
		return
	}

	j := models.JournalEntry{
		ID:            utils.GetUUID(),
		TxnID:         txnID,
		DebitAccount:  senderAcc,
		CreditAccount: recipientAcc,
		Amount:        req.Amount,
		Currency:      "INR",
		CreatedAt:     now,
	}

	if err := p.app.DB.InsertOne(ctx, journalCollection, j); err != nil {
		p.failTxn(ctx, txnID)
		utils.RespondWithError(w, http.StatusInternalServerError, "failed")
		return
	}

	if err := p.app.DB.Inc(ctx, accountsCollection, map[string]any{"_id": senderAcc}, "cached_balance", -req.Amount); err != nil {
		p.failTxn(ctx, txnID)
		utils.RespondWithError(w, http.StatusInternalServerError, "failed")
		return
	}

	if err := p.app.DB.Inc(ctx, accountsCollection, map[string]any{"_id": recipientAcc}, "cached_balance", req.Amount); err != nil {
		p.failTxn(ctx, txnID)
		utils.RespondWithError(w, http.StatusInternalServerError, "failed")
		return
	}

	// derived per-user views (best-effort)
	_ = p.app.DB.InsertMany(ctx, transactionsCollection, []interface{}{
		models.Transaction{
			ID:        utils.GetUUID(),
			ParentTxn: txnID,
			UserID:    senderID,
			Type:      "debit",
			Amount:    req.Amount,
			Status:    "success",
			CreatedAt: now,
		},
		models.Transaction{
			ID:        utils.GetUUID(),
			ParentTxn: txnID,
			UserID:    req.Recipient,
			Type:      "credit",
			Amount:    req.Amount,
			Status:    "success",
			CreatedAt: now,
		},
	})

	p.successTxn(ctx, txnID)

	utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success":        true,
		"transaction_id": txnID,
	})
}

func (p *PaymentService) Refund(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	ctx := r.Context()
	userID := utils.GetUserIDFromRequest(r)

	var req struct {
		TransactionID string `json:"transaction_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TransactionID == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request")
		return
	}

	var orig models.Transaction
	if err := p.app.DB.FindOne(ctx, transactionsCollection, map[string]any{"_id": req.TransactionID}, &orig); err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "not found")
		return
	}

	// Verify user owns the transaction (they are the original payer)
	if orig.UserID != "" && orig.UserID != userID {
		utils.RespondWithError(w, http.StatusForbidden, "unauthorized")
		return
	}

	if orig.Status != "success" {
		utils.RespondWithError(w, http.StatusBadRequest, "not refundable")
		return
	}

	fromAcc := orig.ToAccount
	toAcc := orig.FromAccount

	lockA, lockB := fromAcc, toAcc
	if lockB < lockA {
		lockA, lockB = lockB, lockA
	}

	ok, _ := p.lock(ctx, lockA)
	if !ok {
		utils.RespondWithError(w, http.StatusTooManyRequests, "retry")
		return
	}
	defer p.unlock(ctx, lockA)

	ok, _ = p.lock(ctx, lockB)
	if !ok {
		utils.RespondWithError(w, http.StatusTooManyRequests, "retry")
		return
	}
	defer p.unlock(ctx, lockB)

	txnID := utils.GetUUID()
	now := time.Now()

	refund := models.Transaction{
		ID:          txnID,
		UserID:      orig.UserID,
		Type:        "refund",
		Method:      "wallet",
		FromAccount: fromAcc,
		ToAccount:   toAcc,
		Amount:      orig.Amount,
		Currency:    orig.Currency,
		Status:      "initiated",
		CreatedAt:   now,
		UpdatedAt:   now,
		Meta:        models.Meta{"original_txn": orig.ID},
	}

	if err := p.app.DB.InsertOne(ctx, transactionsCollection, refund); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed")
		return
	}

	j := models.JournalEntry{
		ID:            utils.GetUUID(),
		TxnID:         txnID,
		DebitAccount:  fromAcc,
		CreditAccount: toAcc,
		Amount:        refund.Amount,
		Currency:      refund.Currency,
		CreatedAt:     now,
	}

	if err := p.app.DB.InsertOne(ctx, journalCollection, j); err != nil {
		p.failTxn(ctx, txnID)
		http.Error(w, "failed", http.StatusInternalServerError)
		return
	}

	if err := p.app.DB.Inc(ctx, accountsCollection, map[string]any{"_id": fromAcc}, "cached_balance", -refund.Amount); err != nil {
		p.failTxn(ctx, txnID)
		utils.RespondWithError(w, http.StatusInternalServerError, "failed")
		return
	}

	if err := p.app.DB.Inc(ctx, accountsCollection, map[string]any{"_id": toAcc}, "cached_balance", refund.Amount); err != nil {
		p.failTxn(ctx, txnID)
		utils.RespondWithError(w, http.StatusInternalServerError, "failed")
		return
	}

	p.successTxn(ctx, txnID)

	// mark original reversed (best-effort)
	_ = p.app.DB.UpdateOne(ctx, transactionsCollection,
		map[string]any{"_id": orig.ID},
		map[string]any{"$set": map[string]any{"status": "reversed", "updated_at": now}},
	)

	utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success":        true,
		"transaction_id": txnID,
	})
}

func (p *PaymentService) ListTransactions(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	ctx := r.Context()
	userID := utils.GetUserIDFromRequest(r)

	var txns []models.Transaction
	if err := p.app.DB.FindMany(ctx, transactionsCollection,
		map[string]any{
			"$or": []map[string]any{
				{"userid": userID},
				{"meta.recipient": userID},
			},
		}, &txns,
	); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, txns)
}
