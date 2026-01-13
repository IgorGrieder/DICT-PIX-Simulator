package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/dict-simulator/go/internal/config"
	"github.com/dict-simulator/go/internal/db"
	"github.com/dict-simulator/go/internal/middleware"
	"github.com/dict-simulator/go/internal/models"
	"github.com/dict-simulator/go/internal/modules/auth"
	"github.com/dict-simulator/go/internal/modules/entries"
)

var tracer trace.Tracer

func main() {
	// Load configuration
	config.Load()

	// Initialize OpenTelemetry
	shutdown, err := initTracer()
	if err != nil {
		log.Fatalf("Failed to initialize tracer: %v", err)
	}
	defer shutdown(context.Background())

	tracer = otel.Tracer("dict-simulator")

	// Connect to MongoDB
	if err := db.ConnectMongo(config.Env.MongoDBURI); err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer db.DisconnectMongo()

	// Connect to Redis
	if err := db.ConnectRedis(config.Env.RedisURI); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer db.DisconnectRedis()

	// Ensure indexes
	ctx := context.Background()
	if err := models.EnsureUserIndexes(ctx); err != nil {
		log.Printf("Warning: Failed to create user indexes: %v", err)
	}
	if err := models.EnsureEntryIndexes(ctx); err != nil {
		log.Printf("Warning: Failed to create entry indexes: %v", err)
	}
	if err := models.EnsureIdempotencyIndexes(ctx); err != nil {
		log.Printf("Warning: Failed to create idempotency indexes: %v", err)
	}

	// Initialize handlers
	authHandler := auth.NewHandler()
	entriesHandler := entries.NewHandler()

	// Setup router
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("GET /health", withTracing("health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":    "ok",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	}))

	// Prometheus metrics endpoint
	mux.Handle("GET /metrics", promhttp.Handler())

	// Auth routes (no auth middleware)
	mux.HandleFunc("POST /auth/register", withTracing("auth.register", authHandler.Register))
	mux.HandleFunc("POST /auth/login", withTracing("auth.login", authHandler.Login))

	// Entries routes (with auth + rate limiter + idempotency for POST)
	mux.Handle("POST /entries", chain(
		http.HandlerFunc(withTracing("entries.create", entriesHandler.Create)),
		middleware.AuthMiddleware,
		middleware.RateLimiterMiddleware,
		middleware.IdempotencyMiddleware,
	))

	mux.Handle("GET /entries/{key}", chain(
		http.HandlerFunc(withTracing("entries.get", entriesHandler.Get)),
		middleware.AuthMiddleware,
		middleware.RateLimiterMiddleware,
	))

	mux.Handle("DELETE /entries/{key}", chain(
		http.HandlerFunc(withTracing("entries.delete", entriesHandler.Delete)),
		middleware.AuthMiddleware,
		middleware.RateLimiterMiddleware,
	))

	// Wrap with metrics, CORS and logging
	handler := middleware.MetricsMiddleware(logMiddleware(corsMiddleware(mux)))

	// Create server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.Env.Port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	log.Printf("DICT Simulator running at http://localhost:%d", config.Env.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}

// initTracer initializes the OpenTelemetry tracer
func initTracer() (func(context.Context) error, error) {
	ctx := context.Background()

	// Parse the endpoint URL to get just the host
	endpoint := config.Env.OTELExporterEndpoint
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimSuffix(endpoint, "/v1/traces")

	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("dict-simulator"),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)

	return tp.Shutdown, nil
}

// withTracing wraps a handler with OpenTelemetry tracing
func withTracing(spanName string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), spanName,
			trace.WithAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.url", r.URL.String()),
			),
		)
		defer span.End()

		handler(w, r.WithContext(ctx))
	}
}

// chain applies middlewares in order (last middleware wraps first)
func chain(handler http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

// corsMiddleware adds CORS headers
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Idempotency-Key, X-User-Id")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// logMiddleware logs incoming requests
func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
