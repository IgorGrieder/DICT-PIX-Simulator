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
	Response   any       `bson:"response"`
	StatusCode int       `bson:"statusCode"`
	CreatedAt  time.Time `bson:"createdAt"`
}

func IdempotencyCollection() *mongo.Collection {
	return db.Collection("idempotency")
}

// EnsureIdempotencyIndexes creates necessary indexes for the idempotency collection
func EnsureIdempotencyIndexes(ctx context.Context) error {
	collection := IdempotencyCollection()

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

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	return err
}

// FindIdempotencyRecord finds an existing idempotency record
func FindIdempotencyRecord(ctx context.Context, key string) (*IdempotencyRecord, error) {
	var record IdempotencyRecord
	err := IdempotencyCollection().FindOne(ctx, bson.M{"key": key}).Decode(&record)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &record, nil
}

// SaveIdempotencyRecord saves or updates an idempotency record
func SaveIdempotencyRecord(ctx context.Context, key string, response any, statusCode int) error {
	record := IdempotencyRecord{
		Key:        key,
		Response:   response,
		StatusCode: statusCode,
		CreatedAt:  time.Now(),
	}

	opts := options.Update().SetUpsert(true)
	_, err := IdempotencyCollection().UpdateOne(
		ctx,
		bson.M{"key": key},
		bson.M{"$set": record},
		opts,
	)
	return err
}
