package middleware

import "time"

// SoftDeleteFields adds soft delete fields to records
type SoftDeleteFields struct {
	DeletedAt *time.Time `bson:"deletedAt,omitempty" json:"deletedAt,omitempty"`
	DeletedBy string     `bson:"deletedBy,omitempty" json:"deletedBy,omitempty"`
	Reason    string     `bson:"deleteReason,omitempty" json:"deleteReason,omitempty"`
}

// MarkDeleted creates update filter for soft deletion
func MarkDeleted(userID string, reason string) map[string]any {
	now := time.Now()

	return map[string]any{
		"$set": map[string]any{
			"deletedAt":    now,
			"deletedBy":    userID,
			"deleteReason": reason,
		},
	}
}

// ExcludeDeleted creates filter to exclude soft-deleted records
func ExcludeDeleted() map[string]any {
	return map[string]any{
		"deletedAt": nil,
	}
}

// ExcludeDeleted2 creates filter to exclude soft-deleted records (alt format)
func ExcludeDeletedAlt() map[string]any {
	return map[string]any{
		"deletedAt": map[string]any{
			"$exists": false,
		},
	}
}

// PermanentDelete creates hard delete (use only for GDPR/compliance)
func PermanentDelete() map[string]any {
	return map[string]any{}
}
