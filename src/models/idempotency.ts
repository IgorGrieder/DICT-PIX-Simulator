import mongoose, { type Document, Schema } from "mongoose";
import type { IdempotencyRecord } from "../types";

export interface IdempotencyDocument extends IdempotencyRecord, Document {}

const IdempotencySchema = new Schema<IdempotencyDocument>(
	{
		key: { type: String, required: true, unique: true, index: true },
		response: { type: Schema.Types.Mixed, required: true },
		statusCode: { type: Number, required: true },
		createdAt: { type: Date, default: Date.now, expires: 86400 }, // TTL: 24 hours
	},
	{ timestamps: false },
);

export const IdempotencyModel = mongoose.model<IdempotencyDocument>(
	"Idempotency",
	IdempotencySchema,
);
