package occurrence_test

// This test builds the occurrence context the way CRIMSON_BALROG — the only
// event type that populates one — actually produces it
// (events/crimsonbalrog/config.go OccurrenceContext,
// events/crimsonbalrog/config.go EncodeOccurrenceContext), rather than a
// hand-written flat fixture. It lives in an external _test package because
// events/crimsonbalrog imports event/occurrence (an internal test would be
// an import cycle).

import (
	"atlas-events/event/occurrence"
	"atlas-events/events/crimsonbalrog"
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/monster"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// TestTransformVisualReadsCrimsonBalrogProducedContext proves (or disproves)
// that the FR-API8 REST projection can actually read the context shape
// CRIMSON_BALROG's Evaluate/seed path writes: `visual` is an OBJECT
// ({"name": ...} — crimsonbalrog.VisualConfig, post-B6 no state/subState
// bytes at all) and the music key is `backgroundMusic`, not `bgm`.
func TestTransformVisualReadsCrimsonBalrogProducedContext(t *testing.T) {
	oc := crimsonbalrog.OccurrenceContext{
		RouteId:         uuid.New(),
		VoyageId:        uuid.New(),
		WorldId:         world.Id(1),
		ChannelId:       channel.Id(0),
		AttackMaps:      []crimsonbalrog.AttackMap{{MapId: _map.Id(200090010)}},
		RelatedMapIds:   []_map.Id{200090011},
		MonsterId:       monster.Id(8800000),
		MonsterCount:    3,
		BackgroundMusic: "Bgm04/ArabPirate",
		Visual:          crimsonbalrog.VisualConfig{Name: "CONTI_MOVE"},
	}
	raw, err := crimsonbalrog.EncodeOccurrenceContext(oc)
	if err != nil {
		t.Fatalf("EncodeOccurrenceContext: %v", err)
	}

	m, err := occurrence.NewBuilder(uuid.New(), crimsonbalrog.TypeName).
		SetContext(raw).
		Build()
	if err != nil {
		t.Fatalf("build occurrence model: %v", err)
	}

	got, err := occurrence.TransformVisual(m)
	if err != nil {
		t.Fatalf("TransformVisual: %v", err)
	}
	if got.Visual != "CONTI_MOVE" {
		t.Errorf("Visual = %q, want %q", got.Visual, "CONTI_MOVE")
	}
	if got.Bgm != "Bgm04/ArabPirate" {
		t.Errorf("Bgm = %q, want %q", got.Bgm, "Bgm04/ArabPirate")
	}
}
