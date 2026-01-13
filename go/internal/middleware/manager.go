package middleware

import (
	"github.com/dict-simulator/go/internal/models"
	"github.com/dict-simulator/go/internal/ratelimit"
)

type Manager struct {
	idempotencyRepo *models.IdempotencyRepository
	rateLimiter     *ratelimit.Bucket
}

func NewManager(idempotencyRepo *models.IdempotencyRepository, rateLimiter *ratelimit.Bucket) *Manager {
	return &Manager{
		idempotencyRepo: idempotencyRepo,
		rateLimiter:     rateLimiter,
	}
}
