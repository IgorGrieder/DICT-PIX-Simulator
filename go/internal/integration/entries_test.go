package integration

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dict-simulator/go/internal/models"
)

func TestCreateEntry(t *testing.T) {
	t.Parallel()

	client := NewTestClient(t)
	cpf := GenerateValidCPF()
	correlationID := uuid.New().String()

	req := CreateEntryRequest(cpf)
	headers := map[string]string{
		"X-Correlation-Id":  correlationID,
		"X-Idempotency-Key": uuid.New().String(),
	}

	resp := client.POSTWithHeaders("/entries", req, headers)
	defer resp.Body.Close()

	// Verify status
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Verify correlation ID is returned
	assert.Equal(t, correlationID, resp.Header.Get("X-Correlation-Id"))

	// Parse and verify response structure
	var apiResp struct {
		ResponseTime  time.Time            `json:"responseTime"`
		CorrelationId string               `json:"correlationId"`
		Data          models.EntryResponse `json:"data"`
	}
	err := json.NewDecoder(resp.Body).Decode(&apiResp)
	require.NoError(t, err)

	assert.Equal(t, correlationID, apiResp.CorrelationId)
	assert.Equal(t, cpf, apiResp.Data.Key)
	assert.Equal(t, models.KeyTypeCPF, apiResp.Data.KeyType)
	assert.Equal(t, "12345678", apiResp.Data.Account.Participant)
	assert.Equal(t, "Test User", apiResp.Data.Owner.Name)
	assert.NotZero(t, apiResp.Data.CreatedAt)
	assert.NotZero(t, apiResp.Data.KeyOwnershipDate)
	assert.WithinDuration(t, time.Now(), apiResp.ResponseTime, time.Minute)

	// Cleanup
	client.CleanupEntry(cpf)
}

func TestGetEntry(t *testing.T) {
	t.Parallel()

	client := NewTestClient(t)

	// Create an entry first
	cpf := client.CreateEntry()
	defer client.CleanupEntry(cpf)

	// Get the entry
	resp := client.GET("/entries/" + cpf)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var apiResp struct {
		Data models.EntryResponse `json:"data"`
	}
	err := json.NewDecoder(resp.Body).Decode(&apiResp)
	require.NoError(t, err)

	assert.Equal(t, cpf, apiResp.Data.Key)
	assert.Equal(t, models.KeyTypeCPF, apiResp.Data.KeyType)
	assert.Equal(t, "12345678", apiResp.Data.Account.Participant)
}

func TestGetEntry_NotFound(t *testing.T) {
	t.Parallel()

	client := NewTestClient(t)

	resp := client.GET("/entries/00000000000")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	var apiResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	json.NewDecoder(resp.Body).Decode(&apiResp)
	assert.Equal(t, "ENTRY_NOT_FOUND", apiResp.Error)
}

func TestUpdateEntry(t *testing.T) {
	t.Parallel()

	client := NewTestClient(t)

	// Create an entry first
	cpf := client.CreateEntry()
	defer client.CleanupEntry(cpf)

	// Update the entry
	updateReq := map[string]any{
		"key": cpf,
		"owner": map[string]any{
			"name": "Updated User Name",
		},
		"reason": "USER_REQUESTED",
	}

	resp := client.PUT("/entries/"+cpf, updateReq)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var apiResp struct {
		Data models.EntryResponse `json:"data"`
	}
	err := json.NewDecoder(resp.Body).Decode(&apiResp)
	require.NoError(t, err)

	assert.Equal(t, "Updated User Name", apiResp.Data.Owner.Name)
}

func TestDeleteEntry(t *testing.T) {
	t.Parallel()

	client := NewTestClient(t)

	// Create an entry first
	cpf := client.CreateEntry()

	// Delete the entry
	resp := client.DeleteEntry(cpf, "12345678", "USER_REQUESTED")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var apiResp struct {
		Data struct {
			Message string `json:"message"`
			Key     string `json:"key"`
		} `json:"data"`
	}
	err := json.NewDecoder(resp.Body).Decode(&apiResp)
	require.NoError(t, err)

	assert.Equal(t, cpf, apiResp.Data.Key)
	assert.Contains(t, apiResp.Data.Message, "deleted")

	// Verify entry is gone
	getResp := client.GET("/entries/" + cpf)
	defer getResp.Body.Close()
	assert.Equal(t, http.StatusNotFound, getResp.StatusCode)
}

// =============================================================================
// Validation Tests
// =============================================================================

func TestCreateEntry_InvalidCPF(t *testing.T) {
	t.Parallel()

	client := NewTestClient(t)

	testCases := []struct {
		name string
		cpf  string
	}{
		{"wrong length", "1234567890"},
		{"invalid check digits", "12345678901"},
		{"all zeros", "00000000000"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := CreateEntryRequest(tc.cpf)
			resp := client.POSTWithHeaders("/entries", req, map[string]string{
				"X-Idempotency-Key": uuid.New().String(),
			})
			defer resp.Body.Close()

			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

			var apiResp struct {
				Error string `json:"error"`
			}
			json.NewDecoder(resp.Body).Decode(&apiResp)
			assert.Equal(t, "INVALID_CPF", apiResp.Error)
		})
	}
}

