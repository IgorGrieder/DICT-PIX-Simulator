package config

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	Port                   int
	Environment            string
	MongoDBURI             string
	RedisURI               string
	JWTSecret              string
	OTELExporterEndpoint   string
	RateLimitEnabled       bool
	RateLimitBucketSize    int
	RateLimitRefillSeconds int
}

var Env *Config

func Load() {
	port, _ := strconv.Atoi(getEnvOrDefault("PORT", "3000"))
	rateLimitEnabled := getEnvOrDefault("RATE_LIMIT_ENABLED", "true")
	rateLimitBucketSize, _ := strconv.Atoi(getEnvOrDefault("RATE_LIMIT_BUCKET_SIZE", "60"))
	rateLimitRefillSeconds, _ := strconv.Atoi(getEnvOrDefault("RATE_LIMIT_REFILL_SECONDS", "60"))

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	Env = &Config{
		Port:                   port,
		Environment:            getEnvOrDefault("GO_ENV", "development"),
		MongoDBURI:             getEnvOrDefault("MONGODB_URI", "mongodb://localhost:27017/dict"),
		RedisURI:               getEnvOrDefault("REDIS_URI", "redis://localhost:6379"),
		JWTSecret:              jwtSecret,
		OTELExporterEndpoint:   getEnvOrDefault("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318/v1/traces"),
		RateLimitEnabled:       rateLimitEnabled != "false" && rateLimitEnabled != "0",
		RateLimitBucketSize:    rateLimitBucketSize,
		RateLimitRefillSeconds: rateLimitRefillSeconds,
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
