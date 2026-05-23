package farms

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"naevis/auditlog"
	"naevis/infra"
	"naevis/models"
	"naevis/utils"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

/* ---------------------------------------------------- */
/* DTOs                                                 */
/* ---------------------------------------------------- */

type OrderDisplay struct {
	ID           string `json:"id"`
	Buyer        string `json:"buyer"`
	Contact      string `json:"contact"`
	Crop         string `json:"crop"`
	Qty          int    `json:"qty"`
	Unit         string `json:"unit"`
	OrderDate    string `json:"orderDate"`
	DeliveryDate string `json:"deliveryDate"`
	Address      string `json:"address"`
	Payment      string `json:"payment"`
	Status       string `json:"status"`
}

/* ---------------------------------------------------- */
/* Buy crop                                             */
/* ---------------------------------------------------- */

func BuyCrop(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		farmID := ps.ByName("farmid")
		cropID := ps.ByName("cropid")

		// SECURITY: Use atomic FindOneAndUpdate to prevent race condition
		// This ensures only one concurrent request can successfully decrement inventory
		var updatedCrop bson.M
		err := app.DB.FindOneAndUpdate(
			ctx,
			cropsCollection,
			bson.M{
				"farmid":     farmID,
				"cropid":     cropID,
				"quantity":   bson.M{"$gt": 0}, // Atomic check
				"outOfStock": false,
			},
			bson.M{
				"$inc": bson.M{"quantity": -1},
				"$set": bson.M{"updatedAt": time.Now()},
			},
			&updatedCrop,
		)

		if err != nil {
			utils.RespondWithJSON(
				w,
				http.StatusBadRequest,
				utils.M{"success": false, "message": "Crop not available or already out of stock"},
			)
			return
		}

		// Check if this was the last crop (quantity is now 0)
		if quantity, ok := updatedCrop["quantity"].(int32); ok && quantity == 0 {
			app.DB.UpdateOne(
				ctx,
				cropsCollection,
				bson.M{"farmid": farmID, "cropid": cropID},
				bson.M{"$set": bson.M{"outOfStock": true}},
			)
		}

		utils.RespondWithJSON(w, http.StatusOK, utils.M{"success": true})
	}
}

/* ---------------------------------------------------- */
/* Order status updates                                 */
/* ---------------------------------------------------- */

func updateOrderStatus(
	w http.ResponseWriter,
	r *http.Request,
	orderID string,
	newStatus string,
	app *infra.Deps,
) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// SECURITY: Verify the user is the farm owner
	userID := utils.GetUserIDFromRequest(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Fetch the order to verify the requester is the farm owner
	var order models.FarmOrder
	if err := app.DB.FindOne(ctx, farmOrdersCollection, bson.M{"orderid": orderID}, &order); err != nil {
		utils.RespondWithJSON(
			w,
			http.StatusNotFound,
			utils.M{"success": false, "message": "Order not found"},
		)
		return
	}

	// Verify authorization: only the farm owner can change order status
	var farm models.Farm
	if err := app.DB.FindOne(ctx, farmsCollection, bson.M{"farmid": order.FarmID}, &farm); err != nil {
		utils.RespondWithJSON(
			w,
			http.StatusNotFound,
			utils.M{"success": false, "message": "Farm not found"},
		)
		return
	}

	if farm.CreatedBy != userID {
		http.Error(w, "Forbidden: Only farm owner can update order status", http.StatusForbidden)
		return
	}

	// Prevent invalid status transitions
	validTransitions := map[string][]string{
		"pending":   {"accepted", "rejected"},
		"accepted":  {"paid", "rejected"},
		"paid":      {"delivered"},
		"rejected":  {},
		"delivered": {},
	}

	allowed := false
	for _, validStatus := range validTransitions[string(order.Status)] {
		if validStatus == newStatus {
			allowed = true
			break
		}
	}

	if !allowed {
		utils.RespondWithJSON(
			w,
			http.StatusBadRequest,
			utils.M{"success": false, "message": "Invalid status transition from " + string(order.Status) + " to " + newStatus},
		)
		return
	}

	err := app.DB.UpdateOne(
		ctx,
		farmOrdersCollection,
		bson.M{"orderid": orderID},
		bson.M{"$set": bson.M{"status": newStatus, "updatedat": time.Now()}},
	)

	if err != nil {
		utils.RespondWithJSON(
			w,
			http.StatusBadRequest,
			utils.M{"success": false, "message": "Order not found or unchanged"},
		)
		return
	}

	// SECURITY: Log audit trail for farm order status changes
	auditAction := ""
	switch newStatus {
	case "accepted":
		auditAction = models.AuditActionOrderAccept
	case "rejected":
		auditAction = models.AuditActionOrderReject
	case "paid":
		auditAction = models.AuditActionOrderMarkPaid
	case "delivered":
		auditAction = models.AuditActionOrderMarkDeliver
	}

	if auditAction != "" {
		auditlog.LogAction(
			ctx,
			app,
			r,
			userID,
			auditAction,
			"farm_order",
			orderID,
			"success",
			map[string]interface{}{
				"oldStatus": order.Status,
				"newStatus": newStatus,
			},
		)
	}

	utils.RespondWithJSON(
		w,
		http.StatusOK,
		utils.M{"success": true, "status": newStatus},
	)
}

