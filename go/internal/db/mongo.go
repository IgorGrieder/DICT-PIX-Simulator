package db

import (
	"context"
	"time"

	"github.com/dict-simulator/go/internal/logger"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"
	"go.uber.org/zap"
)

type Mongo struct {
	Client   *mongo.Client
	Database *mongo.Database
}

func ConnectMongo(uri string) (*Mongo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Add OpenTelemetry instrumentation monitor
	clientOptions := options.Client().
		ApplyURI(uri).
		SetMonitor(otelmongo.NewMonitor())

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	m := &Mongo{
		Client:   client,
		Database: client.Database("dict"),
	}

	logger.Info("MongoDB connected", zap.String("uri", uri))
	return m, nil
}

func (m *Mongo) Disconnect() error {
	if m.Client == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return m.Client.Disconnect(ctx)
}

// Collection returns the specified collection
func (m *Mongo) Collection(name string) *mongo.Collection {
	return m.Database.Collection(name)
}

// WithDatabase returns a new Mongo instance pointing to a different database
// reusing the same client connection
func (m *Mongo) WithDatabase(name string) *Mongo {
	return &Mongo{
		Client:   m.Client,
		Database: m.Client.Database(name),
	}
}
