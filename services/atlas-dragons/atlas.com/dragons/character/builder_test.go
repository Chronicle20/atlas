package character

import "testing"

// TestBuild_ZeroId asserts the identity invariant Build() enforces: a
// Builder constructed with id == 0 fails rather than silently producing a
// zero-value character.
func TestBuild_ZeroId(t *testing.T) {
	_, err := NewBuilder(0).Build()
	if err == nil {
		t.Fatal("Build() error = nil, want error for zero id")
	}
}

// TestBuild_WithId asserts Build() succeeds once the identity invariant is
// satisfied.
func TestBuild_WithId(t *testing.T) {
	m, err := NewBuilder(1).Build()
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if m.Id() != 1 {
		t.Errorf("m.Id() = %d, want 1", m.Id())
	}
}
