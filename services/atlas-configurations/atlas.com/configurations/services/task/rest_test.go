package task

import (
	"encoding/json"
	"testing"
)

func TestRestModel_JSONMarshal(t *testing.T) {
	rm := RestModel{
		Type:     "heartbeat",
		Interval: 10000,
		Duration: 5000,
	}

	data, err := json.Marshal(rm)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	var decoded RestModel
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if decoded.Type != rm.Type {
		t.Errorf("expected Type '%s', got '%s'", rm.Type, decoded.Type)
	}
	if decoded.Interval != rm.Interval {
		t.Errorf("expected Interval %d, got %d", rm.Interval, decoded.Interval)
	}
	if decoded.Duration != rm.Duration {
		t.Errorf("expected Duration %d, got %d", rm.Duration, decoded.Duration)
	}
}

func TestRestModel_JSONMarshal_ZeroValues(t *testing.T) {
	rm := RestModel{
		Type:     "cleanup",
		Interval: 60000,
		Duration: 0,
	}

	data, err := json.Marshal(rm)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	var decoded RestModel
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if decoded.Duration != 0 {
		t.Errorf("expected Duration 0, got %d", decoded.Duration)
	}
}

func TestRestModel_EmptyType(t *testing.T) {
	rm := RestModel{
		Type:     "",
		Interval: 1000,
	}

	data, err := json.Marshal(rm)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	var decoded RestModel
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if decoded.Type != "" {
		t.Errorf("expected empty Type, got '%s'", decoded.Type)
	}
}

func TestRestModel_JSONFields(t *testing.T) {
	jsonStr := `{"type":"respawn","interval":5000,"duration":1000}`

	var rm RestModel
	err := json.Unmarshal([]byte(jsonStr), &rm)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if rm.Type != "respawn" {
		t.Errorf("expected Type 'respawn', got '%s'", rm.Type)
	}
	if rm.Interval != 5000 {
		t.Errorf("expected Interval 5000, got %d", rm.Interval)
	}
	if rm.Duration != 1000 {
		t.Errorf("expected Duration 1000, got %d", rm.Duration)
	}
}
