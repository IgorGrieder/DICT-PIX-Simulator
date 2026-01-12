import { z } from "zod/v4";

export namespace EntryModel {
  // Shared enums as const arrays for reuse
  export const keyTypes = ["CPF", "CNPJ", "EMAIL", "PHONE", "EVP"] as const;
  export const accountTypes = ["CACC", "SVGS", "SLRY"] as const;
  export const ownerTypes = ["NATURAL_PERSON", "LEGAL_PERSON"] as const;

  // Derived literal types
  export type KeyType = (typeof keyTypes)[number];
  export type AccountType = (typeof accountTypes)[number];
  export type OwnerType = (typeof ownerTypes)[number];

  // Base nested schemas
  export const account = z.object({
    participant: z.string().regex(/^\d{8}$/, "ISPB must be 8 digits"),
    branch: z.string().regex(/^\d{4}$/, "Branch must be 4 digits"),
    accountNumber: z.string().min(1, "Account number is required"),
    accountType: z.enum(accountTypes),
  });
  export type account = z.infer<typeof account>;

  export const owner = z.object({
    type: z.enum(ownerTypes),
    taxIdNumber: z.string().min(1, "Tax ID is required"),
    name: z.string().min(1, "Name is required"),
  });
  export type owner = z.infer<typeof owner>;

  // Request schemas
  export const createBody = z.object({
    key: z.string().min(1, "Key is required"),
    keyType: z.enum(keyTypes),
    account,
    owner,
  });
  export type createBody = z.infer<typeof createBody>;

  export const keyParam = z.object({
    key: z.string().min(1),
  });
  export type keyParam = z.infer<typeof keyParam>;

  // Response schemas
  export const response = z.object({
    key: z.string(),
    keyType: z.enum(keyTypes),
    account,
    owner,
    createdAt: z.date(),
  });
  export type response = z.infer<typeof response>;

  export const responseWithDates = response.extend({
    updatedAt: z.date(),
  });
  export type responseWithDates = z.infer<typeof responseWithDates>;

  export const error = z.object({
    error: z.string(),
    message: z.string(),
  });
  export type error = z.infer<typeof error>;

  export const deleteResponse = z.object({
    message: z.string(),
    key: z.string(),
  });
  export type deleteResponse = z.infer<typeof deleteResponse>;
}
