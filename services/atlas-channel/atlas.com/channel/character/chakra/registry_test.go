package chakra

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func testTenant(t *testing.T) tenant.Model {
	t.Helper()
	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tn
}

// newRegistry builds an isolated Registry so tests never share the
// process-wide singleton.
func newRegistry() *Registry {
	return &Registry{entries: make(map[Key]Entry)}
}

func TestStartAndGet(t *testing.T) {
	r := newRegistry()
	tn := testTenant(t)
	now := time.Now()

	r.Start(tn, 42, 3, 99, 92, now)

	e, ok := r.Get(tn, 42, now)
	if !ok {
		t.Fatal("Get after Start returned ok=false, want true")
	}
	if e.SkillLevel != 3 || e.X != 99 || e.Y != 92 {
		t.Fatalf("entry = %+v, want SkillLevel=3 X=99 Y=92", e)
	}
}

func TestGetMissingCharacter(t *testing.T) {
	r := newRegistry()
	tn := testTenant(t)
	if _, ok := r.Get(tn, 42, time.Now()); ok {
		t.Fatal("Get on an empty registry returned ok=true, want false")
	}
}

// TestLazyExpiry pins that correctness does not depend on the sweeper: an
// entry older than TTL reads as absent even though it is still in the map.
func TestLazyExpiry(t *testing.T) {
	r := newRegistry()
	tn := testTenant(t)
	start := time.Now()
	r.Start(tn, 42, 1, 200, 9, start)

	if _, ok := r.Get(tn, 42, start.Add(TTL-time.Millisecond)); !ok {
		t.Fatal("entry just inside TTL read as absent")
	}
	if _, ok := r.Get(tn, 42, start.Add(TTL)); ok {
		t.Fatal("entry exactly at TTL read as present, want expired")
	}
	if _, ok := r.Get(tn, 42, start.Add(TTL+time.Second)); ok {
		t.Fatal("entry past TTL read as present, want expired")
	}
}

func TestClearReportsPresence(t *testing.T) {
	r := newRegistry()
	tn := testTenant(t)
	r.Start(tn, 42, 1, 99, 68, time.Now())

	if !r.Clear(tn, 42) {
		t.Fatal("Clear on a present entry returned false, want true")
	}
	if r.Clear(tn, 42) {
		t.Fatal("Clear on an absent entry returned true, want false")
	}
	if _, ok := r.Get(tn, 42, time.Now()); ok {
		t.Fatal("entry still present after Clear")
	}
}

// TestTenantIsolation pins that the same characterId in two tenants are two
// independent windows (NFR: multi-tenancy).
func TestTenantIsolation(t *testing.T) {
	r := newRegistry()
	a := testTenant(t)
	b := testTenant(t)
	now := time.Now()

	r.Start(a, 42, 1, 200, 9, now)

	if _, ok := r.Get(b, 42, now); ok {
		t.Fatal("tenant b sees tenant a's entry")
	}
	if r.Clear(b, 42) {
		t.Fatal("Clear in tenant b reported tenant a's entry")
	}
	if _, ok := r.Get(a, 42, now); !ok {
		t.Fatal("tenant a's entry was disturbed by tenant b")
	}
}

// TestSweepEvictsOnlyExpired pins that the sweeper bounds memory without
// touching live windows.
func TestSweepEvictsOnlyExpired(t *testing.T) {
	r := newRegistry()
	tn := testTenant(t)
	old := time.Now()
	r.Start(tn, 1, 1, 99, 68, old)
	r.Start(tn, 2, 1, 99, 68, old.Add(TTL))

	if n := r.Sweep(old.Add(TTL)); n != 1 {
		t.Fatalf("Sweep evicted %d entries, want 1", n)
	}
	if _, ok := r.Get(tn, 2, old.Add(TTL)); !ok {
		t.Fatal("Sweep evicted a live entry")
	}
}

