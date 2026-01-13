/**
 * Precise Benchmark Test for Elysia vs Go Comparison
 *
 * Three-phase test designed to complete in ~60 seconds:
 * - Phase 1 (INSERT): 10,000 records in ~20s
 * - Phase 2 (MIXED): 70% reads, 30% deletes in ~30s
 * - Phase 3 (CLEANUP): Delete remaining records in ~10s
 *
 * Usage:
 *   k6 run benchmark.test.js -e BASE_URL=http://localhost:3000 -e APP=elysia
 *   k6 run benchmark.test.js -e BASE_URL=http://localhost:3000 -e APP=go
 *
 * With Prometheus export:
 *   k6 run --out experimental-prometheus-rw benchmark.test.js -e APP=elysia
 */

import { randomString } from "https://jslib.k6.io/k6-utils/1.4.0/index.js";
import { check } from "k6";
import { Counter, Rate, Trend } from "k6/metrics";
import http from "k6/http";
import { SharedArray } from "k6/data";

// ============================================================================
// Configuration
// ============================================================================

const BASE_URL = __ENV.BASE_URL || "http://localhost:3000";
const APP_NAME = __ENV.APP || "unknown";
const TARGET_INSERTS = 10000;

// Custom metrics with app tag for comparison
const insertDuration = new Trend("insert_duration", true);
const readDuration = new Trend("read_duration", true);
const deleteDuration = new Trend("delete_duration", true);
const insertCounter = new Counter("inserts_total");
const readCounter = new Counter("reads_total");
const deleteCounter = new Counter("deletes_total");
const errorRate = new Rate("error_rate");

// Shared state for tracking inserted CPFs across VUs
const cpfPool = new SharedArray("cpfs", function () {
	const cpfs = [];
	for (let i = 0; i < TARGET_INSERTS + 1000; i++) {
		cpfs.push(generateValidCPF());
	}
	return cpfs;
});

// ============================================================================
// Test Options
// ============================================================================

export const options = {
	scenarios: {
		insert: {
			executor: "constant-vus",
			vus: 50,
			duration: "20s",
			exec: "insertPhase",
			startTime: "0s",
			tags: { phase: "insert", app: APP_NAME },
		},
		mixed: {
			executor: "constant-vus",
			vus: 50,
			duration: "30s",
			exec: "mixedPhase",
			startTime: "20s",
			tags: { phase: "mixed", app: APP_NAME },
		},
		cleanup: {
			executor: "constant-vus",
			vus: 20,
			duration: "10s",
			exec: "cleanupPhase",
			startTime: "50s",
			tags: { phase: "cleanup", app: APP_NAME },
		},
	},
	thresholds: {
		"insert_duration{phase:insert}": ["p(95)<300"],
		"read_duration{phase:mixed}": ["p(95)<100"],
		"delete_duration{phase:mixed}": ["p(95)<200"],
		error_rate: ["rate<0.05"],
		http_req_failed: ["rate<0.05"],
	},
	tags: {
		app: APP_NAME,
		testType: "benchmark",
	},
};

// ============================================================================
// Utility Functions
// ============================================================================

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

function createEntryPayload(cpf) {
	return JSON.stringify({
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
			name: `Benchmark User ${randomString(6)}`,
		},
	});
}

function getHeaders(token) {
	const headers = {
		"Content-Type": "application/json",
		"x-idempotency-key": `k6-${APP_NAME}-${randomString(16)}`,
	};

	if (token) {
		headers["Authorization"] = token;
	}

	return headers;
}

// ============================================================================
// Test Phases
// ============================================================================

export function insertPhase(data) {
	const vuId = __VU;
	const iter = __ITER;

	const cpfIndex = (vuId * 1000 + iter) % cpfPool.length;
	const cpf = cpfPool[cpfIndex];

	const payload = createEntryPayload(cpf);
	const headers = getHeaders(data?.token); // Use optional chaining just in case

	const startTime = Date.now();
	const res = http.post(`${BASE_URL}/entries`, payload, {
		headers,
		tags: { operation: "create", phase: "insert", app: APP_NAME },
	});
	const duration = Date.now() - startTime;

	insertDuration.add(duration, { app: APP_NAME });
	insertCounter.add(1, { app: APP_NAME });

	const success = check(res, {
		"insert: status 201": (r) => r.status === 201,
		"insert: has key": (r) => {
			try {
				return JSON.parse(r.body).key === cpf;
			} catch {
				return false;
			}
		},
	});

	if (!success) {
		errorRate.add(1, { app: APP_NAME, operation: "create" });
	}
}

