package farms

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"naevis/infra"
	"naevis/models"
	"naevis/utils"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

/* ---------------------------------------------------- */
/* Orders placed BY the current user (buyer)            */
/* ---------------------------------------------------- */

func GetMyFarmOrders(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID := utils.GetUserIDFromRequest(r)

		// Parse pagination parameters
		skip := 0
		limit := 10
		if s := r.URL.Query().Get("skip"); s != "" {
			if parsed, err := strconv.Atoi(s); err == nil && parsed >= 0 {
				skip = parsed
			}
		}
		if l := r.URL.Query().Get("limit"); l != "" {
			if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
				limit = parsed
			}
		}

		var orders []models.FarmOrder
		if err := app.DB.FindMany(
			ctx,
			farmOrdersCollection,
			bson.M{"userid": userID},
			&orders,
		); err != nil {
			utils.RespondWithJSON(w, http.StatusInternalServerError, utils.M{
				"success": false,
				"message": "Failed to fetch orders",
			})
			return
		}

		// Apply pagination
		total := len(orders)
		start := skip
		end := skip + limit
		if start > total {
			start = total
		}
		if end > total {
			end = total
		}

		paginatedOrders := orders[start:end]

		utils.RespondWithJSON(w, http.StatusOK, utils.M{
			"success": true,
			"orders":  paginatedOrders,
			"total":   total,
			"skip":    skip,
			"limit":   limit,
		})
	}
}

/* ---------------------------------------------------- */
/* Orders coming INTO farms owned by the farmer         */
/* ---------------------------------------------------- */

func GetIncomingFarmOrders(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		userID := utils.GetUserIDFromRequest(r)

		// 1. Fetch farms owned by this user
		var farms []models.Farm
		if err := app.DB.FindMany(
			ctx,
			farmsCollection,
			bson.M{"createdBy": userID},
			&farms,
		); err != nil {
			utils.RespondWithJSON(w, http.StatusInternalServerError, utils.M{
				"success": false,
				"message": "Failed to fetch farms",
			})
			return
		}

		farmIDs := make([]string, 0, len(farms))
		for _, f := range farms {
			farmIDs = append(farmIDs, f.FarmID)
		}

		if len(farmIDs) == 0 {
			utils.RespondWithJSON(w, http.StatusOK, utils.M{
				"success": true,
				"orders":  []OrderDisplay{},
			})
			return
		}

		// 2. Build filter query from URL params
		filter := bson.M{"farmid": bson.M{"$in": farmIDs}}

		// Filter by status
		if status := r.URL.Query().Get("status"); status != "" {
			filter["status"] = status
		}

		// Filter by date range
		if dateFrom := r.URL.Query().Get("dateFrom"); dateFrom != "" {
			if t, err := time.Parse("2006-01-02", dateFrom); err == nil {
				filter["createdat"] = bson.M{"$gte": t}
			}
		}

		if dateTo := r.URL.Query().Get("dateTo"); dateTo != "" {
			if t, err := time.Parse("2006-01-02", dateTo); err == nil {
				// Add one day to include all orders on that date
				t = t.Add(24 * time.Hour)
				if dateFrom := r.URL.Query().Get("dateFrom"); dateFrom != "" {
					// If there's already a $gte, we need to use $lte
					if existingDateFilter, ok := filter["createdat"].(bson.M); ok {
						existingDateFilter["$lte"] = t
						filter["createdat"] = existingDateFilter
					}
				} else {
					filter["createdat"] = bson.M{"$lte": t}
				}
			}
		}

		// 2. Fetch orders for those farms
		var orders []models.FarmOrder
		if err := app.DB.FindMany(
			ctx,
			farmOrdersCollection,
			filter,
			&orders,
		); err != nil {
			utils.RespondWithJSON(w, http.StatusInternalServerError, utils.M{
				"success": false,
				"message": "Failed to fetch orders",
			})
			return
		}

		// 3. Build frontend-friendly response and apply client-side filters
		displayOrders := make([]OrderDisplay, 0, len(orders))
		cropFilter := r.URL.Query().Get("crop")
		paymentFilter := r.URL.Query().Get("payment")

		for _, o := range orders {
			user := fetchUserByID(ctx, o.UserID, app)
			crop := fetchCropByID(ctx, o.CropID, app)

			// Client-side filtering for crop (since we filter by crop name)
			if cropFilter != "" && crop.Name != cropFilter {
				continue
			}

			// Client-side filtering for payment status
			// Note: payment status is always "pending" in current implementation
			// This is extensible for future payment tracking
			paymentStatus := "pending"
			if paymentFilter != "" && paymentStatus != paymentFilter {
				continue
			}

			displayOrders = append(displayOrders, OrderDisplay{
				ID:           o.OrderID,
				Buyer:        user.Name,
				Contact:      user.Email,
				Crop:         crop.Name,
				Qty:          o.Quantity,
				Unit:         crop.Unit,
				OrderDate:    o.CreatedAt.Format("2006-01-02"),
				DeliveryDate: estimateDeliveryDate(o.CreatedAt),
				Address:      user.Address,
				Payment:      paymentStatus,
				Status:       string(o.Status),
			})
		}

		utils.RespondWithJSON(w, http.StatusOK, utils.M{
			"success": true,
			"orders":  displayOrders,
		})
	}
}

func fetchUserByID(ctx context.Context, id string, app *infra.Deps) models.User {
	var user models.User

	if id == "" {
		return user
	}

	err := app.DB.FindOne(
		ctx,
		usersCollection,
		bson.M{"userid": id},
		&user,
	)
	if err != nil {
		return models.User{}
	}

	return user
}

func fetchCropByID(ctx context.Context, id string, app *infra.Deps) models.Crop {
	var crop models.Crop

	if id == "" {
		return crop
	}

	err := app.DB.FindOne(
		ctx,
		cropsCollection,
		bson.M{"cropid": id},
		&crop,
	)
	if err != nil {
		return models.Crop{}
	}

	return crop
}
