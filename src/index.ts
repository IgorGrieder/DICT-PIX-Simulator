import { opentelemetry } from "@elysiajs/opentelemetry";
import { OTLPTraceExporter } from "@opentelemetry/exporter-trace-otlp-proto";
import { BatchSpanProcessor } from "@opentelemetry/sdk-trace-node";
import { Elysia } from "elysia";
import { connectDB } from "./db";
import { entriesRoutes } from "./routes/entries";

const PORT = process.env.PORT || 3000;
const OTEL_ENDPOINT =
	process.env.OTEL_EXPORTER_OTLP_ENDPOINT || "http://localhost:4318/v1/traces";

// Connect to MongoDB
await connectDB();

const app = new Elysia()
	// OpenTelemetry integration
	.use(
		opentelemetry({
			serviceName: "dict-simulator",
			spanProcessors: [
				new BatchSpanProcessor(
					new OTLPTraceExporter({
						url: OTEL_ENDPOINT,
					}),
				),
			],
		}),
	)
	.get("/health", () => ({ status: "ok", timestamp: new Date().toISOString() }))
	.use(entriesRoutes)
	.listen(PORT);

console.log(`🦊 DICT Simulator running at http://localhost:${PORT}`);

export type App = typeof app;
