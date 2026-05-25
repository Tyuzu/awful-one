package db

import (
	"context"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

/* -------------------- Types -------------------- */

type MongoDatabase struct {
	db      *mongo.Database
	client  *mongo.Client
	limiter chan struct{}
}

func NewMongoDatabase(db *mongo.Database, client *mongo.Client, maxConcurrent int) *MongoDatabase {
	if maxConcurrent <= 0 {
		maxConcurrent = 50
	}

	return &MongoDatabase{
		db:      db,
		client:  client,
		limiter: make(chan struct{}, maxConcurrent),
	}
}

func (m *MongoDatabase) collection(name string) *mongo.Collection {
	return m.db.Collection(name)
}

/* -------------------- Lifecycle -------------------- */

func (m *MongoDatabase) Ping(ctx context.Context) error {
	c, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return m.client.Ping(c, nil)
}

func (m *MongoDatabase) WithDB(ctx context.Context, op func(ctx context.Context) error) error {
	m.limiter <- struct{}{}
	defer func() { <-m.limiter }()

	var err error
	for i := 0; i < 2; i++ {
		c, cancel := context.WithTimeout(ctx, 5*time.Second)
		err = op(c)
		cancel()

		if err == nil {
			return nil
		}

		if mongo.IsNetworkError(err) || mongo.IsTimeout(err) {
			time.Sleep(200 * time.Millisecond)
			continue
		}

		return err
	}
	return err
}

func (m *MongoDatabase) RunTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	session, err := m.client.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sc mongo.SessionContext) (any, error) {
		return nil, fn(sc)
	})
	return err
}

/* -------------------- Create -------------------- */

func (m *MongoDatabase) Insert(ctx context.Context, collection string, document any) error {
	return m.InsertOne(ctx, collection, document)
}

func (m *MongoDatabase) InsertOne(ctx context.Context, collection string, document any) error {
	_, err := m.collection(collection).InsertOne(ctx, document)
	return err
}

func (m *MongoDatabase) InsertMany(ctx context.Context, collection string, documents []any) error {
	if len(documents) == 0 {
		return nil
	}
	_, err := m.collection(collection).InsertMany(ctx, documents)
	return err
}

func (m *MongoDatabase) BulkWrite(ctx context.Context, collection string, operations []any) error {
	if len(operations) == 0 {
		return nil
	}

	models := make([]mongo.WriteModel, 0, len(operations))
	for _, op := range operations {
		if wm, ok := op.(mongo.WriteModel); ok {
			models = append(models, wm)
		}
	}

	if len(models) == 0 {
		return nil
	}

	_, err := m.collection(collection).BulkWrite(ctx, models)
	return err
}

/* -------------------- Read -------------------- */

func (m *MongoDatabase) FindOne(ctx context.Context, collection string, filter any, result any) error {
	filter = m.normalizeFilter(filter)
	res := m.collection(collection).FindOne(ctx, filter)
	if err := res.Err(); err != nil {
		return err
	}
	return res.Decode(result)
}

func (m *MongoDatabase) FindOneWithProjection(ctx context.Context, collection string, filter any, projection []string, result any) error {
	filter = m.normalizeFilter(filter)

	opts := options.FindOne()
	if len(projection) > 0 {
		opts.SetProjection(buildProjection(projection))
	}

	res := m.collection(collection).FindOne(ctx, filter, opts)
	if err := res.Err(); err != nil {
		return err
	}
	return res.Decode(result)
}

func (m *MongoDatabase) FindMany(ctx context.Context, collection string, filter any, result any) error {
	filter = m.normalizeFilter(filter)
	cur, err := m.collection(collection).Find(ctx, filter)
	if err != nil {
		return err
	}
	defer cur.Close(ctx)
	return cur.All(ctx, result)
}

func (m *MongoDatabase) FindManyWithOptions(
	ctx context.Context,
	collection string,
	filter any,
	opts FindManyOptions,
	result any,
) error {

	filter = m.normalizeFilter(filter)

	findOpts := options.Find()

	if opts.Limit > 0 {
		findOpts.SetLimit(int64(opts.Limit))
	}

	if opts.Skip > 0 {
		findOpts.SetSkip(int64(opts.Skip))
	}

	if len(opts.Sort) > 0 {
		findOpts.SetSort(opts.Sort)
	}

	if len(opts.Projection) > 0 {
		findOpts.SetProjection(buildProjection(opts.Projection))
	}

	cur, err := m.collection(collection).Find(ctx, filter, findOpts)
	if err != nil {
		return err
	}
	defer cur.Close(ctx)

	return cur.All(ctx, result)
}

func (m *MongoDatabase) FindManyWithProjection(
	ctx context.Context,
	collection string,
	filter any,
	projection []string,
	opts FindManyOptions,
	result any,
) error {
	filter = m.normalizeFilter(filter)

	findOpts := options.Find()
	if opts.Limit > 0 {
		findOpts.SetLimit(int64(opts.Limit))
	}
	if opts.Skip > 0 {
		findOpts.SetSkip(int64(opts.Skip))
	}
	if len(opts.Sort) > 0 {
		findOpts.SetSort(opts.Sort)
	}
	if len(projection) > 0 {
		findOpts.SetProjection(buildProjection(projection))
	}

	cur, err := m.collection(collection).Find(ctx, filter, findOpts)
	if err != nil {
		return err
	}
	defer cur.Close(ctx)
	return cur.All(ctx, result)
}

