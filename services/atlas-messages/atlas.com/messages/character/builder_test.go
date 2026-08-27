package character

import "testing"

// TestBuild_MissingId asserts the identity invariant Build() enforces: a
// Builder with no id set fails rather than silently producing a zero-value
// character.
func TestBuild_MissingId(t *testing.T) {
	_, err := NewBuilder().Build()
	if err == nil {
		t.Fatal("Build() error = nil, want error for missing id")
	}
}

// TestBuild_WithId asserts Build() succeeds once the identity invariant is
// satisfied.
func TestBuild_WithId(t *testing.T) {
	m, err := NewBuilder().SetId(1).Build()
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if m.Id() != 1 {
		t.Errorf("m.Id() = %d, want 1", m.Id())
	}
}
