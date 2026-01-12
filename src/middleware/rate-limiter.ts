import { Elysia } from "elysia";
import { checkRateLimit } from "../utils/rate-limiter";

export const rateLimiterMiddleware = new Elysia({
	name: "rate-limiter",
}).derive(async ({ headers, set }) => {
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
