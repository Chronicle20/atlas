package movement

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

// The movement decoder constructs every element as a pointer. A prior version of
// foldMovementSummary matched Teleport/Jump/StartFallDown by value, so those
// fragments were silently ignored (dead cases). These tests pin the fixed
// pointer-matched behavior.

func TestFoldMovementSummary_NormalAppliesPositionAndFh(t *testing.T) {
	start := summary{X: 1, Y: 2, Fh: 3, Stance: 4}
	e := &model.NormalElement{Element: model.Element{X: 100, Y: 200, Fh: 50, BMoveAction: 9}}
	got, err := foldMovementSummary(start, e)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.X != 100 || got.Y != 200 || got.Fh != 50 || got.Stance != 9 {
		t.Fatalf("normal not applied: got %+v", got)
	}
}

func TestFoldMovementSummary_TeleportAppliesPositionAndFh(t *testing.T) {
	start := summary{X: 1, Y: 2, Fh: 3, Stance: 4}
	e := &model.TeleportElement{Element: model.Element{X: 100, Y: 200, Fh: 50, BMoveAction: 9}}
	got, err := foldMovementSummary(start, e)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.X != 100 || got.Y != 200 || got.Fh != 50 || got.Stance != 9 {
		t.Fatalf("teleport not applied (was a dead value-case): got %+v", got)
	}
}

func TestFoldMovementSummary_TeleportZeroFhPreservesPriorFh(t *testing.T) {
	start := summary{X: 1, Y: 2, Fh: 3, Stance: 4}
	e := &model.TeleportElement{Element: model.Element{X: 100, Y: 200, Fh: 0, BMoveAction: 9}}
	got, _ := foldMovementSummary(start, e)
	if got.Fh != 3 {
		t.Fatalf("zero fh should preserve prior fh; got %d", got.Fh)
	}
}

func TestFoldMovementSummary_JumpUpdatesStanceOnly(t *testing.T) {
	start := summary{X: 1, Y: 2, Fh: 3, Stance: 4}
	e := &model.JumpElement{Element: model.Element{X: 100, Y: 200, Fh: 50, BMoveAction: 9}}
	got, _ := foldMovementSummary(start, e)
	if got.X != 1 || got.Y != 2 || got.Fh != 3 || got.Stance != 9 {
		t.Fatalf("jump should update stance only (mid-air): got %+v", got)
	}
}

func TestFoldMovementSummary_StartFallDownUpdatesStanceOnly(t *testing.T) {
	start := summary{X: 1, Y: 2, Fh: 3, Stance: 4}
	e := &model.StartFallDownElement{Element: model.Element{X: 100, Y: 200, Fh: 50, BMoveAction: 9}}
	got, _ := foldMovementSummary(start, e)
	if got.X != 1 || got.Y != 2 || got.Fh != 3 || got.Stance != 9 {
		t.Fatalf("start-fall-down should update stance only (mid-air): got %+v", got)
	}
}