func TestCreateEntry_InvalidEmail(t *testing.T) {
	t.Parallel()

	client := NewTestClient(t)

	testCases := []struct {
		name  string
		email string
	}{
		{"uppercase", "Test@Example.com"},
		{"no @", "testexample.com"},
		{"no domain", "test@"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := map[string]any{
				"key":     tc.email,
				"keyType": "EMAIL",
				"account": map[string]any{
					"participant":   "12345678",
					"branch":        "0001",
					"accountNumber": "0007654321",
					"accountType":   "CACC",
					"openingDate":   time.Now().UTC().Format(time.RFC3339),
				},
				"owner": map[string]any{
					"type":        "NATURAL_PERSON",
					"taxIdNumber": "12345678901",
					"name":        "Test User",
				},
				"reason":    "USER_REQUESTED",
				"requestId": uuid.New().String(),
			}

			resp := client.POSTWithHeaders("/entries", req, map[string]string{
				"X-Idempotency-Key": uuid.New().String(),
			})
			defer resp.Body.Close()

			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

			var apiResp struct {
				Error string `json:"error"`
			}
			json.NewDecoder(resp.Body).Decode(&apiResp)
			assert.Equal(t, "INVALID_EMAIL", apiResp.Error)
		})
	}
}

func TestCreateEntry_InvalidPhone(t *testing.T) {
	t.Parallel()

	client := NewTestClient(t)

	testCases := []struct {
		name  string
		phone string
	}{
		{"missing plus", "5511999999999"},
		{"starts with zero", "+0511999999999"},
		{"too short", "+55"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := map[string]any{
				"key":     tc.phone,
				"keyType": "PHONE",
				"account": map[string]any{
					"participant":   "12345678",
					"branch":        "0001",
					"accountNumber": "0007654321",
					"accountType":   "CACC",
					"openingDate":   time.Now().UTC().Format(time.RFC3339),
				},
				"owner": map[string]any{
					"type":        "NATURAL_PERSON",
					"taxIdNumber": "12345678901",
					"name":        "Test User",
				},
				"reason":    "USER_REQUESTED",
				"requestId": uuid.New().String(),
			}

			resp := client.POSTWithHeaders("/entries", req, map[string]string{
				"X-Idempotency-Key": uuid.New().String(),
			})
			defer resp.Body.Close()

			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	}
}

// =============================================================================
// EVP Key Restrictions
// =============================================================================

func TestEVPKey_CannotBeUpdated(t *testing.T) {
	t.Parallel()

	client := NewTestClient(t)

	// Create an EVP entry
	evpKey := uuid.New().String()
	req := map[string]any{
		"key":     evpKey,
		"keyType": "EVP",
		"account": map[string]any{
			"participant":   "12345678",
			"branch":        "0001",
			"accountNumber": "0007654321",
			"accountType":   "CACC",
			"openingDate":   time.Now().UTC().Format(time.RFC3339),
		},
		"owner": map[string]any{
			"type":        "NATURAL_PERSON",
			"taxIdNumber": "12345678901",
			"name":        "Test User",
		},
		"reason":    "USER_REQUESTED",
		"requestId": uuid.New().String(),
	}

	createResp := client.POSTWithHeaders("/entries", req, map[string]string{
		"X-Idempotency-Key": uuid.New().String(),
	})
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	createResp.Body.Close()

	defer client.CleanupEntry(evpKey)

	// Try to update - should fail
	updateReq := map[string]any{
		"key": evpKey,
		"owner": map[string]any{
			"name": "New Name",
		},
		"reason": "USER_REQUESTED",
	}

	resp := client.PUT("/entries/"+evpKey, updateReq)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var apiResp struct {
		Error string `json:"error"`
	}
	json.NewDecoder(resp.Body).Decode(&apiResp)
	assert.Equal(t, "INVALID_OPERATION", apiResp.Error)
}

// =============================================================================
// Delete Validation
// =============================================================================

func TestDeleteEntry_WrongParticipant(t *testing.T) {
	t.Parallel()

	client := NewTestClient(t)

	cpf := client.CreateEntry()
	defer client.CleanupEntry(cpf)

	// Try to delete with wrong participant
	resp := client.DeleteEntry(cpf, "99999999", "USER_REQUESTED")
	defer resp.Body.Close()

	// With the single-query optimization, we can't distinguish between "key not found"
	// and "participant mismatch", so we return 404 for both (safer defaults anyway)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	var apiResp struct {
		Error string `json:"error"`
	}
	json.NewDecoder(resp.Body).Decode(&apiResp)
	assert.Equal(t, "ENTRY_NOT_FOUND", apiResp.Error)
}

