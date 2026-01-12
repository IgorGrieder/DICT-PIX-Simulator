import type { KeyType } from "../types";

// CPF validation with Módulo 11
function isValidCPF(cpf: string): boolean {
	if (!/^\d{11}$/.test(cpf)) return false;
	if (/^(\d)\1{10}$/.test(cpf)) return false; // All same digits

	const digits = cpf.split("").map(Number);

	// First check digit
	let sum = 0;
	for (let i = 0; i < 9; i++) {
		sum += digits[i]! * (10 - i);
	}
	let remainder = (sum * 10) % 11;
	if (remainder === 10) remainder = 0;
	if (remainder !== digits[9]!) return false;

	// Second check digit
	sum = 0;
	for (let i = 0; i < 10; i++) {
		sum += digits[i]! * (11 - i);
	}
	remainder = (sum * 10) % 11;
	if (remainder === 10) remainder = 0;
	if (remainder !== digits[10]!) return false;

	return true;
}

// CNPJ validation with Módulo 11
function isValidCNPJ(cnpj: string): boolean {
	if (!/^\d{14}$/.test(cnpj)) return false;
	if (/^(\d)\1{13}$/.test(cnpj)) return false; // All same digits

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
	if (firstCheck !== digits[12]!) return false;

	// Second check digit
	sum = 0;
	for (let i = 0; i < 13; i++) {
		sum += digits[i]! * weights2[i]!;
	}
	remainder = sum % 11;
	const secondCheck = remainder < 2 ? 0 : 11 - remainder;
	if (secondCheck !== digits[13]!) return false;

	return true;
}

// Email validation (RFC 5322 simplified)
function isValidEmail(email: string): boolean {
	const emailRegex =
		/^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$/;
	return emailRegex.test(email) && email.length <= 77;
}

// Phone validation (+55 prefix, 10-11 digits)
function isValidPhone(phone: string): boolean {
	const phoneRegex = /^\+55\d{10,11}$/;
	return phoneRegex.test(phone);
}

// EVP validation (UUID v4)
function isValidEVP(evp: string): boolean {
	const uuidRegex =
		/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;
	return uuidRegex.test(evp);
}

export function validateKey(key: string, keyType: KeyType): boolean {
	switch (keyType) {
		case "CPF":
			return isValidCPF(key);
		case "CNPJ":
			return isValidCNPJ(key);
		case "EMAIL":
			return isValidEmail(key);
		case "PHONE":
			return isValidPhone(key);
		case "EVP":
			return isValidEVP(key);
		default:
			return false;
	}
}
