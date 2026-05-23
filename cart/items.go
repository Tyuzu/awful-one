package cart

import (
	"context"
	"errors"

	"naevis/infra"

	"go.mongodb.org/mongo-driver/bson"
)

/* ───────────────────────────────────── Item Details Lookup ───────────────────────────────────── */

type ItemDetails struct {
	Name       string
	Type       string
	Category   string
	Price      float64
	Unit       string
	EntityID   string
	EntityName string
	EntityType string
	Available  int
}

// lookupItemDetails queries all item collections to find and validate the item
// Returns backend-verified item details with current price and availability
func lookupItemDetails(ctx context.Context, itemID string, app *infra.Deps) (*ItemDetails, error) {
	// Try products collection (merchandise, books, etc.)
	product, err := lookupProduct(ctx, itemID, app)
	if err == nil && product != nil {
		return product, nil
	}

	// Try crops collection (linked to farm)
	crop, err := lookupCrop(ctx, itemID, app)
	if err == nil && crop != nil {
		return crop, nil
	}

	// Try menu items (linked to place/restaurant)
	menu, err := lookupMenu(ctx, itemID, app)
	if err == nil && menu != nil {
		return menu, nil
	}

	// Try merchandise collection
	merch, err := lookupMerchandise(ctx, itemID, app)
	if err == nil && merch != nil {
		return merch, nil
	}

	return nil, errors.New("item not found in any collection")
}

// lookupProduct queries the products collection
func lookupProduct(ctx context.Context, productID string, app *infra.Deps) (*ItemDetails, error) {
	var product struct {
		Name     string  `bson:"name"`
		Type     string  `bson:"type"`
		Price    float64 `bson:"price"`
		Unit     string  `bson:"unit"`
		Quantity int     `bson:"quantity"`
	}

	err := app.DB.FindOne(ctx, "products", bson.M{"productid": productID}, &product)
	if err != nil {
		return nil, err
	}

	if product.Quantity <= 0 {
		return nil, errors.New("product out of stock")
	}

	return &ItemDetails{
		Name:       product.Name,
		Type:       product.Type,
		Category:   "products",
		Price:      product.Price,
		Unit:       product.Unit,
		EntityID:   "",
		EntityName: "",
		EntityType: "",
		Available:  product.Quantity,
	}, nil
}

// lookupCrop queries the crops collection (from farms)
func lookupCrop(ctx context.Context, cropID string, app *infra.Deps) (*ItemDetails, error) {
	var crop struct {
		CropID       string  `bson:"cropid"`
		Name         string  `bson:"name"`
		Breed        string  `bson:"breed"`
		Price        float64 `bson:"price"`
		AvailableQty int     `bson:"quantity"`
		FarmID       string  `bson:"farmid"`
		FarmName     string  `bson:"farmName"`
	}

	err := app.DB.FindOne(ctx, "crops", bson.M{"cropid": cropID}, &crop)
	if err != nil {
		return nil, err
	}

	if crop.AvailableQty <= 0 {
		return nil, errors.New("crop out of stock")
	}

	// CRITICAL FIX: If farm name not populated, fetch from farm collection
	farmName := crop.FarmName
	if farmName == "" && crop.FarmID != "" {
		var farm struct {
			Name string `bson:"name"`
		}
		if err := app.DB.FindOne(ctx, "farms", bson.M{"farmid": crop.FarmID}, &farm); err == nil {
			farmName = farm.Name
		}
	}

	return &ItemDetails{
		Name:       crop.Name,
		Type:       crop.Breed,
		Category:   "crops",
		Price:      crop.Price,
		Unit:       "kg",
		EntityID:   crop.FarmID,
		EntityName: farmName,
		EntityType: "farm",
		Available:  crop.AvailableQty,
	}, nil
}

// lookupMenu queries the menu collection (from places)
func lookupMenu(ctx context.Context, menuID string, app *infra.Deps) (*ItemDetails, error) {
	var menu struct {
		MenuID  string  `bson:"menuid"`
		Name    string  `bson:"name"`
		Price   float64 `bson:"price"`
		Stock   int     `bson:"stock"`
		PlaceID string  `bson:"placeid"`
		Place   string  `bson:"place"`
	}

	err := app.DB.FindOne(ctx, "menu", bson.M{"menuid": menuID}, &menu)
	if err != nil {
		return nil, err
	}

	if menu.Stock <= 0 {
		return nil, errors.New("menu item out of stock")
	}

	return &ItemDetails{
		Name:       menu.Name,
		Type:       "menu",
		Category:   "menu",
		Price:      menu.Price,
		Unit:       "unit",
		EntityID:   menu.PlaceID,
		EntityName: menu.Place,
		EntityType: "place",
		Available:  menu.Stock,
	}, nil
}

// lookupMerchandise queries the merchandise collection (from events)
func lookupMerchandise(ctx context.Context, merchID string, app *infra.Deps) (*ItemDetails, error) {
	var merch struct {
		MerchID string  `bson:"merchid"`
		Name    string  `bson:"name"`
		Price   float64 `bson:"price"`
		Stock   int     `bson:"stock"`
	}

	err := app.DB.FindOne(ctx, "merchandise", bson.M{"merchid": merchID}, &merch)
	if err != nil {
		return nil, err
	}

	if merch.Stock <= 0 {
		return nil, errors.New("merchandise out of stock")
	}

	return &ItemDetails{
		Name:       merch.Name,
		Type:       "merchandise",
		Category:   "merchandise",
		Price:      merch.Price,
		Unit:       "unit",
		EntityID:   "",
		EntityName: "",
		EntityType: "",
		Available:  merch.Stock,
	}, nil
}
