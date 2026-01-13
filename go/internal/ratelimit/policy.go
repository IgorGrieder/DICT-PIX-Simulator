package ratelimit

// PolicyName identifies a rate limiting policy as defined in DICT API spec
type PolicyName string

const (
	// PolicyEntriesWrite applies to createEntry and deleteEntry operations
	PolicyEntriesWrite PolicyName = "ENTRIES_WRITE"

	// PolicyEntriesUpdate applies to updateEntry operations
	PolicyEntriesUpdate PolicyName = "ENTRIES_UPDATE"

	// PolicyEntriesReadParticipant applies to getEntry operations (participant antiscan)
	PolicyEntriesReadParticipant PolicyName = "ENTRIES_READ_PARTICIPANT_ANTISCAN"
)

// Scope defines who the rate limit applies to
type Scope string

const (
	// ScopePSP limits are shared across all requests from a participant
	ScopePSP Scope = "PSP"

	// ScopeUser limits are per end-user (PI-PayerId)
	ScopeUser Scope = "USER"
)

// Policy defines the configuration for a rate limiting bucket
// Based on DICT API specification for token bucket algorithm
type Policy struct {
	Name         PolicyName
	Scope        Scope
	RefillRate   int  // tokens replenished per minute
	BucketSize   int  // maximum tokens (bucket capacity)
	SuccessCost  int  // tokens consumed on 2xx response
	NotFoundCost int  // tokens consumed on 404 response
	DefaultCost  int  // tokens consumed on other non-5xx responses
	IgnoreOn5xx  bool // whether to skip token deduction on 5xx errors
}

// CostForStatus returns the token cost based on HTTP status code
func (p Policy) CostForStatus(statusCode int) int {
	// 5xx errors are ignored per DICT spec
	if statusCode >= 500 && p.IgnoreOn5xx {
		return 0
	}

	switch {
	case statusCode >= 200 && statusCode < 300:
		return p.SuccessCost
	case statusCode == 404:
		return p.NotFoundCost
	default:
		return p.DefaultCost
	}
}

// DefaultPolicies returns the DICT API rate limiting policies
// Using Category H for participant antiscan (most restrictive, good for testing)
func DefaultPolicies() map[PolicyName]Policy {
	return map[PolicyName]Policy{
		PolicyEntriesWrite: {
			Name:         PolicyEntriesWrite,
			Scope:        ScopePSP,
			RefillRate:   1200, // 1200 tokens per minute
			BucketSize:   36000,
			SuccessCost:  1,
			NotFoundCost: 1,
			DefaultCost:  1,
			IgnoreOn5xx:  true,
		},
		PolicyEntriesUpdate: {
			Name:         PolicyEntriesUpdate,
			Scope:        ScopePSP,
			RefillRate:   600, // 600 tokens per minute
			BucketSize:   600,
			SuccessCost:  1,
			NotFoundCost: 1,
			DefaultCost:  1,
			IgnoreOn5xx:  true,
		},
		PolicyEntriesReadParticipant: {
			Name:         PolicyEntriesReadParticipant,
			Scope:        ScopePSP,
			RefillRate:   2,  // Category H: 2 tokens per minute
			BucketSize:   50, // Category H: 50 token bucket
			SuccessCost:  1,
			NotFoundCost: 3, // DICT spec: 404 costs 3 tokens for antiscan
			DefaultCost:  1,
			IgnoreOn5xx:  true,
		},
	}
}

// GetPolicy returns a policy by name, or nil if not found
func GetPolicy(name PolicyName) *Policy {
	policies := DefaultPolicies()
	if p, ok := policies[name]; ok {
		return &p
	}
	return nil
}
