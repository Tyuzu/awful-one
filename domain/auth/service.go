package auth

import (
	"context"
	"encoding/json"
	"time"

	"naevis/config/mqevent"
	"naevis/infra/mq"
	"naevis/models"
)

// AuthService contains core authentication business logic
// Handles user registration, login, and event publishing
type AuthService struct {
	repo     UserRepository
	eventBus mq.MQ
}

// NewAuthService creates a new authentication service
func NewAuthService(repo UserRepository, eventBus mq.MQ) *AuthService {
	return &AuthService{
		repo:     repo,
		eventBus: eventBus,
	}
}

// RegisterInput represents user registration request
type RegisterInput struct {
	Username string
	Email    string
	Password string
}

// RegisterUser performs user registration and publishes UserRegistered event
func (s *AuthService) RegisterUser(ctx context.Context, input RegisterInput) (*models.User, error) {
	// Create user (repository handles password hashing)
	user, err := s.repo.Create(ctx, input.Username, input.Email, input.Password)
	if err != nil {
		return nil, err
	}

	// Publish domain event asynchronously
	go s.publishUserRegisteredEvent(context.Background(), user)

	return user, nil
}

// publishUserRegisteredEvent publishes the UserRegistered domain event
func (s *AuthService) publishUserRegisteredEvent(ctx context.Context, user *models.User) {
	publishCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	payload := mqevent.UserRegisteredPayload{
		UserID:    user.UserID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		// Log error - could not serialize event
		return
	}

	_ = s.eventBus.Publish(publishCtx, mqevent.UserRegistered, data)
}

// LoginInput represents user login request
type LoginInput struct {
	Email    string
	Password string
	IP       string
}

// LoginUser authenticates user and publishes UserLoggedIn event
func (s *AuthService) LoginUser(ctx context.Context, input LoginInput) (*models.User, error) {
	// Authenticate user (repository handles password verification)
	user, err := s.repo.FindByEmailAndPassword(ctx, input.Email, input.Password)
	if err != nil {
		return nil, err
	}

	// Publish domain event asynchronously
	go s.publishUserLoggedInEvent(context.Background(), user, input.IP)

	return user, nil
}

// publishUserLoggedInEvent publishes the UserLoggedIn domain event
func (s *AuthService) publishUserLoggedInEvent(ctx context.Context, user *models.User, ip string) {
	publishCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	payload := mqevent.UserLoggedInPayload{
		UserID:     user.UserID,
		Username:   user.Username,
		OccurredAt: time.Now(),
		IP:         ip,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	_ = s.eventBus.Publish(publishCtx, mqevent.UserLoggedIn, data)
}

// LogoutInput represents user logout request
type LogoutInput struct {
	UserID string
	IP     string
}

// LogoutUser logs out user and publishes UserLoggedOut event
func (s *AuthService) LogoutUser(ctx context.Context, input LogoutInput) error {
	// Perform any necessary logout operations (invalidate tokens, etc.)
	// This could be implemented in repository

	// Publish domain event asynchronously
	go s.publishUserLoggedOutEvent(context.Background(), input.UserID)

	return nil
}

// publishUserLoggedOutEvent publishes the UserLoggedOut domain event
func (s *AuthService) publishUserLoggedOutEvent(ctx context.Context, userID string) {
	publishCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	payload := mqevent.UserLoggedOutPayload{
		UserID:     userID,
		OccurredAt: time.Now(),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	_ = s.eventBus.Publish(publishCtx, mqevent.UserLoggedOut, data)
}
