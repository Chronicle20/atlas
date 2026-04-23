package reactor

import (
	"atlas-reactors/reactor/data"
	"atlas-reactors/reactor/data/state"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// timerTestData builds a reactor data.Model where state 0 auto-advances to
// state 1 via a timer, and state 1 has no events.
func timerTestData(t *testing.T, timeoutMs int32) data.Model {
	t.Helper()
	m, err := data.Extract(data.RestModel{
		Name: "timer-test",
		StateInfo: map[int8][]state.RestModel{
			0: {{Type: 101, NextState: 1, ActiveSkills: []uint32{}}},
		},
		TimeoutInfo:          map[int8]int32{0: timeoutMs},
		TimeoutNextStateInfo: map[int8]int8{0: 1},
	})
	if err != nil {
		t.Fatalf("data.Extract: %v", err)
	}
	return m
}

// TestScheduleStateTimeout_FiresAndTransitions verifies the core loop: a state
// with Timeout+TimeoutNextState set arms a timer; on fire, the reactor moves
// to the configured next state.
func TestScheduleStateTimeout_FiresAndTransitions(t *testing.T) {
	setupTestRegistry(t)
	l := setupTestLogger()
	ten := setupTestTenant()
	ctx := setupTestContext(ten)

	d := timerTestData(t, 50) // 50ms

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(1000000)).Build()
	builder := NewModelBuilder(ten, f, 9101000, "moon-bunny").
		SetState(0).SetPosition(100, 100).SetDelay(0).SetData(d)
	created, err := GetRegistry().Create(ten, builder)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	scheduleStateTimeout(l, ctx, created)

	// Wait for the timer to fire and the transition to complete.
	time.Sleep(150 * time.Millisecond)

	got, err := GetRegistry().Get(ten, created.Id())
	if err != nil {
		t.Fatalf("reactor gone after timer fire; timer should transition not destroy for type-101 cyclic: %v", err)
	}
	if got.State() != 1 {
		t.Fatalf("state = %d, want 1 (timer-driven transition)", got.State())
	}

	cancelAllStateTimeouts() // cleanup
}

// TestCancelStateTimeout_StopsPendingFire verifies that cancel prevents the
// transition from happening.
func TestCancelStateTimeout_StopsPendingFire(t *testing.T) {
	setupTestRegistry(t)
	l := setupTestLogger()
	ten := setupTestTenant()
	ctx := setupTestContext(ten)

	d := timerTestData(t, 200) // long enough to cancel before fire

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(1000000)).Build()
	builder := NewModelBuilder(ten, f, 9101000, "moon-bunny").
		SetState(0).SetPosition(100, 100).SetDelay(0).SetData(d)
	created, _ := GetRegistry().Create(ten, builder)

	scheduleStateTimeout(l, ctx, created)
	time.Sleep(50 * time.Millisecond)
	cancelStateTimeout(created.Id())

	time.Sleep(250 * time.Millisecond)

	got, _ := GetRegistry().Get(ten, created.Id())
	if got.State() != 0 {
		t.Fatalf("state = %d, want 0 (timer was cancelled before firing)", got.State())
	}
}

// TestCancelAllStateTimeouts_DoesNotPanicWhenEmpty verifies teardown safety.
func TestCancelAllStateTimeouts_DoesNotPanicWhenEmpty(t *testing.T) {
	cancelAllStateTimeouts() // should be a no-op with no panic
}

// TestHit_CancelsPendingStateTimer verifies that hitting a reactor with a
// pending timer cancels that timer (a new one may be armed for the new state,
// but the original fire MUST NOT happen).
func TestHit_CancelsPendingStateTimer(t *testing.T) {
	setupTestRegistry(t)
	l := setupTestLogger()
	ten := setupTestTenant()
	ctx := setupTestContext(ten)

	// State 0: type-0 hit event -> state 2, AND a type-101 timer -> state 1.
	// State 2 has an event back to state 0 so it's not a terminal state — that
	// keeps the hit from triggering destroy via TriggerAndDestroy, which would
	// remove the reactor before we can assert state.
	// If the timer were to fire we'd see state 1; if the hit lands first and
	// cancels the timer, we see state 2 without any further transition.
	m, err := data.Extract(data.RestModel{
		Name: "hit-cancels-timer",
		StateInfo: map[int8][]state.RestModel{
			0: {{Type: 0, NextState: 2, ActiveSkills: []uint32{}}},
			2: {{Type: 0, NextState: 0, ActiveSkills: []uint32{}}},
		},
		TimeoutInfo:          map[int8]int32{0: 100},
		TimeoutNextStateInfo: map[int8]int8{0: 1},
	})
	if err != nil {
		t.Fatalf("data.Extract: %v", err)
	}

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(1000000)).Build()
	builder := NewModelBuilder(ten, f, 9999, "test").
		SetState(0).SetPosition(100, 100).SetDelay(0).SetData(m)
	created, _ := GetRegistry().Create(ten, builder)
	scheduleStateTimeout(l, ctx, created) // Arm the timer manually since we created via the registry directly.

	// Create has armed a 100ms timer. Hit immediately — should cancel it.
	// Tolerate Kafka producer error (no broker reachable in unit tests);
	// the registry mutations under test happen before the producer call.
	_ = Hit(l)(ctx)(created.Id(), 0, 0)

	// Wait well past the original timer's fire time.
	time.Sleep(200 * time.Millisecond)

	got, err := GetRegistry().Get(ten, created.Id())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	// State must be 2 (hit result). If the timer had fired we'd see 1 or 3.
	if got.State() != 2 {
		t.Fatalf("state = %d, want 2 (hit landed, timer should have been cancelled)", got.State())
	}

	cancelAllStateTimeouts()
}

// TestDestroy_CancelsPendingStateTimer verifies Destroy cancels the timer.
func TestDestroy_CancelsPendingStateTimer(t *testing.T) {
	setupTestRegistry(t)
	l := setupTestLogger()
	ten := setupTestTenant()
	ctx := setupTestContext(ten)

	d := timerTestData(t, 100)

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(1000000)).Build()
	builder := NewModelBuilder(ten, f, 9101000, "moon-bunny").
		SetState(0).SetPosition(100, 100).SetDelay(0).SetData(d)
	created, _ := GetRegistry().Create(ten, builder)
	scheduleStateTimeout(l, ctx, created) // Arm the timer manually since we created via the registry directly.

	// Create armed the timer. Destroy should cancel.
	// Tolerate Kafka producer error (no broker reachable in unit tests);
	// the registry mutations under test (Remove + cancelStateTimeout) happen
	// before the producer call.
	_ = Destroy(l)(ctx)(created)

	// No panic / no attempt to transition a deleted reactor.
	time.Sleep(200 * time.Millisecond)

	if _, err := GetRegistry().Get(ten, created.Id()); err == nil {
		t.Fatal("reactor should be gone after Destroy")
	}
}
