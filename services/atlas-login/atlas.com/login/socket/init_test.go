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
// CreateSocketService wires Create/DestroyByIdWithSpan/SendPing directly
// from this context, bypassing socket/handler.AdaptHandler entirely, so
// this path must also originate the environment from env.Self() — a fix
// round found it missing here even though socket/handler's per-request
// path already had it.
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
