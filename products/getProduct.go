package products

import (
	"context"
	"naevis/infra"
	"naevis/models"
)

func getProductEntity(ctx context.Context, id string, app *infra.Deps) models.Product {
	var product models.Product

	_ = app.DB.FindOne(ctx, productsCollection, map[string]any{
		"productid": id,
	}, &product)

	// If not found or error, zero-value product is returned (same behavior as before)
	return product
}
