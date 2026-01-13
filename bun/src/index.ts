import { fromTypes, openapi } from "@elysiajs/openapi";
import { opentelemetry } from "@elysiajs/opentelemetry";
import { OTLPTraceExporter } from "@opentelemetry/exporter-trace-otlp-proto";
import { BatchSpanProcessor } from "@opentelemetry/sdk-trace-node";
import { Elysia } from "elysia";
import { env } from "./config/env";
import { connectDB } from "./db";
import { authModule } from "./modules/auth";
import { entriesModule } from "./modules/entries";
import { metricsEndpoint, metricsMiddleware } from "./utils/metrics";

// Connect to MongoDB
await connectDB();

const app = new Elysia()
	// Prometheus metrics
	.use(metricsMiddleware)
	.use(metricsEndpoint)
	// OpenTelemetry integration
	.use(
		opentelemetry({
			serviceName: "dict-simulator",
			spanProcessors: [
				new BatchSpanProcessor(
					new OTLPTraceExporter({
						url: env.OTEL_EXPORTER_OTLP_ENDPOINT,
					}),
				),
			],
		}),
	)
	.use(
		openapi({
			references: fromTypes(),
		}),
	)
	.get("/health", () => ({ status: "ok", timestamp: new Date().toISOString() }))
	.use(authModule)
	.use(entriesModule)
	.listen(env.PORT);

console.log(`DICT Simulator running at http://localhost:${env.PORT}`);

export type App = typeof app;
