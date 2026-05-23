package auth

import (
	"context"

	"naevis/models"
)

// UserRepository defines data access interface for user aggregate
// Implementations should handle user persistence and retrieval
type UserRepository interface {
	Create(ctx context.Context, username, email, password string) (*models.User, error)
	FindByID(ctx context.Context, userID string) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByEmailAndPassword(ctx context.Context, email, password string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, userID string) error
	MarkAsVerified(ctx context.Context, userID string) error
	UpdateLastLogin(ctx context.Context, userID string) error
}
