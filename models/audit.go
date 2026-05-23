package models

import "time"

// AuditLog tracks sensitive operations for compliance and security
type AuditLog struct {
	ID         string                 `bson:"_id,omitempty" json:"id"`
	UserID     string                 `bson:"userId" json:"userId"`
	Action     string                 `bson:"action" json:"action"` // e.g., "TICKET_PURCHASE", "MERCH_DELETE", "ORDER_MARK_PAID"
	EntityType string                 `bson:"entityType" json:"entityType"`
	EntityID   string                 `bson:"entityId" json:"entityId"`
	Changes    map[string]interface{} `bson:"changes,omitempty" json:"changes,omitempty"` // What changed
	IPAddress  string                 `bson:"ipAddress" json:"ipAddress"`
	UserAgent  string                 `bson:"userAgent" json:"userAgent"`
	Status     string                 `bson:"status" json:"status"`                     // "success", "failed", "attempted"
	Reason     string                 `bson:"reason,omitempty" json:"reason,omitempty"` // Why it failed
	CreatedAt  time.Time              `bson:"createdAt" json:"createdAt"`
}

// AuditAction constants for common operations
const (
	AuditActionTicketPurchase   = "TICKET_PURCHASE"
	AuditActionTicketCancel     = "TICKET_CANCEL"
	AuditActionMerchCreate      = "MERCH_CREATE"
	AuditActionMerchUpdate      = "MERCH_UPDATE"
	AuditActionMerchDelete      = "MERCH_DELETE"
	AuditActionMerchPurchase    = "MERCH_PURCHASE"
	AuditActionOrderAccept      = "ORDER_ACCEPT"
	AuditActionOrderReject      = "ORDER_REJECT"
	AuditActionOrderMarkPaid    = "ORDER_MARK_PAID"
	AuditActionOrderMarkDeliver = "ORDER_MARK_DELIVERED"
	AuditActionPaymentProcess   = "PAYMENT_PROCESS"
	AuditActionPayment          = "PAYMENT"
	AuditActionTopUp            = "TOPUP"
	AuditActionFarmCreate       = "FARM_CREATE"
	AuditActionFarmDelete       = "FARM_DELETE"
)
