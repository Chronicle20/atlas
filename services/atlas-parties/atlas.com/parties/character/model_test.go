package character

import "testing"

func TestChangeGmPreservesOtherFields(t *testing.T) {
	m := Model{id: 42, name: "Atlas", level: 200, partyId: 7, online: true, gm: 1}
	out := m.ChangeGm(0)
	if out.GM() != 0 {
		t.Errorf("expected gm 0, got %d", out.GM())
	}
	if out.Id() != 42 || out.Name() != "Atlas" || out.Level() != 200 || out.PartyId() != 7 || !out.Online() {
		t.Error("ChangeGm must not mutate unrelated fields")
	}
}
