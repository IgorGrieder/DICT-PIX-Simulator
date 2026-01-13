import { Elysia } from "elysia";
import {
	Counter,
	Histogram,
	Registry,
	collectDefaultMetrics,
} from "prom-client";

// Create a custom registry
export const register = new Registry();

// Add default Node.js metrics (memory, CPU, etc.)
collectDefaultMetrics({ register });

// Custom metrics
export const httpRequestsTotal = new Counter({
	name: "http_requests_total",
	help: "Total number of HTTP requests",
	labelNames: ["method", "path", "status"],
	registers: [register],
});

export const httpRequestDuration = new Histogram({
	name: "http_request_duration_seconds",
	help: "HTTP request duration in seconds",
	labelNames: ["method", "path", "status"],
	buckets: [0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10],
	registers: [register],
});

// Metrics middleware plugin
export const metricsMiddleware = new Elysia({ name: "metrics" })
	.derive(() => {
		return { metricsStart: performance.now() };
	})
	.onAfterHandle(({ request, set, metricsStart }) => {
		const duration = (performance.now() - metricsStart) / 1000;
		const path = new URL(request.url).pathname;
		const method = request.method;
		const status = String(set.status || 200);

		httpRequestsTotal.inc({ method, path: normalizePath(path), status });
		httpRequestDuration.observe(
			{ method, path: normalizePath(path), status },
			duration,
		);
	})
	.onError(({ request, set, metricsStart }) => {
		const duration =
			(performance.now() - (metricsStart ?? performance.now())) / 1000;
		const path = new URL(request.url).pathname;
		const method = request.method;
		const status = String(set.status || 500);

		httpRequestsTotal.inc({ method, path: normalizePath(path), status });
		httpRequestDuration.observe(
			{ method, path: normalizePath(path), status },
			duration,
		);
	});

// Normalize paths to avoid high cardinality (e.g., /entries/12345 -> /entries/:key)
function normalizePath(path: string): string {
	// Replace UUIDs
	path = path.replace(
		/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/gi,
		":id",
	);
	// Replace CPF patterns (11 digits)
	path = path.replace(/\/\d{11}\b/g, "/:key");
	// Replace MongoDB ObjectIds
	path = path.replace(/[0-9a-f]{24}/gi, ":id");
	// Replace any remaining numeric IDs
	path = path.replace(/\/\d+\b/g, "/:id");
	return path;
}

// Metrics endpoint handler
export const metricsEndpoint = new Elysia({ name: "metrics-endpoint" }).get(
	"/metrics",
	async ({ set }) => {
		set.headers["Content-Type"] = register.contentType;
		return await register.metrics();
	},
);
