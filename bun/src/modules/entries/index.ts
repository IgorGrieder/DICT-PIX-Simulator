import { record } from "@elysiajs/opentelemetry";
import { Elysia } from "elysia";
import { authMiddleware } from "../../middleware/auth";
import { rateLimiterMiddleware } from "../../middleware/rate-limiter";
import { EntryModel as EntryDB } from "../../models/entry";
import { idempotencyMiddleware } from "../../utils/idempotency";
import { validateKey } from "../../utils/validators";
import { EntryModel } from "./model";

export const entriesModule = new Elysia({ prefix: "/entries" })
	.use(authMiddleware)
	.use(idempotencyMiddleware)
	.use(rateLimiterMiddleware)

	// Create Entry
	.post(
		"/",
		async ({ body, set }) => {
			return record("handler.createEntry", async () => {
				// Validate key format (body is already validated by Elysia)
				const validationResult = record("validation.key", () =>
					validateKey(body.key, body.keyType),
				);

				if (validationResult.success === false) {
					set.status = 400;
					return {
						error: validationResult.error.type,
						message: validationResult.error.message,
					};
				}

				// Check if key already exists
				const existing = await record("db.findExisting", () =>
					EntryDB.findOne({ key: body.key }),
				);

				if (existing) {
					set.status = 409;
					return {
						error: "KEY_ALREADY_EXISTS",
						message: "This key is already registered in the directory",
					};
				}

				// Create entry
				const entry = await record("db.create", () => EntryDB.create(body));

				set.status = 201;
				return {
					key: entry.key,
					keyType: entry.keyType,
					account: entry.account,
					owner: entry.owner,
					createdAt: entry.createdAt,
				};
			});
		},
		{
			idempotent: true,
			body: EntryModel.createBody,
			response: {
				201: EntryModel.response,
				400: EntryModel.error,
				409: EntryModel.error,
			},
		},
	)

	// Get Entry
	.get(
		"/:key",
		async ({ params, set }) => {
			return record("handler.getEntry", async () => {
				const entry = await record("db.findOne", () =>
					EntryDB.findOne({ key: params.key }),
				);

				if (!entry) {
					set.status = 404;
					return {
						error: "ENTRY_NOT_FOUND",
						message: "No entry found for this key",
					};
				}

				return {
					key: entry.key,
					keyType: entry.keyType,
					account: entry.account,
					owner: entry.owner,
					createdAt: entry.createdAt,
					updatedAt: entry.updatedAt,
				};
			});
		},
		{
			params: EntryModel.keyParam,
			response: {
				200: EntryModel.responseWithDates,
				404: EntryModel.error,
			},
		},
	)

	// Delete Entry
	.delete(
		"/:key",
		async ({ params, set }) => {
			return record("handler.deleteEntry", async () => {
				const entry = await record("db.findOneAndDelete", () =>
					EntryDB.findOneAndDelete({ key: params.key }),
				);

				if (!entry) {
					set.status = 404;
					return {
						error: "ENTRY_NOT_FOUND",
						message: "No entry found for this key",
					};
				}

				set.status = 200;
				return {
					message: "Entry deleted successfully",
					key: entry.key,
				};
			});
		},
		{
			params: EntryModel.keyParam,
			response: {
				200: EntryModel.deleteResponse,
				404: EntryModel.error,
			},
		},
	);
