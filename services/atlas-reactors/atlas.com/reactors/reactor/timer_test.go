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
