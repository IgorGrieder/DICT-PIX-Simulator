package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"

	"github.com/dict-simulator/go/internal/config"
	"github.com/dict-simulator/go/internal/db"
	"github.com/dict-simulator/go/internal/middleware"
	"github.com/dict-simulator/go/internal/models"
	"github.com/dict-simulator/go/internal/modules/auth"
	"github.com/dict-simulator/go/internal/modules/entries"
	"github.com/dict-simulator/go/internal/ratelimit"
	"github.com/dict-simulator/go/internal/router"
)

// Global test infrastructure - shared across all tests via TestMain
var (
	testMongoDB *db.Mongo
	testRedisDB *db.Redis
)

// TestMain sets up shared test infrastructure once for all tests
func TestMain(m *testing.M) {
	ctx := context.Background()

	// Start MongoDB container
	mongoContainer, err := mongodb.Run(ctx, "mongo:7")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start MongoDB container: %v\n", err)
		os.Exit(1)
	}
	defer mongoContainer.Terminate(ctx)

	mongoURI, err := mongoContainer.ConnectionString(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get MongoDB connection string: %v\n", err)
		os.Exit(1)
	}

	// Start Redis container
	redisContainer, err := tcredis.Run(ctx, "redis:7")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start Redis container: %v\n", err)
		os.Exit(1)
	}
	defer redisContainer.Terminate(ctx)

	redisURI, err := redisContainer.ConnectionString(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get Redis connection string: %v\n", err)
		os.Exit(1)
	}

	// Connect to databases
	testMongoDB, err = db.ConnectMongo(mongoURI)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to MongoDB: %v\n", err)
		os.Exit(1)
	}
	defer testMongoDB.Disconnect()

	testRedisDB, err = db.ConnectRedis(redisURI)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to Redis: %v\n", err)
		os.Exit(1)
	}
	defer testRedisDB.Disconnect()

	// Run tests
	code := m.Run()

	os.Exit(code)
}

// createTestServer creates a new server instance with specific config and isolated database
func createTestServer(t *testing.T, cfg *config.Config, dbName string) *httptest.Server {
	t.Helper()

	// Create isolated database connection
	isolatedMongo := testMongoDB.WithDatabase(dbName)

	// Initialize repositories with isolated DB
	entryRepo := models.NewEntryRepository(isolatedMongo)
	userRepo := models.NewUserRepository(isolatedMongo)
	idempotencyRepo := models.NewIdempotencyRepository(isolatedMongo)

	// Ensure indexes on the new isolated DB
	ctx := context.Background()
	if err := entryRepo.EnsureIndexes(ctx); err != nil {
		t.Fatalf("Failed to ensure entry indexes: %v", err)
	}
	if err := userRepo.EnsureIndexes(ctx); err != nil {
		t.Fatalf("Failed to ensure user indexes: %v", err)
	}
	if err := idempotencyRepo.EnsureIndexes(ctx); err != nil {
		t.Fatalf("Failed to ensure idempotency indexes: %v", err)
	}

	// Initialize rate limiter (shared Redis is fine, keys are isolated by user/request)
	rateLimitBucket := ratelimit.NewBucket(testRedisDB.Client)
	mwManager := middleware.NewManager(idempotencyRepo, rateLimitBucket, cfg.RateLimitEnabled)

	// Initialize handlers
	authHandler := auth.NewHandler(userRepo, cfg.JWTSecret)
	entriesHandler := entries.NewHandler(entryRepo)

	// Setup router with default policies
	handler := router.Setup(cfg, authHandler, entriesHandler, mwManager, ratelimit.DefaultPolicies())

	srv := httptest.NewServer(handler)

	// Register cleanup: Close server first, then Drop DB
	// t.Cleanup runs in reverse order of registration
	t.Cleanup(func() {
		if err := isolatedMongo.Database.Drop(context.Background()); err != nil {
			t.Logf("Failed to drop test database %s: %v", dbName, err)
		}
	})
	t.Cleanup(srv.Close)

	return srv
}

// StartRateLimitedServer starts a new server with rate limiting enabled
func StartRateLimitedServer(t *testing.T) *httptest.Server {
	t.Helper()
	cfg := &config.Config{
		Port:                   3000,
		Environment:            "test",
		JWTSecret:              "test-jwt-secret-for-integration-tests",
		RateLimitEnabled:       true,
		RateLimitBucketSize:    60,
		RateLimitRefillSeconds: 60,
	}
	dbName := "test_dict_ratelimit_" + uuid.New().String()
	return createTestServer(t, cfg, dbName)
}

// TestClient provides HTTP client methods for a specific test
type TestClient struct {
	t         *testing.T
	authToken string
	baseURL   string
}

// NewTestClient creates a client for a test with its own auth token and isolated server
func NewTestClient(t *testing.T) *TestClient {
	t.Helper()

	cfg := &config.Config{
		Port:                   3000,
		Environment:            "test",
		JWTSecret:              "test-jwt-secret-for-integration-tests",
		RateLimitEnabled:       false,
		RateLimitBucketSize:    60,
		RateLimitRefillSeconds: 60,
	}
	dbName := "test_dict_" + uuid.New().String()
	server := createTestServer(t, cfg, dbName)

	return NewTestClientForServer(t, server)
}

// NewTestClientForServer creates a client for a specific server
func NewTestClientForServer(t *testing.T, server *httptest.Server) *TestClient {
	t.Helper()

	client := &TestClient{
		t:       t,
		baseURL: server.URL,
	}

	// Register a unique user for this test
	client.authToken = client.registerTestUser()

	return client
}

