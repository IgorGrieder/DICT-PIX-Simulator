package main

import (
	"context"

	"go.uber.org/zap"

	"github.com/dict-simulator/go/internal/config"
	"github.com/dict-simulator/go/internal/db"
	"github.com/dict-simulator/go/internal/logger"
	"github.com/dict-simulator/go/internal/middleware"
	"github.com/dict-simulator/go/internal/models"
	"github.com/dict-simulator/go/internal/modules/auth"
	"github.com/dict-simulator/go/internal/modules/entries"
	"github.com/dict-simulator/go/internal/ratelimit"
	"github.com/dict-simulator/go/internal/router"
	"github.com/dict-simulator/go/internal/server"
	"github.com/dict-simulator/go/internal/telemetry"
)

func main() {
	// Load configuration
	config.Load()

	// Initialize logger
	if err := logger.Init(config.Env.Environment); err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	defer logger.Sync()

	// Initialize tracing
	shutdownTracing, err := telemetry.InitTracer()
	if err != nil {
		logger.Fatal("Failed to initialize tracer", zap.Error(err))
	}
	defer shutdownTracing(context.Background())

	// Connect to MongoDB
	mongoDB, err := db.ConnectMongo(config.Env.MongoDBURI)
	if err != nil {
		logger.Fatal("Failed to connect to MongoDB", zap.Error(err))
	}
	defer mongoDB.Disconnect()

	// Connect to Redis
	redisDB, err := db.ConnectRedis(config.Env.RedisURI)
	if err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	defer redisDB.Disconnect()

	// Initialize repositories
	entryRepo := models.NewEntryRepository(mongoDB)
	userRepo := models.NewUserRepository(mongoDB)
	idempotencyRepo := models.NewIdempotencyRepository(mongoDB)

	// Ensure database indexes
	ctx := context.Background()
	if err := entryRepo.EnsureIndexes(ctx); err != nil {
		logger.Fatal("Failed to ensure entry indexes", zap.Error(err))
	}
	if err := userRepo.EnsureIndexes(ctx); err != nil {
		logger.Fatal("Failed to ensure user indexes", zap.Error(err))
	}
	if err := idempotencyRepo.EnsureIndexes(ctx); err != nil {
		logger.Fatal("Failed to ensure idempotency indexes", zap.Error(err))
	}

	// Initialize services/components
	rateLimitBucket := ratelimit.NewBucket(redisDB.Client)
	mwManager := middleware.NewManager(idempotencyRepo, rateLimitBucket)

	// Initialize handlers
	authHandler := auth.NewHandler(userRepo)
	entriesHandler := entries.NewHandler(entryRepo)

	// Setup router
	r := router.Setup(authHandler, entriesHandler, mwManager)

	// Start server
	srv := server.New(r, config.Env.Port)
	srv.ListenAndServeWithGracefulShutdown()

}
