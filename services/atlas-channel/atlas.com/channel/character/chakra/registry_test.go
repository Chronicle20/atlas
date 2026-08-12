package chakra

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

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
