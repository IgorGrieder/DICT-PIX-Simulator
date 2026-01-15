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

	// 1. Initialize basic logger (stdout only, for early errors before OTEL is ready)
	if err := logger.Init(config.Env.Environment, nil); err != nil {
		panic("failed to initialize logger: " + err.Error())
	}

	// 2. Initialize tracing
	shutdownTracing, err := telemetry.InitTracer(config.Env.OTELExporterEndpoint)
	if err != nil {
		logger.Fatal("Failed to initialize tracer", zap.Error(err))
	}
	defer shutdownTracing(context.Background())

	// 3. Initialize logging provider for OTEL export
	shutdownLogging, err := telemetry.InitLoggerProvider(config.Env.OTELExporterEndpoint)
	if err != nil {
		logger.Fatal("Failed to initialize log provider", zap.Error(err))
	}
	defer shutdownLogging(context.Background())

	// 4. Re-initialize logger with OTEL export (dual output: stdout + OTEL)
	if err := logger.Init(config.Env.Environment, telemetry.LoggerProvider); err != nil {
		logger.Fatal("Failed to reinitialize logger with OTEL", zap.Error(err))
	}
	defer logger.Sync()

	// 5. Connect to MongoDB (now with otelmongo instrumentation)
	mongoDB, err := db.ConnectMongo(config.Env.MongoDBURI)
	if err != nil {
		logger.Fatal("Failed to connect to MongoDB", zap.Error(err))
	}
	defer mongoDB.Disconnect()

	// 6. Connect to Redis (now with redisotel instrumentation)
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
	mwManager := middleware.NewManager(idempotencyRepo, rateLimitBucket, config.Env.RateLimitEnabled)

	// Initialize handlers
	authHandler := auth.NewHandler(userRepo, config.Env.JWTSecret)
	entriesHandler := entries.NewHandler(entryRepo)

	// Setup router (now with otelhttp instrumentation)
	// Pass default rate limiting policies
	r := router.Setup(config.Env, authHandler, entriesHandler, mwManager, ratelimit.DefaultPolicies())

	// Start server
	srv := server.New(r, config.Env.Port)
	srv.ListenAndServeWithGracefulShutdown()
}
