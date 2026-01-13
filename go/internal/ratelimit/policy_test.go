package ratelimit

import (
	"testing"
)

func TestPolicyCostForStatus(t *testing.T) {
	tests := []struct {
		name       string
		policy     Policy
		statusCode int
		want       int
	}{
		{
			name: "success 200 costs SuccessCost",
			policy: Policy{
				SuccessCost:  1,
				NotFoundCost: 3,
				DefaultCost:  1,
				IgnoreOn5xx:  true,
			},
			statusCode: 200,
			want:       1,
		},
		{
			name: "404 costs NotFoundCost (antiscan)",
			policy: Policy{
				SuccessCost:  1,
				NotFoundCost: 3,
				DefaultCost:  1,
				IgnoreOn5xx:  true,
			},
			statusCode: 404,
			want:       3,
		},
		{
			name: "500 ignored when IgnoreOn5xx is true",
			policy: Policy{
				SuccessCost:  1,
				NotFoundCost: 3,
				DefaultCost:  1,
				IgnoreOn5xx:  true,
			},
			statusCode: 500,
			want:       0,
		},
		{
			name: "500 counted when IgnoreOn5xx is false",
			policy: Policy{
				SuccessCost:  1,
				NotFoundCost: 3,
				DefaultCost:  1,
				IgnoreOn5xx:  false,
			},
			statusCode: 500,
			want:       1,
		},
		{
			name: "400 costs DefaultCost",
			policy: Policy{
				SuccessCost:  1,
				NotFoundCost: 3,
				DefaultCost:  2,
				IgnoreOn5xx:  true,
			},
			statusCode: 400,
			want:       2,
		},
		{
			name: "201 Created costs SuccessCost",
			policy: Policy{
				SuccessCost:  1,
				NotFoundCost: 3,
				DefaultCost:  1,
				IgnoreOn5xx:  true,
			},
			statusCode: 201,
			want:       1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.policy.CostForStatus(tt.statusCode)
			if got != tt.want {
				t.Errorf("CostForStatus(%d) = %d, want %d", tt.statusCode, got, tt.want)
			}
		})
	}
}

func TestDefaultPolicies(t *testing.T) {
	policies := DefaultPolicies()

	// Test ENTRIES_WRITE policy
	entriesWrite, ok := policies[PolicyEntriesWrite]
	if !ok {
		t.Fatal("ENTRIES_WRITE policy not found")
	}
	if entriesWrite.RefillRate != 1200 {
		t.Errorf("ENTRIES_WRITE RefillRate = %d, want 1200", entriesWrite.RefillRate)
	}
	if entriesWrite.BucketSize != 36000 {
		t.Errorf("ENTRIES_WRITE BucketSize = %d, want 36000", entriesWrite.BucketSize)
	}

	// Test ENTRIES_UPDATE policy
	entriesUpdate, ok := policies[PolicyEntriesUpdate]
	if !ok {
		t.Fatal("ENTRIES_UPDATE policy not found")
	}
	if entriesUpdate.RefillRate != 600 {
		t.Errorf("ENTRIES_UPDATE RefillRate = %d, want 600", entriesUpdate.RefillRate)
	}
	if entriesUpdate.BucketSize != 600 {
		t.Errorf("ENTRIES_UPDATE BucketSize = %d, want 600", entriesUpdate.BucketSize)
	}

	// Test ENTRIES_READ_PARTICIPANT_ANTISCAN policy (Category H)
	entriesRead, ok := policies[PolicyEntriesReadParticipant]
	if !ok {
		t.Fatal("ENTRIES_READ_PARTICIPANT_ANTISCAN policy not found")
	}
	if entriesRead.RefillRate != 2 {
		t.Errorf("ENTRIES_READ RefillRate = %d, want 2 (Category H)", entriesRead.RefillRate)
	}
	if entriesRead.BucketSize != 50 {
		t.Errorf("ENTRIES_READ BucketSize = %d, want 50 (Category H)", entriesRead.BucketSize)
	}
	if entriesRead.NotFoundCost != 3 {
		t.Errorf("ENTRIES_READ NotFoundCost = %d, want 3 (antiscan penalty)", entriesRead.NotFoundCost)
	}
}

func TestGetPolicy(t *testing.T) {
	// Test existing policy
	p := GetPolicy(PolicyEntriesWrite)
	if p == nil {
		t.Fatal("GetPolicy(PolicyEntriesWrite) returned nil")
	}
	if p.Name != PolicyEntriesWrite {
		t.Errorf("GetPolicy returned wrong policy name: %s", p.Name)
	}

	// Test non-existing policy
	p = GetPolicy("NON_EXISTENT")
	if p != nil {
		t.Error("GetPolicy(NON_EXISTENT) should return nil")
	}
}
