package db

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
)

type FindManyOptions struct {
	Limit      int
	Skip       int
	Sort       bson.D
	Projection []string
}

type Database interface {
	/* Lifecycle */
	Ping(ctx context.Context) error
	WithDB(ctx context.Context, op func(ctx context.Context) error) error
	RunTransaction(ctx context.Context, fn func(ctx context.Context) error) error

	/* Create */
	Insert(ctx context.Context, collection string, document any) error
	InsertOne(ctx context.Context, collection string, document any) error
	InsertMany(ctx context.Context, collection string, documents []any) error
	BulkWrite(ctx context.Context, collection string, operations []any) error

	/* Read */
	FindOne(ctx context.Context, collection string, filter any, result any) error
	FindOneWithProjection(ctx context.Context, collection string, filter any, projection []string, result any) error

	FindMany(ctx context.Context, collection string, filter any, result any) error
	FindManyWithOptions(ctx context.Context, collection string, filter any, opts FindManyOptions, result any) error
	FindManyWithProjection(
		ctx context.Context,
		collection string,
		filter any,
		projection []string,
		opts FindManyOptions,
		result any,
	) error

	Distinct(ctx context.Context, collection string, field string, filter any, result any) error

	/* Update */
	Update(ctx context.Context, collection string, filter any, update any) error
	UpdateOne(ctx context.Context, collection string, filter any, update any) error
	UpdateMany(ctx context.Context, collection string, filter any, update any) error
	Upsert(ctx context.Context, collection string, filter any, document any) error
	Inc(ctx context.Context, collection string, filter any, field string, value int64) error
	AddToSet(ctx context.Context, collection string, filter any, field string, value any) error

	/* Delete */
	Delete(ctx context.Context, collection string, filter any) (int64, error)
	DeleteOne(ctx context.Context, collection string, filter any) (int64, error)
	DeleteMany(ctx context.Context, collection string, filter any) error

	/* Atomic */
	FindOneAndUpdate(ctx context.Context, collection string, filter any, update any, result any) error

	/* Aggregate / Count */
	Aggregate(ctx context.Context, collection string, pipeline any, result any) error
	Count(ctx context.Context, collection string, filter any) (int64, error)
	CountDocuments(ctx context.Context, collection string, filter any) (int64, error)
	EstimatedDocumentCount(ctx context.Context, collection string) (int64, error)
}
