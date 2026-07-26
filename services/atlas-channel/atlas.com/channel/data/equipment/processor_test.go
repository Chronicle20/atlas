package equipment

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
)

// equipmentResponse is a real JSON:API equipment response, mirroring what
// atlas-data's equipment resource marshals (resource type "statistics", id =
// template id, attribute petAbilities). It is served verbatim so the test
// exercises the same api2go unmarshal path the live client uses (the
// relationship-stub gotcha in libs/atlas-rest).
const equipmentResponse = `{
  "data": { "type": "statistics", "id": "1812000", "attributes": { "petAbilities": ["consumeHP", "consumeMP"] } }
}`

// TestGetById_DecodesPetAbilities stands up an httptest server emulating
// atlas-data's equipment lookup and asserts the processor decodes the pet
// abilities.
func TestGetById_DecodesPetAbilities(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(equipmentResponse))
	}))
	defer srv.Close()

	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	m, err := NewProcessor(logrus.New(), context.Background()).GetById(1812000)
	if err != nil {
		t.Fatalf("GetById returned error: %v", err)
	}

	if gotPath != "/data/equipment/1812000" {
		t.Errorf("request path: want /data/equipment/1812000, got %q", gotPath)
	}

	want := []string{"consumeHP", "consumeMP"}
	got := m.PetAbilities()
	if len(got) != len(want) {
		t.Fatalf("PetAbilities: want %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("PetAbilities[%d]: want %q, got %q", i, want[i], got[i])
		}
	}
}

// TestGetById_NoAbilities asserts an equip with no pet-ability attributes
// decodes to an empty (non-nil-required) abilities slice, matching the
// omitempty JSON tag on the wire.
func TestGetById_NoAbilities(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{"data": {"type": "statistics", "id": "1302000", "attributes": {}}}`))
	}))
	defer srv.Close()

	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	m, err := NewProcessor(logrus.New(), context.Background()).GetById(1302000)
	if err != nil {
		t.Fatalf("GetById returned error: %v", err)
	}
	if len(m.PetAbilities()) != 0 {
		t.Errorf("PetAbilities: want empty, got %v", m.PetAbilities())
	}
}
