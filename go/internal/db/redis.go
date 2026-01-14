package db

import (
	"context"
	"time"

	"github.com/dict-simulator/go/internal/logger"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type Redis struct {
	Client *redis.Client
}

func ConnectRedis(uri string) (*Redis, error) {
	opts, err := redis.ParseURL(uri)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opts)

	// Add OpenTelemetry tracing instrumentation
	if err := redisotel.InstrumentTracing(client); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	logger.Info("Redis connected", zap.String("uri", uri))
	return &Redis{Client: client}, nil
}

func (r *Redis) Disconnect() error {
	if r.Client == nil {
		return nil
	}
	return r.Client.Close()
}
