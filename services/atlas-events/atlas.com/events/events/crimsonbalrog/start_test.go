package crimsonbalrog

import (
	"atlas-events/event/registry"
	event "atlas-events/kafka/message/event"
	monster "atlas-events/kafka/message/monster"
	"encoding/json"
	"os"
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

// emitted records every message Start's tests produce. Installed once per
// package (producer.Manager caches one writer per topic for the lifetime of
// the singleton — producertest.Capture's own doc comment); each test that
// reads it goes through newStartFakes, which resets it first.
var emitted *producertest.Capture

func TestMain(m *testing.M) {
	os.Setenv(string(monster.EnvCommandTopic), string(monster.EnvCommandTopic))
	os.Setenv(string(event.EnvEventTopicEventVisual), string(event.EnvEventTopicEventVisual))
	emitted = producertest.InstallCapturing()
	os.Exit(m.Run())
}

// newStartFakes resets the captured-message state and returns the same
// fakes fixture evaluate_test.go's tests use (Start doesn't need the
// transports/maps fakes, but reusing the one harness keeps the package
// consistent rather than inventing a second one).
func newStartFakes(t *testing.T) *fakes {
	t.Helper()
	emitted.Reset()
	return newFakes(t)
}

// emitted decodes every message captured on topic whose Type equals
// wantType as a monster.FieldCommand[monster.SpawnFieldCommandBody] — the
// only body shape Start ever produces on the monster command topic.
func (f *fakes) emitted(topic topic.Token, wantType string) []monster.FieldCommand[monster.SpawnFieldCommandBody] {
	f.t.Helper()
	var out []monster.FieldCommand[monster.SpawnFieldCommandBody]
	for _, m := range emitted.Messages(string(topic)) {
		var c monster.FieldCommand[monster.SpawnFieldCommandBody]
		if err := json.Unmarshal(m.Value, &c); err != nil {
			f.t.Fatalf("decode monster command: %v", err)
		}
		if c.Type != wantType {
			continue
		}
		out = append(out, c)
	}
	return out
}

// emittedVisuals decodes every visual event captured whose Type equals
// wantType as an event.VisualEvent[event.ShowVisualBody].
func (f *fakes) emittedVisuals(wantType string) []event.VisualEvent[event.ShowVisualBody] {
	f.t.Helper()
	var out []event.VisualEvent[event.ShowVisualBody]
	for _, m := range emitted.Messages(string(event.EnvEventTopicEventVisual)) {
		var e event.VisualEvent[event.ShowVisualBody]
		if err := json.Unmarshal(m.Value, &e); err != nil {
			f.t.Fatalf("decode visual event: %v", err)
		}
		if e.Type != wantType {
			continue
		}
		out = append(out, e)
	}
	return out
}

// occurrenceOption customizes the OccurrenceContext behind activeOccurrence's
// registry.Occurrence.
type occurrenceOption func(*OccurrenceContext)

// withAttackMaps replaces the default attack map with one per mapId, each
// seeded with two DISTINCT default spawn positions — not the seed's grounded,
// duplicated-position shape (ruling 1: a test fixture may do that freely,
// the seed may not). A test that wants the grounded shape follows with
// withSpawnPositions.
func withAttackMaps(mapIds ..._map.Id) occurrenceOption {
	return func(oc *OccurrenceContext) {
		ams := make([]AttackMap, 0, len(mapIds))
		for i, id := range mapIds {
			ams = append(ams, AttackMap{
				MapId: id,
				SpawnPositions: []Position{
					{X: int16(100 + i*10), Y: int16(100 + i*10)},
					{X: int16(200 + i*10), Y: int16(200 + i*10)},
				},
			})
		}
		oc.AttackMaps = ams
	}
}

// withSpawnPositions overrides one attack map's spawn positions — used to pin
// the grounded, duplicated-position shape from
// deploy/seed/shared/all/events/definitions/event-crimson-balrog.json: both
// monsters spawn at the SAME configured point per map (ruling 1).
func withSpawnPositions(mapId _map.Id, positions ...Position) occurrenceOption {
	return func(oc *OccurrenceContext) {
		for i := range oc.AttackMaps {
			if oc.AttackMaps[i].MapId == mapId {
				oc.AttackMaps[i].SpawnPositions = positions
			}
		}
	}
}

func withRelatedMaps(mapIds ..._map.Id) occurrenceOption {
	return func(oc *OccurrenceContext) { oc.RelatedMapIds = mapIds }
}

func monsterCount(n uint32) occurrenceOption {
	return func(oc *OccurrenceContext) { oc.MonsterCount = n }
}

// activeOccurrence builds a registry.Occurrence carrying an OccurrenceContext
// seeded with the grounded defaults (deploy/seed/shared/all/events/definitions/
// event-crimson-balrog.json, same values as newFakes' Config), then applies
// opts.
func activeOccurrence(t *testing.T, opts ...occurrenceOption) registry.Occurrence {
	t.Helper()

	oc := OccurrenceContext{
		RouteId:   uuid.New(),
		VoyageId:  uuid.New(),
		WorldId:   world.Id(1),
		ChannelId: channel.Id(4),
		AttackMaps: []AttackMap{
			{MapId: 200090010, SpawnPositions: []Position{{X: 339, Y: 148}, {X: 339, Y: 148}}},
		},
		RelatedMapIds:   []_map.Id{200090011},
		MonsterId:       8150000,
		MonsterCount:    2,
		BackgroundMusic: "Bgm04/ArabPirate",
		Visual: VisualConfig{
			Name: "CONTI_MOVE",
		},
	}
	for _, opt := range opts {
		opt(&oc)
	}

	raw, err := EncodeOccurrenceContext(oc)
	if err != nil {
		t.Fatalf("encode occurrence context: %v", err)
	}
	return registry.Occurrence{
		Id:           uuid.New(),
		DefinitionId: uuid.New(),
		Type:         TypeName,
		Stage:        StageAttacking,
		Context:      raw,
		WorldId:      oc.WorldId,
		ChannelId:    oc.ChannelId,
		VoyageId:     oc.VoyageId,
	}
}

// FR-B11/FR-B22: monsterCount monsters, in the ATTACK map only, each carrying
// the provenance pair that makes the occurrence's cleanup possible. Positions
// are consumed from the configured list in order, one per monster — ruling 1
// replaces the brief's "not a single position reused" assertion (which would
// fail against the real, deliberately-duplicated seed) with an ordering
// assertion driven by a fixture that carries two distinct positions.
func TestStartSpawnsExactlyMonsterCountInTheAttackMapWithProvenance(t *testing.T) {
	f := newStartFakes(t)
	o := activeOccurrence(t, withAttackMaps(200090010), withRelatedMaps(200090011), monsterCount(2))
	oc, err := DecodeOccurrenceContext(o.Context)
	if err != nil {
		t.Fatalf("decode occurrence context: %v", err)
	}

	if _, err := f.handler().Start(f.ctx, o); err != nil {
		t.Fatalf("Start: %v", err)
	}

	spawns := f.emitted(monster.EnvCommandTopic, monster.CommandTypeSpawnField)
	if len(spawns) != 2 {
		t.Fatalf("emitted %d spawns, want 2", len(spawns))
	}
	for i, s := range spawns {
		if s.MapId != 200090010 {
			t.Fatalf("spawn in map %d — the cabin must get no monsters (FR-B13)", s.MapId)
		}
		if s.Body.SpawnSourceType != "EVENT" || s.Body.SpawnSourceId != o.Id.String() {
			t.Fatalf("provenance = %s/%s, want EVENT/%s", s.Body.SpawnSourceType, s.Body.SpawnSourceId, o.Id)
		}
		want := oc.AttackMaps[0].SpawnPositions[i]
		if s.Body.X != want.X || s.Body.Y != want.Y {
			t.Fatalf("spawn[%d] = (%d,%d), want configured position (%d,%d)", i, s.Body.X, s.Body.Y, want.X, want.Y)
		}
	}
	// The configured positions are used, in order — not a single position reused.
	if spawns[0].Body.X == spawns[1].Body.X && spawns[0].Body.Y == spawns[1].Body.Y {
		t.Fatalf("both monsters spawned at the same configured position")
	}
}

// Pins the grounded, deliberately-duplicated-position shape from the seed:
// both monsters land at the SAME configured point (ruling 1). This is not a
// bug to fix — it is what the reference data actually specifies.
func TestStartSpawnsBothMonstersAtTheSameConfiguredPositionWhenTheSeedDuplicatesIt(t *testing.T) {
	f := newStartFakes(t)
	o := activeOccurrence(t,
		withAttackMaps(200090010),
		withSpawnPositions(200090010, Position{X: 339, Y: 148}, Position{X: 339, Y: 148}),
		monsterCount(2),
	)

	if _, err := f.handler().Start(f.ctx, o); err != nil {
		t.Fatalf("Start: %v", err)
	}

	spawns := f.emitted(monster.EnvCommandTopic, monster.CommandTypeSpawnField)
	if len(spawns) != 2 {
		t.Fatalf("emitted %d spawns, want 2", len(spawns))
	}
	for _, s := range spawns {
		if s.Body.X != 339 || s.Body.Y != 148 {
			t.Fatalf("spawn = (%d,%d), want the grounded (339,148)", s.Body.X, s.Body.Y)
		}
	}
}

// FR-B11/FR-B12: the visual is an EVENT, and it carries the gameplay content
// (which visual, which state bytes, which music) rather than a packet.
func TestStartEmitsTheVisualForTheAttackMapOnly(t *testing.T) {
	f := newStartFakes(t)
	o := activeOccurrence(t, withAttackMaps(200090010), withRelatedMaps(200090011))

	if _, err := f.handler().Start(f.ctx, o); err != nil {
		t.Fatalf("Start: %v", err)
	}

	vs := f.emittedVisuals(event.VisualTypeShow)
	if len(vs) != 1 {
		t.Fatalf("emitted %d SHOW events, want 1", len(vs))
	}
	if vs[0].MapId != 200090010 {
		t.Fatalf("SHOW sent to map %d, want the attack map", vs[0].MapId)
	}
	if vs[0].Body.Visual != "CONTI_MOVE" {
		t.Fatalf("visual = %+v", vs[0].Body)
	}
	if vs[0].Body.Bgm != "Bgm04/ArabPirate" {
		t.Fatalf("bgm = %q", vs[0].Body.Bgm)
	}
}

// Start settles the occurrence into ATTACKING with no scheduled transition:
// completion is externally driven (monsters die, or the vessel arrives), not
// timed.
func TestStartReturnsAttackingWithNoScheduledTransition(t *testing.T) {
	f := newStartFakes(t)
	p, err := f.handler().Start(f.ctx, activeOccurrence(t))
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if p.Stage != StageAttacking || p.Terminal || p.NextTransitionAt != nil {
		t.Fatalf("progress = %+v", p)
	}
}
