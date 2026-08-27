package reactor

import (
	"atlas-reactors/character"
	"atlas-reactors/character/mock"
	"atlas-reactors/reactor/data/area"
	"atlas-reactors/reactor/data/point"
	"atlas-reactors/reactor/data/state"
	"context"
	"errors"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// errTestPosition is the sentinel error returned by the position-error test
// case's mocked characterProcessor.
var errTestPosition = errors.New("position lookup failed")

// stateBoxTouchArea is the state-0 touch area used by every TestTouch case:
// TL(-50,-50) / BR(50,50) around a reactor placed at (100,100), i.e. a
// world-space box of [50,150] x [50,150].
func stateBoxTouchArea() area.RestModel {
	return area.RestModel{
		TL: point.RestModel{X: -50, Y: -50},
		BR: point.RestModel{X: 50, Y: 50},
	}
}

// testCharacterProcessorFunc builds a characterProcessor seam replacement
// whose Position always returns the given coordinates/error.
func testCharacterProcessorFunc(x int16, y int16, err error) func(l logrus.FieldLogger, ctx context.Context) character.Processor {
	return func(_ logrus.FieldLogger, _ context.Context) character.Processor {
		return &mock.ProcessorMock{
			PositionFunc: func(_ uint32) (int16, int16, error) {
				return x, y, err
			},
		}
	}
}

// testCharacterProcessorThatFatals builds a characterProcessor seam
// replacement whose Position call fails the test immediately -- used to
// assert a code path never reaches the character read.
func testCharacterProcessorThatFatals(t *testing.T) func(l logrus.FieldLogger, ctx context.Context) character.Processor {
	return func(_ logrus.FieldLogger, _ context.Context) character.Processor {
		return &mock.ProcessorMock{
			PositionFunc: func(_ uint32) (int16, int16, error) {
				t.Fatal("Position should not have been called")
				return 0, 0, nil
			},
		}
	}
}

func TestTouch(t *testing.T) {
	tests := []struct {
		name            string
		unknownReactor  bool
		activateByTouch bool
		touchAreaInfo   map[int8]area.RestModel
		stateInfo       map[int8][]state.RestModel
		charX           int16
		charY           int16
		positionErr     error
		expectExists    bool
		expectState     int8
	}{
		{
			name:            "accepts",
			activateByTouch: true,
			touchAreaInfo:   map[int8]area.RestModel{0: stateBoxTouchArea()},
			stateInfo: map[int8][]state.RestModel{
				0: {{Type: 6, NextState: 1, ActiveSkills: []uint32{}}},
			},
			charX:        100,
			charY:        100,
			expectExists: true,
			expectState:  1,
		},
		{
			name:            "rejects_flag_unset",
			activateByTouch: false,
			touchAreaInfo:   map[int8]area.RestModel{0: stateBoxTouchArea()},
			stateInfo: map[int8][]state.RestModel{
				0: {{Type: 6, NextState: 1, ActiveSkills: []uint32{}}},
			},
			charX:        100,
			charY:        100,
			expectExists: true,
			expectState:  0,
		},
		{
			name:            "rejects_outside_bounds",
			activateByTouch: true,
			touchAreaInfo:   map[int8]area.RestModel{0: stateBoxTouchArea()},
			stateInfo: map[int8][]state.RestModel{
				0: {{Type: 6, NextState: 1, ActiveSkills: []uint32{}}},
			},
			charX:        500,
			charY:        500,
			expectExists: true,
			expectState:  0,
		},
		{
			name:            "rejects_on_left_edge",
			activateByTouch: true,
			touchAreaInfo:   map[int8]area.RestModel{0: stateBoxTouchArea()},
			stateInfo: map[int8][]state.RestModel{
				0: {{Type: 6, NextState: 1, ActiveSkills: []uint32{}}},
			},
			charX:        49,
			charY:        100,
			expectExists: true,
			expectState:  0,
		},
		{
			name:            "accepts_on_boundary",
			activateByTouch: true,
			touchAreaInfo:   map[int8]area.RestModel{0: stateBoxTouchArea()},
			stateInfo: map[int8][]state.RestModel{
				0: {{Type: 6, NextState: 1, ActiveSkills: []uint32{}}},
			},
			charX:        50,
			charY:        100,
			expectExists: true,
			expectState:  1,
		},
		{
			name:            "rejects_missing_area",
			activateByTouch: true,
			touchAreaInfo:   map[int8]area.RestModel{},
			stateInfo: map[int8][]state.RestModel{
				0: {{Type: 6, NextState: 1, ActiveSkills: []uint32{}}},
			},
			charX:        100,
			charY:        100,
			expectExists: true,
			expectState:  0,
		},
		{
			name:            "rejects_position_error",
			activateByTouch: true,
			touchAreaInfo:   map[int8]area.RestModel{0: stateBoxTouchArea()},
			stateInfo: map[int8][]state.RestModel{
				0: {{Type: 6, NextState: 1, ActiveSkills: []uint32{}}},
			},
			positionErr:  errTestPosition,
			expectExists: true,
			expectState:  0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setupTestRegistry(t)
			l := setupTestLogger()
			ten := setupTestTenant()
			ctx := setupTestContext(ten)

			original := characterProcessor
			characterProcessor = testCharacterProcessorFunc(tc.charX, tc.charY, tc.positionErr)
			t.Cleanup(func() { characterProcessor = original })

			d := newTestData(t, tc.stateInfo, nil, nil, tc.touchAreaInfo, tc.activateByTouch)

			f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(1000000)).Build()
			builder := NewBuilder(ten, f, 6109013, "touch-reactor").
				SetState(0).SetPosition(100, 100).SetDelay(0).SetData(d)
			created, err := GetRegistry().Create(ten, builder)
			if err != nil {
				t.Fatalf("Create failed: %v", err)
			}

			// Producer errors are tolerated the same way the Hit tests
			// tolerate them; assertions are on registry state.
			_ = NewProcessor(l, ctx).Touch(created.Id(), 1000, true)

			got, err := GetRegistry().Get(ten, created.Id())
			if !tc.expectExists {
				if err == nil {
					t.Fatal("expected reactor to not exist")
				}
				return
			}
			if err != nil {
				t.Fatalf("expected reactor to still exist; got error: %v", err)
			}
			if got.State() != tc.expectState {
				t.Fatalf("state = %d, want %d", got.State(), tc.expectState)
			}
		})
	}

	t.Run("rejects_unknown_reactor", func(t *testing.T) {
		setupTestRegistry(t)
		l := setupTestLogger()
		ten := setupTestTenant()
		ctx := setupTestContext(ten)

		original := characterProcessor
		characterProcessor = testCharacterProcessorFunc(100, 100, nil)
		t.Cleanup(func() { characterProcessor = original })

		err := NewProcessor(l, ctx).Touch(999999999, 1000, true)
		if err == nil {
			t.Fatal("expected error for unknown reactor, got nil")
		}
	})
}

