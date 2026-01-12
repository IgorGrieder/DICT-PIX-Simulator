import { record } from "@elysiajs/opentelemetry";
import { EntryModel } from "../models/entry";
import type { KeyType } from "../types";
import { validateKey } from "../utils/validators";

export async function createEntry(ctx: {
	body: {
		key: string;
		keyType: "CPF" | "CNPJ" | "EMAIL" | "PHONE" | "EVP";
		account: {
			participant: string;
			branch: string;
			accountNumber: string;
			accountType: "CACC" | "SVGS" | "SLRY";
		};
		owner: {
			type: "NATURAL_PERSON" | "LEGAL_PERSON";
			taxIdNumber: string;
			name: string;
		};
	};
	set: { status?: number | string };
}) {
	return record("handler.createEntry", async () => {
		const { body, set } = ctx;

		// Validate key format (body is already validated by Elysia)
		const isValidKey = await record("validation.key", () =>
			validateKey(body.key, body.keyType as KeyType),
		);

		if (!isValidKey) {
			set.status = 400;
			return {
				error: "INVALID_KEY_FORMAT",
				message: `Invalid ${body.keyType} format`,
			};
		}

		// Check if key already exists
		const existing = await record("db.findExisting", () =>
			EntryModel.findOne({ key: body.key }),
		);

		if (existing) {
			set.status = 409;
			return {
				error: "KEY_ALREADY_EXISTS",
				message: "This key is already registered in the directory",
			};
		}

		// Create entry
		const entry = await record("db.create", () => EntryModel.create(body));

		set.status = 201;
		return {
			key: entry.key,
			keyType: entry.keyType,
			account: entry.account,
			owner: entry.owner,
			createdAt: entry.createdAt,
		};
	});
}

export async function getEntry(ctx: {
	params: { key: string };
	set: { status?: number | string };
}) {
	return record("handler.getEntry", async () => {
		const { params, set } = ctx;

		const entry = await record("db.findOne", () =>
			EntryModel.findOne({ key: params.key }),
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
}

export async function deleteEntry(ctx: {
	params: { key: string };
	set: { status?: number | string };
}) {
	return record("handler.deleteEntry", async () => {
		const { params, set } = ctx;

		const entry = await record("db.findOneAndDelete", () =>
			EntryModel.findOneAndDelete({ key: params.key }),
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
}
