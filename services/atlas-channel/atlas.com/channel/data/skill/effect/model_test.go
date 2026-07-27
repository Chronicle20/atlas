package effect

import "testing"

// TestModelY pins the Y() getter added for MP Recovery (task-151): y is
// already populated from REST (rest.go); only the getter was missing.
func TestModelY(t *testing.T) {
	m := Model{y: 55}
	if got := m.Y(); got != 55 {
		t.Fatalf("Y() = %d, want 55", got)
	}
}
