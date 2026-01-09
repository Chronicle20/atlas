package seed

import (
	"encoding/json"
	"testing"
)

func TestSeedResultJSONSerialization(t *testing.T) {
	result := SeedResult{
		DeletedShops:       10,
		DeletedCommodities: 100,
		CreatedShops:       5,
		CreatedCommodities: 50,
		FailedCount:        2,
		Errors:             []string{"error1", "error2"},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal SeedResult: %v", err)
	}

	var unmarshaled SeedResult
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal SeedResult: %v", err)
	}

	if unmarshaled.DeletedShops != 10 {
		t.Errorf("Expected DeletedShops 10, got %d", unmarshaled.DeletedShops)
	}
	if unmarshaled.DeletedCommodities != 100 {
		t.Errorf("Expected DeletedCommodities 100, got %d", unmarshaled.DeletedCommodities)
	}
	if unmarshaled.CreatedShops != 5 {
		t.Errorf("Expected CreatedShops 5, got %d", unmarshaled.CreatedShops)
	}
	if unmarshaled.CreatedCommodities != 50 {
		t.Errorf("Expected CreatedCommodities 50, got %d", unmarshaled.CreatedCommodities)
	}
	if unmarshaled.FailedCount != 2 {
		t.Errorf("Expected FailedCount 2, got %d", unmarshaled.FailedCount)
	}
	if len(unmarshaled.Errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(unmarshaled.Errors))
	}
}

func TestSeedResultErrorsOmitEmpty(t *testing.T) {
	result := SeedResult{
		DeletedShops:       10,
		DeletedCommodities: 100,
		CreatedShops:       5,
		CreatedCommodities: 50,
		FailedCount:        0,
		Errors:             nil,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal SeedResult: %v", err)
	}

	jsonStr := string(data)

	// Errors should be omitted when nil/empty
	if contains(jsonStr, "errors") {
		t.Error("Expected 'errors' field to be omitted when nil")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
