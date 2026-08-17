package socket

import (
	"context"
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

// TestNewListenerContextCarriesThisPodsEnvironment pins the buildListener
// socket-registration path (main.go calls socket.NewListenerContext):
// CreateSocketService wires Create/Destroy/SendPing directly from this
// context, bypassing socket/handler.AdaptHandler entirely, so this path
// must also originate the environment from env.Self() -- a second,
// parallel context path alongside the per-request handler path.
func TestNewListenerContextCarriesThisPodsEnvironment(t *testing.T) {
	t.Setenv(env.SelfVar, "pr-123")
	ctx := NewListenerContext(context.Background(), testTenant(t))
	if got := env.MustFromContext(ctx); got != env.Id("pr-123") {
		t.Fatalf("environment = %q, want \"pr-123\" from ATLAS_ENVIRONMENT", got)
	}
}

func TestNewListenerContextOnMainIsTheLegacyValue(t *testing.T) {
	t.Setenv(env.SelfVar, "")
	ctx := NewListenerContext(context.Background(), testTenant(t))
	if got := env.MustFromContext(ctx); got != env.Id("") {
		t.Fatalf("environment = %q, want the empty id", got)
	}
}

// TestWithSelfEnvironmentCarriesThisPodsEnvironment pins the helper that
// lets a package outside env-domain-guard's permitted list (like
// character/combo's DecayTick) originate env.Self() on a per-event
// context without importing atlas-env directly.
func TestWithSelfEnvironmentCarriesThisPodsEnvironment(t *testing.T) {
	t.Setenv(env.SelfVar, "pr-123")
	ctx := WithSelfEnvironment(context.Background())
	if got := env.MustFromContext(ctx); got != env.Id("pr-123") {
		t.Fatalf("environment = %q, want \"pr-123\" from ATLAS_ENVIRONMENT", got)
	}
}
