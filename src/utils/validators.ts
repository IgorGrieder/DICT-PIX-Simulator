import type { EntryModel } from "../modules/entries/model";

type ValidationResult<T = void> =
	| { success: true; value: T }
	| { success: false; error: ValidationError };

type ValidationError =
	| { type: Keys.INVALID_CPF; message: string }
	| { type: Keys.INVALID_CNPJ; message: string }
	| { type: Keys.INVALID_EMAIL; message: string }
	| { type: Keys.INVALID_PHONE; message: string }
	| { type: Keys.INVALID_EVP; message: string };

export enum Keys {
	INVALID_CPF,
	INVALID_CNPJ,
	INVALID_EMAIL,
	INVALID_PHONE,
	INVALID_EVP,
}

// CPF validation with Módulo 11
function validateCPF(cpf: string): ValidationResult {
	if (!/^\d{11}$/.test(cpf))
		return {
			success: false,
			error: { type: Keys.INVALID_CPF, message: "Invalid CPF format" },
		};
	if (/^(\d)\1{10}$/.test(cpf))
		return {
			success: false,
			error: { type: Keys.INVALID_CPF, message: "Invalid CPF format" },
		}; // All same digits

	const digits = cpf.split("").map(Number);

	// First check digit
	let sum = 0;
	for (let i = 0; i < 9; i++) {
		sum += digits[i]! * (10 - i);
	}
	let remainder = (sum * 10) % 11;
	if (remainder === 10) remainder = 0;
	if (remainder !== digits[9]!)
		return {
			success: false,
			error: { type: Keys.INVALID_CPF, message: "Invalid CPF format" },
		};

	// Second check digit
	sum = 0;
	for (let i = 0; i < 10; i++) {
		sum += digits[i]! * (11 - i);
	}
	remainder = (sum * 10) % 11;
	if (remainder === 10) remainder = 0;
	if (remainder !== digits[10]!)
		return {
			success: false,
			error: { type: Keys.INVALID_CPF, message: "Invalid CPF format" },
		};

	return { success: true, value: undefined };
}

// CNPJ validation with Módulo 11
function validateCNPJ(cnpj: string): ValidationResult {
	if (!/^\d{14}$/.test(cnpj))
		return {
			success: false,
			error: { type: Keys.INVALID_CNPJ, message: "Invalid CNPJ format" },
		};
	if (/^(\d)\1{13}$/.test(cnpj))
		return {
			success: false,
			error: { type: Keys.INVALID_CNPJ, message: "Invalid CNPJ format" },
		}; // All same digits

	const digits = cnpj.split("").map(Number);
	const weights1 = [5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2];
	const weights2 = [6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2];

	// First check digit
	let sum = 0;
	for (let i = 0; i < 12; i++) {
		sum += digits[i]! * weights1[i]!;
	}
	let remainder = sum % 11;
	const firstCheck = remainder < 2 ? 0 : 11 - remainder;
	if (firstCheck !== digits[12]!)
		return {
			success: false,
			error: { type: Keys.INVALID_CNPJ, message: "Invalid CNPJ format" },
		};

	// Second check digit
	sum = 0;
	for (let i = 0; i < 13; i++) {
		sum += digits[i]! * weights2[i]!;
	}
	remainder = sum % 11;
	const secondCheck = remainder < 2 ? 0 : 11 - remainder;
	if (secondCheck !== digits[13]!)
		return {
			success: false,
			error: { type: Keys.INVALID_CNPJ, message: "Invalid CNPJ format" },
		};

	return { success: true, value: undefined };
}

// Email validation (RFC 5322 simplified)
function validateEmail(email: string): ValidationResult {
	const emailRegex =
		/^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$/;
	if (!emailRegex.test(email) || email.length > 77)
		return {
			success: false,
			error: { type: Keys.INVALID_EMAIL, message: "Invalid email format" },
		};
	return { success: true, value: undefined };
}

// Phone validation (+55 prefix, 10-11 digits)
function validatePhone(phone: string): ValidationResult {
	const phoneRegex = /^\+55\d{10,11}$/;
	if (!phoneRegex.test(phone))
		return {
			success: false,
			error: { type: Keys.INVALID_PHONE, message: "Invalid phone format" },
		};
	return { success: true, value: undefined };
}

// EVP validation (UUID v4)
function validateEVP(evp: string): ValidationResult {
	const uuidRegex =
		/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;
	if (!uuidRegex.test(evp))
		return {
			success: false,
			error: { type: Keys.INVALID_EVP, message: "Invalid EVP format" },
		};
	return { success: true, value: undefined };
}

export function validateKey(
	key: string,
	keyType: EntryModel.KeyType,
): ValidationResult {
	switch (keyType) {
		case "CPF":
			return validateCPF(key);
		case "CNPJ":
			return validateCNPJ(key);
		case "EMAIL":
			return validateEmail(key);
		case "PHONE":
			return validatePhone(key);
		case "EVP":
			return validateEVP(key);
	}
}
