package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// withRegistry installs r as the process-wide registry for the duration of
// the test and restores the legacy default afterward -- env.SetRegistry is
// process-wide state (registry.go), so tests must not leak it.
func withRegistry(t *testing.T, r env.Registry) {
	t.Helper()
	env.SetRegistry(r)
	t.Cleanup(func() { env.SetRegistry(nil) })
}

func TestTenantEnvironmentResolvesTheOwningEnvironment(t *testing.T) {
	// The bug this pins: a baseline pod's background sweep for a
	// sparse-environment tenant must stamp that tenant's environment, not
	// the pod's own.
	r := env.NewMapRegistry(env.Id("main"), time.Now)
	r.ApplyTenant("t-1", env.Id("pr-1412"))
	withRegistry(t, r)

	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("build tenant: %v", err)
	}
	r.ApplyTenant(tn.Id().String(), env.Id("pr-1412"))

	ctx := tenant.WithContext(context.Background(), tn)
	got := TenantEnvironment(ctx)

	if id := env.MustFromContext(got); id != env.Id("pr-1412") {
		t.Fatalf("TenantEnvironment stamped %q, want %q", id, "pr-1412")
	}
}

func TestTenantEnvironmentFallsBackToSelfWhenNoTenantIsOnContext(t *testing.T) {
	r := env.NewMapRegistry(env.Id("main"), time.Now)
	withRegistry(t, r)

	got := TenantEnvironment(context.Background())

	if id := env.MustFromContext(got); id != env.Self() {
		t.Fatalf("TenantEnvironment stamped %q, want self %q", id, env.Self())
	}
}

func TestTenantEnvironmentFallsBackToSelfForAnUnknownTenant(t *testing.T) {
	r := env.NewMapRegistry(env.Id("main"), time.Now)
	withRegistry(t, r)

	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("build tenant: %v", err)
	}

	ctx := tenant.WithContext(context.Background(), tn)
	got := TenantEnvironment(ctx)

	if id := env.MustFromContext(got); id != env.Self() {
		t.Fatalf("TenantEnvironment stamped %q, want self %q", id, env.Self())
	}
}

func TestTenantEnvironmentFallsBackToSelfForALegacyTenant(t *testing.T) {
	r := env.NewMapRegistry(env.Id("main"), time.Now)
	withRegistry(t, r)

	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("build tenant: %v", err)
	}
	r.ApplyTenant(tn.Id().String(), env.Id(""))

	ctx := tenant.WithContext(context.Background(), tn)
	got := TenantEnvironment(ctx)

	if id := env.MustFromContext(got); id != env.Self() {
		t.Fatalf("TenantEnvironment stamped %q, want self %q", id, env.Self())
	}
}
