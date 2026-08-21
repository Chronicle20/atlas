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

// TestRestModel_Equipment asserts that the include=inventory block's
// "equipment" included resources decode into the equipped compartment.
func TestRestModel_Equipment(t *testing.T) {
	payload := []byte(`{
		"data": {
			"type": "characters",
			"id": "1001",
			"attributes": {"name": "Statue"}
		},
		"included": [
			{"type": "equipment", "id": "1", "attributes": {"slot": -1, "templateId": 1302000}},
			{"type": "equipment", "id": "2", "attributes": {"slot": -101, "templateId": 1040002}}
		]
	}`)

	var rm RestModel
	if err := jsonapi.Unmarshal(payload, &rm); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	equipped := m.Equipment()
	if len(equipped) != 2 {
		t.Fatalf("len(Equipment()) = %d, want 2", len(equipped))
	}
	// Sorted by slot ascending: -101 then -1.
	if equipped[0].Slot() != -101 || equipped[0].TemplateId() != 1040002 {
		t.Errorf("equipped[0] = %+v, want slot -101 templateId 1040002", equipped[0])
	}
	if equipped[1].Slot() != -1 || equipped[1].TemplateId() != 1302000 {
		t.Errorf("equipped[1] = %+v, want slot -1 templateId 1302000", equipped[1])
	}
}

// TestRestModel_Equipment_Absent asserts that a response with no included
// block leaves Equipment empty rather than erroring.
func TestRestModel_Equipment_Absent(t *testing.T) {
	payload := []byte(`{
		"data": {
			"type": "characters",
			"id": "1001",
			"attributes": {"name": "Statue"}
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
	if len(m.Equipment()) != 0 {
		t.Errorf("len(Equipment()) = %d, want 0", len(m.Equipment()))
	}
}
