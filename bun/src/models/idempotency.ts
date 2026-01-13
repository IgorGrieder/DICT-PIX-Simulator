import mongoose, { type Document, Schema } from "mongoose";

export type IdempotencyRecord = {
	key: string;
	response: unknown;
	statusCode: number;
	createdAt: Date;
};

export type IdempotencyDocument = IdempotencyRecord & Document;

const IdempotencySchema = new Schema<IdempotencyDocument>(
	{
		key: { type: String, required: true, unique: true },
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
