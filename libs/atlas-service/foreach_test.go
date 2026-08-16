package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func testTenant(t *testing.T) tenant.Model {
	t.Helper()
	m, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("unable to build test tenant: %v", err)
	}
	return m
}

func oneTenantPerEnvironment(t *testing.T) TenantLister {
	t.Helper()
	return func(context.Context) ([]tenant.Model, error) {
		return []tenant.Model{testTenant(t)}, nil
	}
}

func twoTenantsPerEnvironment(t *testing.T) TenantLister {
	t.Helper()
	return func(context.Context) ([]tenant.Model, error) {
		return []tenant.Model{testTenant(t), testTenant(t)}, nil
	}
}

func TestForEachOwnedEnvironmentRunsOncePerTenantPerOwnedEnvironment(t *testing.T) {
	r := env.NewMapRegistry(env.Id("main"), time.Now)
	r.Apply(env.Record{Name: "main", Baseline: "main", Namespace: "atlas-main", Phase: env.PhaseActive})
	r.Apply(env.Record{Name: "pr-123", Baseline: "main", Namespace: "atlas-pr-123", Phase: env.PhaseActive})
	env.SetRegistry(r)
	t.Cleanup(func() { env.SetRegistry(nil) })

	seen := map[string]int{}
	ForEachOwnedEnvironment(testLogger(t), context.Background(), "atlas-monsters",
		twoTenantsPerEnvironment(t), func(ctx context.Context) {
			seen[string(env.MustFromContext(ctx))]++
		})

	if seen["main"] != 2 || seen["pr-123"] != 2 {
		t.Fatalf("seen = %v, want 2 iterations each for main and pr-123", seen)
	}
}

func TestForEachOwnedEnvironmentRunsSerially(t *testing.T) {
	// The helper must preserve each loop's existing shape: today every
	// class-1 loop is `for _, t := range tenants { work }`. An unsynchronised
	// counter is the assertion — this test is run under -race, so a
	// concurrent implementation fails it rather than passing flakily.
	r := env.NewMapRegistry(env.Id("main"), time.Now)
	r.Apply(env.Record{Name: "main", Baseline: "main", Namespace: "atlas-main", Phase: env.PhaseActive})
	r.Apply(env.Record{Name: "pr-123", Baseline: "main", Namespace: "atlas-pr-123", Phase: env.PhaseActive})
	env.SetRegistry(r)
	t.Cleanup(func() { env.SetRegistry(nil) })

	inFlight := 0
	maxInFlight := 0
	ForEachOwnedEnvironment(testLogger(t), context.Background(), "atlas-monsters",
		twoTenantsPerEnvironment(t), func(context.Context) {
			inFlight++
			if inFlight > maxInFlight {
				maxInFlight = inFlight
			}
			time.Sleep(time.Millisecond)
			inFlight--
		})

	if maxInFlight != 1 {
		t.Fatalf("maxInFlight = %d, want 1 — the default helper must not "+
			"parallelise tenants that the caller's loop ran serially", maxInFlight)
	}
}

func TestForEachOwnedEnvironmentSkipsEnvironmentsThisDeploymentDoesNotOwn(t *testing.T) {
	// FR-6.3: a baseline deployment must not originate work for an
	// environment that overrides its service.
	r := env.NewMapRegistry(env.Id("main"), time.Now)
	r.Apply(env.Record{Name: "main", Baseline: "main", Namespace: "atlas-main", Phase: env.PhaseActive})
	r.Apply(env.Record{
		Name: "pr-123", Baseline: "main", Namespace: "atlas-pr-123",
		Overrides: map[string]string{"atlas-monsters": "atlas-pr-123"}, Phase: env.PhaseActive,
	})
	env.SetRegistry(r)
	t.Cleanup(func() { env.SetRegistry(nil) })

	seen := map[string]int{}
	ForEachOwnedEnvironment(testLogger(t), context.Background(), "atlas-monsters",
		oneTenantPerEnvironment(t), func(ctx context.Context) {
			seen[string(env.MustFromContext(ctx))]++
		})

	if seen["pr-123"] != 0 {
		t.Fatalf("baseline originated %d iterations for an environment that overrides it", seen["pr-123"])
	}
	if seen["main"] != 1 {
		t.Fatalf("seen[main] = %d, want 1", seen["main"])
	}
}

func TestForEachOwnedEnvironmentWithNoRecordsDoesExactlyTodaysWork(t *testing.T) {
	// FR-6.6 / NFR-2: a deployment owning only main performs the same work
	// it does today, and no extra background work.
	env.SetRegistry(nil)

	iterations := 0
	ForEachOwnedEnvironment(testLogger(t), context.Background(), "atlas-monsters",
		func(context.Context) ([]tenant.Model, error) { return []tenant.Model{testTenant(t)}, nil },
		func(context.Context) { iterations++ })

	if iterations != 1 {
		t.Fatalf("iterations = %d, want 1", iterations)
	}
}

