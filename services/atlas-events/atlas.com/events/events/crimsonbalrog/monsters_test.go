package crimsonbalrog

import (
	"atlas-events/event/occurrence"
	"atlas-events/event/registry"
	event "atlas-events/kafka/message/event"
	monster "atlas-events/kafka/message/monster"
	monsterstatus "atlas-events/kafka/message/monsterstatus"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

// newEmitCapture resets the package-wide producertest.Capture (installed once
// by start_test.go's TestMain) and returns the same emission-reading fixture
// start_test.go's tests use, so completeIfEliminated's HIDE emissions are
// readable the same way Start's SHOW/spawn emissions are.
func newEmitCapture(t *testing.T) *fakes {
	t.Helper()
	emitted.Reset()
	return newFakes(t)
}

// emittedHideVisuals decodes every visual event captured whose Type equals
// wantType as an event.VisualEvent[event.HideVisualBody]. A separate method
// from fakes.emittedVisuals (start_test.go) because that one is fixed to
// ShowVisualBody, the only body Start ever produces.
func (f *fakes) emittedHideVisuals(wantType string) []event.VisualEvent[event.HideVisualBody] {
	f.t.Helper()
	var out []event.VisualEvent[event.HideVisualBody]
	for _, m := range emitted.Messages(event.EnvEventTopicEventVisual) {
		var e event.VisualEvent[event.HideVisualBody]
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

// seedActiveOccurrence persists an ACTIVE CRIMSON_BALROG occurrence (via
// occurrence.Processor.CreateFromSeed, the same path Evaluate/Start use in
// production) with the grounded defaults from activeOccurrence
// (start_test.go), customized by opts. Unlike activeOccurrence (which only
// builds an in-memory registry.Occurrence for Start's tests), this writes a
// real row so OnMonsterStatus's op.GetById(occurrenceId) can find it.
func seedActiveOccurrence(t *testing.T, db *gorm.DB, opts ...occurrenceOption) occurrence.Model {
	t.Helper()
	ensureHandlerRegistered()

	oc := OccurrenceContext{
		RouteId:   uuid.New(),
		VoyageId:  uuid.New(),
		WorldId:   1,
		ChannelId: 4,
		AttackMaps: []AttackMap{
			{MapId: 200090010, SpawnPositions: []Position{{X: 339, Y: 148}, {X: 339, Y: 148}}},
		},
		RelatedMapIds:   []_map.Id{200090011},
		MonsterId:       8150000,
		MonsterCount:    1,
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

	d := seedDefinition(t, db, TypeName, routes("route-a"))

	scope := make([]registry.MapScope, 0, len(oc.AttackMaps)+len(oc.RelatedMapIds))
	for _, am := range oc.AttackMaps {
		scope = append(scope, registry.MapScope{MapId: am.MapId, Visual: true})
	}
	for _, mapId := range oc.RelatedMapIds {
		scope = append(scope, registry.MapScope{MapId: mapId, Visual: false})
	}

	seed := registry.Seed{
		Stage:     StageAttacking,
		Context:   raw,
		WorldId:   oc.WorldId,
		ChannelId: oc.ChannelId,
		VoyageId:  oc.VoyageId,
		Maps:      scope,
	}

	o, err := occurrence.NewProcessor(testLogger(t), testCtx(t), db).CreateFromSeed(d, seed, "seed")
	if err != nil {
		t.Fatalf("CreateFromSeed: %v", err)
	}
	return o
}

// must fails the test immediately if err is non-nil. OnMonsterStatus never
// returns an error for a foreign/unowned monster (it is ignored, not
// rejected — see OnMonsterStatus's doc comment), so every call in these
// tests is expected to succeed.
func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("OnMonsterStatus: %v", err)
	}
}

// readOccurrence re-reads an occurrence's current row.
func readOccurrence(t *testing.T, db *gorm.DB, id uuid.UUID) occurrence.Model {
	t.Helper()
	m, err := occurrence.NewProcessor(testLogger(t), testCtx(t), db).GetById(id)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	return m
}

// state reports the current State() of an occurrence.
func state(t *testing.T, db *gorm.DB, id uuid.UUID) string {
	t.Helper()
	return readOccurrence(t, db, id).State()
}

// statusEvent builds a monsterstatus.StatusEvent carrying this occurrence's
// provenance, an empty Body (OnMonsterStatus never reads it).
func statusEvent(theType string, occurrenceId uuid.UUID, uniqueId uint32) monsterstatus.StatusEvent[json.RawMessage] {
	return monsterstatus.StatusEvent[json.RawMessage]{
		UniqueId:        uniqueId,
		MonsterId:       8150000,
		Type:            theType,
		SpawnSourceType: monsterSourceEvent,
		SpawnSourceId:   occurrenceId.String(),
		Body:            json.RawMessage("{}"),
	}
}

func created(occurrenceId uuid.UUID, uniqueId uint32) monsterstatus.StatusEvent[json.RawMessage] {
	return statusEvent(monsterstatus.EventMonsterStatusCreated, occurrenceId, uniqueId)
}

func killed(occurrenceId uuid.UUID, uniqueId uint32) monsterstatus.StatusEvent[json.RawMessage] {
	return statusEvent(monsterstatus.EventMonsterStatusKilled, occurrenceId, uniqueId)
}

// createdWithSource builds a CREATED event whose provenance names a
// DIFFERENT occurrence (some other, unseeded occurrence id) — a monster this
// test's occurrence must not track.
func createdWithSource(sourceId string, uniqueId uint32) monsterstatus.StatusEvent[json.RawMessage] {
	return monsterstatus.StatusEvent[json.RawMessage]{
		UniqueId:        uniqueId,
		MonsterId:       8150000,
		Type:            monsterstatus.EventMonsterStatusCreated,
		SpawnSourceType: monsterSourceEvent,
		SpawnSourceId:   sourceId,
		Body:            json.RawMessage("{}"),
	}
}

// cyclicCreated builds a CREATED event with NO provenance at all — a
// naturally-spawned (cyclic) monster, which no occurrence ever owns.
func cyclicCreated(uniqueId uint32) monsterstatus.StatusEvent[json.RawMessage] {
	return monsterstatus.StatusEvent[json.RawMessage]{
		UniqueId:  uniqueId,
		MonsterId: 9300102,
		Type:      monsterstatus.EventMonsterStatusCreated,
		Body:      json.RawMessage("{}"),
	}
}

// FR-B18: completion fires only when every spawned monster is accounted for
// AND none is alive. The first conjunct is what stops a completion firing in
// the window after the first spawn's CREATED but before the second's.
func TestCompletionWaitsForTheFullSpawnSet(t *testing.T) {
	db := newTestDB(t)
	o := seedActiveOccurrence(t, db, monsterCount(2))
	p := NewMonsterProcessor(testLogger(t), testCtx(t), db)

	must(t, p.OnMonsterStatus(created(o.Id(), 1)))
	must(t, p.OnMonsterStatus(killed(o.Id(), 1)))

	if state(t, db, o.Id()) != occurrence.StateActive {
		t.Fatalf("completed after one of two monsters died")
	}

	must(t, p.OnMonsterStatus(created(o.Id(), 2)))
	must(t, p.OnMonsterStatus(killed(o.Id(), 2)))

	got := readOccurrence(t, db, o.Id())
	if got.State() != occurrence.StateCompleted || got.CompletionReason() != occurrence.ReasonMonstersEliminated {
		t.Fatalf("state=%s reason=%s", got.State(), got.CompletionReason())
	}
}

// A monster with someone else's provenance, or none, is ignored entirely.
func TestForeignMonstersAreIgnored(t *testing.T) {
	db := newTestDB(t)
	o := seedActiveOccurrence(t, db, monsterCount(1))
	p := NewMonsterProcessor(testLogger(t), testCtx(t), db)

	must(t, p.OnMonsterStatus(createdWithSource(uuid.New().String(), 9)))
	must(t, p.OnMonsterStatus(cyclicCreated(10)))

	total, alive, err := occurrence.NewProcessor(testLogger(t), testCtx(t), db).MonsterTally(o.Id())
	if err != nil {
		t.Fatalf("MonsterTally: %v", err)
	}
	if total != 0 || alive != 0 {
		t.Fatalf("foreign monsters were tracked: total=%d alive=%d", total, alive)
	}
}

// FR-B18 cleanup: the visual is removed on this path. The BGM is deliberately
// NOT restored — see design §15.4: atlas-data does not expose Map.wz info/bgm,
// so no service can name the map's default, and "restore the default" would
// mean hard-coding a guessed string.
func TestEliminationCleanupHidesTheVisualAndLeavesTheMusic(t *testing.T) {
	db := newTestDB(t)
	f := newEmitCapture(t)
	o := seedActiveOccurrence(t, db, monsterCount(1), withAttackMaps(200090010))
	p := NewMonsterProcessor(testLogger(t), testCtx(t), db)

	must(t, p.OnMonsterStatus(created(o.Id(), 1)))
	must(t, p.OnMonsterStatus(killed(o.Id(), 1)))

	hides := f.emittedHideVisuals(event.VisualTypeHide)
	if len(hides) != 1 || hides[0].MapId != 200090010 {
		t.Fatalf("HIDE = %+v", hides)
	}
	if hides[0].Body.Visual != "CONTI_MOVE" {
		t.Fatalf("hide visual = %+v, want CONTI_MOVE", hides[0].Body)
	}
	if len(f.emitted(monster.EnvCommandTopic, monster.CommandTypeDestroyBySource)) != 0 {
		t.Fatalf("elimination must not issue DESTROY_BY_SOURCE — nothing is left")
	}
}

// Redelivery of the final KILLED must not complete twice or re-run cleanup:
// occurrence.Complete's guarded UPDATE (WHERE state = 'ACTIVE') makes the
// second call a no-win, and completeIfEliminated skips hideVisuals on a loss.
func TestRedeliveredFinalKillIsANoOp(t *testing.T) {
	db := newTestDB(t)
	f := newEmitCapture(t)
	o := seedActiveOccurrence(t, db, monsterCount(1), withAttackMaps(200090010))
	p := NewMonsterProcessor(testLogger(t), testCtx(t), db)

	must(t, p.OnMonsterStatus(created(o.Id(), 1)))
	must(t, p.OnMonsterStatus(killed(o.Id(), 1)))

	first := readOccurrence(t, db, o.Id())
	if first.State() != occurrence.StateCompleted {
		t.Fatalf("state = %s, want COMPLETED", first.State())
	}
	if len(f.emittedHideVisuals(event.VisualTypeHide)) != 1 {
		t.Fatalf("expected exactly 1 HIDE after first completion")
	}

	// Redelivery of the SAME final KILLED.
	must(t, p.OnMonsterStatus(killed(o.Id(), 1)))

	again := readOccurrence(t, db, o.Id())
	if again.State() != occurrence.StateCompleted || again.CompletionReason() != occurrence.ReasonMonstersEliminated {
		t.Fatalf("state=%s reason=%s after redelivery", again.State(), again.CompletionReason())
	}
	if again.CompletedAt() == nil || first.CompletedAt() == nil || !again.CompletedAt().Equal(*first.CompletedAt()) {
		t.Fatalf("completedAt changed on redelivery: first=%v again=%v", first.CompletedAt(), again.CompletedAt())
	}
	if got := len(f.emittedHideVisuals(event.VisualTypeHide)); got != 1 {
		t.Fatalf("emitted %d HIDE events after redelivery, want 1 (no re-run of cleanup)", got)
	}
}