export function mixedPhase(data) {
	const vuId = __VU;
	const iter = __ITER;

	const cpfIndex =
		(vuId * 500 + iter) % Math.min(cpfPool.length, TARGET_INSERTS);
	const cpf = cpfPool[cpfIndex];
	const headers = getHeaders(data?.token);

	const isRead = Math.random() < 0.7;

	if (isRead) {
		const startTime = Date.now();
		const res = http.get(`${BASE_URL}/entries/${cpf}`, {
			headers,
			tags: { operation: "read", phase: "mixed", app: APP_NAME },
		});
		const duration = Date.now() - startTime;

		readDuration.add(duration, { app: APP_NAME });
		readCounter.add(1, { app: APP_NAME });

		const success = check(res, {
			"read: status 200 or 404": (r) => r.status === 200 || r.status === 404,
		});

		if (!success) {
			errorRate.add(1, { app: APP_NAME, operation: "read" });
		}
	} else {
		const startTime = Date.now();
		const res = http.del(`${BASE_URL}/entries/${cpf}`, null, {
			headers,
			tags: { operation: "delete", phase: "mixed", app: APP_NAME },
		});
		const duration = Date.now() - startTime;

		deleteDuration.add(duration, { app: APP_NAME });
		deleteCounter.add(1, { app: APP_NAME });

		const success = check(res, {
			"delete: status 200 or 404": (r) => r.status === 200 || r.status === 404,
		});

		if (!success) {
			errorRate.add(1, { app: APP_NAME, operation: "delete" });
		}
	}
}

export function cleanupPhase(data) {
	const vuId = __VU;
	const iter = __ITER;

	const cpfIndex =
		(vuId * 200 + iter) % Math.min(cpfPool.length, TARGET_INSERTS);
	const cpf = cpfPool[cpfIndex];
	const headers = getHeaders(data?.token);

	const res = http.del(`${BASE_URL}/entries/${cpf}`, null, {
		headers,
		tags: { operation: "cleanup", phase: "cleanup", app: APP_NAME },
	});

	check(res, {
		"cleanup: status 200 or 404": (r) => r.status === 200 || r.status === 404,
	});
}

// ============================================================================
// Lifecycle Hooks
// ============================================================================

export function setup() {
	console.log(`\n========================================`);
	console.log(`  Benchmark Test: ${APP_NAME.toUpperCase()}`);
	console.log(`  Target URL: ${BASE_URL}`);
	console.log(`  Target Inserts: ${TARGET_INSERTS}`);
	console.log(`========================================\n`);

	// 1. Health Check
	const healthCheck = http.get(`${BASE_URL}/health`);
	if (healthCheck.status !== 200) {
		throw new Error(`Health check failed: ${healthCheck.status}`);
	}
	console.log(`Health check passed.`);

	// 2. Authentication
	// Create a unique user for this benchmark run
	const userCpf = generateValidCPF();
	const userEmail = `admin_${randomString(5)}@benchmark.com`;
	const userPayload = JSON.stringify({
		name: `Benchmark Admin`,
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
	} else {
		console.log(`User registered successfully.`);
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

	const token = JSON.parse(loginRes.body).token;
	console.log(`Authentication successful. Token obtained.`);

	return {
		startTime: Date.now(),
		token: token,
	};
}

export function teardown(data) {
	const totalTime = (Date.now() - data.startTime) / 1000;
	console.log(`\n========================================`);
	console.log(`  Benchmark Complete: ${APP_NAME.toUpperCase()}`);
	console.log(`  Total Time: ${totalTime.toFixed(2)}s`);
	console.log(`========================================\n`);
}
