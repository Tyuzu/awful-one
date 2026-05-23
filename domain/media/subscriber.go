package media

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"naevis/config"
	"naevis/config/mqevent"
	"naevis/infra"
	"naevis/infra/mq"
	"naevis/infra/mq/subscriber"
)

// Subscriber registers all media domain event handlers
type Subscriber struct {
	deps *infra.Deps
}

// NewSubscriber creates a new media domain subscriber
func NewSubscriber(deps *infra.Deps) subscriber.Subscriber {
	return &Subscriber{
		deps: deps,
	}
}

// Register subscribes to all media-related domain events
func (s *Subscriber) Register(
	ctx context.Context,
	bus mq.MQ,
) error {

	handler := NewEventHandler(s.deps)

	// Subscribe to media.uploaded events from dropify service
	if err := bus.QueueSubscribe(
		ctx,
		mqevent.MediaUploaded,
		"media-processor",
		handler.HandleMediaUploaded,
	); err != nil {

		return fmt.Errorf(
			"failed to subscribe to %s: %w",
			mqevent.MediaUploaded,
			err,
		)
	}

	log.Printf(
		"[MediaSubscriber] registered for subject=%s",
		mqevent.MediaUploaded,
	)

	return nil
}

// EventHandler handles media domain events
type EventHandler struct {
	deps *infra.Deps
}

// NewEventHandler creates a new event handler
func NewEventHandler(deps *infra.Deps) *EventHandler {
	return &EventHandler{
		deps: deps,
	}
}

// HandleMediaUploaded processes media.uploaded events from dropify
func (h *EventHandler) HandleMediaUploaded(
	ctx context.Context,
	data []byte,
) error {

	var payload mqevent.MediaUploadedPayload

	// -------------------------
	// Decode payload
	// -------------------------

	if err := json.Unmarshal(data, &payload); err != nil {

		log.Printf(
			"[MediaSubscriber] failed to unmarshal media.uploaded event: %v",
			err,
		)

		return err
	}

	log.Printf(
		"[MediaSubscriber] processing entity=%s id=%s file=%s path=%s",
		payload.EntityType,
		payload.EntityID,
		payload.FileName,
		payload.FilePath,
	)

	// -------------------------
	// Validate payload
	// -------------------------

	if payload.EntityType == "" {

		log.Printf(
			"[MediaSubscriber] missing entity type",
		)

		return nil
	}

	if payload.EntityID == "" {

		log.Printf(
			"[MediaSubscriber] missing entity id",
		)

		return nil
	}

	if payload.FilePath == "" {

		log.Printf(
			"[MediaSubscriber] missing file path",
		)

		return nil
	}

	// -------------------------
	// Resolve metadata
	// -------------------------

	meta, ok := getEntityMetadata(payload.EntityType)
	if !ok {

		log.Printf(
			"[MediaSubscriber] unsupported entity type: %s",
			payload.EntityType,
		)

		// poison messages should not retry forever
		return nil
	}

	// -------------------------
	// Build update
	// -------------------------

	update := buildUpdate(&payload)

	filter := bson.M{
		meta.keyField: payload.EntityID,
	}

	log.Printf(
		"[MediaSubscriber] updating collection=%s filter=%v update=%v",
		meta.collectionName,
		filter,
		update,
	)

	// -------------------------
	// Update database
	// -------------------------

	if err := h.deps.DB.UpdateOne(
		ctx,
		meta.collectionName,
		filter,
		update,
	); err != nil {

		log.Printf(
			"[MediaSubscriber] failed to update entity: %v",
			err,
		)

		return err
	}

	// -------------------------
	// Cache invalidation
	// -------------------------

	if meta.cacheKey != "" {

		cacheKey := meta.cacheKey + payload.EntityID

		if err := h.deps.Cache.Del(
			ctx,
			cacheKey,
		); err != nil {

			log.Printf(
				"[MediaSubscriber] cache invalidation failed key=%s err=%v",
				cacheKey,
				err,
			)

			// do not fail request because of cache issues
		}
	}

	log.Printf(
		"[MediaSubscriber] successfully processed entity=%s id=%s",
		payload.EntityType,
		payload.EntityID,
	)

	return nil
}

// entityMetadata holds collection and field information
type entityMetadata struct {
	collectionName string
	keyField       string
	cacheKey       string
	imageField     string
	isArray        bool
}

// getEntityMetadata returns metadata for an entity type
func getEntityMetadata(entityType string) (entityMetadata, bool) {

	metaMap := map[string]entityMetadata{

		"place": {
			collectionName: config.Collections.PlacesCollection,
			keyField:       "placeid",
			cacheKey:       "place:",
			imageField:     "banner",
			isArray:        false,
		},

		"event": {
			collectionName: config.Collections.EventsCollection,
			keyField:       "eventid",
			cacheKey:       "event:",
			imageField:     "banner",
			isArray:        false,
		},

		"baito": {
			collectionName: config.Collections.BaitoCollection,
			keyField:       "baitoid",
			cacheKey:       "baito:",
			imageField:     "imageUrls",
			isArray:        true,
		},

		"baito_worker": {
			collectionName: config.Collections.BaitoWorkerCollection,
			keyField:       "baitoUserId",
			cacheKey:       "worker:",
			imageField:     "imageUrls",
			isArray:        true,
		},

		"artist": {
			collectionName: config.Collections.ArtistsCollection,
			keyField:       "artistid",
			cacheKey:       "artist:",
			imageField:     "banner",
			isArray:        false,
		},

		"farm": {
			collectionName: config.Collections.FarmsCollection,
			keyField:       "farmid",
			cacheKey:       "farm:",
			imageField:     "banner",
			isArray:        false,
		},

		"crop": {
			collectionName: config.Collections.CropsCollection,
			keyField:       "cropid",
			cacheKey:       "crop:",
			imageField:     "banner",
			isArray:        false,
		},

		"feedpost": {
			collectionName: config.Collections.FeedPostsCollection,
			keyField:       "postid",
			cacheKey:       "feedpost:",
			imageField:     "imageUrls",
			isArray:        true,
		},

		"user": {
			collectionName: config.Collections.UserCollection,
			keyField:       "userid",
			cacheKey:       "profile:",
			imageField:     "avatar",
			isArray:        false,
		},

		"recipe": {
			collectionName: config.Collections.RecipeCollection,
			keyField:       "recipeid",
			cacheKey:       "recipe:",
			imageField:     "banner",
			isArray:        false,
		},
	}

	meta, ok := metaMap[entityType]

	return meta, ok
}

// buildUpdate creates Mongo update operators
func buildUpdate(
	payload *mqevent.MediaUploadedPayload,
) bson.M {

	meta, _ := getEntityMetadata(payload.EntityType)

	updatedAt := time.Now().UTC()

	// array image fields
	if meta.isArray {

		return bson.M{
			"$addToSet": bson.M{
				meta.imageField: payload.FilePath,
			},
			"$set": bson.M{
				"updated_at": updatedAt,
			},
		}
	}

	// scalar image fields
	return bson.M{
		"$set": bson.M{
			meta.imageField: payload.FilePath,
			"updated_at":    updatedAt,
		},
	}
}
