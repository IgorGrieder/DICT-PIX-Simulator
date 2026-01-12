import { z } from "zod/v4";

const envSchema = z.object({
	PORT: z.coerce.number().default(3000),
	NODE_ENV: z
		.enum(["development", "production", "test"])
		.default("development"),

	// Database
	MONGODB_URI: z.string().default("mongodb://localhost:27017/dict"),
	REDIS_URI: z.string().default("redis://localhost:6379"),

	// Auth
	JWT_SECRET: z.string().min(1, "JWT_SECRET environment variable is required"),

	// OpenTelemetry
	OTEL_EXPORTER_OTLP_ENDPOINT: z
		.string()
		.default("http://localhost:4318/v1/traces"),

	// Rate Limiting
	RATE_LIMIT_BUCKET_SIZE: z.coerce.number().default(60),
	RATE_LIMIT_REFILL_SECONDS: z.coerce.number().default(60),
});

// Validate process.env
// We accept partial here because we might want to override defaults but if JWT_SECRET is missing it should fail
const _env = envSchema.safeParse(process.env);

if (!_env.success) {
	console.error("❌ Invalid environment variables:");
	console.error(JSON.stringify(_env.error.format(), null, 2));
	process.exit(1);
}

export const env = _env.data;
