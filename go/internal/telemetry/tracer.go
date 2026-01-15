package telemetry

import (
	"context"
	"net/http"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	// Tracer is the global OpenTelemetry tracer
	Tracer trace.Tracer
	// TracerProvider is exposed for use with otelhttp and other instrumentation
	TracerProvider *sdktrace.TracerProvider
	// LoggerProvider is exposed for use with otelzap
	LoggerProvider log.LoggerProvider
)

// parseEndpoint extracts the host from the OTEL endpoint URL
func parseEndpoint(endpoint string) string {
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimSuffix(endpoint, "/v1/traces")
	endpoint = strings.TrimSuffix(endpoint, "/v1/logs")
	return endpoint
}

// InitTracer initializes the OpenTelemetry tracer and returns a shutdown function
func InitTracer(otelEndpoint string) (func(context.Context) error, error) {
	ctx := context.Background()

	endpoint := parseEndpoint(otelEndpoint)

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

	// Set up W3C TraceContext propagator for distributed tracing
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Store and set the provider globally
	TracerProvider = tp
	otel.SetTracerProvider(tp)
	Tracer = otel.Tracer("dict-simulator")

	return tp.Shutdown, nil
}

// InitLoggerProvider initializes the OpenTelemetry log provider for otelzap
func InitLoggerProvider(otelEndpoint string) (func(context.Context) error, error) {
	ctx := context.Background()

	endpoint := parseEndpoint(otelEndpoint)

	exporter, err := otlploghttp.New(ctx,
		otlploghttp.WithEndpoint(endpoint),
		otlploghttp.WithInsecure(),
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

	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
		sdklog.WithResource(res),
	)

	// Store the provider for use with otelzap
	LoggerProvider = lp

	return lp.Shutdown, nil
}

// WithTracing wraps a handler with OpenTelemetry tracing
// Deprecated: Use otelhttp.NewHandler instead for automatic HTTP instrumentation
func WithTracing(spanName string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := Tracer.Start(r.Context(), spanName,
			trace.WithAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.url", r.URL.String()),
			),
		)
		defer span.End()

		handler(w, r.WithContext(ctx))
	}
}
