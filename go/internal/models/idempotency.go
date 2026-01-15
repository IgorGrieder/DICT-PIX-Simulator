package models

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/dict-simulator/go/internal/db"
)

// IdempotencyRecord represents a stored idempotent request response
type IdempotencyRecord struct {
	Key        string    `bson:"key"`
	Response   string    `bson:"response"` // Store as raw JSON string to preserve format
	StatusCode int       `bson:"statusCode"`
	CreatedAt  time.Time `bson:"createdAt"`
}

// IdempotencyRepository handles database operations for idempotency records
type IdempotencyRepository struct {
	collection *mongo.Collection
}

// NewIdempotencyRepository creates a new idempotency repository
func NewIdempotencyRepository(db *db.Mongo) *IdempotencyRepository {
	return &IdempotencyRepository{
		collection: db.Collection("idempotency"),
	}
}

// EnsureIndexes creates necessary indexes for the idempotency collection
func (r *IdempotencyRepository) EnsureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "key", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "createdAt", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(86400), // TTL: 24 hours
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	return err
}

// FindByKey finds an existing idempotency record
func (r *IdempotencyRepository) FindByKey(ctx context.Context, key string) (*IdempotencyRecord, error) {
	var record IdempotencyRecord
	err := r.collection.FindOne(ctx, bson.M{"key": key}).Decode(&record)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &record, nil
}

// ClaimKey attempts to atomically claim an idempotency key
// Returns (true, nil, nil) if claimed (newly inserted)
// Returns (false, record, nil) if already exists
func (r *IdempotencyRepository) ClaimKey(ctx context.Context, key string) (bool, *IdempotencyRecord, error) {
	// First, check if a completed record exists
	record, err := r.FindByKey(ctx, key)
	if err == nil && record != nil {
		return false, record, nil
	}

	if err != nil && err != mongo.ErrNoDocuments { // Unexpected error
		return false, nil, err
	}

	record = &IdempotencyRecord{
		Key:        key,
		StatusCode: 0,
		CreatedAt:  time.Now().UTC(),
	}

	filter := bson.M{"key": key}
	update := bson.M{
		"$setOnInsert": record,
	}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.Before)

	var existing IdempotencyRecord
	err = r.collection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&existing)

	if err == mongo.ErrNoDocuments {
		// We successfully inserted (claimed) the key because "Before" document was null
		return true, nil, nil
	}

	if err != nil {
		return false, nil, err
	}

	// Key already existed
	return false, &existing, nil
}

// Save saves or updates an idempotency record
func (r *IdempotencyRepository) Save(ctx context.Context, key string, response string, statusCode int) error {
	record := IdempotencyRecord{
		Key:        key,
		Response:   response,
		StatusCode: statusCode,
		CreatedAt:  time.Now().UTC(),
	}

	opts := options.Update().SetUpsert(true)
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"key": key},
		bson.M{"$set": record},
		opts,
	)
	return err
}
