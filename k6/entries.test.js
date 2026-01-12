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

	const headers = {
		"Content-Type": "application/json",
		"x-idempotency-key": idempotencyKey,
	};

	const payload = JSON.stringify({
		key: cpf,
		keyType: "CPF",
		account: {
			participant: "12345678",
			branch: "0001",
			accountNumber: `${Math.floor(Math.random() * 1000000)}`,
			accountType: "CACC",
		},
		owner: {
			type: "NATURAL_PERSON",
			taxIdNumber: cpf,
			name: `Test User ${randomString(8)}`,
		},
	});

	// Create Entry
	const createRes = http.post(`${BASE_URL}/entries`, payload, { headers });
	check(createRes, {
		"create: status is 201": (r) => r.status === 201,
		"create: has key": (r) => JSON.parse(r.body).key === cpf,
	});

	sleep(0.5);

	// Get Entry
	const getRes = http.get(`${BASE_URL}/entries/${cpf}`);
	check(getRes, {
		"get: status is 200": (r) => r.status === 200,
		"get: correct key": (r) => JSON.parse(r.body).key === cpf,
	});

	sleep(0.5);

	// Delete Entry
	const deleteRes = http.del(`${BASE_URL}/entries/${cpf}`);
	check(deleteRes, {
		"delete: status is 200": (r) => r.status === 200,
	});

	sleep(1);
}
