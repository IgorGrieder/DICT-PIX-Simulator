# DICT Simulator API - Architecture Documentation

## Overview

This is a **DICT Simulator API** - a simulated implementation of the Brazilian Central Bank's **DICT (Diretorio de Identificadores de Contas Transacionais)** system for managing **Pix keys**.

DICT is the authoritative directory that maps Pix keys (CPF, CNPJ, email, phone, or EVP) to bank account information, enabling instant payments in Brazil's Pix ecosystem.

---

## Business Domain

### What is DICT?

DICT is a centralized directory maintained by the Central Bank of Brazil (BCB) that:

- Stores Pix key registrations linking keys to bank accounts
- Enables key lookup for instant payment routing
- Enforces uniqueness of keys across all financial institutions
- Provides anti-fraud protections via rate limiting

### Core Business Operations

| Operation        | Description                                                    |
| ---------------- | -------------------------------------------------------------- |
| **Create Entry** | Register a new Pix key in the directory                        |
| **Get Entry**    | Lookup a Pix key to find associated account info               |
| **Update Entry** | Modify account info or owner name (EVP keys cannot be updated) |
| **Delete Entry** | Remove a Pix key from the directory                            |

---

## Database Schema

### MongoDB Database: `dict`

#### Collection: `entries`

Stores Pix key registrations.

```javascript
{
  "_id": ObjectId,
  "key": String,              // The Pix key value (unique)
  "keyType": String,          // "CPF" | "CNPJ" | "EMAIL" | "PHONE" | "EVP"
  "account": {
    "participant": String,    // 8-digit ISPB code (bank identifier)
    "branch": String,         // 4-digit branch number
    "accountNumber": String,  // Account number
    "accountType": String,    // "CACC" | "SVGS" | "SLRY"
    "openingDate": Date       // Account opening date
  },
  "owner": {
    "type": String,           // "NATURAL_PERSON" | "LEGAL_PERSON"
    "taxIdNumber": String,    // CPF (11 digits) or CNPJ (14 digits)
    "name": String,           // Owner's name
    "tradeName": String       // Optional: trade name for LEGAL_PERSON
  },
  "createdAt": Date,
  "updatedAt": Date,
  "keyOwnershipDate": Date    // When ownership was established
}
```

#### Collection: `users`

Stores API users for authentication.

```javascript
{
  "_id": ObjectId,
  "email": String,            // Unique email
  "password": String,         // bcrypt hashed password
  "name": String,
  "createdAt": Date,
  "updatedAt": Date
}
```

**Indexes:**

- `{ email: 1 }` - Unique index for login

---

#### Collection: `idempotency`

Stores idempotent request responses for deduplication.

```javascript
{
  "key": String,              // Idempotency key from header
  "response": String,         // Cached JSON response body
  "statusCode": Number,       // HTTP status code
  "createdAt": Date           // TTL: auto-expires after 24 hours
}
```

---

### Redis (Rate Limiting)

Token bucket state per policy and participant.

**Key Pattern:** `rate_limit:{policy}:{identifier}:tokens`
**Key Pattern:** `rate_limit:{policy}:{identifier}:last_refill`

Example:

```
rate_limit:ENTRIES_WRITE:12345678:tokens = "35999"
rate_limit:ENTRIES_WRITE:12345678:last_refill = "1737312000"
```

---

## API Routes

### Public Routes (No Authentication)

| Method | Path             | Handler                  | Description        |
| ------ | ---------------- | ------------------------ | ------------------ |
| `GET`  | `/health`        | `health.Handler.Health`  | Health check       |
| `GET`  | `/metrics`       | `health.Handler.Metrics` | Prometheus metrics |
| `GET`  | `/swagger/*`     | Swagger UI               | API documentation  |
| `POST` | `/auth/register` | `auth.Handler.Register`  | User registration  |
| `POST` | `/auth/login`    | `auth.Handler.Login`     | User login         |

### Protected Routes (JWT Required)

| Method | Path                    | Handler                  | Middleware Chain                        |
| ------ | ----------------------- | ------------------------ | --------------------------------------- |
| `POST` | `/entries`              | `entries.Handler.Create` | Auth -> RateLimit(WRITE) -> Idempotency |
| `GET`  | `/entries/{key}`        | `entries.Handler.Get`    | Auth -> RateLimit(READ_ANTISCAN)        |
| `PUT`  | `/entries/{key}`        | `entries.Handler.Update` | Auth -> RateLimit(UPDATE)               |
| `POST` | `/entries/{key}/delete` | `entries.Handler.Delete` | Auth -> RateLimit(WRITE)                |

---

## Request/Response Flow

### Middleware Chain (Order of Execution)

```
Request -> OpenTelemetry Tracing
        -> Metrics Recording
        -> Request Logging
        -> CORS Headers
        -> Route Handler
           -> JWT Authentication (protected routes)
           -> Rate Limiting (per policy)
           -> Idempotency Check (POST /entries only)
           -> Business Logic Handler
        <- Response
```

### API Response Format (DICT-Compliant)

**Success Response:**

