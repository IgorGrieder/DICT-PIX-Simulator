export type KeyType = "CPF" | "CNPJ" | "EMAIL" | "PHONE" | "EVP";
export type AccountType = "CACC" | "SVGS" | "SLRY";
export type OwnerType = "NATURAL_PERSON" | "LEGAL_PERSON";

export type Account = {
	participant: string; // ISPB (8 digits)
	branch: string; // Agency (4 digits)
	accountNumber: string;
	accountType: AccountType;
}

export type Owner = {
	type: OwnerType;
	taxIdNumber: string; // CPF or CNPJ
	name: string;
}

export type Entry = {
	key: string;
	keyType: KeyType;
	account: Account;
	owner: Owner;
	createdAt: Date;
	updatedAt: Date;
}

export type IdempotencyRecord = {
	key: string;
	response: unknown;
	statusCode: number;
	createdAt: Date;
}
