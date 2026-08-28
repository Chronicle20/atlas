package npc

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// TestInMapModelProvider_FiltersImitateTemplates asserts that a map NPC list
// mixing Player NPC imitate-pool placeholders (design §4.2:
// 9901000-9906599) with ordinary WZ NPC templates yields only the ordinary
// ones. This is the fix for task-251 bug report §2: those placeholders exist
// purely so the client's CNpcPool overlay can paint a deployed Player NPC's
// look onto them, and leaving them in the list the channel spawns from would
// double-spawn (and double-elect a controller for) every deployed Player
// NPC.
func TestInMapModelProvider_FiltersImitateTemplates(t *testing.T) {
	npcs := []Model{
		{id: 1, template: 9901000}, // imitate placeholder, occupied
		{id: 2, template: 9901001}, // imitate placeholder, unoccupied
		{id: 3, template: 9906599}, // imitate placeholder, upper bound
		{id: 4, template: 100100},  // ordinary WZ NPC
		{id: 5, template: 9900999}, // just below the imitate pool
		{id: 6, template: 9906600}, // just above the imitate pool
	}

	filtered, err := model.FilteredProvider[Model](model.FixedProvider[[]Model](npcs), model.Filters[Model](notImitateTemplate))()
	if err != nil {
		t.Fatalf("FilteredProvider() unexpected err = %v", err)
	}

	wantIds := map[uint32]bool{4: true, 5: true, 6: true}
	if len(filtered) != len(wantIds) {
		t.Fatalf("filtered = %+v, want %d entries matching %v", filtered, len(wantIds), wantIds)
	}
	for _, m := range filtered {
		if !wantIds[m.Id()] {
			t.Errorf("unexpected NPC [%d] template [%d] survived the imitate-template filter", m.Id(), m.Template())
		}
	}
}
