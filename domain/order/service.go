package order

import (
	"context"
	"encoding/json"
	"time"

	"naevis/config/mqevent"
	"naevis/infra/mq"
)

// OrderService contains core order business logic
// Handles order creation, payment, and event publishing
type OrderService struct {
	repo     OrderRepository
	eventBus mq.MQ
}

// NewOrderService creates a new order service
func NewOrderService(repo OrderRepository, eventBus mq.MQ) *OrderService {
	return &OrderService{
		repo:     repo,
		eventBus: eventBus,
	}
}

// CreateOrderInput represents order creation request
type CreateOrderInput struct {
	UserID   string
	Amount   int64
	Items    []OrderItem
	Metadata map[string]string
}

// OrderItem represents an item in the order
type OrderItem struct {
	ProductID string
	Quantity  int
	Price     int64
}

// CreateOrder creates new order and publishes OrderCreated event
func (s *OrderService) CreateOrder(ctx context.Context, input CreateOrderInput) (string, error) {
	// Create order in repository
	orderID, err := s.repo.Create(ctx, input.UserID, input.Amount, input.Items, input.Metadata)
	if err != nil {
		return "", err
	}

	// Publish domain event asynchronously
	go s.publishOrderCreatedEvent(context.Background(), orderID, input.UserID, input.Amount)

	return orderID, nil
}

// publishOrderCreatedEvent publishes the OrderCreated domain event
func (s *OrderService) publishOrderCreatedEvent(ctx context.Context, orderID, userID string, amount int64) {
	publishCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	payload := mqevent.OrderCreatedPayload{
		OrderID:    orderID,
		UserID:     userID,
		Amount:     amount,
		OccurredAt: time.Now(),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	_ = s.eventBus.Publish(publishCtx, mqevent.OrderCreated, data)
}

// MarkOrderAsPaid marks order as paid and publishes OrderPaid event
func (s *OrderService) MarkOrderAsPaid(ctx context.Context, orderID string) error {
	// Update order status in repository
	err := s.repo.UpdateStatus(ctx, orderID, "paid")
	if err != nil {
		return err
	}

	// Fetch order details to publish event
	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return err
	}

	// Publish domain event asynchronously
	go s.publishOrderPaidEvent(context.Background(), orderID, order.UserID)

	return nil
}

// publishOrderPaidEvent publishes the OrderPaid domain event
func (s *OrderService) publishOrderPaidEvent(ctx context.Context, orderID, userID string) {
	publishCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	payload := mqevent.OrderPaidPayload{
		OrderID: orderID,
		UserID:  userID,
		PaidAt:  time.Now(),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	_ = s.eventBus.Publish(publishCtx, mqevent.OrderPaid, data)
}

// AcceptOrderAsSeller marks order as accepted by seller and publishes event
func (s *OrderService) AcceptOrderAsSeller(ctx context.Context, orderID, sellerID string) error {
	// Update order in repository
	err := s.repo.UpdateSellerStatus(ctx, orderID, sellerID, "accepted")
	if err != nil {
		return err
	}

	// Publish domain event asynchronously
	go s.publishOrderSellerAcceptedEvent(context.Background(), orderID, sellerID)

	return nil
}

// publishOrderSellerAcceptedEvent publishes the OrderSellerAccepted domain event
func (s *OrderService) publishOrderSellerAcceptedEvent(ctx context.Context, orderID, sellerID string) {
	publishCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	payload := mqevent.OrderSellerAcceptedPayload{
		OrderID:    orderID,
		SellerID:   sellerID,
		OccurredAt: time.Now(),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	_ = s.eventBus.Publish(publishCtx, mqevent.OrderSellerAccepted, data)
}