func AcceptOrder(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		updateOrderStatus(w, r, ps.ByName("id"), "accepted", app)
	}
}

func RejectOrder(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		updateOrderStatus(w, r, ps.ByName("id"), "rejected", app)
	}
}

func MarkOrderDelivered(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		updateOrderStatus(w, r, ps.ByName("id"), "delivered", app)
	}
}

func MarkOrderPaid(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		updateOrderStatus(w, r, ps.ByName("id"), "paid", app)
	}
}

/* ---------------------------------------------------- */
/* Bulk order status updates                            */
/* ---------------------------------------------------- */

type BulkOrdersRequest struct {
	OrderIDs []string `json:"orderIds"`
}

type BulkOrdersResponse struct {
	Success bool     `json:"success"`
	Message string   `json:"message"`
	Updated int      `json:"updated"`
	Failed  int      `json:"failed"`
	Errors  []string `json:"errors,omitempty"`
}

func BulkAcceptOrders(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		bulkUpdateOrders(w, r, "accepted", app)
	}
}

func BulkRejectOrders(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		bulkUpdateOrders(w, r, "rejected", app)
	}
}

func BulkMarkOrdersDelivered(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		bulkUpdateOrders(w, r, "delivered", app)
	}
}

func bulkUpdateOrders(w http.ResponseWriter, r *http.Request, newStatus string, app *infra.Deps) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	userID := utils.GetUserIDFromRequest(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req BulkOrdersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondWithJSON(w, http.StatusBadRequest, utils.M{
			"success": false,
			"message": "Invalid request body",
		})
		return
	}

	if len(req.OrderIDs) == 0 {
		utils.RespondWithJSON(w, http.StatusBadRequest, utils.M{
			"success": false,
			"message": "No order IDs provided",
		})
		return
	}

	// Fetch user's farms for authorization
	var farms []models.Farm
	if err := app.DB.FindMany(ctx, farmsCollection, bson.M{"createdBy": userID}, &farms); err != nil {
		utils.RespondWithJSON(w, http.StatusInternalServerError, utils.M{
			"success": false,
			"message": "Failed to fetch farms",
		})
		return
	}

	farmIDs := make([]string, len(farms))
	for i, f := range farms {
		farmIDs[i] = f.FarmID
	}

	response := BulkOrdersResponse{Success: true}
	var errors []string

	for _, orderID := range req.OrderIDs {
		// Fetch the order
		var order models.FarmOrder
		if err := app.DB.FindOne(ctx, farmOrdersCollection, bson.M{"orderid": orderID}, &order); err != nil {
			response.Failed++
			errors = append(errors, fmt.Sprintf("Order %s not found", orderID))
			continue
		}

		// Verify farm ownership
		authorized := false
		for _, farmID := range farmIDs {
			if order.FarmID == farmID {
				authorized = true
				break
			}
		}
		if !authorized {
			response.Failed++
			errors = append(errors, fmt.Sprintf("Order %s unauthorized", orderID))
			continue
		}

		// Validate status transition
		validTransitions := map[string][]string{
			"pending":   {"accepted", "rejected", "delivered"},
			"accepted":  {"paid", "rejected", "delivered"},
			"paid":      {"delivered"},
			"rejected":  {},
			"delivered": {},
		}

		allowed := false
		for _, validStatus := range validTransitions[string(order.Status)] {
			if validStatus == newStatus {
				allowed = true
				break
			}
		}

		if !allowed {
			response.Failed++
			errors = append(errors, fmt.Sprintf("Order %s: invalid transition from %s to %s", orderID, order.Status, newStatus))
			continue
		}

		// Update order status
		if err := app.DB.UpdateOne(
			ctx,
			farmOrdersCollection,
			bson.M{"orderid": orderID},
			bson.M{"$set": bson.M{"status": newStatus, "updatedat": time.Now()}},
		); err != nil {
			response.Failed++
			errors = append(errors, fmt.Sprintf("Order %s: update failed", orderID))
			continue
		}

		response.Updated++

		// Log audit action
		auditAction := ""
		switch newStatus {
		case "accepted":
			auditAction = models.AuditActionOrderAccept
		case "rejected":
			auditAction = models.AuditActionOrderReject
		case "delivered":
			auditAction = models.AuditActionOrderMarkDeliver
		}

		if auditAction != "" {
			auditlog.LogAction(
				ctx,
				app,
				r,
				userID,
				auditAction,
				"farm_order",
				orderID,
				"success",
				map[string]interface{}{
					"oldStatus": order.Status,
					"newStatus": newStatus,
				},
			)
		}
	}

	if len(errors) > 0 {
		response.Errors = errors
	}

	if response.Updated > 0 {
		response.Message = fmt.Sprintf("Successfully updated %d order(s)", response.Updated)
	} else {
		response.Success = false
		response.Message = "No orders were updated"
	}

	utils.RespondWithJSON(w, http.StatusOK, response)
}

