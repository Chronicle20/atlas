package crimsonbalrog

import (
	"atlas-events/event/occurrence"
	"atlas-events/event/transition"
	event "atlas-events/kafka/message/event"
	monster "atlas-events/kafka/message/monster"
	transport "atlas-events/kafka/message/transport"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// voyageId is shared across this file's tests, the same way testTenantId is
// (trigger_test.go) — each test gets its own fresh in-memory DB (newTestDB),
// so reusing one voyage id across tests carries no cross-test coupling.
var voyageId = uuid.New()

// withVoyage sets the (voyageId, worldId, channelId) scope an occurrence is
// seeded with — seedActiveOccurrence copies OccurrenceContext's VoyageId/
// WorldId/ChannelId straight onto the seeded row's scope, so this affects
// both the stored context and what GetActiveByVoyage matches against.
func withVoyage(voyageId uuid.UUID, worldId world.Id, channelId channel.Id) occurrenceOption {
	return func(oc *OccurrenceContext) {
		oc.VoyageId = voyageId
		oc.WorldId = worldId
		oc.ChannelId = channelId
	}
}

// seedCompletedOccurrence seeds an ACTIVE occurrence, then drives it through
// the real guarded-UPDATE completion path (occurrence.Processor.Complete)
// with the given reason — the same path the monster-elimination path
// (monsters.go) and this task's arrival path both use — rather than writing
// a COMPLETED row directly, so the fixture exercises the same production
// code the assertions depend on.
func seedCompletedOccurrence(t *testing.T, db *gorm.DB, reason string, opts ...occurrenceOption) occurrence.Model {
	t.Helper()
	o := seedActiveOccurrence(t, db, opts...)
	won, err := occurrence.NewProcessor(testLogger(t), testCtx(t), db).
		Complete(o.Id(), reason, transition.TriggerTypeMonsterKilled, "test-setup")
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if !won {
		t.Fatalf("seedCompletedOccurrence: Complete unexpectedly lost the race")
	}
	return readOccurrence(t, db, o.Id())
}

// arrived builds a VOYAGE_ARRIVED event for (voyageId, worldId, channelId).
// RouteId is left zero — ArrivalProcessor.OnVoyageArrived never reads it,
// only e.Body's voyage/world/channel scope (unlike OnVoyageDeparted, which
// matches Config.ApplicableRouteIds against e.RouteId).
func arrived(voyageId uuid.UUID, worldId world.Id, channelId channel.Id) transport.StatusEvent[transport.VoyageStatusEventBody] {
	return transport.StatusEvent[transport.VoyageStatusEventBody]{
		Type: transport.EventStatusVoyageArrived,
		Body: transport.VoyageStatusEventBody{
			VoyageId:  voyageId,
			WorldId:   worldId,
			ChannelId: channelId,
		},
	}
}

// FR-B19: arrival despawns everything remaining, removes the visual and
// completes with VESSEL_ARRIVED.
func TestArrivalCompletesAndCleansUp(t *testing.T) {
	db := newTestDB(t)
	f := newEmitCapture(t)
	o := seedActiveOccurrence(t, db, withVoyage(voyageId, 1, 4), withAttackMaps(200090010))

	must(t, NewArrivalProcessor(testLogger(t), testCtx(t), db).OnVoyageArrived(arrived(voyageId, 1, 4)))

	got := readOccurrence(t, db, o.Id())
	if got.State() != occurrence.StateCompleted || got.CompletionReason() != occurrence.ReasonVesselArrived {
		t.Fatalf("state=%s reason=%s", got.State(), got.CompletionReason())
	}
	ds := f.emitted(monster.EnvCommandTopic, monster.CommandTypeDestroyBySource)
	if len(ds) != 1 || ds[0].MapId != 200090010 {
		t.Fatalf("DESTROY_BY_SOURCE = %+v", ds)
	}
	if ds[0].Body.SpawnSourceId != o.Id().String() {
		t.Fatalf("destroy targeted %q, want the occurrence id", ds[0].Body.SpawnSourceId)
	}
	if len(f.emittedVisuals(event.VisualTypeHide)) != 1 {
		t.Fatalf("expected one HIDE")
	}
}

// FR-B20: arrival cleanup must succeed when every Balrog was killed a second
// earlier. Zero matches is the ORDINARY case, not an error path.
func TestArrivalAfterEliminationIsANoOp(t *testing.T) {
	db := newTestDB(t)
	f := newEmitCapture(t)
	o := seedCompletedOccurrence(t, db, occurrence.ReasonMonstersEliminated, withVoyage(voyageId, 1, 4))

	must(t, NewArrivalProcessor(testLogger(t), testCtx(t), db).OnVoyageArrived(arrived(voyageId, 1, 4)))

	got := readOccurrence(t, db, o.Id())
	if got.CompletionReason() != occurrence.ReasonMonstersEliminated {
		t.Fatalf("reason overwritten to %q — the first completion must win", got.CompletionReason())
	}
	if len(f.emittedVisuals(event.VisualTypeHide)) != 0 {
		t.Fatalf("cleanup ran twice")
	}
}

// FR-N11: an arrival in world 1 channel 4 must not touch an occurrence in
// channel 5 of the same voyage.
func TestArrivalIsScopedToItsChannel(t *testing.T) {
	db := newTestDB(t)
	f := newEmitCapture(t)
	o := seedActiveOccurrence(t, db, withVoyage(voyageId, 1, 5), withAttackMaps(200090010))

	must(t, NewArrivalProcessor(testLogger(t), testCtx(t), db).OnVoyageArrived(arrived(voyageId, 1, 4)))

	got := readOccurrence(t, db, o.Id())
	if got.State() != occurrence.StateActive {
		t.Fatalf("state=%s, want ACTIVE — a channel 4 arrival must not complete a channel 5 occurrence", got.State())
	}
	if len(f.emitted(monster.EnvCommandTopic, monster.CommandTypeDestroyBySource)) != 0 {
		t.Fatalf("cleanup must not run for a different channel's occurrence")
	}
	if len(f.emittedVisuals(event.VisualTypeHide)) != 0 {
		t.Fatalf("HIDE must not run for a different channel's occurrence")
	}
}
