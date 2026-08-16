package requests

import (
	"context"
	"testing"
	"time"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
)

func TestRootUrlForWithNoEnvironmentIsUnchanged(t *testing.T) {
	// NFR-7: byte-identical to today for a legacy operation.
	t.Setenv("BASE_SERVICE_URL", "http://atlas-ingress.atlas-main.svc.cluster.local:80/api/")

	got, err := RootUrlFor(context.Background(), "characters")
	if err != nil {
		t.Fatalf("RootUrlFor: %v", err)
	}
	if want := RootUrl("characters"); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestRootUrlForTargetsTheEnvironmentsIngress(t *testing.T) {
	t.Setenv("BASE_SERVICE_URL", "http://atlas-ingress.atlas-main.svc.cluster.local:80/api/")
	reg := env.NewMapRegistry(env.Id("main"), time.Now)
	reg.Apply(env.Record{
		Name: "main", Baseline: "main",
		Namespace: "atlas-main", Phase: env.PhaseActive,
	})
	reg.Apply(env.Record{
		Name: "pr-123", Baseline: "main",
		Namespace: "atlas-pr-123", Phase: env.PhaseActive,
	})
	env.SetRegistry(reg)
	t.Cleanup(func() { env.SetRegistry(nil) })

	ctx := env.WithContext(context.Background(), env.Id("pr-123"))
	got, err := RootUrlFor(ctx, "characters")
	if err != nil {
		t.Fatalf("RootUrlFor: %v", err)
	}
	want := "http://atlas-ingress.atlas-pr-123.svc.cluster.local:80/api/"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestRootUrlForANonOverriddenServiceStillTargetsTheEnvironmentsIngress(t *testing.T) {
	// The regression this whole mechanism turns on. pr-123 overrides only
	// atlas-character. A call to atlas-monsters must STILL leave through
	// pr-123's ingress — that ingress's NS_ATLAS_MONSTERS then points at
	// atlas-main, so the request reaches the baseline pod WITH the
	// ENVIRONMENT header intact. Resolving the upstream namespace here
	// instead would strip the environment from the operation.
	t.Setenv("BASE_SERVICE_URL", "http://atlas-ingress.atlas-main.svc.cluster.local:80/api/")
	reg := env.NewMapRegistry(env.Id("main"), time.Now)
	reg.Apply(env.Record{
		Name: "main", Baseline: "main",
		Namespace: "atlas-main", Phase: env.PhaseActive,
	})
	reg.Apply(env.Record{
		Name: "pr-123", Baseline: "main", Namespace: "atlas-pr-123",
		Overrides: map[string]string{"atlas-character": "atlas-pr-123"},
		Phase:     env.PhaseActive,
	})
	env.SetRegistry(reg)
	t.Cleanup(func() { env.SetRegistry(nil) })

	ctx := env.WithContext(context.Background(), env.Id("pr-123"))
	got, err := RootUrlFor(ctx, "monsters")
	if err != nil {
		t.Fatalf("RootUrlFor: %v", err)
	}
	want := "http://atlas-ingress.atlas-pr-123.svc.cluster.local:80/api/"
	if got != want {
		t.Fatalf("got %q, want %q — a non-overridden service must not be "+
			"resolved to the baseline's ingress", got, want)
	}
}

func TestRootUrlForAnUnknownEnvironmentErrorsAndNeverFallsBack(t *testing.T) {
	// G4 / FR-3.5: an operation must never silently transition to main.
	t.Setenv("BASE_SERVICE_URL", "http://atlas-ingress.atlas-main.svc.cluster.local:80/api/")
	reg := env.NewMapRegistry(env.Id("main"), time.Now)
	reg.Apply(env.Record{
		Name: "main", Baseline: "main",
		Namespace: "atlas-main", Phase: env.PhaseActive,
	})
	env.SetRegistry(reg)
	t.Cleanup(func() { env.SetRegistry(nil) })

	ctx := env.WithContext(context.Background(), env.Id("pr-999"))
	got, err := RootUrlFor(ctx, "characters")
	if err == nil {
		t.Fatalf("got %q with no error; want an error, never a baseline URL", got)
	}
}

func TestRootUrlForHonoursAPerDomainOverride(t *testing.T) {
	// A *_SERVICE_URL override bypasses the ingress entirely (local debug).
	// It must keep winning, and must NOT be namespace-rewritten.
	t.Setenv("BASE_SERVICE_URL", "http://atlas-ingress.atlas-main.svc.cluster.local:80/api/")
	t.Setenv("CHARACTERS_SERVICE_URL", "http://localhost:9999/api/")

	ctx := env.WithContext(context.Background(), env.Id("pr-123"))
	got, err := RootUrlFor(ctx, "characters")
	if err != nil {
		t.Fatalf("RootUrlFor: %v", err)
	}
	if got != "http://localhost:9999/api/" {
		t.Fatalf("got %q, want the per-domain override verbatim", got)
	}
}
