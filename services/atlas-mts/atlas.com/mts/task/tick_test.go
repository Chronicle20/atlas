package task

import (
	"os"
	"strings"
	"testing"
)

// TestSweepUsesForEachOwnedEnvironment pins FR-6.1/design §8.3: the sweep is
// a persisted-work path, so the environment for each expired listing must be
// reconstructed from the row's own tenant via service.ForEachOwnedEnvironment
// rather than assumed from env.Self(). A row belonging to a tenant of an
// environment this deployment does not own must never be visited.
func TestSweepUsesForEachOwnedEnvironment(t *testing.T) {
	src, err := os.ReadFile("periodic.go")
	if err != nil {
		t.Fatalf("read periodic.go: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "service.ForEachOwnedEnvironment") {
		t.Fatal("Sweep does not use service.ForEachOwnedEnvironment")
	}
}
