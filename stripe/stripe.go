package stripe

import (
	"errors"
	"fmt"
	"os"
)

type TicketSession struct {
	URL      string
	TicketID string
	EventID  string
	Quantity int
}

func CreateTicketSession(ticketId string, eventId string, quantity int) (TicketSession, error) {
	// NOTE: PRODUCTION REQUIRED: Integrate with actual Stripe API
	// Current implementation returns a client-side checkout URL.
	// In production, this should:
	// 1. Call stripe.checkout.sessions.create() API
	// 2. Return the official Stripe checkout URL
	// 3. Store session ID in database for webhook verification
	// Reference: https://stripe.com/docs/payments/checkout/how-to
	if ticketId == "" || eventId == "" || quantity <= 0 {
		return TicketSession{}, errors.New("invalid ticket parameters")
	}

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:5173"
	}

	s := TicketSession{
		URL:      fmt.Sprintf("%s/checkout?type=ticket&ticketId=%s&eventId=%s&quantity=%d", frontendURL, ticketId, eventId, quantity),
		TicketID: ticketId,
		EventID:  eventId,
		Quantity: quantity,
	}
	return s, nil
}

type MerchSession struct {
	URL     string
	MerchID string
	EventID string
	Stock   int
}

func CreateMerchSession(merchId string, eventId string, stock int) (MerchSession, error) {
	// NOTE: PRODUCTION REQUIRED: Integrate with actual Stripe API
	// Current implementation returns a client-side checkout URL.
	// In production, this should:
	// 1. Call stripe.checkout.sessions.create() API
	// 2. Return the official Stripe checkout URL
	// 3. Store session ID in database for webhook verification
	// Reference: https://stripe.com/docs/payments/checkout/how-to
	if merchId == "" || eventId == "" || stock <= 0 {
		return MerchSession{}, errors.New("invalid merch parameters")
	}

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:5173"
	}

	s := MerchSession{
		URL:     fmt.Sprintf("%s/checkout?type=merch&merchId=%s&eventId=%s&quantity=%d", frontendURL, merchId, eventId, stock),
		MerchID: merchId,
		EventID: eventId,
		Stock:   stock,
	}
	return s, nil
}

type MenuSession struct {
	URL     string
	MenuID  string
	PlaceID string
	Stock   int
}

func CreateMenuSession(menuId string, placeId string, stock int) (MenuSession, error) {
	// NOTE: PRODUCTION REQUIRED: Integrate with actual Stripe API
	// Current implementation returns a client-side checkout URL.
	// In production, this should:
	// 1. Call stripe.checkout.sessions.create() API
	// 2. Return the official Stripe checkout URL
	// 3. Store session ID in database for webhook verification
	// Reference: https://stripe.com/docs/payments/checkout/how-to
	if menuId == "" || placeId == "" || stock <= 0 {
		return MenuSession{}, errors.New("invalid menu parameters")
	}

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:5173"
	}

	s := MenuSession{
		URL:     fmt.Sprintf("%s/checkout?type=menu&menuId=%s&placeId=%s&quantity=%d", frontendURL, menuId, placeId, stock),
		MenuID:  menuId,
		PlaceID: placeId,
		Stock:   stock,
	}
	return s, nil
}
