package handler

import (
	"testing"

	"github.com/google/uuid"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func testTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Register(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("register test tenant: %v", err)
	}
	return tm
}

// TestSocketSessionContextCarriesThisPodsEnvironment pins FR-2.2: a login
// socket session originates the environment from this pod's own
// ATLAS_ENVIRONMENT, not from any inbound header — a game client connects
// to the PR's own login service.
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
