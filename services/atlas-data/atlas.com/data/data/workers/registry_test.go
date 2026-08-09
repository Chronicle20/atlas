package workers

import "testing"

// The run-record roster (task-203) is derived from this list, so a twelfth
// worker must widen the denominator without any second edit.
func TestRegisteredNamesMatchesRegistered(t *testing.T) {
	got := RegisteredNames()
	if len(got) != len(Registered) {
		t.Fatalf("RegisteredNames returned %d names, want %d", len(got), len(Registered))
	}
	for i, w := range Registered {
		if got[i] != w.Name() {
			t.Fatalf("index %d: got %q, want %q", i, got[i], w.Name())
		}
		if got[i] == "" {
			t.Fatalf("index %d: worker has an empty name", i)
		}
	}
}