func TestForEachOwnedEnvironmentIsolatesFaults(t *testing.T) {
	// FR-6.5: one environment's panic does not stop another's iteration.
	r := env.NewMapRegistry(env.Id("main"), time.Now)
	r.Apply(env.Record{Name: "main", Baseline: "main", Namespace: "atlas-main", Phase: env.PhaseActive})
	r.Apply(env.Record{Name: "pr-123", Baseline: "main", Namespace: "atlas-pr-123", Phase: env.PhaseActive})
	env.SetRegistry(r)
	t.Cleanup(func() { env.SetRegistry(nil) })

	completed := 0
	ForEachOwnedEnvironment(testLogger(t), context.Background(), "atlas-monsters",
		oneTenantPerEnvironment(t), func(ctx context.Context) {
			if env.MustFromContext(ctx) == env.Id("main") {
				panic("boom")
			}
			completed++
		})

	if completed != 1 {
		t.Fatalf("completed = %d; a panic in one environment stopped another", completed)
	}
}

func TestForEachOwnedEnvironmentConcurrentlyRunsBodiesInParallel(t *testing.T) {
	// The opt-in variant, for loops that ALREADY parallelised their tenants.
	r := env.NewMapRegistry(env.Id("main"), time.Now)
	r.Apply(env.Record{Name: "main", Baseline: "main", Namespace: "atlas-main", Phase: env.PhaseActive})
	r.Apply(env.Record{Name: "pr-123", Baseline: "main", Namespace: "atlas-pr-123", Phase: env.PhaseActive})
	env.SetRegistry(r)
	t.Cleanup(func() { env.SetRegistry(nil) })

	started := make(chan struct{}, 4)
	release := make(chan struct{})
	go func() {
		for i := 0; i < 4; i++ {
			<-started
		}
		close(release)
	}()

	ForEachOwnedEnvironmentConcurrently(testLogger(t), context.Background(), "atlas-monsters",
		twoTenantsPerEnvironment(t), func(context.Context) {
			started <- struct{}{}
			<-release // deadlocks unless all four run concurrently
		})
}

func TestForEachOwnedEnvironmentFiltersListerToItsOwnEnvironmentsTenants(t *testing.T) {
	// A populated registry with tenants projected across two environments:
	// each tenant must be visited exactly once, under its own environment,
	// even though the lister below is well-behaved and already scopes its
	// results per-context. This pins the happy path before the misbehaving
	// lister below pins the enforcement.
	r := env.NewMapRegistry(env.Id("main"), time.Now)
	r.Apply(env.Record{Name: "main", Baseline: "main", Namespace: "atlas-main", Phase: env.PhaseActive})
	r.Apply(env.Record{Name: "pr-123", Baseline: "main", Namespace: "atlas-pr-123", Phase: env.PhaseActive})

	mainTenant := testTenant(t)
	prTenant := testTenant(t)
	r.ApplyTenant(mainTenant.Id().String(), env.Id("main"))
	r.ApplyTenant(prTenant.Id().String(), env.Id("pr-123"))
	env.SetRegistry(r)
	t.Cleanup(func() { env.SetRegistry(nil) })

	lister := func(ctx context.Context) ([]tenant.Model, error) {
		switch env.MustFromContext(ctx) {
		case env.Id("main"):
			return []tenant.Model{mainTenant}, nil
		case env.Id("pr-123"):
			return []tenant.Model{prTenant}, nil
		default:
			return nil, nil
		}
	}

	seen := map[string]map[string]int{"main": {}, "pr-123": {}}
	ForEachOwnedEnvironment(testLogger(t), context.Background(), "atlas-monsters",
		lister, func(ctx context.Context) {
			e := string(env.MustFromContext(ctx))
			tm := tenant.MustFromContext(ctx)
			seen[e][tm.Id().String()]++
		})

	if seen["main"][mainTenant.Id().String()] != 1 {
		t.Fatalf("seen[main][mainTenant] = %d, want 1", seen["main"][mainTenant.Id().String()])
	}
	if seen["pr-123"][prTenant.Id().String()] != 1 {
		t.Fatalf("seen[pr-123][prTenant] = %d, want 1", seen["pr-123"][prTenant.Id().String()])
	}
	if len(seen["main"]) != 1 || len(seen["pr-123"]) != 1 {
		t.Fatalf("seen = %v, want exactly one tenant visited per environment", seen)
	}
}

