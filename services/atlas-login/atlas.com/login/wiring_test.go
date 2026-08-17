// services/atlas-login/atlas.com/login/wiring_test.go
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