// registerTestUser creates a unique test user and returns the auth token
func (c *TestClient) registerTestUser() string {
	email := fmt.Sprintf("test-%s@example.com", uuid.New().String()[:8])

	body := map[string]string{
		"email":    email,
		"password": "testpassword123",
		"name":     "Test User",
	}

	resp := c.PostNoAuth("/auth/register", body)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		c.t.Fatalf("Failed to register test user: status %d", resp.StatusCode)
	}

	var result struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		c.t.Fatalf("Failed to decode auth response: %v", err)
	}

	return result.Data.Token
}

// Request makes an HTTP request
func (c *TestClient) Request(method, path string, body any, headers map[string]string) *http.Response {
	c.t.Helper()

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			c.t.Fatalf("Failed to marshal request body: %v", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		c.t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Add auth token
	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}

	// Add custom headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.t.Fatalf("Failed to make request: %v", err)
	}

	return resp
}

// PostNoAuth makes a POST request without auth (for register/login)
func (c *TestClient) PostNoAuth(path string, body any) *http.Response {
	c.t.Helper()

	jsonBody, err := json.Marshal(body)
	if err != nil {
		c.t.Fatalf("Failed to marshal request body: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, bytes.NewReader(jsonBody))
	if err != nil {
		c.t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.t.Fatalf("Failed to make request: %v", err)
	}

	return resp
}

// GET makes a GET request
func (c *TestClient) GET(path string) *http.Response {
	return c.Request(http.MethodGet, path, nil, nil)
}

// GETWithHeaders makes a GET request with custom headers
func (c *TestClient) GETWithHeaders(path string, headers map[string]string) *http.Response {
	return c.Request(http.MethodGet, path, nil, headers)
}

// POST makes a POST request
func (c *TestClient) POST(path string, body any) *http.Response {
	return c.Request(http.MethodPost, path, body, nil)
}

// POSTWithHeaders makes a POST request with custom headers
func (c *TestClient) POSTWithHeaders(path string, body any, headers map[string]string) *http.Response {
	return c.Request(http.MethodPost, path, body, headers)
}

// PUT makes a PUT request
func (c *TestClient) PUT(path string, body any) *http.Response {
	return c.Request(http.MethodPut, path, body, nil)
}

// DeleteEntry makes a POST request to delete an entry (DICT spec uses POST)
func (c *TestClient) DeleteEntry(key, participant, reason string) *http.Response {
	body := map[string]string{
		"key":         key,
		"participant": participant,
		"reason":      reason,
	}
	return c.POST("/entries/"+key+"/delete", body)
}

// CreateEntry creates an entry and returns the CPF used
func (c *TestClient) CreateEntry() string {
	c.t.Helper()

	cpf := GenerateValidCPF()
	req := CreateEntryRequest(cpf)

	resp := c.POSTWithHeaders("/entries", req, map[string]string{
		"X-Idempotency-Key": uuid.New().String(),
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		c.t.Fatalf("Failed to create entry: status %d", resp.StatusCode)
	}

	return cpf
}

// CleanupEntry deletes an entry (call in defer)
func (c *TestClient) CleanupEntry(cpf string) {
	resp := c.DeleteEntry(cpf, "12345678", "USER_REQUESTED")
	resp.Body.Close()
}

// ParseResponse parses a JSON response into the given struct
func ParseResponse[T any](t *testing.T, resp *http.Response) T {
	t.Helper()
	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	return result
}

// GenerateValidCPF generates a valid CPF using Módulo 11 algorithm
func GenerateValidCPF() string {
	// Use timestamp + random for uniqueness
	now := time.Now().UnixNano()
	digits := make([]int, 11)

	// Generate 9 semi-random digits based on timestamp
	for i := 0; i < 9; i++ {
		digits[i] = int((now >> (i * 3)) % 10)
	}

	// Calculate first check digit
	sum := 0
	for i := 0; i < 9; i++ {
		sum += digits[i] * (10 - i)
	}
	remainder := (sum * 10) % 11
	if remainder == 10 {
		remainder = 0
	}
	digits[9] = remainder

	// Calculate second check digit
	sum = 0
	for i := 0; i < 10; i++ {
		sum += digits[i] * (11 - i)
	}
	remainder = (sum * 10) % 11
	if remainder == 10 {
		remainder = 0
	}
	digits[10] = remainder

	return fmt.Sprintf("%d%d%d%d%d%d%d%d%d%d%d",
		digits[0], digits[1], digits[2], digits[3], digits[4],
		digits[5], digits[6], digits[7], digits[8], digits[9], digits[10])
}

// CreateEntryRequest creates a valid entry request body
func CreateEntryRequest(cpf string) map[string]any {
	return map[string]any{
		"key":     cpf,
		"keyType": "CPF",
		"account": map[string]any{
			"participant":   "12345678",
			"branch":        "0001",
			"accountNumber": "0007654321",
			"accountType":   "CACC",
			"openingDate":   time.Now().UTC().Format(time.RFC3339),
		},
		"owner": map[string]any{
			"type":        "NATURAL_PERSON",
			"taxIdNumber": cpf,
			"name":        "Test User",
		},
		"reason":    "USER_REQUESTED",
		"requestId": uuid.New().String(),
	}
}

// FlushRedis is removed as we use isolated databases/keys now