func TestForEachOwnedEnvironmentFiltersOutTenantsWronglyReturnedForOtherEnvironments(t *testing.T) {
	// FR-7.3 / the defect this fix closes: a lister that ignores the
	// context environment and returns EVERY tenant it knows about (as the
	// marriages/asset-expiration local DB/session-backed listers do) must
	// still produce exactly one visit per (correct environment, tenant)
	// pair. The filter, not the lister, is what enforces the contract.
	r := env.NewMapRegistry(env.Id("main"), time.Now)
	r.Apply(env.Record{Name: "main", Baseline: "main", Namespace: "atlas-main", Phase: env.PhaseActive})
	r.Apply(env.Record{Name: "pr-123", Baseline: "main", Namespace: "atlas-pr-123", Phase: env.PhaseActive})

	mainTenant := testTenant(t)
	prTenant := testTenant(t)
	r.ApplyTenant(mainTenant.Id().String(), env.Id("main"))
	r.ApplyTenant(prTenant.Id().String(), env.Id("pr-123"))
	env.SetRegistry(r)
	t.Cleanup(func() { env.SetRegistry(nil) })

	// Misbehaving lister: ignores ctx, returns both tenants every time.
	allTenants := []tenant.Model{mainTenant, prTenant}
	lister := func(context.Context) ([]tenant.Model, error) {
		return allTenants, nil
	}

	seen := map[string]map[string]int{"main": {}, "pr-123": {}}
	ForEachOwnedEnvironment(testLogger(t), context.Background(), "atlas-monsters",
		lister, func(ctx context.Context) {
			e := string(env.MustFromContext(ctx))
			tm := tenant.MustFromContext(ctx)
			seen[e][tm.Id().String()]++
		})

	if seen["main"][mainTenant.Id().String()] != 1 {
		t.Fatalf("seen[main][mainTenant] = %d, want 1", seen["main"][mainTenant.Id().String()])
	}
	if seen["main"][prTenant.Id().String()] != 0 {
		t.Fatalf("seen[main][prTenant] = %d, want 0 — prTenant leaked into main's iteration", seen["main"][prTenant.Id().String()])
	}
	if seen["pr-123"][prTenant.Id().String()] != 1 {
		t.Fatalf("seen[pr-123][prTenant] = %d, want 1", seen["pr-123"][prTenant.Id().String()])
	}
	if seen["pr-123"][mainTenant.Id().String()] != 0 {
		t.Fatalf("seen[pr-123][mainTenant] = %d, want 0 — mainTenant leaked into pr-123's iteration", seen["pr-123"][mainTenant.Id().String()])
	}
}

func TestForEachOwnedEnvironmentLegacyRegistryVisitsEveryTenantUnfiltered(t *testing.T) {
	// The regression this fix could most easily (and most invisibly) cause:
	// against the legacy registry (no records, no tenant projections), the
	// unknown-tenant passthrough must admit every tenant. Dropping it would
	// silently filter every legacy iteration down to zero, stopping every
	// ticker on an unmigrated / never-configured deployment.
	env.SetRegistry(nil)

	seen := map[string]int{}
	ForEachOwnedEnvironment(testLogger(t), context.Background(), "atlas-monsters",
		twoTenantsPerEnvironment(t), func(ctx context.Context) {
			tm := tenant.MustFromContext(ctx)
			seen[tm.Id().String()]++
		})

	if len(seen) != 2 {
		t.Fatalf("seen = %v, want exactly 2 distinct tenants visited (legacy path unfiltered)", seen)
	}
	for id, n := range seen {
		if n != 1 {
			t.Fatalf("seen[%s] = %d, want 1", id, n)
		}
	}
}

func TestForEachOwnedEnvironmentReresolvesOwnershipEveryCall(t *testing.T) {
	// FR-6.4: a loop must not cache an ownership set across ticks. This is
	// the C4 defect stated as a test.
	r := env.NewMapRegistry(env.Id("main"), time.Now)
	r.Apply(env.Record{Name: "main", Baseline: "main", Namespace: "atlas-main", Phase: env.PhaseActive})
	env.SetRegistry(r)
	t.Cleanup(func() { env.SetRegistry(nil) })

	count := func() int {
		n := 0
		ForEachOwnedEnvironment(testLogger(t), context.Background(), "atlas-monsters",
			oneTenantPerEnvironment(t), func(context.Context) { n++ })
		return n
	}

	if got := count(); got != 1 {
		t.Fatalf("first tick = %d, want 1", got)
	}
	r.Apply(env.Record{Name: "pr-123", Baseline: "main", Namespace: "atlas-pr-123", Phase: env.PhaseActive})
	if got := count(); got != 2 {
		t.Fatalf("second tick = %d, want 2 — a new environment must be picked up without a restart", got)
	}
}
