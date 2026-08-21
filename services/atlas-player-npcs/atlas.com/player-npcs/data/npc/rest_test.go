package npc

import (
	"testing"

	"github.com/jtumidanski/api2go/jsonapi"
)

// TestRestModel_Decode_ImitateTrue asserts that imitate: true decodes and
// populates Imitate().
func TestRestModel_Decode_ImitateTrue(t *testing.T) {
	payload := []byte(`{
		"data": {
			"type": "npcs",
			"id": "9010000",
			"attributes": {
				"name": "Maple Administrator",
				"imitate": true
			}
		}
	}`)

	var rm RestModel
	if err := jsonapi.Unmarshal(payload, &rm); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	if m.Id() != 9010000 {
		t.Errorf("Id() = %d, want 9010000", m.Id())
	}
	if m.Name() != "Maple Administrator" {
		t.Errorf("Name() = %q, want Maple Administrator", m.Name())
	}
	if !m.Imitate() {
		t.Errorf("Imitate() = false, want true")
	}
}

// TestRestModel_Decode_ImitateAbsent asserts that an absent imitate field
// decodes to false, matching the server's default.
func TestRestModel_Decode_ImitateAbsent(t *testing.T) {
	payload := []byte(`{
		"data": {
			"type": "npcs",
			"id": "1052",
			"attributes": {
				"name": "Snail"
			}
		}
	}`)

	var rm RestModel
	if err := jsonapi.Unmarshal(payload, &rm); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	if m.Imitate() {
		t.Errorf("Imitate() = true, want false")
	}
}