// TestTouch_SkillGatedStateAdvances is the FR-16 regression guard: Touch must
// not consult ActiveSkills. If it reused Hit's skill-gating predicate, it
// would fall through to TriggerAndDestroy on the "no matching event" branch
// -- the exact inversion this task exists to prevent.
func TestTouch_SkillGatedStateAdvances(t *testing.T) {
	setupTestRegistry(t)
	l := setupTestLogger()
	ten := setupTestTenant()
	ctx := setupTestContext(ten)

	original := characterProcessor
	characterProcessor = testCharacterProcessorFunc(100, 100, nil)
	t.Cleanup(func() { characterProcessor = original })

	d := newTestData(t,
		map[int8][]state.RestModel{
			0: {{Type: 6, NextState: 1, ActiveSkills: []uint32{9001000}}},
			1: {{Type: 7, NextState: 0, ActiveSkills: []uint32{9001000}}},
		},
		nil, nil,
		map[int8]area.RestModel{0: stateBoxTouchArea()},
		true,
	)

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(1000000)).Build()
	builder := NewBuilder(ten, f, 6109013, "skill-gated-touch-reactor").
		SetState(0).SetPosition(100, 100).SetDelay(0).SetData(d)
	created, err := GetRegistry().Create(ten, builder)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// skillId is not in play at all on the touch path.
	_ = NewProcessor(l, ctx).Touch(created.Id(), 1000, true)

	got, err := GetRegistry().Get(ten, created.Id())
	if err != nil {
		t.Fatalf("reactor should still exist after touch; got error: %v", err)
	}
	if got.State() != 1 {
		t.Fatalf("state = %d, want 1", got.State())
	}
}

