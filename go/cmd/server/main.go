package main

import (
	"context"
	"net/http"

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

// databases holds database connections
type databases struct {
	mongo *db.Mongo
	redis *db.Redis
}

// repositories holds all repository instances
type repositories struct {
	entry       *models.EntryRepository
	user        *models.UserRepository
	idempotency *models.IdempotencyRepository
}

func main() {
	config.Load()

	shutdownTelemetry := setupTelemetry()
	defer shutdownTelemetry()

	dbs := setupDatabases()
	defer dbs.mongo.Disconnect()
	defer dbs.redis.Disconnect()

	repos := setupRepositories(dbs.mongo)

	handler := setupApp(repos, dbs.redis)

	srv := server.New(handler, config.Env.Port)
	srv.ListenAndServeWithGracefulShutdown()
}

// setupTelemetry initializes OpenTelemetry tracing and logging providers.
// Returns a cleanup function that should be deferred.
func setupTelemetry() func() {
	shutdownTracing, err := telemetry.InitTracer(config.Env.OTELExporterEndpoint)
	if err != nil {
		logger.Fatal("Failed to initialize tracer", zap.Error(err))
	}

	shutdownLogging, err := telemetry.InitLoggerProvider(config.Env.OTELExporterEndpoint)
	if err != nil {
		logger.Fatal("Failed to initialize log provider", zap.Error(err))
	}

	if err := logger.Init(config.Env.Environment, telemetry.LoggerProvider); err != nil {
		panic("failed to initialize logger: " + err.Error())
	}

	return func() {
		ctx := context.Background()
		shutdownTracing(ctx)
		shutdownLogging(ctx)
		logger.Sync()
	}
}

// setupDatabases establishes connections to MongoDB and Redis.
// Fatals on connection failure.
func setupDatabases() *databases {
	mongoDB, err := db.ConnectMongo(config.Env.MongoDBURI)
	if err != nil {
		logger.Fatal("Failed to connect to MongoDB", zap.Error(err))
	}

	redisDB, err := db.ConnectRedis(config.Env.RedisURI)
	if err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}

	return &databases{
		mongo: mongoDB,
		redis: redisDB,
	}
}

// setupRepositories creates all repository instances and ensures database indexes.
// Fatals on index creation failure.
func setupRepositories(mongoDB *db.Mongo) *repositories {
	entryRepo := models.NewEntryRepository(mongoDB)
	userRepo := models.NewUserRepository(mongoDB)
	idempotencyRepo := models.NewIdempotencyRepository(mongoDB)

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

	return &repositories{
		entry:       entryRepo,
		user:        userRepo,
		idempotency: idempotencyRepo,
	}
}

// setupApp initializes handlers, middleware, and the HTTP router.
// Returns the fully configured HTTP handler ready to serve requests.
func setupApp(repos *repositories, redisDB *db.Redis) http.Handler {
	rateLimitBucket := ratelimit.NewBucket(redisDB.Client)
	mwManager := middleware.NewManager(repos.idempotency, rateLimitBucket, config.Env.RateLimitEnabled)

	authHandler := auth.NewHandler(repos.user, config.Env.JWTSecret)
	entriesHandler := entries.NewHandler(repos.entry)

	return router.Setup(config.Env, authHandler, entriesHandler, mwManager, ratelimit.DefaultPolicies())
}
