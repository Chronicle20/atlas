package position

import (
	"errors"
	"testing"
)

// fr47Tuning is the FR-4.7 default tuning used throughout this file unless
// a test says otherwise.
var fr47Tuning = Tuning{InitialX: 262, InitialY: 262, AreaX: 320, AreaY: 160, AreaSteps: 4}

// identitySnap resolves every candidate to itself with a fixed foothold id,
// standing in for the injected ground-resolution round trip.
func identitySnap(points []Point) ([]Point, error) {
	out := make([]Point, len(points))
	for i, p := range points {
		out[i] = Point{X: p.X, CY: p.CY, Fh: 1}
	}
	return out, nil
}

func TestGridPositions(t *testing.T) {
	t.Run("pitch at step 0", func(t *testing.T) {
		dx, dy := GridPitch(fr47Tuning, 0)
		if dx != 320 || dy != 160 {
			t.Fatalf("got dx=%d dy=%d, want dx=320 dy=160", dx, dy)
		}
	})

	t.Run("pitch at step 1", func(t *testing.T) {
		dx, dy := GridPitch(fr47Tuning, 1)
		if dx != 160 || dy != 120 {
			t.Fatalf("got dx=%d dy=%d, want dx=160 dy=120", dx, dy)
		}
	})

	t.Run("pitch at step 2", func(t *testing.T) {
		dx, dy := GridPitch(fr47Tuning, 2)
		if dx != 106 || dy != 100 {
			t.Fatalf("got dx=%d dy=%d, want dx=106 dy=100", dx, dy)
		}
	})

	t.Run("pitch at step 3", func(t *testing.T) {
		dx, dy := GridPitch(fr47Tuning, 3)
		if dx != 80 || dy != 90 {
			t.Fatalf("got dx=%d dy=%d, want dx=80 dy=90", dx, dy)
		}
	})

	t.Run("first placement on an empty map", func(t *testing.T) {
		bounds := Rect{X: 0, Y: 0, W: 912, H: 592}

		p, err := NextGridPosition(fr47Tuning, bounds, 0, nil, identitySnap)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		wantX := bounds.X + fr47Tuning.InitialX
		wantY := bounds.Y + fr47Tuning.InitialY
		if p.X != wantX || p.CY != wantY {
			t.Fatalf("got (%d,%d), want (%d,%d)", p.X, p.CY, wantX, wantY)
		}
	})

	t.Run("second placement avoids the first", func(t *testing.T) {
		bounds := Rect{X: 0, Y: 0, W: 912, H: 592}
		dx, dy := GridPitch(fr47Tuning, 0)

		first, err := NextGridPosition(fr47Tuning, bounds, 0, nil, identitySnap)
		if err != nil {
			t.Fatalf("unexpected error placing first: %v", err)
		}
		firstRect := Rect{X: first.X, Y: first.CY, W: dx, H: dy}

		second, err := NextGridPosition(fr47Tuning, bounds, 0, []Rect{firstRect}, identitySnap)
		if err != nil {
			t.Fatalf("unexpected error placing second: %v", err)
		}
		secondRect := Rect{X: second.X, Y: second.CY, W: dx, H: dy}

		if rectsOverlap(firstRect, secondRect) {
			t.Fatalf("second placement %+v intersects first %+v", secondRect, firstRect)
		}
	})

	t.Run("no free slot at areaSteps", func(t *testing.T) {
		step := byte(4)
		dx, dy := GridPitch(fr47Tuning, step)
		bounds := Rect{
			X: 0, Y: 0,
			W: fr47Tuning.InitialX + dx,
			H: fr47Tuning.InitialY + dy,
		}
		occupied := Rect{
			X: bounds.X + fr47Tuning.InitialX,
			Y: bounds.Y + fr47Tuning.InitialY,
			W: dx, H: dy,
		}

		_, err := NextGridPosition(fr47Tuning, bounds, step, []Rect{occupied}, identitySnap)
		if !errors.Is(err, ErrMapFull) {
			t.Fatalf("got err=%v, want ErrMapFull", err)
		}
	})
}

func TestPodiumPositions(t *testing.T) {
	tests := []struct {
		name    string
		rank    uint32
		step    byte
		wantX   int16
		wantY   int16
		wantErr error
	}{
		{name: "platform 0, first slot", rank: 0, step: 1, wantX: 0, wantY: -47},
		{name: "platform 0, second slot", rank: 1, step: 2, wantX: 16, wantY: -47},
		{name: "platform 1", rank: 2, step: 1, wantX: 120, wantY: 40},
		{name: "platform 1 boundary", rank: 1, step: 1, wantX: -120, wantY: 40},
		{name: "step 0 is an error, not a panic", rank: 0, step: 0, wantErr: ErrInvalidStep},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := PodiumPosition(tt.rank, tt.step)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("got err=%v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if p.X != tt.wantX || p.CY != tt.wantY {
				t.Fatalf("got (%d,%d), want (%d,%d)", p.X, p.CY, tt.wantX, tt.wantY)
			}
		})
	}
}

func TestPodiumStepRaise(t *testing.T) {
	t.Run("decode", func(t *testing.T) {
		step, count := DecodePodiumState(EncodePodiumState(1, 3))
		if step != 1 || count != 3 {
			t.Fatalf("got step=%d count=%d, want step=1 count=3", step, count)
		}
		if got := EncodePodiumState(1, 3); got != 97 {
			t.Fatalf("EncodePodiumState(1, 3) = %d, want 97", got)
		}
	})

	t.Run("raise when count >= 3*step", func(t *testing.T) {
		newStep, raised, err := RaisePodiumStep(fr47Tuning, 1, 3)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !raised || newStep != 2 {
			t.Fatalf("got newStep=%d raised=%v, want newStep=2 raised=true", newStep, raised)
		}
	})

	t.Run("no raise below the threshold", func(t *testing.T) {
		newStep, raised, err := RaisePodiumStep(fr47Tuning, 2, 5)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if raised || newStep != 2 {
			t.Fatalf("got newStep=%d raised=%v, want newStep=2 raised=false", newStep, raised)
		}
	})

	t.Run("bounded by areaSteps", func(t *testing.T) {
		_, raised, err := RaisePodiumStep(fr47Tuning, 4, 12)
		if !errors.Is(err, ErrMapFull) {
			t.Fatalf("got err=%v, want ErrMapFull", err)
		}
		if raised {
			t.Fatalf("got raised=true, want false")
		}
	})
}

func TestReorganize(t *testing.T) {
	bounds := Rect{X: 0, Y: 0, W: 912, H: 592}
	existing := []Placement{
		{ScriptId: 9901002},
		{ScriptId: 9901000},
		{ScriptId: 9901001},
	}

	result, err := Reorganize(fr47Tuning, bounds, 0, existing, identitySnap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("ordering", func(t *testing.T) {
		if len(result) != 3 {
			t.Fatalf("got %d results, want 3", len(result))
		}
		want := []uint32{9901000, 9901001, 9901002}
		for i, w := range want {
			if result[i].ScriptId != w {
				t.Fatalf("result[%d].ScriptId = %d, want %d", i, result[i].ScriptId, w)
			}
		}
	})

	t.Run("all repositioned", func(t *testing.T) {
		if len(result) != len(existing) {
			t.Fatalf("got %d results, want %d (one per input)", len(result), len(existing))
		}
		for _, p := range result {
			if p.Step != 1 {
				t.Fatalf("Placement %d Step = %d, want 1 (step+1)", p.ScriptId, p.Step)
			}
		}
	})
}
