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
	// atlas-character imports github.com/Chronicle20/atlas/libs/atlas-service
	// under the local alias "lifecycle" (a local "atlas-character/service"
	// package already occupies the "service" identifier) — matching
	// tools/env-bootstrap-guard.sh's alias-aware scan rather than the
	// recipe's literal "service." string.
	if !strings.Contains(string(src), "lifecycle.WithEnvironmentRegistry(serviceName)") {
		t.Fatal("main.go does not pass lifecycle.WithEnvironmentRegistry to Bootstrap")
	}
}
