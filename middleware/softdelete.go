package middleware

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

// SoftDeleteFields adds soft delete fields to records
type SoftDeleteFields struct {
	DeletedAt *time.Time `bson:"deletedAt,omitempty" json:"deletedAt,omitempty"`
	DeletedBy string     `bson:"deletedBy,omitempty" json:"deletedBy,omitempty"`
	Reason    string     `bson:"deleteReason,omitempty" json:"deleteReason,omitempty"`
}

// MarkDeleted creates update filter for soft deletion
func MarkDeleted(userID string, reason string) bson.M {
	now := time.Now()
	return bson.M{
		"$set": bson.M{
			"deletedAt":    now,
			"deletedBy":    userID,
			"deleteReason": reason,
		},
	}
}

// ExcludeDeleted creates filter to exclude soft-deleted records
func ExcludeDeleted() bson.M {
	return bson.M{
		"deletedAt": nil,
	}
}

// ExcludeDeleted2 creates filter to exclude soft-deleted records (alt format)
func ExcludeDeletedAlt() bson.M {
	return bson.M{
		"deletedAt": bson.M{"$exists": false},
	}
}

// PermanentDelete creates hard delete (use only for GDPR/compliance)
func PermanentDelete() bson.M {
	return bson.M{}
}
