package _map

import (
	"testing"

	"github.com/jtumidanski/api2go/jsonapi"
)

// TestRestModel_Decode_MapArea asserts the JSON:API decode of a captured
// data/maps/{id} payload's mapArea bounds.
func TestRestModel_Decode_MapArea(t *testing.T) {
	payload := []byte(`{
		"data": {
			"type": "maps",
			"id": "100000000",
			"attributes": {
				"mapArea": {"x": -500, "y": -300, "width": 1000, "height": 600}
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

	if m.Id() != 100000000 {
		t.Errorf("Id() = %d, want 100000000", m.Id())
	}
	area := m.MapArea()
	if area == nil {
		t.Fatalf("MapArea() = nil, want non-nil")
	}
	if area.X() != -500 || area.Y() != -300 || area.Width() != 1000 || area.Height() != 600 {
		t.Errorf("MapArea() = %+v, want {-500 -300 1000 600}", area)
	}
}

// TestGroundResultRestModel_Decode asserts the ground response decodes into
// GroundResult values.
func TestGroundResultRestModel_Decode(t *testing.T) {
	payload := []byte(`{
		"data": [
			{"type": "grounds", "id": "0", "attributes": {"x": 0, "y": -50, "fh": 1, "found": true}},
			{"type": "grounds", "id": "1", "attributes": {"x": 30000, "y": 0, "fh": 0, "found": false}}
		]
	}`)

	var rms []GroundResultRestModel
	if err := jsonapi.Unmarshal(payload, &rms); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	if len(rms) != 2 {
		t.Fatalf("len(rms) = %d, want 2", len(rms))
	}

	first, err := ExtractGroundResult(rms[0])
	if err != nil {
		t.Fatalf("ExtractGroundResult returned error: %v", err)
	}
	if !first.Found() || first.Y() != -50 || first.Fh() != 1 {
		t.Errorf("first = %+v, want found=true y=-50 fh=1", first)
	}

	second, err := ExtractGroundResult(rms[1])
	if err != nil {
		t.Fatalf("ExtractGroundResult returned error: %v", err)
	}
	if second.Found() {
		t.Errorf("second.Found() = true, want false")
	}
}