func (m *MongoDatabase) Distinct(ctx context.Context, collection string, field string, filter any, result any) error {
	filter = m.normalizeFilter(filter)
	values, err := m.collection(collection).Distinct(ctx, field, filter)
	if err != nil {
		return err
	}
	bsonBytes, _ := bson.Marshal(values)
	return bson.Unmarshal(bsonBytes, result)
}

/* -------------------- Update -------------------- */

func (m *MongoDatabase) Update(ctx context.Context, collection string, filter any, update any) error {
	return m.UpdateOne(ctx, collection, filter, update)
}

func (m *MongoDatabase) UpdateOne(ctx context.Context, collection string, filter any, update any) error {
	filter = m.normalizeFilter(filter)
	_, err := m.collection(collection).
		UpdateOne(ctx, filter, update)
	return err
}

func (m *MongoDatabase) UpdateMany(ctx context.Context, collection string, filter any, update any) error {
	filter = m.normalizeFilter(filter)
	_, err := m.collection(collection).
		UpdateMany(ctx, filter, bson.M{"$set": update})
	return err
}

func (m *MongoDatabase) Upsert(ctx context.Context, collection string, filter any, update any) error {

	filter = m.normalizeFilter(filter)

	opts := options.Update().SetUpsert(true)

	_, err := m.collection(collection).UpdateOne(
		ctx,
		filter,
		update,
		opts,
	)

	return err
}

func (m *MongoDatabase) Inc(ctx context.Context, collection string, filter any, field string, value int64) error {
	filter = m.normalizeFilter(filter)
	_, err := m.collection(collection).
		UpdateOne(ctx, filter, bson.M{"$inc": bson.M{field: value}})
	return err
}

func (m *MongoDatabase) AddToSet(ctx context.Context, collection string, filter any, field string, value any) error {
	filter = m.normalizeFilter(filter)
	_, err := m.collection(collection).
		UpdateOne(ctx, filter, bson.M{"$addToSet": bson.M{field: value}})
	return err
}

/* -------------------- Delete -------------------- */

func (m *MongoDatabase) Delete(ctx context.Context, collection string, filter any) (int64, error) {
	return m.DeleteOne(ctx, collection, filter)
}

func (m *MongoDatabase) DeleteOne(ctx context.Context, collection string, filter any) (int64, error) {
	filter = m.normalizeFilter(filter)

	res, err := m.collection(collection).DeleteOne(ctx, filter)
	if err != nil {
		return 0, err
	}

	return res.DeletedCount, nil
}

func (m *MongoDatabase) DeleteMany(ctx context.Context, collection string, filter any) error {
	filter = m.normalizeFilter(filter)
	_, err := m.collection(collection).DeleteMany(ctx, filter)
	return err
}

/* -------------------- Atomic -------------------- */

func (m *MongoDatabase) FindOneAndUpdate(ctx context.Context, collection string, filter any, update any, result any) error {
	filter = m.normalizeFilter(filter)
	res := m.collection(collection).FindOneAndUpdate(
		ctx,
		filter,
		bson.M{"$set": update},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)
	if err := res.Err(); err != nil {
		return err
	}
	return res.Decode(result)
}

/* -------------------- Aggregate / Count -------------------- */

func (m *MongoDatabase) Aggregate(ctx context.Context, collection string, pipeline any, result any) error {
	cur, err := m.collection(collection).Aggregate(ctx, pipeline)
	if err != nil {
		return err
	}
	defer cur.Close(ctx)
	return cur.All(ctx, result)
}

func (m *MongoDatabase) Count(ctx context.Context, collection string, filter any) (int64, error) {
	return m.CountDocuments(ctx, collection, filter)
}

func (m *MongoDatabase) CountDocuments(ctx context.Context, collection string, filter any) (int64, error) {
	filter = m.normalizeFilter(filter)
	return m.collection(collection).CountDocuments(ctx, filter)
}

func (m *MongoDatabase) EstimatedDocumentCount(ctx context.Context, collection string) (int64, error) {
	return m.collection(collection).EstimatedDocumentCount(ctx)
}

/* -------------------- Helpers -------------------- */

func (m *MongoDatabase) normalizeFilter(filter any) any {
	if mf, ok := filter.(map[string]any); ok {
		return m.translateFilter(mf)
	}
	return filter
}

func (m *MongoDatabase) translateFilter(filter map[string]any) bson.M {
	out := bson.M{}
	for k, v := range filter {
		switch {
		case strings.HasSuffix(k, "_ne"):
			out[strings.TrimSuffix(k, "_ne")] = bson.M{"$ne": v}
		case strings.HasSuffix(k, "_contains"):
			out[strings.TrimSuffix(k, "_contains")] = bson.M{
				"$regex":   v,
				"$options": "i",
			}
		default:
			out[k] = v
		}
	}
	return out
}

func buildProjection(fields []string) bson.M {
	p := bson.M{}
	for _, f := range fields {
		p[f] = 1
	}
	return p
}
