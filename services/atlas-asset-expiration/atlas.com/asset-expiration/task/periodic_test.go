package task

import (
	"os"
	"strings"
	"testing"
)

// TestCheckAllSessionsUsesForEachOwnedEnvironment pins design C4/FR-6.1:
// tenant resolution must go through service.ForEachOwnedEnvironment so both
// the owned-environment set and each environment's tenant set are resolved
// fresh on every tick, not cached and closed over. A tenant provisioned
// after this pod started must be picked up without a restart, because a
// baseline pod cannot be redeployed to serve an ephemeral environment (G7,
// NG6).
func TestCheckAllSessionsUsesForEachOwnedEnvironment(t *testing.T) {
	src, err := os.ReadFile("periodic.go")
	if err != nil {
		t.Fatalf("read periodic.go: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "service.ForEachOwnedEnvironment") {
		t.Fatal("checkAllSessions does not use service.ForEachOwnedEnvironment")
	}
}
