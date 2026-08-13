package cash

import (
	"encoding/json"
	"testing"
)

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
