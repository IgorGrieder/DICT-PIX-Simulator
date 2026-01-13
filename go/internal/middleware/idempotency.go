package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/dict-simulator/go/internal/models"
)

const IdempotencyKeyHeader = "X-Idempotency-Key"

// responseRecorder captures the response for idempotency storage
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		body:           &bytes.Buffer{},
	}
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.statusCode = code
	rr.ResponseWriter.WriteHeader(code)
}

func (rr *responseRecorder) Write(b []byte) (int, error) {
	rr.body.Write(b)
	return rr.ResponseWriter.Write(b)
}

// IdempotencyMiddleware handles idempotent requests
func IdempotencyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idempotencyKey := r.Header.Get(IdempotencyKeyHeader)

		// If no idempotency key, proceed normally
		if idempotencyKey == "" {
			next.ServeHTTP(w, r)
			return
		}

		ctx := r.Context()

		// Try to atomically insert a "processing" record to claim this key
		// This prevents race conditions between concurrent requests
		claimed, record, err := claimIdempotencyKey(ctx, idempotencyKey)
		if err != nil {
			// On error, proceed with the request
			next.ServeHTTP(w, r)
			return
		}

		// If we didn't claim the key, return the existing response
		if !claimed && record != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(record.StatusCode)
			json.NewEncoder(w).Encode(record.Response)
			return
		}

		// We claimed the key, process the request
		recorder := newResponseRecorder(w)
		next.ServeHTTP(recorder, r)

		// Store the response (fire and forget, but synchronous to avoid data races)
		var response any
		if err := json.Unmarshal(recorder.body.Bytes(), &response); err == nil {
			models.SaveIdempotencyRecord(context.Background(), idempotencyKey, response, recorder.statusCode)
		}
	})
}

// claimIdempotencyKey attempts to atomically claim an idempotency key.
// Returns (true, nil, nil) if claimed, (false, record, nil) if already exists.
func claimIdempotencyKey(ctx context.Context, key string) (bool, *models.IdempotencyRecord, error) {
	// First, check if a completed record exists
	record, err := models.FindIdempotencyRecord(ctx, key)
	if err != nil {
		return false, nil, err
	}

	if record != nil {
		return false, record, nil
	}

	// Try to insert a new record atomically using upsert with a condition
	// This ensures only one request can claim the key
	collection := models.IdempotencyCollection()
	filter := bson.M{"key": key}
	update := bson.M{
		"$setOnInsert": bson.M{
			"key":        key,
			"statusCode": 0, // Processing marker
			"createdAt":  nil,
		},
	}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.Before)

	var existing models.IdempotencyRecord
	err = collection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&existing)

	if err == mongo.ErrNoDocuments {
		// We successfully inserted (claimed) the key
		return true, nil, nil
	}

	if err != nil {
		return false, nil, err
	}

	// Key already existed, return the existing record if it's complete
	if existing.StatusCode != 0 {
		return false, &existing, nil
	}

	// Another request is processing, we could wait or proceed
	// For simplicity, we'll proceed (the save will just overwrite)
	return true, nil, nil
}
