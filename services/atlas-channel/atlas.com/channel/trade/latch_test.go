package trade

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func testTenant(t *testing.T) tenant.Model {
	t.Helper()
	m, err := tenant.Register(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("register tenant: %v", err)
	}
	return m
}

func newLatch() *CreateLatch {
	return &CreateLatch{inFlight: make(map[latchKey]chan struct{})}
}

// TestAwaitSettledWaitsForAnArmedCreate is the defect: the invite arm must not
// return while the create it belongs to is still on its way to the topic.
func TestAwaitSettledWaitsForAnArmedCreate(t *testing.T) {
	r := newLatch()
	tn := testTenant(t)
	release := r.Begin(tn, character.Id(2))

	released := make(chan struct{})
	go func() {
		time.Sleep(30 * time.Millisecond)
		close(released)
		release()
	}()

	if !r.AwaitSettled(tn, character.Id(2), 50*time.Millisecond, time.Second) {
		t.Fatal("AwaitSettled: got false, want true (a create was in flight)")
	}
	select {
	case <-released:
	default:
		t.Error("AwaitSettled returned before the create released the latch")
	}
}

// TestAwaitSettledWaitsForACreateThatHasNotArmedYet pins the reason an
// "is it armed right now?" test is not enough: atlas-socket dispatches one
// goroutine per packet, so the invite can reach the latch before the create
// does.
func TestAwaitSettledWaitsForACreateThatHasNotArmedYet(t *testing.T) {
	r := newLatch()
	tn := testTenant(t)

	go func() {
		time.Sleep(20 * time.Millisecond)
		release := r.Begin(tn, character.Id(2))
		time.Sleep(20 * time.Millisecond)
		release()
	}()

	start := time.Now()
	if !r.AwaitSettled(tn, character.Id(2), 500*time.Millisecond, time.Second) {
		t.Fatal("AwaitSettled: got false, want true (a create armed during the arrive window)")
	}
	if elapsed := time.Since(start); elapsed < 40*time.Millisecond {
		t.Errorf("AwaitSettled returned after %s; want it to outlast the create", elapsed)
	}
}

// TestAwaitSettledGivesUpWhenNoCreateArrives pins the bound: an invite with no
// create behind it is delayed by the arrive window and then proceeds, so a
// client that never sends mode 0 still gets its refusal rather than a hang.
func TestAwaitSettledGivesUpWhenNoCreateArrives(t *testing.T) {
	r := newLatch()
	tn := testTenant(t)

	start := time.Now()
	if r.AwaitSettled(tn, character.Id(2), 20*time.Millisecond, time.Second) {
		t.Error("AwaitSettled: got true, want false (no create was ever in flight)")
	}
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Errorf("AwaitSettled blocked for %s; want it bounded by the arrive window", elapsed)
	}
}

// TestAwaitSettledIsBoundedByTheSettleTimeout pins that a create which never
// releases — a panicking arm, a wedged REST probe — cannot strand the invite.
func TestAwaitSettledIsBoundedByTheSettleTimeout(t *testing.T) {
	r := newLatch()
	tn := testTenant(t)
	_ = r.Begin(tn, character.Id(2))

	start := time.Now()
	if !r.AwaitSettled(tn, character.Id(2), 50*time.Millisecond, 30*time.Millisecond) {
		t.Fatal("AwaitSettled: got false, want true (a create was in flight)")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("AwaitSettled blocked for %s; want it bounded by the settle timeout", elapsed)
	}
}

// TestBeginIsScopedPerCharacter pins that one character's create cannot delay
// another's invite — the latch is keyed, not global.
func TestBeginIsScopedPerCharacter(t *testing.T) {
	r := newLatch()
	tn := testTenant(t)
	_ = r.Begin(tn, character.Id(1))

	if r.Armed(tn, character.Id(2)) {
		t.Error("character 2 armed by character 1's create")
	}
	if !r.Armed(tn, character.Id(1)) {
		t.Error("character 1 not armed by its own create")
	}
}

// TestBeginIsScopedPerTenant pins the same for tenants sharing a pod.
func TestBeginIsScopedPerTenant(t *testing.T) {
	r := newLatch()
	one := testTenant(t)
	two := testTenant(t)
	_ = r.Begin(one, character.Id(2))

	if r.Armed(two, character.Id(2)) {
		t.Error("tenant two armed by tenant one's create")
	}
}

// TestNestedBeginCannotOpenTheGateEarly pins that a second create for the same
// character gets a no-op release: only the latch's owner clears it, so an
// overlapping create cannot let an invite past the first one.
func TestNestedBeginCannotOpenTheGateEarly(t *testing.T) {
	r := newLatch()
	tn := testTenant(t)
	first := r.Begin(tn, character.Id(2))
	second := r.Begin(tn, character.Id(2))

	second()
	if !r.Armed(tn, character.Id(2)) {
		t.Error("the nested release cleared the first create's latch")
	}
	first()
	if r.Armed(tn, character.Id(2)) {
		t.Error("the owner's release did not clear the latch")
	}
}

// TestReleaseIsIdempotent pins that a doubled release — a defer plus an early
// return on the same path — does not panic closing an already-closed channel,
// and does not clear a LATER create's latch.
func TestReleaseIsIdempotent(t *testing.T) {
	r := newLatch()
	tn := testTenant(t)
	release := r.Begin(tn, character.Id(2))
	release()
	if r.Armed(tn, character.Id(2)) {
		t.Fatal("latch still armed after release")
	}

	again := r.Begin(tn, character.Id(2))
	release()
	if !r.Armed(tn, character.Id(2)) {
		t.Error("the stale release cleared the second create's latch")
	}
	again()
	if r.Armed(tn, character.Id(2)) {
		t.Error("latch still armed after the second create released")
	}
}
