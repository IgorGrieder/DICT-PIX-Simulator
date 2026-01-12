import http from "k6/http";
import { check, sleep } from "k6";
import { uuidv4 } from "https://jslib.k6.io/k6-utils/1.4.0/index.js";

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

export default function () {
	const cpf = generateValidCPF();

	const headers = {
		"Content-Type": "application/json",
		"x-idempotency-key": uuidv4(),
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
		const getRes = http.get(`${BASE_URL}/entries/${cpf}`);
		check(getRes, {
			"get success": (r) => r.status === 200,
		});

		// Delete
		const delRes = http.del(`${BASE_URL}/entries/${cpf}`);
		check(delRes, {
			"delete success": (r) => r.status === 200 || r.status === 404,
		});
	}

	sleep(0.1);
}
