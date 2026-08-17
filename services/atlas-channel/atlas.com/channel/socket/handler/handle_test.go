package handler

import (
	"testing"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
)

// TestSocketSessionContextCarriesThisPodsEnvironment pins FR-2.2: a channel
// socket session originates the environment from this pod's own
// ATLAS_ENVIRONMENT, not from any inbound header — a game client connects
// to the PR's own channel service.
func TestSocketSessionContextCarriesThisPodsEnvironment(t *testing.T) {
	t.Setenv(env.SelfVar, "pr-123")
	ctx := newSessionContext(testTenant(t))
	if got := env.MustFromContext(ctx); got != env.Id("pr-123") {
		t.Fatalf("environment = %q, want \"pr-123\" from ATLAS_ENVIRONMENT", got)
	}
}

func TestSocketSessionContextOnMainIsTheLegacyValue(t *testing.T) {
	t.Setenv(env.SelfVar, "")
	ctx := newSessionContext(testTenant(t))
	if got := env.MustFromContext(ctx); got != env.Id("") {
		t.Fatalf("environment = %q, want the empty id", got)
	}
}
