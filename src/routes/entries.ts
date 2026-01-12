import { Elysia } from "elysia";
import { z } from "zod/v4";
import { createEntry, deleteEntry, getEntry } from "../handlers/entries";
import { idempotencyMiddleware } from "../utils/idempotency";

// Zod schemas for request validation (using Elysia's Standard Schema support)
const AccountSchema = z.object({
	participant: z.string().regex(/^\d{8}$/, "ISPB must be 8 digits"),
	branch: z.string().regex(/^\d{4}$/, "Branch must be 4 digits"),
	accountNumber: z.string().min(1, "Account number is required"),
	accountType: z.enum(["CACC", "SVGS", "SLRY"]),
});

const OwnerSchema = z.object({
	type: z.enum(["NATURAL_PERSON", "LEGAL_PERSON"]),
	taxIdNumber: z.string().min(1, "Tax ID is required"),
	name: z.string().min(1, "Name is required"),
});

const CreateEntryBodySchema = z.object({
	key: z.string().min(1, "Key is required"),
	keyType: z.enum(["CPF", "CNPJ", "EMAIL", "PHONE", "EVP"]),
	account: AccountSchema,
	owner: OwnerSchema,
});

export const entriesRoutes = new Elysia({ prefix: "/entries" })
	.use(idempotencyMiddleware)
	// Create Entry with Zod validation via Standard Schema
	.post("/", createEntry, {
		idempotent: true,
		body: CreateEntryBodySchema,
	})
	// Get Entry
	.get("/:key", getEntry)
	// Delete Entry
	.delete("/:key", deleteEntry);
