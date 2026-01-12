import { RedisClient } from "bun";
import { env } from "../config/env";

const redis = new RedisClient(env.REDIS_URI);

const bucketSize = env.RATE_LIMIT_BUCKET_SIZE;
const refillSeconds = env.RATE_LIMIT_REFILL_SECONDS;

export interface RateLimitResult {
	allowed: boolean;
	limit: number;
	remaining: number;
	reset: number;
}

export async function checkRateLimit(userId: string): Promise<RateLimitResult> {
	const key = `rate_limit:${userId}`;

	const currentTokens = await redis.get(key);

	if (currentTokens === null) {
		await redis.set(key, String(bucketSize - 1));
		return {
			allowed: true,
			limit: bucketSize,
			remaining: bucketSize - 1,
			reset: Math.floor(Date.now() / 1000) + refillSeconds,
		};
	}

	const tokens = Number.parseInt(currentTokens, 10);

	if (tokens <= 0) {
		return {
			allowed: false,
			limit: bucketSize,
			remaining: 0,
			reset: Math.floor(Date.now() / 1000) + refillSeconds,
		};
	}

	const remaining = tokens - 1;
	await redis.set(key, String(remaining));

	return {
		allowed: true,
		limit: bucketSize,
		remaining,
		reset: Math.floor(Date.now() / 1000) + refillSeconds,
	};
}

export { redis };
