package db

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

func ConnectRedis(uri string) error {
	opts, err := redis.ParseURL(uri)
	if err != nil {
		return err
	}

	RedisClient = redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := RedisClient.Ping(ctx).Err(); err != nil {
		return err
	}

	log.Printf("Redis connected: %s", uri)
	return nil
}

func DisconnectRedis() error {
	if RedisClient == nil {
		return nil
	}
	return RedisClient.Close()
}
