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

	// First check digit
	let sum = 0;
	for (let i = 0; i < 9; i++) {
		sum += digits[i] * (10 - i);
	}
	let remainder = (sum * 10) % 11;
	if (remainder === 10) remainder = 0;
	digits.push(remainder);

	// Second check digit
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

// Test configuration
export const options = {
	scenarios: {
		// Smoke test
		smoke: {
			executor: "constant-vus",
			vus: 1,
			duration: "10s",
			startTime: "0s",
		},
		// Load test
		load: {
			executor: "ramping-vus",
			startVUs: 0,
			stages: [
				{ duration: "30s", target: 10 },
				{ duration: "1m", target: 10 },
				{ duration: "30s", target: 0 },
			],
			startTime: "15s",
		},
	},
	thresholds: {
		http_req_duration: ["p(95)<500"], // 95% of requests should be below 500ms
		http_req_failed: ["rate<0.01"], // Less than 1% failure rate
	},
};

export default function () {
	const cpf = generateValidCPF();
	const idempotencyKey = `k6-${randomString(16)}`;
	const requestId = generateUUID();
	const correlationId = generateUUID();

	const headers = {
		"Content-Type": "application/json",
		"X-Idempotency-Key": idempotencyKey,
		"X-Correlation-Id": correlationId,
	};

	// Create Entry payload - now includes openingDate and requestId per DICT spec
	const payload = JSON.stringify({
		key: cpf,
		keyType: "CPF",
		account: {
			participant: "12345678",
			branch: "0001",
			accountNumber: `${Math.floor(Math.random() * 1000000)}`,
			accountType: "CACC",
			openingDate: new Date().toISOString(), // NEW: required per DICT spec
		},
		owner: {
			type: "NATURAL_PERSON",
			taxIdNumber: cpf,
			name: `Test User ${randomString(8)}`,
		},
		reason: "USER_REQUESTED",      // NEW: required per DICT spec
		requestId: requestId,           // NEW: required per DICT spec (idempotency)
	});

	// Create Entry
	const createRes = http.post(`${BASE_URL}/entries`, payload, { headers });
	check(createRes, {
		"create: status is 201": (r) => r.status === 201,
		"create: has data.key": (r) => {
			const body = JSON.parse(r.body);
			return body.data && body.data.key === cpf;
		},
		"create: has correlationId": (r) => {
			const body = JSON.parse(r.body);
			return body.correlationId === correlationId;
		},
		"create: has responseTime": (r) => {
			const body = JSON.parse(r.body);
			return body.responseTime !== undefined;
		},
	});

	sleep(0.5);

	// Get Entry
	const getRes = http.get(`${BASE_URL}/entries/${cpf}`, { headers: { "X-Correlation-Id": correlationId } });
	check(getRes, {
		"get: status is 200": (r) => r.status === 200,
		"get: correct key": (r) => {
			const body = JSON.parse(r.body);
			return body.data && body.data.key === cpf;
		},
		"get: has keyOwnershipDate": (r) => {
			const body = JSON.parse(r.body);
			return body.data && body.data.keyOwnershipDate !== undefined;
		},
	});

	sleep(0.5);

	// Delete Entry - now uses POST /entries/{key}/delete per DICT spec
	const deletePayload = JSON.stringify({
		key: cpf,
		participant: "12345678",
		reason: "USER_REQUESTED",
	});
	const deleteRes = http.post(`${BASE_URL}/entries/${cpf}/delete`, deletePayload, { 
		headers: { 
			"Content-Type": "application/json",
			"X-Correlation-Id": correlationId,
		} 
	});
	check(deleteRes, {
		"delete: status is 200": (r) => r.status === 200,
		"delete: has data.key": (r) => {
			const body = JSON.parse(r.body);
			return body.data && body.data.key === cpf;
		},
	});

	sleep(1);
}
