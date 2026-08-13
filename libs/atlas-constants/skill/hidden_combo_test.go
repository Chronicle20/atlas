package skill

import "testing"

func TestAranHiddenComboParent(t *testing.T) {
	cases := []struct {
		name   string
		id     Id
		parent Id
		ok     bool
	}{
		{"full swing double swing", AranStage3FullSwingDoubleSwingId, AranStage3FullSwingId, true},
		{"full swing triple swing", AranStage3FullSwingTripleSwingId, AranStage3FullSwingId, true},
		{"over swing double swing", AranStage4OverswingDoubleSwingId, AranStage4OverSwingId, true},
		{"over swing triple swing", AranStage4OverswingTripleSwingId, AranStage4OverSwingId, true},
		{"the parent itself is not a variant", AranStage4OverSwingId, 0, false},
		{"unrelated aran skill", AranStage1ComboAbilityId, 0, false},
		{"unrelated skill", Id(1001004), 0, false},
		{"zero", Id(0), 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			parent, ok := AranHiddenComboParent(tc.id)
			if ok != tc.ok {
				t.Fatalf("AranHiddenComboParent(%d): ok = %v, want %v", tc.id, ok, tc.ok)
			}
			if parent != tc.parent {
				t.Fatalf("AranHiddenComboParent(%d): parent = %d, want %d", tc.id, parent, tc.parent)
			}
		})
	}
}

// Every hidden combo id the SP-reset exclusion list names must resolve to a
// parent: the two lists describe the same four skills, and a variant that is
// excluded from SP reset but has no parent to read a level from would be
// rejected by the attack pipeline's ownership gate.
func TestAranHiddenComboParentCoversPointResetExclusions(t *testing.T) {
	for _, id := range []Id{Id(21110007), Id(21110008), Id(21120009), Id(21120010)} {
		if !IsPointResetExcluded(id) {
			t.Fatalf("skill [%d] is expected to be point-reset excluded", id)
		}
		if _, ok := AranHiddenComboParent(id); !ok {
			t.Fatalf("skill [%d] is point-reset excluded as an Aran hidden combo but has no parent", id)
		}
	}
}
