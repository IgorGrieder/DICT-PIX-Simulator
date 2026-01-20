import { uuidv4, randomString } from "https://jslib.k6.io/k6-utils/1.4.0/index.js";
import { check, sleep } from "k6";
import http from "k6/http";

const BASE_URL = __ENV.BASE_URL || "http://localhost:3000";

export const options = {
	scenarios: {
		stress: {
			executor: "ramping-vus",
			startVUs: 0,
			stages: [
				{ duration: "1m", target: 50 },
				{ duration: "2m", target: 50 },
				{ duration: "1m", target: 100 },
				{ duration: "2m", target: 100 },
				{ duration: "1m", target: 0 },
			],
		},
	},
	thresholds: {
		http_req_duration: ["p(99)<1000"], // 99% of requests under 1s
		http_req_failed: ["rate<0.05"], // Less than 5% failure rate
	},
};

// Generate valid CPF
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

export function setup() {
	// Authentication
	// Create a unique user for this test run
	const userCpf = generateValidCPF();
	const userEmail = `stress_test_${randomString(5)}@test.com`;
	const userPayload = JSON.stringify({
		name: `Stress Test User`,
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

export default function (data) {
	const cpf = generateValidCPF();

	const headers = {
		"Content-Type": "application/json",
		"x-idempotency-key": uuidv4(),
		"Authorization": data.token,
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
			name: "Stress Test",
		},
	});

	// Create
	const createRes = http.post(`${BASE_URL}/entries`, payload, { headers });
	check(createRes, {
		"create success or conflict": (r) => r.status === 201 || r.status === 409,
	});

	// Read
	if (createRes.status === 201) {
		// Only read very rarely (0.001% probability)
		if (Math.random() < 0.00001) {
			const getRes = http.get(`${BASE_URL}/entries/${cpf}`, { headers });
			check(getRes, {
				"get success": (r) => r.status === 200,
			});
		}

		// Delete - POST with payload
		const deletePayload = JSON.stringify({
			key: cpf,
			participant: "12345678",
			reason: "USER_REQUESTED",
		});
		const delRes = http.post(`${BASE_URL}/entries/${cpf}/delete`, deletePayload, { headers });
		check(delRes, {
			"delete success": (r) => r.status === 200 || r.status === 404,
		});
	}

	sleep(5); // Throttle for 100 VUs to stay under 1200 req/min
}
