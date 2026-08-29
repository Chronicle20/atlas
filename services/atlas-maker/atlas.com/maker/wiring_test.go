package main

import (
	"os"
	"strings"
	"testing"
)

// TestMainWiresTheEnvironmentRegistry pins the one line every service must
// carry. It is a source assertion rather than a behavioural one because the
// wiring's effect is inert until an Environment record exists (FR-1.8), so
// there is nothing observable to assert at this point in the migration.
func TestMainWiresTheEnvironmentRegistry(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	if !strings.Contains(string(src), "service.WithEnvironmentRegistry(serviceName)") {
		t.Fatal("main.go does not pass service.WithEnvironmentRegistry to Bootstrap")
	}
}

// TestServiceBootstraps asserts the router main.go builds resolves the
// service's base route prefix without panicking. Domain route tests live
// with their domains (Tasks 17, 18, 24); this is a smoke test only.
func TestServiceBootstraps(t *testing.T) {
	s := GetServer()
	if s.GetPrefix() != "/api/" {
		t.Fatalf("expected base route prefix /api/, got %q", s.GetPrefix())
	}
	if s.GetBaseURL() != "" {
		t.Fatalf("expected empty base URL, got %q", s.GetBaseURL())
	}
}
