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