```json
{
  "responseTime": "2024-01-15T10:30:00Z",
  "correlationId": "550e8400-e29b-41d4-a716-446655440000",
  "code": "ENTRY_CREATED",
  "data": { ... }
}
```

**Error Response:**

```json
{
  "responseTime": "2024-01-15T10:30:00Z",
  "correlationId": "550e8400-e29b-41d4-a716-446655440000",
  "error": "KEY_ALREADY_EXISTS",
  "message": "This key is already registered in the directory"
}
```

---

## Rate Limiting (DICT Spec Compliance)

Uses **Token Bucket Algorithm** with Redis for distributed rate limiting.

### Rate Limit Policies

| Policy Name                         | Applies To     | Refill Rate | Bucket Size | Success Cost | 404 Cost |
| ----------------------------------- | -------------- | ----------- | ----------- | ------------ | -------- |
| `ENTRIES_WRITE`                     | Create, Delete | 1200/min    | 36,000      | 1            | 1        |
| `ENTRIES_UPDATE`                    | Update         | 600/min     | 600         | 1            | 1        |
| `ENTRIES_READ_PARTICIPANT_ANTISCAN` | Get (lookup)   | 2/min       | 50          | 1            | **3**    |

**Anti-Scan Protection:** The READ policy penalizes 404 responses with 3x token cost to prevent enumeration attacks.

**5xx Errors:** Token deduction is skipped on server errors (fail-open for reliability).

### Rate Limit Headers

```http
X-RateLimit-Limit: 50
X-RateLimit-Remaining: 47
X-RateLimit-Reset: 1737312060
X-RateLimit-Policy: ENTRIES_READ_PARTICIPANT_ANTISCAN
```

---

## Pix Key Validation

### Key Types and Formats

| Key Type | Format                             | Validation                       |
| -------- | ---------------------------------- | -------------------------------- |
| `CPF`    | 11 digits                          | Modulo 11 algorithm              |
| `CNPJ`   | 14 digits                          | Modulo 11 algorithm              |
| `EMAIL`  | Lowercase, max 77 chars            | DICT regex pattern               |
| `PHONE`  | E.164 format: `+{country}{number}` | `^\+[1-9]\d{6,14}$`              |
| `EVP`    | UUID v4 lowercase                  | `^[0-9a-f]{8}-...-[0-9a-f]{12}$` |

### Validation Examples

```
CPF:    "12345678909"                     (valid with check digits)
CNPJ:   "11222333000181"                  (valid with check digits)
EMAIL:  "user@example.com"                (must be lowercase)
PHONE:  "+5511999999999"                  (E.164 international)
EVP:    "550e8400-e29b-41d4-a716-446655440000"
```

---

## Business Rules

### Entry Creation (`POST /entries`)

1. Validate request body schema
2. Validate key format matches keyType
3. Check if key already exists -> 409 Conflict
4. Create entry with current timestamp as ownership date

### Entry Lookup (`GET /entries/{key}`)

1. Extract key from path
2. Find entry by key -> 404 if not found
3. Return entry data

### Entry Update (`PUT /entries/{key}`)

1. Validate request body
2. Key in path must match key in body (if provided)
3. EVP keys cannot be updated -> 400 Bad Request
4. Only these fields can be updated:
   - `account.*` (all account fields)
   - `owner.name`
   - `owner.tradeName`
5. `owner.taxIdNumber` is immutable

### Entry Deletion (`POST /entries/{key}/delete`)

1. Extract key from path and participant from body
2. Validate participant in request matches entry's participant -> 403 Forbidden
3. Delete entry and return confirmation

### Valid Reasons

**Create:** `USER_REQUESTED`, `RECONCILIATION`

**Update:** `USER_REQUESTED`, `BRANCH_TRANSFER`, `RECONCILIATION`, `RFB_VALIDATION`

**Delete:** `USER_REQUESTED`, `ACCOUNT_CLOSURE`, `RECONCILIATION`, `FRAUD`, `RFB_VALIDATION`

---

## Authentication

### JWT Token Structure

```json
{
  "user_id": "507f1f77bcf86cd799439011",
  "email": "user@example.com",
  "name": "John Doe",
  "exp": 1737916800,
  "iat": 1737312000
}
```

**Token Lifetime:** 7 days

**Header Format:** `Authorization: Bearer <token>`

### Auth Flow

1. `POST /auth/register` - Create user, return JWT
2. `POST /auth/login` - Validate credentials, return JWT
3. Protected endpoints extract `X-User-Id` from validated token

---

## Idempotency

Applied to `POST /entries` (entry creation).

### Flow

1. Check `X-Idempotency-Key` header
2. If key exists in database, return cached response
3. If new key, atomically claim it (prevents race conditions)
4. Process request and cache response
5. Records expire after 24 hours

---

## Observability

### Telemetry Stack

- **Tracing:** OpenTelemetry with OTLP exporter
- **Metrics:** Prometheus via `/metrics` endpoint
- **Logging:** Zap logger with OTEL integration

### Prometheus Metrics

| Metric                          | Type      | Labels               |
| ------------------------------- | --------- | -------------------- |
| `http_requests_total`           | Counter   | method, path, status |
| `http_request_duration_seconds` | Histogram | method, path, status |

