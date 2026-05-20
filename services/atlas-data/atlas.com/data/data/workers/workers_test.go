package workers

import "testing"

func TestRegisteredSize(t *testing.T) {
	if len(Registered) != 10 {
		t.Fatalf("registered = %d, want 10", len(Registered))
	}
}

func TestRegisteredUniqueArchives(t *testing.T) {
	seen := map[string]bool{}
	for _, w := range Registered {
		if seen[w.ArchiveName()] {
			t.Fatalf("duplicate archive: %s", w.ArchiveName())
		}
		seen[w.ArchiveName()] = true
	}
}

func TestRegisteredUniqueNames(t *testing.T) {
	seen := map[string]bool{}
	for _, w := range Registered {
		if seen[w.Name()] {
			t.Fatalf("duplicate name: %s", w.Name())
		}
		seen[w.Name()] = true
	}
}
