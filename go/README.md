# Pix DICT Simulator (Go)

A high-performance Go implementation of the **Directory of Transactional Identifiers (DICT)**, simulating the core engine behind the Brazilian Pix ecosystem.

## Features

- **Fast Key Lookups:** MongoDB Single Field Indexing for sub-second responses
- **Idempotency:** `X-Idempotency-Key` header support for safe retries
- **Validation:** Módulo 11 for CPF/CNPJ, regex for Email/Phone, UUID v4 for EVP
- **Observability:** OpenTelemetry integration with Jaeger
- **Authentication:** JWT-based authentication with bcrypt password hashing
- **Rate Limiting:** Token bucket algorithm using Redis
- **Type Safety:** Strongly typed Go structs

## Tech Stack

- **Runtime:** Go 1.23+
- **HTTP Server:** `net/http` (standard library)
- **Database:** MongoDB via `go.mongodb.org/mongo-driver`
- **Cache:** Redis via `github.com/redis/go-redis/v9`
- **Auth:** JWT via `github.com/golang-jwt/jwt/v5`
- **Password Hashing:** bcrypt via `golang.org/x/crypto/bcrypt`
- **Observability:** OpenTelemetry with Jaeger

## Quick Start

### With Docker (Recommended)

```bash
docker-compose up --build
```

Services available:
- **API:** http://localhost:3000
- **Jaeger UI:** http://localhost:16686

### Local Development

```bash
# Start MongoDB, Redis, and Jaeger (required)
docker run -d -p 27017:27017 mongo:7.0
docker run -d -p 6379:6379 redis:7.2-alpine
docker run -d -p 16686:16686 -p 4318:4318 -e COLLECTOR_OTLP_ENABLED=true jaegertracing/jaeger:2.6.0

# Set environment variables
export JWT_SECRET=your-secret-key

# Run the server
go run ./cmd/server
```

## API Endpoints

### Authentication

#### Register
```bash
curl -X POST http://localhost:3000/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123",
    "name": "John Doe"
  }'
```

#### Login
```bash
curl -X POST http://localhost:3000/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123"
  }'
```

### Entries (Requires Authentication)

#### Create Entry
```bash
curl -X POST http://localhost:3000/entries \
  -H "Content-Type: application/json" \
  -H "Authorization: <your-jwt-token>" \
  -H "X-Idempotency-Key: unique-request-id" \
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
    }
  }'
```

#### Get Entry
```bash
curl http://localhost:3000/entries/12345678909 \
  -H "Authorization: <your-jwt-token>"
```

#### Delete Entry
```bash
curl -X DELETE http://localhost:3000/entries/12345678909 \
  -H "Authorization: <your-jwt-token>"
```

### Health Check
```bash
curl http://localhost:3000/health
```

## Key Types

| Type | Format | Validation |
|------|--------|------------|
| CPF | 11 digits | Módulo 11 |
| CNPJ | 14 digits | Módulo 11 |
| EMAIL | RFC 5322 | Regex (max 77 chars) |
| PHONE | +55XXXXXXXXXXX | +55 prefix + 10-11 digits |
| EVP | UUID v4 | UUID format |

## Project Structure

```
go/
├── cmd/
│   └── server/
│       └── main.go           # Entry point with HTTP router
├── internal/
│   ├── config/
│   │   └── env.go            # Environment configuration
│   ├── db/
│   │   ├── mongo.go          # MongoDB connection
│   │   └── redis.go          # Redis connection
│   ├── middleware/
│   │   ├── auth.go           # JWT authentication
│   │   ├── rate_limiter.go   # Rate limiting
│   │   └── idempotency.go    # Idempotency support
│   ├── models/
│   │   ├── user.go           # User model
│   │   ├── entry.go          # Entry model
│   │   └── idempotency.go    # Idempotency record
│   └── modules/
│       ├── auth/
│       │   └── handler.go    # Auth handlers
│       └── entries/
│           ├── handler.go    # Entry handlers
│           └── validator.go  # Key validation
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
├── .env.example
└── README.md
```

## Observability

The app uses OpenTelemetry for distributed tracing. Traces are exported to Jaeger via OTLP.

### Viewing Traces

1. Start the stack with `docker-compose up`
2. Make some API requests
3. Open Jaeger UI at http://localhost:16686
4. Select "dict-simulator" service

Each request creates spans for:
- HTTP handlers (`auth.register`, `entries.create`, etc.)
- Middleware operations
- Database operations

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| PORT | 3000 | Server port |
| MONGODB_URI | mongodb://localhost:27017/dict | MongoDB connection string |
| REDIS_URI | redis://localhost:6379 | Redis connection string |
| JWT_SECRET | (required) | Secret key for JWT signing |
| OTEL_EXPORTER_OTLP_ENDPOINT | http://localhost:4318/v1/traces | OpenTelemetry endpoint |
| RATE_LIMIT_BUCKET_SIZE | 60 | Max requests per window |
| RATE_LIMIT_REFILL_SECONDS | 60 | Rate limit window in seconds |

## Development

```bash
# Build
go build -o server ./cmd/server

# Run tests
go test ./...

# Format code
go fmt ./...

# Lint (requires golangci-lint)
golangci-lint run
```

## License

MIT
