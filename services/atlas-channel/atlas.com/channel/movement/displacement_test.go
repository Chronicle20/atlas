package movement

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

func TestDisplaces_StationaryPathDoesNotDisplace(t *testing.T) {
	// The client flushes a move path whose fragments land the character back
	// on the coordinates it started from. This is what a standing player
	// emits when a new action begins — it must not read as movement.
	m := model.Movement{
		StartX: 400,
		StartY: -20,
		Elements: []model.MovementCodec{
			&model.NormalElement{Element: model.Element{X: 400, Y: -20, Fh: 7, BMoveAction: 4}},
			&model.Element{BMoveAction: 4},
		},
	}
	if Displaces(m) {
		t.Fatalf("a path that ends where it started must not count as displacement")
	}
}

func TestDisplaces_WalkDisplaces(t *testing.T) {
	m := model.Movement{
		StartX: 400,
		StartY: -20,
		Elements: []model.MovementCodec{
			&model.NormalElement{Element: model.Element{X: 460, Y: -20, Fh: 7, BMoveAction: 2}},
		},
	}
	if !Displaces(m) {
		t.Fatalf("a walk to a new x must count as displacement")
	}
}

func TestDisplaces_VerticalOnlyDisplaces(t *testing.T) {
	m := model.Movement{
		StartX: 400,
		StartY: -20,
		Elements: []model.MovementCodec{
			&model.NormalElement{Element: model.Element{X: 400, Y: -200, Fh: 9, BMoveAction: 2}},
		},
	}
	if !Displaces(m) {
		t.Fatalf("a change in y alone must count as displacement")
	}
}

func TestDisplaces_TeleportDisplaces(t *testing.T) {
	m := model.Movement{
		StartX: 400,
		StartY: -20,
		Elements: []model.MovementCodec{
			&model.TeleportElement{Element: model.Element{X: 1200, Y: -600, Fh: 3, BMoveAction: 2}},
		},
	}
	if !Displaces(m) {
		t.Fatalf("a teleport must count as displacement")
	}
}

func TestDisplaces_MidAirOnlyFragmentsDoNotDisplace(t *testing.T) {
	// Jump and StartFallDown carry no landing coordinates — the fold keeps the
	// start position for them, so a path made only of those has not (yet)
	// moved the character anywhere the server can observe.
	m := model.Movement{
		StartX: 400,
		StartY: -20,
		Elements: []model.MovementCodec{
			&model.JumpElement{Element: model.Element{Vx: 30, Vy: -40, BMoveAction: 3}},
			&model.StartFallDownElement{Element: model.Element{Vx: 30, Vy: 40, BMoveAction: 3}},
		},
	}
	if Displaces(m) {
		t.Fatalf("coordinate-less mid-air fragments must not count as displacement")
	}
}

func TestDisplaces_JumpThenLandDisplaces(t *testing.T) {
	m := model.Movement{
		StartX: 400,
		StartY: -20,
		Elements: []model.MovementCodec{
			&model.JumpElement{Element: model.Element{Vx: 30, Vy: -40, BMoveAction: 3}},
			&model.NormalElement{Element: model.Element{X: 520, Y: -20, Fh: 7, BMoveAction: 4}},
		},
	}
	if !Displaces(m) {
		t.Fatalf("a jump that lands elsewhere must count as displacement")
	}
}

func TestDisplaces_NoElementsDoesNotDisplace(t *testing.T) {
	if Displaces(model.Movement{StartX: 400, StartY: -20}) {
		t.Fatalf("an empty path must not count as displacement")
	}
}
