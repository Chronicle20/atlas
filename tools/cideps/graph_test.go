package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseAtlasRequires_DirectAndIndirect(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "go.mod")
	contents := `module atlas-svc

go 1.25.5

require (
	github.com/Chronicle20/atlas/libs/atlas-kafka v0.0.0
	github.com/Chronicle20/atlas/libs/atlas-tenant v0.0.0
	github.com/google/uuid v1.6.0
)

require (
	github.com/Chronicle20/atlas/libs/atlas-retry v0.0.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
)
`
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := parseAtlasRequires(path)
	if err != nil {
		t.Fatalf("parseAtlasRequires: %v", err)
	}
	want := map[string]struct{}{
		"atlas-kafka":  {},
		"atlas-tenant": {},
		"atlas-retry":  {},
	}
	if len(got) != len(want) {
		t.Fatalf("len(got)=%d want=%d: %v", len(got), len(want), got)
	}
	for k := range want {
		if _, ok := got[k]; !ok {
			t.Errorf("missing %q", k)
		}
	}
}

func TestParseAtlasRequires_MalformedFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "go.mod")
	if err := os.WriteFile(path, []byte("this is not a go.mod"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := parseAtlasRequires(path); err == nil {
		t.Fatal("expected error for malformed go.mod, got nil")
	}
}

func TestBuildGraph_Simple(t *testing.T) {
	g, err := BuildGraph("testdata/simple")
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}
	if got := g.Libs(); !equalSet(got, []string{"lib-a", "lib-b"}) {
		t.Errorf("libs=%v want [lib-a lib-b]", got)
	}
	if got := g.Services(); !equalSet(got, []string{"svc-a"}) {
		t.Errorf("services=%v want [svc-a]", got)
	}
	if got := g.DirectDeps("svc-a"); !equalSet(got, []string{"lib-b"}) {
		t.Errorf("deps(svc-a)=%v want [lib-b]", got)
	}
	if got := g.DirectDeps("lib-b"); !equalSet(got, []string{"lib-a"}) {
		t.Errorf("deps(lib-b)=%v want [lib-a]", got)
	}
	if got := g.DirectDeps("lib-a"); len(got) != 0 {
		t.Errorf("deps(lib-a)=%v want empty", got)
	}
}

// equalSet returns true if a and b contain the same elements (order-insensitive).
func equalSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	m := make(map[string]int)
	for _, s := range a {
		m[s]++
	}
	for _, s := range b {
		m[s]--
	}
	for _, v := range m {
		if v != 0 {
			return false
		}
	}
	return true
}