func TestDeleteEntry_InvalidReason(t *testing.T) {
	t.Parallel()

	client := NewTestClient(t)

	cpf := client.CreateEntry()
	defer client.CleanupEntry(cpf)

	resp := client.DeleteEntry(cpf, "12345678", "INVALID_REASON")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestDeleteEntry_ValidReasons(t *testing.T) {
	t.Parallel()

	client := NewTestClient(t)

	validReasons := []string{
		"USER_REQUESTED",
		"ACCOUNT_CLOSURE",
		"RECONCILIATION",
		"FRAUD",
		"RFB_VALIDATION",
	}

	for _, reason := range validReasons {
		t.Run(reason, func(t *testing.T) {
			cpf := client.CreateEntry()

			resp := client.DeleteEntry(cpf, "12345678", reason)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode, "Reason %s should be valid", reason)
		})
	}
}

// =============================================================================
// Idempotency Tests
// =============================================================================

func TestIdempotency_SameKeyReturnsCachedResponse(t *testing.T) {
	t.Parallel()

	client := NewTestClient(t)

	cpf := GenerateValidCPF()
	idempotencyKey := uuid.New().String()
	req := CreateEntryRequest(cpf)

	headers := map[string]string{
		"X-Idempotency-Key": idempotencyKey,
	}

	// First request
	resp1 := client.POSTWithHeaders("/entries", req, headers)
	body1, _ := json.Marshal(ParseResponse[any](t, resp1))
	resp1.Body.Close()

	assert.Equal(t, http.StatusCreated, resp1.StatusCode)

	// Second request with same idempotency key - should return cached
	resp2 := client.POSTWithHeaders("/entries", req, headers)
	body2, _ := json.Marshal(ParseResponse[any](t, resp2))
	resp2.Body.Close()

	assert.Equal(t, http.StatusCreated, resp2.StatusCode)
	assert.JSONEq(t, string(body1), string(body2))

	// Cleanup
	client.CleanupEntry(cpf)
}

func TestIdempotency_DifferentKeyCausesConflict(t *testing.T) {
	t.Parallel()

	client := NewTestClient(t)

	cpf := GenerateValidCPF()
	req := CreateEntryRequest(cpf)

	// First request
	resp1 := client.POSTWithHeaders("/entries", req, map[string]string{
		"X-Idempotency-Key": uuid.New().String(),
	})
	resp1.Body.Close()
	require.Equal(t, http.StatusCreated, resp1.StatusCode)

	defer client.CleanupEntry(cpf)

	// Second request with different idempotency key but same CPF - conflict
	req["requestId"] = uuid.New().String() // New request ID
	resp2 := client.POSTWithHeaders("/entries", req, map[string]string{
		"X-Idempotency-Key": uuid.New().String(),
	})
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusConflict, resp2.StatusCode)
}

func TestCorrelationId_ReturnsProvidedId(t *testing.T) {
	t.Parallel()

	client := NewTestClient(t)

	correlationID := uuid.New().String()
	headers := map[string]string{
		"X-Correlation-Id": correlationID,
	}

	resp := client.GETWithHeaders("/entries/nonexistent", headers)
	defer resp.Body.Close()

	// Header should contain correlation ID
	assert.Equal(t, correlationID, resp.Header.Get("X-Correlation-Id"))

	// Body should also contain it
	var apiResp struct {
		CorrelationId string `json:"correlationId"`
	}
	json.NewDecoder(resp.Body).Decode(&apiResp)
	assert.Equal(t, correlationID, apiResp.CorrelationId)
}

func TestCorrelationId_GeneratedWhenNotProvided(t *testing.T) {
	t.Parallel()

	client := NewTestClient(t)

	resp := client.GET("/entries/nonexistent")
	defer resp.Body.Close()

	// Should have generated a correlation ID
	assert.NotEmpty(t, resp.Header.Get("X-Correlation-Id"))

	var apiResp struct {
		CorrelationId string `json:"correlationId"`
	}
	json.NewDecoder(resp.Body).Decode(&apiResp)
	assert.NotEmpty(t, apiResp.CorrelationId)
}

func TestResponseTime_IncludedInAllResponses(t *testing.T) {
	t.Parallel()

	client := NewTestClient(t)

	resp := client.GET("/entries/nonexistent")
	defer resp.Body.Close()

	var apiResp struct {
		ResponseTime time.Time `json:"responseTime"`
	}
	err := json.NewDecoder(resp.Body).Decode(&apiResp)
	require.NoError(t, err)

	// Response time should be recent (within last minute)
	assert.WithinDuration(t, time.Now(), apiResp.ResponseTime, time.Minute)
}

func TestRateLimiting_HeadersPresent(t *testing.T) {
	t.Parallel()

	// Use isolated server with rate limiting enabled
	server := StartRateLimitedServer(t)

	client := NewTestClientForServer(t, server)

	resp := client.GET("/entries/nonexistent")
	defer resp.Body.Close()

	assert.NotEmpty(t, resp.Header.Get("X-RateLimit-Limit"))
	assert.NotEmpty(t, resp.Header.Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, resp.Header.Get("X-RateLimit-Policy"))
}
