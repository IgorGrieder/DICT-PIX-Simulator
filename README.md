# Pix DICT Simulator

A high-performance Bun implementation of the **Directory of Transactional Identifiers (DICT)**, simulating the core engine behind the Brazilian Pix ecosystem.

## 🚀 Features

- **Fast Key Lookups:** MongoDB Single Field Indexing for sub-second responses
- **Idempotency:** `x-idempotency-key` header support for safe retries
- **Validation:** Módulo 11 for CPF/CNPJ, regex for Email/Phone, UUID v4 for EVP
- **Observability:** OpenTelemetry integration with Elysia plugin
- **Type Safety:** Zod schemas with Elysia's Standard Schema support

## 🛠 Tech Stack

- **Runtime:** [Bun](https://bun.sh) + [Elysia](https://elysiajs.com)
- **Database:** MongoDB via Mongoose
- **Validation:** Zod (via Elysia Standard Schema)
- **Observability:** OpenTelemetry with Jaeger
- **Linting:** Biome

## 🏃 Quick Start

### With Docker (Recommended)

```bash
docker-compose up --build
```

Services available:
- **API:** http://localhost:3000
- **Jaeger UI:** http://localhost:16686

### Local Development

```bash
# Install dependencies
bun install

# Start MongoDB and Jaeger (required)
docker run -d -p 27017:27017 mongo:7.0
docker run -d -p 16686:16686 -p 4318:4318 -e COLLECTOR_OTLP_ENABLED=true jaegertracing/jaeger:2.6.0

# Run development server
bun run dev
```

## 📡 API Endpoints

### Create Entry

```bash
curl -X POST http://localhost:3000/entries \
  -H "Content-Type: application/json" \
  -H "x-idempotency-key: unique-request-id" \
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

### Get Entry

```bash
curl http://localhost:3000/entries/12345678909
```

### Delete Entry

```bash
curl -X DELETE http://localhost:3000/entries/12345678909
```

### Health Check

```bash
curl http://localhost:3000/health
```

## 🔑 Key Types

| Type | Format | Validation |
|------|--------|------------|
| CPF | 11 digits | Módulo 11 |
| CNPJ | 14 digits | Módulo 11 |
| EMAIL | RFC 5322 | Regex (max 77 chars) |
| PHONE | +55XXXXXXXXXXX | +55 prefix + 10-11 digits |
| EVP | UUID v4 | UUID format |

## 📁 Project Structure

```
src/
├── index.ts           # App entry point with OpenTelemetry
├── db.ts              # MongoDB connection
├── handlers/
│   └── entries.ts     # Request handlers with tracing
├── models/
│   ├── entry.ts       # Entry schema
│   └── idempotency.ts # Idempotency tracking
├── routes/
│   └── entries.ts     # API routes with Zod validation
├── utils/
│   ├── validators.ts  # Key validation (CPF/CNPJ/etc.)
│   └── idempotency.ts # Idempotency middleware
└── types/
    └── index.ts       # TypeScript types
```

## 🔭 Observability

The app uses Elysia's native OpenTelemetry plugin. Traces are exported to Jaeger via OTLP.

### Viewing Traces

1. Start the stack with `docker-compose up`
2. Make some API requests
3. Open Jaeger UI at http://localhost:16686
4. Select "dict-simulator" service

Each request creates spans for:
- Route handlers (`handler.createEntry`, etc.)
- Validations (`validation.key`)
- Database operations (`db.create`, `db.findOne`, etc.)

## 🧪 Scripts

```bash
bun run dev      # Development with hot reload
bun run start    # Production start
bun run format   # Format code with Biome
bun run lint     # Lint code with Biome
bun run check    # Full Biome check
```

## 📝 Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| PORT | 3000 | Server port |
| MONGODB_URI | mongodb://localhost:27017/dict | MongoDB connection string |
| OTEL_EXPORTER_OTLP_ENDPOINT | http://localhost:4318/v1/traces | OpenTelemetry endpoint |

## License

MIT