/* ---------------------------------------------------- */
/* Download receipt                                     */
/* ---------------------------------------------------- */

func DownloadReceipt(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		orderID := ps.ByName("id")

		var order models.FarmOrder
		if err := app.DB.FindOne(ctx, farmOrdersCollection, bson.M{"orderid": orderID}, &order); err != nil {
			utils.RespondWithJSON(
				w,
				http.StatusNotFound,
				utils.M{"success": false, "message": "Order not found"},
			)
			return
		}

		utils.RespondWithJSON(
			w,
			http.StatusOK,
			utils.M{"success": true, "receipt": order},
		)
	}
}

/* ---------------------------------------------------- */
/* Incoming orders                                      */
/* ---------------------------------------------------- */

func GetIncomingOrders(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		var orders []models.FarmOrder
		if err := app.DB.FindMany(ctx, farmOrdersCollection, bson.M{}, &orders); err != nil {
			log.Println("GetIncomingOrders error:", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		incoming := make([]models.IncomingOrder, 0, len(orders))
		for _, o := range orders {
			user := getUserByID(ctx, o.UserID, app)
			crop := getCropByID(ctx, o.CropID, app)

			incoming = append(incoming, models.IncomingOrder{
				ID:           o.OrderID,
				Buyer:        user.Name,
				Contact:      user.Email,
				Crop:         crop.Name,
				Qty:          o.Quantity,
				Unit:         crop.Unit,
				OrderDate:    o.CreatedAt.Format("2006-01-02"),
				DeliveryDate: estimateDeliveryDate(o.CreatedAt),
				Address:      user.Address,
				Payment:      "pending",
				Status:       string(o.Status),
			})
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(
			map[string]interface{}{
				"success": true,
				"orders":  incoming,
			},
		)
	}
}

/* ---------------------------------------------------- */
/* Helpers                                              */
/* ---------------------------------------------------- */

func getUserByID(ctx context.Context, id string, app *infra.Deps) models.User {
	var user models.User
	_ = app.DB.FindOne(ctx, usersCollection, bson.M{"userid": id}, &user)
	return user
}

func getCropByID(ctx context.Context, id string, app *infra.Deps) models.Crop {
	var crop models.Crop
	_ = app.DB.FindOne(ctx, cropsCollection, bson.M{"cropid": id}, &crop)
	return crop
}

func estimateDeliveryDate(created time.Time) string {
	return created.Add(72 * time.Hour).Format("2006-01-02")
}
