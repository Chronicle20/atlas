package consumable

import (
	"context"
	"encoding/json"
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

// The 243 handler resolves the dialogue's avatar from the consumable's npc
// field. Until task-230 the channel's RestModel carried only `spec`, so the
// value never reached the handler even after atlas-data started parsing it
// correctly.
func TestConsumableExtractCarriesNpcScriptAndRunOnPickup(t *testing.T) {
	rm := RestModel{
		Id:          2430008,
		Spec:        map[SpecType]int32{},
		Npc:         2084002,
		Script:      "compassUse",
		RunOnPickup: false,
	}
	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if m.Npc() != 2084002 {
		t.Errorf("Npc: got %d want 2084002", m.Npc())
	}
	if m.Script() != "compassUse" {
		t.Errorf("Script: got %q want %q", m.Script(), "compassUse")
	}
	if m.RunOnPickup() {
		t.Error("RunOnPickup: got true want false")
	}
}

// The json tags must match what atlas-data serves
// (services/atlas-data/atlas.com/data/consumable/rest.go:74-76). A mismatch
// decodes to zero silently and looks exactly like a content gap.
func TestConsumableRestModelJsonTags(t *testing.T) {
	var rm RestModel
	if err := json.Unmarshal([]byte(`{"npc":2084002,"script":"compassUse","runOnPickup":true,"spec":{}}`), &rm); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if rm.Npc != 2084002 || rm.Script != "compassUse" || !rm.RunOnPickup {
		t.Errorf("decoded: %+v", rm)
	}
}
