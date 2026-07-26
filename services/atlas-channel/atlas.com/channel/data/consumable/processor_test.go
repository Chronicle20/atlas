package consumable

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
)

// consumableResponse is a real JSON:API consumables response, mirroring what
// atlas-data's consumable resource marshals (resource type "consumables", id =
// template id, attribute spec with hp/hpR/mp/mpR keys). It is served verbatim
// so the test exercises the same api2go unmarshal path the live client uses
// (the relationship-stub gotcha in libs/atlas-rest).
const consumableResponse = `{
  "data": { "type": "consumables", "id": "2000000", "attributes": { "spec": { "hp": 50, "hpR": 10, "mp": 0 } } }
}`

// TestGetById_DecodesSpec stands up an httptest server emulating atlas-data's
// consumable lookup and asserts the processor decodes the spec map, including
// distinguishing a present-zero value ("mp": 0) from an absent key ("mpR").
func TestGetById_DecodesSpec(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(consumableResponse))
	}))
	defer srv.Close()

	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	m, err := NewProcessor(logrus.New(), context.Background()).GetById(2000000)
	if err != nil {
		t.Fatalf("GetById returned error: %v", err)
	}

	if gotPath != "/data/consumables/2000000" {
		t.Errorf("request path: want /data/consumables/2000000, got %q", gotPath)
	}

	if v, ok := m.GetSpec(SpecTypeHP); !ok || v != 50 {
		t.Errorf("GetSpec(hp): want (50, true), got (%d, %v)", v, ok)
	}
	if v, ok := m.GetSpec(SpecTypeHPRecovery); !ok || v != 10 {
		t.Errorf("GetSpec(hpR): want (10, true), got (%d, %v)", v, ok)
	}
	if v, ok := m.GetSpec(SpecTypeMP); !ok || v != 0 {
		t.Errorf("GetSpec(mp): want (0, true), got (%d, %v)", v, ok)
	}
	if v, ok := m.GetSpec(SpecTypeMPRecovery); ok {
		t.Errorf("GetSpec(mpR): want (0, false) for an absent key, got (%d, %v)", v, ok)
	}
}
