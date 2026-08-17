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
//
// atlas-storage imports github.com/Chronicle20/atlas/libs/atlas-service as
// "lifecycle" (main.go already has a local "atlas-storage/service" package
// imported as "service", so the shared library needs a different local
// name) -- so the literal call here is lifecycle.WithEnvironmentRegistry,
// not service.WithEnvironmentRegistry.
func TestMainWiresTheEnvironmentRegistry(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	if !strings.Contains(string(src), "lifecycle.WithEnvironmentRegistry(serviceName)") {
		t.Fatal("main.go does not pass lifecycle.WithEnvironmentRegistry to Bootstrap")
	}
}
