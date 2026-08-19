package session

import "testing"

// TestGm_Accessor — /find reads the requester's GM flag from the session
// (PRD FR-3). The field existed but had no getter.
func TestGm_Accessor(t *testing.T) {
	var s Model
	if s.Gm() {
		t.Error("zero-value session reports Gm() = true")
	}
	promoted := s.setGm(true)
	if !promoted.Gm() {
		t.Error("after setGm(true), Gm() = false")
	}
	if s.Gm() {
		t.Error("setGm mutated the receiver; Model is meant to be immutable")
	}
}
