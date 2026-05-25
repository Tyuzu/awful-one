package auditlog

import (
	"context"
	"net/http"
	"time"

	"naevis/infra"
	"naevis/models"
	"naevis/utils"
)

const auditCollection = "auditlogs"

// LogAction records an audit event
func LogAction(
	ctx context.Context,
	app *infra.Deps,
	r *http.Request,
	userID string,
	action string,
	entityType string,
	entityID string,
	status string,
	changes map[string]interface{},
) {
	if app == nil || app.DB == nil {
		return // Skip if DB not available
	}

	// Extract IP address (handles proxies)
	ipAddr := r.Header.Get("X-Forwarded-For")
	if ipAddr == "" {
		ipAddr = r.Header.Get("X-Real-IP")
	}
	if ipAddr == "" {
		ipAddr = r.RemoteAddr
	}

	log := models.AuditLog{
		ID:         utils.GetUUID(),
		UserID:     userID,
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		Changes:    changes,
		IPAddress:  ipAddr,
		UserAgent:  r.Header.Get("User-Agent"),
		Status:     status,
		CreatedAt:  time.Now(),
	}

	if err := app.DB.InsertOne(ctx, auditCollection, log); err != nil {
		return // Failed to write audit log, continue without error
	}
}

// LogActionWithReason records an audit event with failure reason
func LogActionWithReason(
	ctx context.Context,
	app *infra.Deps,
	r *http.Request,
	userID string,
	action string,
	entityType string,
	entityID string,
	status string,
	reason string,
) {
	if app == nil || app.DB == nil {
		return
	}

	ipAddr := r.Header.Get("X-Forwarded-For")
	if ipAddr == "" {
		ipAddr = r.Header.Get("X-Real-IP")
	}
	if ipAddr == "" {
		ipAddr = r.RemoteAddr
	}

	audit := models.AuditLog{
		ID:         utils.GetUUID(),
		UserID:     userID,
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		IPAddress:  ipAddr,
		UserAgent:  r.Header.Get("User-Agent"),
		Status:     status,
		Reason:     reason,
		CreatedAt:  time.Now(),
	}

	if err := app.DB.InsertOne(ctx, auditCollection, audit); err != nil {
		return // Failed to write audit log, continue without error
	}
}
