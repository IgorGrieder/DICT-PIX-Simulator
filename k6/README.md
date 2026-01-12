# k6 Load Tests for DICT Simulator

Performance and load testing scripts for the Pix DICT Simulator API.

## Prerequisites

Install k6: https://grafana.com/docs/k6/latest/set-up/install-k6/

```bash
# macOS
brew install k6

# Docker alternative
docker run --rm -i grafana/k6 run - <script.js
```

## Running Tests

### 1. CRUD Flow Test (Smoke + Load)

Tests the full create → get → delete flow with valid CPF generation.

```bash
k6 run k6/entries.test.js
```

### 2. Idempotency Test

Verifies that duplicate requests with the same `x-idempotency-key` return identical responses.

```bash
k6 run k6/idempotency.test.js
```

### 3. Stress Test

Ramps up to 100 concurrent users to test system limits.

```bash
k6 run k6/stress.test.js
```

## Custom Base URL

```bash
k6 run -e BASE_URL=http://your-server:3000 k6/entries.test.js
```

## Docker Compose Integration

Run tests against the Docker Compose stack:

```bash
# Start the stack
docker-compose up -d

# Wait for services
sleep 10

# Run tests
k6 run k6/entries.test.js
```

## Thresholds

| Test | Metric | Threshold |
|------|--------|-----------|
| entries | p95 latency | < 500ms |
| entries | failure rate | < 1% |
| stress | p99 latency | < 1000ms |
| stress | failure rate | < 5% |
| idempotency | check rate | = 100% |
