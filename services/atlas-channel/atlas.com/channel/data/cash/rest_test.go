package cash

import (
	"encoding/json"
	"testing"
)

// The channel mirror of atlas-data's cash_items resource is partial by design —
// it carries only the fields the channel actually uses. This pins that `meso`
// (the 0520 meso-sack award amount) is one of them: without it the type-19
// handler cannot resolve an amount and every sack use fails closed.
func TestRestModelDecodesMeso(t *testing.T) {
	var rm RestModel
	if err := json.Unmarshal([]byte(`{"stateChangeItem":0,"bgmPath":"","protectTime":0,"meso":1000000}`), &rm); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if rm.Meso != 1000000 {
		t.Fatalf("Meso = %d, want 1000000", rm.Meso)
	}
}

// atlas-data omits `meso` (omitempty) for every non-sack cash item; the mirror
// must decode that as 0 rather than erroring.
func TestRestModelMesoAbsentIsZero(t *testing.T) {
	var rm RestModel
	if err := json.Unmarshal([]byte(`{"stateChangeItem":0,"bgmPath":"","protectTime":7}`), &rm); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if rm.Meso != 0 {
		t.Fatalf("Meso = %d, want 0", rm.Meso)
	}
}

// TestRestModel_DecodesNpc locks the channel-side mirror of atlas-data's
// cash_items "npc" attribute (task-221). A missing field here decodes to 0
// silently and the remote-merchant arm would reject every use.
func TestRestModel_DecodesNpc(t *testing.T) {
	var m RestModel
	if err := json.Unmarshal([]byte(`{"npc":9090000,"protectTime":0}`), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m.Npc != 9090000 {
		t.Errorf("Npc = %d, want 9090000", m.Npc)
	}
}
