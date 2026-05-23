package reports

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func EnsureReportIndexes(coll *mongo.Collection) error {
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "reportedBy", Value: 1},
			{Key: "targetType", Value: 1},
			{Key: "targetId", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}
	_, err := coll.Indexes().CreateOne(context.Background(), indexModel)
	return err
}

// Call this once at startup, e.g.:
// err := ensureReportIndexes(db.ReportsCollection)
// if err != nil { log.Fatal(err) }

// This way, even if two requests slip through nearly concurrently, MongoDB itself will prevent a duplicate.\
