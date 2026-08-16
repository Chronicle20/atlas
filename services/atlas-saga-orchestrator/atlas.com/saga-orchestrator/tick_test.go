package main

import (
	"os"
	"strings"
	"testing"
)

// TestReapAndRecoverUseForEachOwnedEnvironment pins FR-6.1/design §8.3: the
// saga sweeper is a persisted-work path — recoverSagas and reapTimedOutSagas
// must reconstruct each row's environment from the tenant on the row via
// service.ForEachOwnedEnvironment, not from env.Self(). A saga row belonging
// to a tenant of an environment this deployment does not own must never be
// visited.
func TestReapAndRecoverUseForEachOwnedEnvironment(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	s := string(src)
	if strings.Count(s, "service.ForEachOwnedEnvironment") < 2 {
		t.Fatal("recoverSagas and reapTimedOutSagas do not both use service.ForEachOwnedEnvironment")
	}
}
