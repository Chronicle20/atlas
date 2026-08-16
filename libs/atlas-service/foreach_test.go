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
