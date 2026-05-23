package userdata

import (
	"context"
	"log"
	"naevis/infra"
	"naevis/models"
	"time"
)

var ValidEntityTypes = map[string]bool{
	"userhome":   true,
	"place":      true,
	"event":      true,
	"feedpost":   true,
	"media":      true,
	"ticket":     true,
	"merch":      true,
	"review":     true,
	"comment":    true,
	"like":       true,
	"favourite":  true,
	"booking":    true,
	"blogpost":   true,
	"collection": true,
}

func IsValidEntityType(entityType string) bool {
	return ValidEntityTypes[entityType]
}

func SetUserData(dataType, dataId, userId, itemType, itemId string, app *infra.Deps) {
	AddUserData(dataType, dataId, userId, itemType, itemId, app)
}

func DelUserData(dataType, dataId, userId string, app *infra.Deps) {
	RemUserData(dataType, dataId, userId, app)
}

func AddUserData(entityType, entityId, userId, itemType, itemId string, app *infra.Deps) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	content := models.UserData{
		EntityID:   entityId,
		EntityType: entityType,
		ItemID:     itemId,
		ItemType:   itemType,
		UserID:     userId,
		CreatedAt:  time.Now().Format(time.RFC3339),
	}

	if err := app.DB.InsertOne(ctx, userdataCollection, content); err != nil {
		log.Printf("Error inserting user data: %v", err)
	}
}

func RemUserData(entityType, entityId, userId string, app *infra.Deps) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := map[string]any{
		"entity_id":   entityId,
		"entity_type": entityType,
		"userid":      userId,
	}

	if err := app.DB.DeleteMany(ctx, userdataCollection, filter); err != nil {
		log.Printf("Error deleting user data: %v", err)
	}
}

func AddUserDataBatch(docs []models.UserData, app *infra.Deps) {
	if len(docs) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var toInsert []any
	for _, doc := range docs {
		toInsert = append(toInsert, doc)
	}

	if err := app.DB.InsertMany(ctx, userdataCollection, toInsert); err != nil {
		log.Printf("Error inserting batch user data: %v", err)
	}
}
