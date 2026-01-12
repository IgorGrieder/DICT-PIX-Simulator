import mongoose, { type Document, Schema } from "mongoose";
import type { Account, Entry, Owner } from "../types";

export interface EntryDocument
	extends Omit<Entry, "createdAt" | "updatedAt">,
		Document {
	createdAt: Date;
	updatedAt: Date;
}

const AccountSchema = new Schema<Account>(
	{
		participant: { type: String, required: true }, // ISPB
		branch: { type: String, required: true },
		accountNumber: { type: String, required: true },
		accountType: {
			type: String,
			enum: ["CACC", "SVGS", "SLRY"],
			required: true,
		},
	},
	{ _id: false },
);

const OwnerSchema = new Schema<Owner>(
	{
		type: {
			type: String,
			enum: ["NATURAL_PERSON", "LEGAL_PERSON"],
			required: true,
		},
		taxIdNumber: { type: String, required: true },
		name: { type: String, required: true },
	},
	{ _id: false },
);

const EntrySchema = new Schema<EntryDocument>(
	{
		key: { type: String, required: true, unique: true, index: true },
		keyType: {
			type: String,
			enum: ["CPF", "CNPJ", "EMAIL", "PHONE", "EVP"],
			required: true,
		},
		account: { type: AccountSchema, required: true },
		owner: { type: OwnerSchema, required: true },
	},
	{
		timestamps: true,
	},
);

// Index for owner lookups
EntrySchema.index({ "owner.taxIdNumber": 1 });

export const EntryModel = mongoose.model<EntryDocument>("Entry", EntrySchema);
