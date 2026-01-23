package seed

import (
	"encoding/json"
	"testing"
)

func TestSeedResultJSON(t *testing.T) {
	tests := []struct {
		name     string
		result   SeedResult
		expected string
	}{
		{
			name: "empty result",
			result: SeedResult{
				DeletedCount: 0,
				CreatedCount: 0,
				FailedCount:  0,
				Errors:       nil,
			},
			expected: `{"deletedCount":0,"createdCount":0,"failedCount":0}`,
		},
		{
			name: "successful seed",
			result: SeedResult{
				DeletedCount: 100,
				CreatedCount: 150,
				FailedCount:  0,
				Errors:       nil,
			},
			expected: `{"deletedCount":100,"createdCount":150,"failedCount":0}`,
		},
		{
			name: "seed with errors",
			result: SeedResult{
				DeletedCount: 50,
				CreatedCount: 45,
				FailedCount:  5,
				Errors:       []string{"file1.json: parse error", "file2.json: invalid format"},
			},
			expected: `{"deletedCount":50,"createdCount":45,"failedCount":5,"errors":["file1.json: parse error","file2.json: invalid format"]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBytes, err := json.Marshal(tt.result)
			if err != nil {
				t.Fatalf("Failed to marshal SeedResult: %v", err)
			}

			if string(jsonBytes) != tt.expected {
				t.Errorf("Expected JSON:\n%s\nGot:\n%s", tt.expected, string(jsonBytes))
			}
		})
	}
}

func TestCombinedSeedResultJSON(t *testing.T) {
	tests := []struct {
		name     string
		result   CombinedSeedResult
		expected string
	}{
		{
			name: "empty combined result",
			result: CombinedSeedResult{
				MonsterDrops: SeedResult{
					DeletedCount: 0,
					CreatedCount: 0,
					FailedCount:  0,
				},
				ContinentDrops: SeedResult{
					DeletedCount: 0,
					CreatedCount: 0,
					FailedCount:  0,
				},
				ReactorDrops: SeedResult{
					DeletedCount: 0,
					CreatedCount: 0,
					FailedCount:  0,
				},
			},
			expected: `{"monsterDrops":{"deletedCount":0,"createdCount":0,"failedCount":0},"continentDrops":{"deletedCount":0,"createdCount":0,"failedCount":0},"reactorDrops":{"deletedCount":0,"createdCount":0,"failedCount":0}}`,
		},
		{
			name: "successful combined seed",
			result: CombinedSeedResult{
				MonsterDrops: SeedResult{
					DeletedCount: 1000,
					CreatedCount: 1500,
					FailedCount:  0,
				},
				ContinentDrops: SeedResult{
					DeletedCount: 4,
					CreatedCount: 4,
					FailedCount:  0,
				},
				ReactorDrops: SeedResult{
					DeletedCount: 50,
					CreatedCount: 146,
					FailedCount:  0,
				},
			},
			expected: `{"monsterDrops":{"deletedCount":1000,"createdCount":1500,"failedCount":0},"continentDrops":{"deletedCount":4,"createdCount":4,"failedCount":0},"reactorDrops":{"deletedCount":50,"createdCount":146,"failedCount":0}}`,
		},
		{
			name: "combined seed with errors",
			result: CombinedSeedResult{
				MonsterDrops: SeedResult{
					DeletedCount: 100,
					CreatedCount: 95,
					FailedCount:  5,
					Errors:       []string{"error1"},
				},
				ContinentDrops: SeedResult{
					DeletedCount: 4,
					CreatedCount: 3,
					FailedCount:  1,
					Errors:       []string{"error2"},
				},
				ReactorDrops: SeedResult{
					DeletedCount: 10,
					CreatedCount: 8,
					FailedCount:  2,
					Errors:       []string{"error3"},
				},
			},
			expected: `{"monsterDrops":{"deletedCount":100,"createdCount":95,"failedCount":5,"errors":["error1"]},"continentDrops":{"deletedCount":4,"createdCount":3,"failedCount":1,"errors":["error2"]},"reactorDrops":{"deletedCount":10,"createdCount":8,"failedCount":2,"errors":["error3"]}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBytes, err := json.Marshal(tt.result)
			if err != nil {
				t.Fatalf("Failed to marshal CombinedSeedResult: %v", err)
			}

			if string(jsonBytes) != tt.expected {
				t.Errorf("Expected JSON:\n%s\nGot:\n%s", tt.expected, string(jsonBytes))
			}
		})
	}
}

func TestSeedResultErrorsOmitEmpty(t *testing.T) {
	// Test that nil errors array is omitted from JSON
	result := SeedResult{
		DeletedCount: 10,
		CreatedCount: 10,
		FailedCount:  0,
		Errors:       nil,
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal SeedResult: %v", err)
	}

	jsonStr := string(jsonBytes)

	// Should not contain "errors" key when nil
	if contains(jsonStr, `"errors"`) {
		t.Errorf("Expected errors to be omitted when nil, got: %s", jsonStr)
	}
}

func TestSeedResultEmptyErrorsArray(t *testing.T) {
	// Test that empty errors array is omitted from JSON
	result := SeedResult{
		DeletedCount: 10,
		CreatedCount: 10,
		FailedCount:  0,
		Errors:       []string{},
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal SeedResult: %v", err)
	}

	jsonStr := string(jsonBytes)

	// Empty slice should also be omitted due to omitempty
	if contains(jsonStr, `"errors"`) {
		t.Errorf("Expected errors to be omitted when empty, got: %s", jsonStr)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
