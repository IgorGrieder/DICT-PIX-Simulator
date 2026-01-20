import { randomString } from "https://jslib.k6.io/k6-utils/1.4.0/index.js";
import { check, sleep } from "k6";
import http from "k6/http";

const BASE_URL = __ENV.BASE_URL || "http://localhost:3000";

// Generate valid CPF with Módulo 11
function generateValidCPF() {
	const digits = [];
	for (let i = 0; i < 9; i++) {
		digits.push(Math.floor(Math.random() * 10));
	}

	let sum = 0;
	for (let i = 0; i < 9; i++) {
		sum += digits[i] * (10 - i);
	}
	let remainder = (sum * 10) % 11;
	if (remainder === 10) remainder = 0;
	digits.push(remainder);

	sum = 0;
	for (let i = 0; i < 10; i++) {
		sum += digits[i] * (11 - i);
	}
	remainder = (sum * 10) % 11;
	if (remainder === 10) remainder = 0;
	digits.push(remainder);

	return digits.join("");
}

// Generate UUID v4 for requestId
function generateUUID() {
	return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
		const r = Math.random() * 16 | 0;
		const v = c === 'x' ? r : (r & 0x3 | 0x8);
		return v.toString(16);
	});
}

export function setup() {
	// Authentication
	// Create a unique user for this test run
	const userCpf = generateValidCPF();
	const userEmail = `idempotency_test_${randomString(5)}@test.com`;
	const userPayload = JSON.stringify({
		name: `Idempotency Test User`,
		email: userEmail,
		password: "securepassword123",
		cpf: userCpf,
	});

	// Register
	const registerRes = http.post(`${BASE_URL}/auth/register`, userPayload, {
		headers: { "Content-Type": "application/json" },
	});

	if (registerRes.status !== 201) {
		console.log(`Registration response: ${registerRes.body}`);
	}

	// Login
	const loginRes = http.post(
		`${BASE_URL}/auth/login`,
		JSON.stringify({
			email: userEmail,
			password: "securepassword123",
		}),
		{
			headers: { "Content-Type": "application/json" },
		},
	);

	if (loginRes.status !== 200) {
		throw new Error(`Login failed: ${loginRes.status} ${loginRes.body}`);
	}

	const token = JSON.parse(loginRes.body).data.token;
	return { token: token };
}

export const options = {
	scenarios: {
		idempotency_test: {
			executor: "per-vu-iterations",
			vus: 5,
			iterations: 10,
		},
	},
	thresholds: {
		checks: ["rate==1.0"], // All checks must pass
	},
};

export default function (data) {
	const cpf = generateValidCPF();
	const idempotencyKey = `idem-${randomString(16)}`;
	const requestId = generateUUID();

	const headers = {
		"Content-Type": "application/json",
		"X-Idempotency-Key": idempotencyKey,
		"Authorization": data.token,
	};

	const payload = JSON.stringify({
		key: cpf,
		keyType: "CPF",
		account: {
			participant: "12345678",
			branch: "0001",
			accountNumber: "123456",
			accountType: "CACC",
			openingDate: new Date().toISOString(),
		},
		owner: {
			type: "NATURAL_PERSON",
			taxIdNumber: cpf,
			name: "Idempotency Test",
		},
		reason: "USER_REQUESTED",
		requestId: requestId,
	});

	// First request - should create
	const res1 = http.post(`${BASE_URL}/entries`, payload, { headers });
	check(res1, {
		"first request: status is 201": (r) => r.status === 201,
	});

	sleep(0.2);

	// Second request with SAME idempotency key - should return cached
	const res2 = http.post(`${BASE_URL}/entries`, payload, { headers });
	check(res2, {
		"second request: same status": (r) => r.status === res1.status,
		"second request: same body": (r) => r.body === res1.body,
	});

	// Cleanup - now uses POST /entries/{key}/delete per DICT spec
	const deletePayload = JSON.stringify({
		key: cpf,
		participant: "12345678",
		reason: "USER_REQUESTED",
	});
	http.post(`${BASE_URL}/entries/${cpf}/delete`, deletePayload, { 
		headers: { 
			"Content-Type": "application/json",
			"Authorization": data.token,
		} 
	});

	sleep(0.5);
}
