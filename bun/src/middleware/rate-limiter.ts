import { Elysia } from "elysia";
import { env } from "../config/env";
import { checkRateLimit } from "../utils/rate-limiter";

export const rateLimiterMiddleware = new Elysia({
	name: "rate-limiter",
}).derive(async ({ headers, set }) => {
	// Skip rate limiting if disabled (for benchmarks)
	if (!env.RATE_LIMIT_ENABLED) {
		return {
			rateLimitResult: { allowed: true, limit: 0, remaining: 0, reset: 0 },
		};
	}

	const userId = headers["x-user-id"] || "anonymous";
	const result = await checkRateLimit(userId);

	set.headers["X-RateLimit-Limit"] = String(result.limit);
	set.headers["X-RateLimit-Remaining"] = String(result.remaining);
	set.headers["X-RateLimit-Reset"] = String(result.reset);

	if (!result.allowed) {
		set.status = 429;
		return {
			error: "TOO_MANY_REQUESTS",
			message: "Rate limit exceeded. Please try again later.",
		};
	}

	return { rateLimitResult: result };
});
