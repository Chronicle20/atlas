package validation

import (
	"strings"
	"testing"
)

func TestValidateConditionInput(t *testing.T) {
	tests := []struct {
		name          string
		input         ConditionInput
		wantError     bool
		errorContains string
	}{
		{
			name:      "inventorySpace valid",
			input:     ConditionInput{Type: "inventorySpace", Operator: ">=", Value: 1, ReferenceId: 2010000},
			wantError: false,
		},
		{
			name:          "inventorySpace missing referenceId",
			input:         ConditionInput{Type: "inventorySpace", Operator: ">=", Value: 1},
			wantError:     true,
			errorContains: "referenceId (itemId) is required for inventorySpace conditions",
		},
		{
			name:          "inventorySpace quantity less than 1",
			input:         ConditionInput{Type: "inventorySpace", Operator: ">=", Value: 0, ReferenceId: 2010000},
			wantError:     true,
			errorContains: "inventorySpace value (quantity) must be >= 1",
		},
		{
			name:      "mapCapacity valid",
			input:     ConditionInput{Type: "mapCapacity", Operator: "<", Value: 30, ReferenceId: 100000000},
			wantError: false,
		},
		{
			name:          "mapCapacity missing referenceId",
			input:         ConditionInput{Type: "mapCapacity", Operator: "<", Value: 30},
			wantError:     true,
			errorContains: "referenceId is required for mapCapacity conditions",
		},
		{
			name:      "transportAvailable valid",
			input:     ConditionInput{Type: "transportAvailable", Operator: "=", Value: 1, ReferenceId: 1},
			wantError: false,
		},
		{
			name:          "transportAvailable missing referenceId",
			input:         ConditionInput{Type: "transportAvailable", Operator: "=", Value: 1},
			wantError:     true,
			errorContains: "referenceId is required for transportAvailable conditions",
		},
		{
			name:      "skillLevel valid",
			input:     ConditionInput{Type: "skillLevel", Operator: ">=", Value: 1, ReferenceId: 1004},
			wantError: false,
		},
		{
			name:          "skillLevel missing referenceId",
			input:         ConditionInput{Type: "skillLevel", Operator: ">=", Value: 1},
			wantError:     true,
			errorContains: "referenceId is required for skillLevel conditions",
		},
		{
			name:      "buff valid",
			input:     ConditionInput{Type: "buff", Operator: "=", Value: 1, ReferenceId: 2001002},
			wantError: false,
		},
		{
			name:          "buff missing referenceId",
			input:         ConditionInput{Type: "buff", Operator: "=", Value: 1},
			wantError:     true,
			errorContains: "referenceId is required for buff conditions",
		},
		{
			name:      "excessSp valid",
			input:     ConditionInput{Type: "excessSp", Operator: ">", Value: 0, ReferenceId: 10},
			wantError: false,
		},
		{
			name:          "excessSp missing referenceId",
			input:         ConditionInput{Type: "excessSp", Operator: ">", Value: 0},
			wantError:     true,
			errorContains: "referenceId (base level) is required for excessSp conditions",
		},
		{
			name:      "pqCustomData valid",
			input:     ConditionInput{Type: "pqCustomData", Operator: "=", Value: 1, Step: "stage"},
			wantError: false,
		},
		{
			name:          "pqCustomData missing step",
			input:         ConditionInput{Type: "pqCustomData", Operator: "=", Value: 1},
			wantError:     true,
			errorContains: "step (custom data key) is required for pqCustomData conditions",
		},
		{
			name:      "buddyCapacity valid",
			input:     ConditionInput{Type: "buddyCapacity", Operator: ">=", Value: 20},
			wantError: false,
		},
		{
			name:      "petCount valid",
			input:     ConditionInput{Type: "petCount", Operator: ">=", Value: 1},
			wantError: false,
		},
		{
			name:      "guildLeader valid",
			input:     ConditionInput{Type: "guildLeader", Operator: "=", Value: 1},
			wantError: false,
		},
		{
			name:      "partyId valid",
			input:     ConditionInput{Type: "partyId", Operator: "=", Value: 42},
			wantError: false,
		},
		{
			name:      "partyLeader valid",
			input:     ConditionInput{Type: "partyLeader", Operator: "=", Value: 1},
			wantError: false,
		},
		{
			name:      "partySize valid",
			input:     ConditionInput{Type: "partySize", Operator: ">=", Value: 2},
			wantError: false,
		},
		{
			name:          "unknown type still rejected",
			input:         ConditionInput{Type: "bogus", Operator: "=", Value: 1},
			wantError:     true,
			errorContains: "unsupported condition type: bogus",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConditionInput(tt.input)
			if tt.wantError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Fatalf("expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}
