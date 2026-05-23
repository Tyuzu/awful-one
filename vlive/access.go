package vlive

import (
	"context"
	"naevis/infra"

	"go.mongodb.org/mongo-driver/bson"
)

// func CheckEntityAccess(
// 	ctx context.Context,
// 	app *infra.Deps,
// 	userID string,
// 	entityType string,
// 	entityID string,
// ) bool {

// 	// Entity owner
// 	count, err := app.DB.CountDocuments(
// 		ctx,
// 		liveViewersCollection,
// 		bson.M{
// 			"_id":      entityID,
// 			"owner_id": userID,
// 		},
// 	)
// 	if err == nil && count > 0 {
// 		return true
// 	}

// 	// Explicit membership
// 	count, err = app.DB.CountDocuments(
// 		ctx,
// 		entityMembersCollection,
// 		bson.M{
// 			"entity_type": entityType,
// 			"entity_id":   entityID,
// 			"user_id":     userID,
// 			"role": bson.M{
// 				"$in": []string{"admin", "member", "viewer"},
// 			},
// 		},
// 	)

// 	return err == nil && count > 0
// }

func CheckEntityAccess(
	ctx context.Context,
	app *infra.Deps,
	userID string,
	entityType string,
	entityID string,
) bool {

	if userID == "" || entityType == "" || entityID == "" {
		return false
	}

	// Entity owner
	count, err := app.DB.CountDocuments(
		ctx,
		liveViewersCollection,
		bson.M{
			"_id":      entityID,
			"owner_id": userID,
		},
	)
	if err == nil && count > 0 {
		return true
	}

	// Explicit membership
	count, err = app.DB.CountDocuments(
		ctx,
		entityMembersCollection,
		bson.M{
			"entity_type": entityType,
			"entity_id":   entityID,
			"user_id":     userID,
			"role": bson.M{
				"$in": []string{"admin", "member", "viewer"},
			},
		},
	)

	return err == nil && count > 0
}