### Trace Span Names

| Route Pattern                | Span Name        |
| ---------------------------- | ---------------- |
| `GET /health`                | `health`         |
| `POST /auth/register`        | `auth.register`  |
| `POST /auth/login`           | `auth.login`     |
| `POST /entries`              | `entries.create` |
| `GET /entries/{key}`         | `entries.get`    |
| `PUT /entries/{key}`         | `entries.update` |
| `POST /entries/{key}/delete` | `entries.delete` |

---

## Configuration

### Environment Variables

| Variable                      | Required | Default                         | Description                   |
| ----------------------------- | -------- | ------------------------------- | ----------------------------- |
| `JWT_SECRET`                  | Yes      | -                               | Secret for signing JWT tokens |
| `PORT`                        | No       | 3000                            | HTTP server port              |
| `GO_ENV`                      | No       | development                     | Environment name              |
| `MONGODB_URI`                 | No       | mongodb://localhost:27017/dict  | MongoDB connection string     |
| `REDIS_URI`                   | No       | redis://localhost:6379          | Redis connection string       |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | No       | http://localhost:4318/v1/traces | OTEL Traces collector endpoint       |
| `RATE_LIMIT_ENABLED`          | No       | true                            | Enable/disable rate limiting  |

---

## Error Codes

### Common Errors

| Code                | HTTP Status | Description                                  |
| ------------------- | ----------- | -------------------------------------------- |
| `INVALID_REQUEST`   | 400         | Malformed request body or validation failure |
| `UNAUTHORIZED`      | 401         | Missing or invalid authentication            |
| `FORBIDDEN`         | 403         | Participant mismatch                         |
| `INTERNAL_ERROR`    | 500         | Server error                                 |
| `TOO_MANY_REQUESTS` | 429         | Rate limit exceeded                          |

### Entry-Specific Errors

| Code                 | HTTP Status | Description                |
| -------------------- | ----------- | -------------------------- |
| `ENTRY_NOT_FOUND`    | 404         | Key not found in directory |
| `KEY_ALREADY_EXISTS` | 409         | Key already registered     |
| `INVALID_OPERATION`  | 400         | EVP key update attempt     |

### Auth Errors

| Code                  | HTTP Status | Description              |
| --------------------- | ----------- | ------------------------ |
| `INVALID_CREDENTIALS` | 401         | Wrong email or password  |
| `USER_ALREADY_EXISTS` | 409         | Email already registered |

---

## Success Codes

| Code              | HTTP Status | Description                |
| ----------------- | ----------- | -------------------------- |
| `ENTRY_CREATED`   | 201         | Entry successfully created |
| `ENTRY_FOUND`     | 200         | Entry retrieved            |
| `ENTRY_UPDATED`   | 200         | Entry updated              |
| `ENTRY_DELETED`   | 200         | Entry deleted              |
| `USER_REGISTERED` | 201         | User registered            |
| `LOGIN_SUCCESS`   | 200         | Login successful           |

---

## Testing

### Run Tests

```bash
cd go
go test ./... -v
```

### Integration Tests

Located in `internal/integration/`:

- `entries_test.go` - Full CRUD flow tests
- `setup_test.go` - Test infrastructure setup

---

## Dependencies

### Core

- `net/http` - Go 1.22+ HTTP routing with pattern matching
- `go.mongodb.org/mongo-driver` - MongoDB driver
- `github.com/redis/go-redis/v9` - Redis client
- `github.com/golang-jwt/jwt/v5` - JWT handling
- `github.com/go-playground/validator/v10` - Struct validation
- `golang.org/x/crypto/bcrypt` - Password hashing

### Observability

- `go.opentelemetry.io/otel` - Tracing
- `go.uber.org/zap` - Structured logging
- `github.com/prometheus/client_golang` - Metrics

### Documentation

- `github.com/swaggo/http-swagger` - Swagger UI
- `github.com/swaggo/swag` - Swagger generation

---

## API Examples

### Register User

```bash
curl -X POST http://localhost:3000/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123",
    "name": "John Doe"
  }'
```

### Login

```bash
curl -X POST http://localhost:3000/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123"
  }'
```

### Create Entry

```bash
curl -X POST http://localhost:3000/entries \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -H "X-Idempotency-Key: unique-request-id" \
  -H "X-Participant-Id: 12345678" \
  -d '{
    "key": "+5511999999999",
    "keyType": "PHONE",
    "account": {
      "participant": "12345678",
      "branch": "0001",
      "accountNumber": "123456789",
      "accountType": "CACC",
      "openingDate": "2023-01-15T00:00:00Z"
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

### Get Entry

```bash
curl http://localhost:3000/entries/+5511999999999 \
  -H "Authorization: Bearer <token>" \
  -H "X-Participant-Id: 12345678"
```

### Delete Entry

```bash
curl -X POST http://localhost:3000/entries/+5511999999999/delete \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "key": "+5511999999999",
    "participant": "12345678",
    "reason": "USER_REQUESTED"
  }'
```
