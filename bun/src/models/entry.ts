import mongoose, { type Document, Schema } from "mongoose";
import { EntryModel as EntrySchema } from "../modules/entries/model";

export type EntryDocument = EntrySchema.createBody &
	Document & {
		createdAt: Date;
		updatedAt: Date;
	};

const AccountMongoSchema = new Schema<EntrySchema.account>(
	{
		participant: { type: String, required: true }, // ISPB
		branch: { type: String, required: true },
		accountNumber: { type: String, required: true },
		accountType: {
			type: String,
			enum: EntrySchema.accountTypes,
			required: true,
		},
	},
	{ _id: false },
);

const OwnerMongoSchema = new Schema<EntrySchema.owner>(
	{
		type: {
			type: String,
			enum: EntrySchema.ownerTypes,
			required: true,
		},
		taxIdNumber: { type: String, required: true },
		name: { type: String, required: true },
	},
	{ _id: false },
);

const EntryMongoSchema = new Schema<EntryDocument>(
	{
		key: { type: String, required: true, unique: true, index: true },
		keyType: {
			type: String,
			enum: EntrySchema.keyTypes,
			required: true,
		},
		account: { type: AccountMongoSchema, required: true },
		owner: { type: OwnerMongoSchema, required: true },
	},
	{
		timestamps: true,
	},
);

// Index for owner lookups
EntryMongoSchema.index({ "owner.taxIdNumber": 1 });

export const EntryModel = mongoose.model<EntryDocument>(
	"Entry",
	EntryMongoSchema,
);