// TestTouch_EmptyStateIsNoOp is the OQ-6 guard: an empty state on touch must
// not fall through to TriggerAndDestroy -- that stays Hit's alone.
func TestTouch_EmptyStateIsNoOp(t *testing.T) {
	setupTestRegistry(t)
	l := setupTestLogger()
	ten := setupTestTenant()
	ctx := setupTestContext(ten)

	original := characterProcessor
	characterProcessor = testCharacterProcessorFunc(100, 100, nil)
	t.Cleanup(func() { characterProcessor = original })

	d := newTestData(t,
		map[int8][]state.RestModel{},
		nil, nil,
		map[int8]area.RestModel{0: stateBoxTouchArea()},
		true,
	)

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(1000000)).Build()
	builder := NewBuilder(ten, f, 6109013, "empty-state-touch-reactor").
		SetState(0).SetPosition(100, 100).SetDelay(0).SetData(d)
	created, err := GetRegistry().Create(ten, builder)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	err = NewProcessor(l, ctx).Touch(created.Id(), 1000, true)
	if err != nil {
		t.Fatalf("Touch on empty state should be a no-op, got error: %v", err)
	}

	got, err := GetRegistry().Get(ten, created.Id())
	if err != nil {
		t.Fatalf("reactor should still exist; got error: %v", err)
	}
	if got.State() != 0 {
		t.Fatalf("state = %d, want 0", got.State())
	}
}

// TestTouch_Idempotence walks the cyclic 6109013 shape: state 0 -(type 6)->
// state 1, state 1 -(type 7)-> state 0, touch areas on both states.
func TestTouch_Idempotence(t *testing.T) {
	setupTestRegistry(t)
	l := setupTestLogger()
	ten := setupTestTenant()
	ctx := setupTestContext(ten)

	original := characterProcessor
	characterProcessor = testCharacterProcessorFunc(100, 100, nil)
	t.Cleanup(func() { characterProcessor = original })

	d := newTestData(t,
		map[int8][]state.RestModel{
			0: {{Type: 6, NextState: 1, ActiveSkills: []uint32{}}},
			1: {{Type: 7, NextState: 0, ActiveSkills: []uint32{}}},
		},
		nil, nil,
		map[int8]area.RestModel{
			0: stateBoxTouchArea(),
			1: stateBoxTouchArea(),
		},
		true,
	)

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(1000000)).Build()
	builder := NewBuilder(ten, f, 6109013, "idempotent-touch-reactor").
		SetState(0).SetPosition(100, 100).SetDelay(0).SetData(d)
	created, err := GetRegistry().Create(ten, builder)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	p := NewProcessor(l, ctx)

	// Step 1: enter -> state 1.
	_ = p.Touch(created.Id(), 1000, true)
	got, _ := GetRegistry().Get(ten, created.Id())
	if got.State() != 1 {
		t.Fatalf("after step 1: state = %d, want 1", got.State())
	}

	// Step 2: enter again -> still latched, still state 1.
	_ = p.Touch(created.Id(), 1000, true)
	got, _ = GetRegistry().Get(ten, created.Id())
	if got.State() != 1 {
		t.Fatalf("after step 2: state = %d, want 1", got.State())
	}

	// Step 3: leave -> clears the latch, no error, state unchanged.
	if err := p.Touch(created.Id(), 1000, false); err != nil {
		t.Fatalf("leave should not error, got: %v", err)
	}
	got, _ = GetRegistry().Get(ten, created.Id())
	if got.State() != 1 {
		t.Fatalf("after step 3: state = %d, want 1", got.State())
	}

	// Step 4: enter again -> state 1's type-7 event cycles back to state 0.
	_ = p.Touch(created.Id(), 1000, true)
	got, _ = GetRegistry().Get(ten, created.Id())
	if got.State() != 0 {
		t.Fatalf("after step 4: state = %d, want 0", got.State())
	}
}

// TestTouch_LeavingIsCheap asserts the leave path short-circuits before any
// character read (design §6.1): a leave touch on a non-touch-activated
// reactor must never call Position.
func TestTouch_LeavingIsCheap(t *testing.T) {
	setupTestRegistry(t)
	l := setupTestLogger()
	ten := setupTestTenant()
	ctx := setupTestContext(ten)

	original := characterProcessor
	characterProcessor = testCharacterProcessorThatFatals(t)
	t.Cleanup(func() { characterProcessor = original })

	d := newTestData(t,
		map[int8][]state.RestModel{
			0: {{Type: 6, NextState: 1, ActiveSkills: []uint32{}}},
		},
		nil, nil, nil, false,
	)

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(1000000)).Build()
	builder := NewBuilder(ten, f, 6109013, "leave-touch-reactor").
		SetState(0).SetPosition(100, 100).SetDelay(0).SetData(d)
	created, err := GetRegistry().Create(ten, builder)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := NewProcessor(l, ctx).Touch(created.Id(), 1000, false); err != nil {
		t.Fatalf("leave should not error, got: %v", err)
	}
}
