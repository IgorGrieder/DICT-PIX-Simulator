import { Elysia } from "elysia";
import { IdempotencyModel } from "../models/idempotency";

export const idempotencyMiddleware = new Elysia({ name: "idempotency" })
	.derive(({ request }) => {
		const idempotencyKey = request.headers.get("x-idempotency-key");
		return { idempotencyKey };
	})
	.macro({
		idempotent: (enabled: boolean) => ({
			async beforeHandle({ idempotencyKey, set }) {
				if (!enabled || !idempotencyKey) return;

				const existing = await IdempotencyModel.findOne({
					key: idempotencyKey,
				});
				if (existing) {
					set.status = existing.statusCode;
					return existing.response;
				}
			},
			async afterHandle({ idempotencyKey, response, set }) {
				if (!enabled || !idempotencyKey) return;

				// Store the response for future idempotent requests
				await IdempotencyModel.findOneAndUpdate(
					{ key: idempotencyKey },
					{
						key: idempotencyKey,
						response,
						statusCode: typeof set.status === "number" ? set.status : 200,
						createdAt: new Date(),
					},
					{ upsert: true },
				);
			},
		}),
	});
