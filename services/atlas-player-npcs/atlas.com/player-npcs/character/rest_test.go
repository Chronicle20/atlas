package character

import (
	"testing"

	"github.com/jtumidanski/api2go/jsonapi"
)

// TestRestModel_Decode asserts the JSON:API decode of a captured
// characters/{id} payload: name, gender, skinColor, face, hair, jobId,
// level and gm all populate.
func TestRestModel_Decode(t *testing.T) {
	payload := []byte(`{
		"data": {
			"type": "characters",
			"id": "1001",
			"attributes": {
				"name": "Statue",
				"gender": 1,
				"skinColor": 3,
				"face": 20000,
				"hair": 30030,
				"jobId": 112,
				"level": 120,
				"gm": 1
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

	if m.Id() != 1001 {
		t.Errorf("Id() = %d, want 1001", m.Id())
	}
	if m.Name() != "Statue" {
		t.Errorf("Name() = %q, want Statue", m.Name())
	}
	if m.Gender() != 1 {
		t.Errorf("Gender() = %d, want 1", m.Gender())
	}
	if m.SkinColor() != 3 {
		t.Errorf("SkinColor() = %d, want 3", m.SkinColor())
	}
	if m.Face() != 20000 {
		t.Errorf("Face() = %d, want 20000", m.Face())
	}
	if m.Hair() != 30030 {
		t.Errorf("Hair() = %d, want 30030", m.Hair())
	}
	if uint16(m.JobId()) != 112 {
		t.Errorf("JobId() = %d, want 112", m.JobId())
	}
	if m.Level() != 120 {
		t.Errorf("Level() = %d, want 120", m.Level())
	}
	if !m.Gm() {
		t.Errorf("Gm() = false, want true")
	}
}
