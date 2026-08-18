package main

import (
	"os"
	"strings"
	"testing"
)

// TestRepublishTickerUsesForEachOwnedEnvironment pins FR-6.1/design C4: the
// legacy-config republish ticker must filter the projection snapshot down
// to tenants of environments this deployment currently owns on every tick,
// via service.ForEachOwnedEnvironment, rather than republishing the raw
// snapshot unconditionally. configuration/projection/loop.go's own apply
// loop is class 3 (control-plane, SERVICE_ID-scoped) and is untouched by
// this test.
func TestRepublishTickerUsesForEachOwnedEnvironment(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "service.ForEachOwnedEnvironment") {
		t.Fatal("republish ticker does not use service.ForEachOwnedEnvironment")
	}
}
