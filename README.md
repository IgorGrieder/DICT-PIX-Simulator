# Pix DICT Simulator

A high-performance, production-grade implementation of the **Directory of Transactional Identifiers (DICT)** — the core address resolution engine for the Brazilian **Pix** payment ecosystem.

Built with **Go**, this project demonstrates robust distributed systems patterns including distributed tracing, idempotency, rate limiting, and hexagonal architecture.

## 🚀 Key Features

- **High Performance:** Optimized for sub-second responses using **MongoDB** with single-field indexing.
- **Resilience & Safety:**
  - **Idempotency:** Native support for `X-Idempotency-Key` headers to ensure safe retries.
  - **Rate Limiting:** Token bucket algorithm (via **Redis**) with sophisticated policies (e.g., stricter limits for 404s to prevent enumeration attacks).
  - **Validation:** Strict adherence to BCB standards (Modulo 11 for CPF/CNPJ, regex for Email/Phone).
- **Observability:**
  - **Distributed Tracing:** Full OpenTelemetry integration exporting to **Jaeger**.
  - **Metrics:** Prometheus metrics exposed for scraping.
  - **Dashboards:** Pre-configured **Grafana** dashboards.
- **Security:** JWT-based authentication with bcrypt password hashing.

## 🛠 Tech Stack

- **Core:** Go 1.23+
- **Database:** MongoDB 7.0 (Mongoose/Driver)
- **Cache & Limits:** Redis 7.2 (Alpine)
- **Observability:** OpenTelemetry, Jaeger, Prometheus, Grafana
- **Testing:** k6 (Load & Stress Testing)
- **Infrastructure:** Docker Compose

## ⚡️ Quick Start

The easiest way to run the full stack is using Docker Compose.

### Prerequisites
- Docker & Docker Compose
- (Optional) Go 1.23+ for local development
- (Optional) k6 for running load tests

### Running the Stack

```bash
# Start all services (API, Mongo, Redis, Jaeger, Prometheus, Grafana)
docker-compose up -d --build

# View logs
docker-compose logs -f app
```

### Accessing Services

| Service | URL | Description |
|---------|-----|-------------|
| **API** | `http://localhost:3000` | Main DICT HTTP Service |
| **Grafana** | `http://localhost:3001` | Dashboards (User/Pass: `admin`/`admin`) |
| **Jaeger** | `http://localhost:16686` | Distributed Tracing UI |
| **Prometheus** | `http://localhost:9090` | Metrics Browser |

## 📡 API Reference

### Authentication

**Register**
```bash
curl -X POST http://localhost:3000/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email": "dev@example.com", "password": "secure123", "name": "Developer"}'
```

**Login**
```bash
curl -X POST http://localhost:3000/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "dev@example.com", "password": "secure123"}'
```
*Returns a JWT token required for all entry operations.*

### Entries

**Create Entry**
```bash
curl -X POST http://localhost:3000/entries \
  -H "Authorization: Bearer <YOUR_JWT>" \
  -H "X-Idempotency-Key: <UNIQUE_UUID>" \
  -H "Content-Type: application/json" \
  -d '{
    "key": "12345678909",
    "keyType": "CPF",
    "account": {
      "participant": "12345678",
      "branch": "0001",
      "accountNumber": "123456",
      "accountType": "CACC"
    },
    "owner": {
      "type": "NATURAL_PERSON",
      "taxIdNumber": "12345678909",
      "name": "John Doe"
    },
    "reason": "USER_REQUESTED",
    "requestId": "550e8400-e29b-41d4-a716-446655440000"
  }'
```

**Get Entry**
```bash
curl http://localhost:3000/entries/12345678909 \
  -H "Authorization: Bearer <YOUR_JWT>"
```

**Delete Entry**
```bash
curl -X DELETE http://localhost:3000/entries/12345678909 \
  -H "Authorization: Bearer <YOUR_JWT>"
```

### Key Types

| Type | Format | Notes |
|------|--------|-------|
| `CPF` | 11 digits | Validated via Modulo 11 |
| `CNPJ` | 14 digits | Validated via Modulo 11 |
| `EMAIL` | RFC 5322 | Max 77 chars |
| `PHONE` | +55XXXXXXXXXXX | E.164 format (Brazil only) |
| `EVP` | UUID v4 | Random key |

## 🛡 Rate Limiting Policies

The system implements specific policies compatible with DICT standards:

- **Writes (Create/Delete):** 1200 req/min (Bucket: 36,000)
- **Updates:** 600 req/min (Bucket: 600)
- **Reads (Anti-Scan):** 2 req/min (Bucket: 50).
  - *Note:* A `404 Not Found` costs **3 tokens**, penalizing key enumeration attempts.

## 🧪 Performance Testing

Load tests are located in the `k6/` directory.

```bash
# Run a standard CRUD flow test
k6 run k6/entries.test.js

# Run a stress test (100 concurrent users)
k6 run k6/stress.test.js

# Verify idempotency behavior
k6 run k6/idempotency.test.js
```

## 📂 Project Structure

```
.
├── go/                     # Backend Source Code
│   ├── cmd/server/         # Application entry point
│   ├── internal/           # Private application code
│   │   ├── config/         # Environment configuration
│   │   ├── db/             # Mongo & Redis adapters
│   │   ├── middleware/     # Auth, Rate Limit, Idempotency
│   │   ├── models/         # Data structures (Entry, Account, Owner)
│   │   └── modules/        # Domain logic (Handlers, Services)
│   └── monitoring/         # Prometheus & Grafana configs
└── k6/                     # Load testing scripts
```

## License

MIT