// TestClearIsIdempotentAcrossInterruptSources pins PRD FR-5.1/FR-5.5: the
// move, map-change and session-destroy paths all call Clear, and only the
// first one to arrive reports an interrupt — so the log line is emitted once
// and a second caller is a harmless no-op.
func TestClearIsIdempotentAcrossInterruptSources(t *testing.T) {
	r := newRegistry()
	tn := testTenant(t)
	r.Start(tn, 42, 1, 99, 68, time.Now())

	first := r.Clear(tn, 42)  // movement
	second := r.Clear(tn, 42) // map change arriving after
	third := r.Clear(tn, 42)  // session destroy

	if !first {
		t.Fatal("the first interrupt did not report an open window")
	}
	if second || third {
		t.Fatal("a follow-up interrupt reported an open window")
	}
}

// TestConcurrentAccess is the -race guard: the registry is written by the
// prepare path and read/cleared by the damage, move and use paths plus the
// sweeper, all concurrently.
func TestConcurrentAccess(t *testing.T) {
	r := newRegistry()
	tn := testTenant(t)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		id := uint32(i%3 + 1)
		wg.Add(4)
		go func() { defer wg.Done(); r.Start(tn, id, 1, 99, 68, time.Now()) }()
		go func() { defer wg.Done(); r.Get(tn, id, time.Now()) }()
		go func() { defer wg.Done(); r.Clear(tn, id) }()
		go func() { defer wg.Done(); r.Sweep(time.Now()) }()
	}
	wg.Wait()
}

// TestStartSweeperIsIdempotent pins the fix for the fan-out bug found in
// review: every atlas-channel socket listener calls StartSweeper on the same
// process-wide registry, so without a guard each one would spawn its own
// ticker against the same singleton. sweeperOnce must make only the first
// caller actually spawn the loop, regardless of how many callers race to
// start it — asserted here by overriding spawnSweeper (the seam StartSweeper
// delegates to) with a counter instead of merely checking StartSweeper
// doesn't panic.
func TestStartSweeperIsIdempotent(t *testing.T) {
	origSpawn := spawnSweeper
	t.Cleanup(func() { spawnSweeper = origSpawn })

	var mu sync.Mutex
	spawnCount := 0
	spawnSweeper = func(_ *Registry, _ logrus.FieldLogger, _ context.Context) {
		mu.Lock()
		spawnCount++
		mu.Unlock()
	}

	r := newRegistry()
	l := logrus.New()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	const callers = 8
	var wg sync.WaitGroup
	wg.Add(callers)
	for i := 0; i < callers; i++ {
		go func() {
			defer wg.Done()
			r.StartSweeper(l, ctx)
		}()
	}
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if spawnCount != 1 {
		t.Fatalf("StartSweeper called by %d concurrent listeners spawned the sweeper %d times, want exactly 1", callers, spawnCount)
	}
}

// TestStartSweeperOutlivesCallerContext pins the fix for the review finding
// that the sweeper was bound to the first caller's per-listener context: a
// tenant/listener teardown (a supported platform capability) must not stop
// the process-wide sweeper for every other tenant still writing to the
// singleton registry. StartSweeper must detach the context it hands to
// spawnSweeper from the caller's cancelation before the caller's context is
// cancelled, not merely happen to still be running afterward.
func TestStartSweeperOutlivesCallerContext(t *testing.T) {
	origSpawn := spawnSweeper
	t.Cleanup(func() { spawnSweeper = origSpawn })

	var mu sync.Mutex
	var spawnedCtx context.Context
	spawnSweeper = func(_ *Registry, _ logrus.FieldLogger, c context.Context) {
		mu.Lock()
		spawnedCtx = c
		mu.Unlock()
	}

	r := newRegistry()
	l := logrus.New()
	ctx, cancel := context.WithCancel(context.Background())

	r.StartSweeper(l, ctx)

	// Simulate the caller's listener tearing down immediately after
	// starting the sweeper.
	cancel()

	mu.Lock()
	defer mu.Unlock()
	if spawnedCtx == nil {
		t.Fatal("spawnSweeper was never called")
	}
	select {
	case <-spawnedCtx.Done():
		t.Fatal("sweeper context was cancelled when the caller's context was cancelled, want a detached context that survives caller teardown")
	default:
	}
	if err := spawnedCtx.Err(); err != nil {
		t.Fatalf("spawnedCtx.Err() = %v, want nil", err)
	}
}
