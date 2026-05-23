package order

import (
	"context"
)

// OrderAggregate represents the order aggregate root
type OrderAggregate struct {
	ID        string
	UserID    string
	SellerID  string
	Amount    int64
	Status    string
	Items     []OrderItem
	Metadata  map[string]string
	CreatedAt int64
	UpdatedAt int64
}

// OrderRepository defines data access interface for order aggregate
// Implementations should handle order persistence and retrieval
type OrderRepository interface {
	Create(ctx context.Context, userID string, amount int64, items []OrderItem, metadata map[string]string) (string, error)
	GetByID(ctx context.Context, orderID string) (*OrderAggregate, error)
	FindByUserID(ctx context.Context, userID string, limit, skip int) ([]*OrderAggregate, error)
	FindBySellerID(ctx context.Context, sellerID string, limit, skip int) ([]*OrderAggregate, error)
	UpdateStatus(ctx context.Context, orderID string, status string) error
	UpdateSellerStatus(ctx context.Context, orderID, sellerID, status string) error
	Delete(ctx context.Context, orderID string) error
}
