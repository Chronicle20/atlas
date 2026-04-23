package main

import (
	"os"
	"path/filepath"
	"testing"
)

// repoRoot returns the path to the Atlas repo root, or skips the test if we
// can't locate it (e.g. the tool is vendored into another repo).
func repoRoot(t *testing.T) string {
	t.Helper()
	// tools/cideps is two levels deep from repo root.
	abs, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(abs, "go.work")); err != nil {
		t.Skipf("no go.work at %s, skipping real-repo test", abs)
	}
	return abs
}

func TestRealRepo_KnownEdges(t *testing.T) {
	root := repoRoot(t)
	g, err := BuildGraph(root)
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}

	// Non-empty collections.
	if len(g.Services()) < 10 {
		t.Errorf("expected many services, got %d: %v", len(g.Services()), g.Services())
	}
	if len(g.Libs()) < 10 {
		t.Errorf("expected many libs, got %d: %v", len(g.Libs()), g.Libs())
	}

	// Known edges — update these if the corresponding go.mod files change.
	mustDep := func(mod, dep string) {
		t.Helper()
		for _, d := range g.DirectDeps(mod) {
			if d == dep {
				return
			}
		}
		t.Errorf("%s does not directly require %s; deps=%v", mod, dep, g.DirectDeps(mod))
	}
	mustDep("atlas-saga", "atlas-constants")
	mustDep("atlas-account", "atlas-kafka")
	mustDep("atlas-account", "atlas-tenant")
	mustDep("atlas-account", "atlas-rest")
}
