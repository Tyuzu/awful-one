package tickets

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"naevis/globals"
	"naevis/infra"
	"naevis/models"
	"naevis/stripe"
	"naevis/userdata"
	"naevis/utils"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

// ------------------------------------------------------------------
// Realtime Event Channels with Cleanup
// ------------------------------------------------------------------

var eventUpdateChannels = struct {
	sync.RWMutex
	channels    map[string]chan map[string]any
	subscribers map[string]int // Track number of active subscribers per channel
}{
	channels:    make(map[string]chan map[string]any),
	subscribers: make(map[string]int),
}

func GetUpdatesChannel(eventId string) chan map[string]any {
	eventUpdateChannels.Lock()
	defer eventUpdateChannels.Unlock()

	if ch, ok := eventUpdateChannels.channels[eventId]; ok {
		eventUpdateChannels.subscribers[eventId]++
		return ch
	}

	ch := make(chan map[string]any, 10)
	eventUpdateChannels.channels[eventId] = ch
	eventUpdateChannels.subscribers[eventId] = 1
	return ch
}

// CloseUpdatesChannel marks a subscriber as finished. Channel is cleaned up when all subscribers are gone.
func CloseUpdatesChannel(eventId string) {
	eventUpdateChannels.Lock()
	defer eventUpdateChannels.Unlock()

	if count, ok := eventUpdateChannels.subscribers[eventId]; ok {
		count--
		if count <= 0 {
			if ch, ok := eventUpdateChannels.channels[eventId]; ok {
				close(ch)
			}
			delete(eventUpdateChannels.channels, eventId)
			delete(eventUpdateChannels.subscribers, eventId)
		} else {
			eventUpdateChannels.subscribers[eventId] = count
		}
	}
}

// ------------------------------------------------------------------
// Stripe Session
// ------------------------------------------------------------------

func CreateTicketPaymentSession(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ticketId := ps.ByName("ticketid")
		eventId := ps.ByName("eventid")

		var body struct {
			Quantity int `json:"quantity"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Quantity < 1 {
			http.Error(w, "Invalid quantity", http.StatusBadRequest)
			return
		}

		session, err := stripe.CreateTicketSession(ticketId, eventId, body.Quantity)
		if err != nil {
			http.Error(w, "Failed to create payment session", http.StatusInternalServerError)
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"data": map[string]any{
				"paymentUrl": session.URL,
				"eventId":    session.EventID,
				"ticketId":   session.TicketID,
				"stock":      session.Quantity,
			},
		})
	}
}

// ------------------------------------------------------------------
// SSE Updates
// ------------------------------------------------------------------

func EventUpdates(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		eventId := ps.ByName("eventId")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		ch := GetUpdatesChannel(eventId)
		defer CloseUpdatesChannel(eventId)

		for {
			select {
			case update := <-ch:
				data, _ := json.Marshal(update)
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			case <-r.Context().Done():
				return
			}
		}
	}
}

func BroadcastTicketUpdate(eventId, ticketId string, remaining int) {
	ch := GetUpdatesChannel(eventId)
	select {
	case ch <- map[string]any{
		"type":             "ticket_update",
		"ticketId":         ticketId,
		"remainingTickets": remaining,
	}:
	default:
		log.Printf("event %s update channel full", eventId)
	}
}

// ------------------------------------------------------------------
// Purchase Flow
// ------------------------------------------------------------------

type TicketPurchaseRequest struct {
	TicketID string `json:"ticketId"`
	EventID  string `json:"eventId"`
	Quantity int    `json:"quantity"`
}

func ConfirmTicketPurchase(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		var req TicketPurchaseRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		// Extract eventid and ticketid from URL parameters
		req.EventID = ps.ByName("eventid")
		req.TicketID = ps.ByName("ticketid")

		buyTicket(w, r, req, app)
	}
}

func PurchaseTicket(eventID, ticketID, userID string, qty int, app *infra.Deps) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var ticket models.Ticket
	if err := app.DB.FindOne(ctx, ticketsCollection, bson.M{
		"eventid":  eventID,
		"ticketid": ticketID,
	}, &ticket); err != nil {
		return nil, fmt.Errorf("ticket not found")
	}

	if ticket.Quantity < qty {
		return nil, fmt.Errorf("not enough tickets")
	}

	if err := app.DB.UpdateOne(ctx, ticketsCollection,
		bson.M{"eventid": eventID, "ticketid": ticketID},
		bson.M{"$inc": bson.M{"quantity": -qty, "sold": qty}},
	); err != nil {
		return nil, err
	}

	codes := make([]string, qty)
	for i := 0; i < qty; i++ {
		codes[i] = utils.GetUUID()
	}
	return codes, nil
}

func StorePurchasedTickets(eventID, ticketID, userID string, codes []string, app *infra.Deps) error {
	now := time.Now()

	purchased := make([]any, 0, len(codes))
	userData := make([]models.UserData, 0, len(codes))

	for _, code := range codes {
		purchased = append(purchased, models.PurchasedTicket{
			EventID:      eventID,
			TicketID:     ticketID,
			UserID:       userID,
			UniqueCode:   code,
			PurchaseDate: now,
		})

		userData = append(userData, models.UserData{
			UserID:     userID,
			EntityID:   code,
			EntityType: "ticket",
			ItemID:     ticketID,
			ItemType:   "ticket",
			CreatedAt:  now.Format(time.RFC3339),
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := app.DB.InsertMany(ctx, purchasedTicketsCollection, purchased); err != nil {
		return err
	}

	userdata.AddUserDataBatch(userData, app)
	return nil
}

func buyTicket(w http.ResponseWriter, r *http.Request, req TicketPurchaseRequest, app *infra.Deps) {
	userID, ok := r.Context().Value(globals.UserIDKey).(string)
	if !ok || userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	codes, err := PurchaseTicket(req.EventID, req.TicketID, userID, req.Quantity, app)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := StorePurchasedTickets(req.EventID, req.TicketID, userID, codes, app); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]any{
		"success":     true,
		"message":     "Tickets purchased successfully",
		"uniqueCodes": codes,
	})
}
