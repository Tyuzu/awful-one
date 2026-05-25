package tickets

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"time"

	"naevis/infra"
	"naevis/models"
	"naevis/utils"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

func ListMyTickets(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		eventID := ps.ByName("eventid")
		requestingUserId := utils.GetUserIDFromRequest(r)

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		// ---- Fetch purchased tickets ----
		var tickets []models.PurchasedTicket
		if err := app.DB.FindMany(
			ctx,
			purchasedTicketsCollection,
			bson.M{
				"eventid": eventID,
				"userid":  requestingUserId,
			},
			&tickets,
		); err != nil {
			log.Printf("Error fetching tickets: %v", err)
			http.Error(w, "Failed to fetch tickets", http.StatusInternalServerError)
			return
		}

		if tickets == nil {
			tickets = []models.PurchasedTicket{}
		}

		// ---- Collect canceled unique codes ----
		canceledCodes := []string{}
		for _, t := range tickets {
			if t.Canceled {
				canceledCodes = append(canceledCodes, t.UniqueCode)
			}
		}

		// ---- Batch fetch refunds ----
		refundMap := map[string]string{} // uniqueCode -> status
		if len(canceledCodes) > 0 {
			var refunds []models.RefundRequest
			if err := app.DB.FindMany(
				ctx,
				refundsCollection,
				bson.M{
					"userid":     requestingUserId,
					"eventid":    eventID,
					"uniquecode": bson.M{"$in": canceledCodes},
				},
				&refunds,
			); err == nil {
				for _, r := range refunds {
					refundMap[r.UniqueCode] = r.Status
				}
			}
		}

		// ---- Build response ----
		response := make([]map[string]any, 0, len(tickets))

		for _, t := range tickets {
			status := "Active"
			refundStatus := ""

			if t.Canceled {
				status = "Cancelled"
				if val, ok := refundMap[t.UniqueCode]; ok {
					refundStatus = val
				} else {
					refundStatus = "not_requested"
				}
			} else if t.TransferredTo != "" {
				status = "Transferred"
			}

			response = append(response, map[string]any{
				"ticketID":      t.TicketID,
				"uniqueCode":    t.UniqueCode,
				"buyerName":     t.BuyerName,
				"purchaseDate":  t.PurchaseDate,
				"canceled":      t.Canceled,
				"transferred":   t.TransferredTo != "",
				"status":        status,
				"refundStatus":  refundStatus,
				"transferredTo": t.TransferredTo,
			})
		}

		// ---- Stable status ordering ----
		sort.Slice(response, func(i, j int) bool {
			order := map[string]int{
				"Active":      0,
				"Transferred": 1,
				"Cancelled":   2,
			}
			return order[response[i]["status"].(string)] <
				order[response[j]["status"].(string)]
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
