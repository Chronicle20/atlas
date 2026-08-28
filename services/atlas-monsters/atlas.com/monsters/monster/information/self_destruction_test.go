package information

import "testing"

func TestSelfDestructionPredicates(t *testing.T) {
	tests := []struct {
		name        string
		present     bool
		action      byte
		removeAfter int32
		hp          int32
		wantPresent bool
		wantOnHp    bool
		wantOnTimer bool
	}{
		{"absent", false, 0, -1, -1, false, false, false},
		{"hp threshold (Boomer 5100002)", true, 1, -1, 1800, true, true, false},
		{"timer only (9300166)", true, 4, 0, -1, true, false, true},
		{"both (9400566)", true, 3, 5, 1, true, true, false},
		{"present, hp 0", true, 1, -1, 0, true, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sd := NewSelfDestruction(tt.present, tt.action, tt.removeAfter, tt.hp)
			if got := sd.Present(); got != tt.wantPresent {
				t.Errorf("Present() = %v, want %v", got, tt.wantPresent)
			}
			if got := sd.OnHpThreshold(); got != tt.wantOnHp {
				t.Errorf("OnHpThreshold() = %v, want %v", got, tt.wantOnHp)
			}
			if got := sd.OnTimer(); got != tt.wantOnTimer {
				t.Errorf("OnTimer() = %v, want %v", got, tt.wantOnTimer)
			}
		})
	}
}

func TestExtractMapsSelfDestruction(t *testing.T) {
	tests := []struct {
		name            string
		dto             selfDestruction
		wantPresent     bool
		wantAction      byte
		wantRemoveAfter int32
		wantHp          int32
	}{
		{"absent block", selfDestruction{Action: 0, RemoveAfter: -1, Hp: -1}, false, 0, -1, -1},
		{"Boomer", selfDestruction{Action: 1, RemoveAfter: -1, Hp: 1800}, true, 1, -1, 1800},
		{"timer mob", selfDestruction{Action: 4, RemoveAfter: 0, Hp: -1}, true, 4, 0, -1},
		// Design D2 claims the mandated predicate (Hp > -1 || RemoveAfter > -1) is
		// false under the OLD pre-task-1 atlas-data absent sentinel {0,0,0}, making a
		// rolling deploy safe either order. That claim does not hold arithmetically:
		// 0 > -1 is true, so this DTO reports Present() == true under the mandated
		// formula. Pinned here as the VERIFIED behavior (not the design doc's claim) —
		// see task-4-report.md for the discrepancy raised to the controller.
		{"legacy absent (pre-Task-1 atlas-data)", selfDestruction{Action: 0, RemoveAfter: 0, Hp: 0}, true, 0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rm := RestModel{SelfDestruction: tt.dto}
			m, err := Extract(rm)
			if err != nil {
				t.Fatalf("Extract() error = %v", err)
			}
			sd := m.SelfDestruction()
			if got := sd.Present(); got != tt.wantPresent {
				t.Errorf("Present() = %v, want %v", got, tt.wantPresent)
			}
			if got := sd.Action(); got != tt.wantAction {
				t.Errorf("Action() = %v, want %v", got, tt.wantAction)
			}
			if got := sd.RemoveAfter(); got != tt.wantRemoveAfter {
				t.Errorf("RemoveAfter() = %v, want %v", got, tt.wantRemoveAfter)
			}
			if got := sd.Hp(); got != tt.wantHp {
				t.Errorf("Hp() = %v, want %v", got, tt.wantHp)
			}
		})
	}
}

func TestBuilderSetsSelfDestruction(t *testing.T) {
	m := NewBuilder().SetSelfDestruction(NewSelfDestruction(true, 3, -1, 5000)).Build()
	sd := m.SelfDestruction()
	if !sd.Present() {
		t.Errorf("Present() = false, want true")
	}
	if sd.Action() != 3 {
		t.Errorf("Action() = %v, want 3", sd.Action())
	}
	if sd.Hp() != 5000 {
		t.Errorf("Hp() = %v, want 5000", sd.Hp())
	}
}
