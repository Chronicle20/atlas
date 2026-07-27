package skill

import "testing"

func TestResurrectionIds(t *testing.T) {
	if GmResurrectionId != Id(9001005) {
		t.Fatalf("GmResurrectionId = %d, want 9001005", GmResurrectionId)
	}
	if BishopResurrectionId != Id(2321006) {
		t.Fatalf("BishopResurrectionId = %d, want 2321006", BishopResurrectionId)
	}
	if SuperGmResurrectionId != Id(9101005) {
		t.Fatalf("SuperGmResurrectionId = %d, want 9101005", SuperGmResurrectionId)
	}
}

func TestResurrectionRegistryEntries(t *testing.T) {
	for _, id := range []Id{BishopResurrectionId, GmResurrectionId, SuperGmResurrectionId} {
		s, ok := Skills[id]
		if !ok {
			t.Fatalf("Skills[%d] missing", id)
		}
		if s.Id() != id {
			t.Fatalf("Skills[%d].Id() = %d, want %d", id, s.Id(), id)
		}
	}
}
