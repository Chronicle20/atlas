package crimsonbalrog

import (
	"atlas-events/event/occurrence"
	"atlas-events/event/transition"
	"atlas-events/kafka/message"
	event "atlas-events/kafka/message/event"
	monster "atlas-events/kafka/message/monster"
	transport "atlas-events/kafka/message/transport"
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// ArrivalProcessor turns a VOYAGE_ARRIVED transport event into the second of
// this occurrence type's two completion paths (Task 26's monster-elimination
// path is the first — FR-B17/FR-B20). It is a processor, not consumer code
// (FR-N18) — the consumer's job is only to decode, guard on type, and
// delegate here.
type ArrivalProcessor interface {
	OnVoyageArrived(e transport.StatusEvent[transport.VoyageStatusEventBody]) error
}

// ArrivalProcessorImpl is the CRIMSON_BALROG ArrivalProcessor.
type ArrivalProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
}

// NewArrivalProcessor constructs an ArrivalProcessor.
func NewArrivalProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) *ArrivalProcessorImpl {
	return &ArrivalProcessorImpl{l: l, ctx: ctx, db: db}
}

var _ ArrivalProcessor = (*ArrivalProcessorImpl)(nil)

// OnVoyageArrived completes any ACTIVE occurrence belonging to this
// (voyageId, worldId, channelId). occurrence.Processor.GetActiveByVoyage
// already scopes to state=ACTIVE (event/occurrence/provider.go:29-38), so a
// stale row (already completed by the monster-elimination path a moment
// earlier) is simply absent from the result and this call is a no-op for
// it, exactly the "zero matches is ORDINARY, not an error" behavior
// FR-B20/ruling 1 require.
//
// For any occurrence still returned, completion is the guarded UPDATE
// (occurrence.Processor.Complete -> event/occurrence/administrator.go:126-145:
// "WHERE id = ? AND state = 'ACTIVE'" inside a SELECT ... FOR UPDATE
// transaction) — that guard, not this method, is what makes a genuine race
// against the elimination path safe: whichever caller's UPDATE lands first
// gets RowsAffected > 0 (won=true); the loser gets RowsAffected == 0
// (won=false) and skips cleanup below rather than re-running it. No
// additional "is it still active?" read is added here — the guard is the
// single source of truth for who won.
func (p *ArrivalProcessorImpl) OnVoyageArrived(e transport.StatusEvent[transport.VoyageStatusEventBody]) error {
	op := occurrence.NewProcessor(p.l, p.ctx, p.db)
	os, err := op.GetActiveByVoyage(e.Body.VoyageId, e.Body.WorldId, e.Body.ChannelId)
	if err != nil {
		return err
	}
	for _, o := range os {
		if o.Type() != TypeName {
			continue
		}
		won, err := op.Complete(o.Id(), occurrence.ReasonVesselArrived, transition.TriggerTypeVoyageArrived, e.Body.VoyageId.String())
		if err != nil {
			return err
		}
		if !won {
			continue
		}
		if err := p.cleanup(o); err != nil {
			return err
		}
	}
	return nil
}

// cleanup despawns everything the occurrence owns, then removes the visual,
// for every attack map (FR-B19). It runs ONLY after Complete's guarded
// UPDATE reports this call won (ruling 2): the state transition is the
// durable fact, and cleanup is a best-effort side effect of it, not the
// other way around — running cleanup unconditionally before the guard would
// let a losing caller (the elimination path already won) re-emit a second
// DESTROY_BY_SOURCE/HIDE, which ruling 1 forbids outright. The accepted
// trade-off (also made by Task 26's completeIfEliminated) is that a crash
// between the guarded UPDATE and this emit leaves live monsters with no
// ACTIVE occurrence to clean them up; that window is the same one every
// other completion path in this package already accepts, so this method
// stays consistent with it rather than inventing a different ordering.
func (p *ArrivalProcessorImpl) cleanup(o occurrence.Model) error {
	return emitCleanup(p.l, p.ctx, o.Id(), o.Context())
}

// emitCleanup despawns everything an occurrence owns, then removes the
// visual, for every attack map (FR-B19). Shared by ArrivalProcessorImpl's
// cleanup (a completion driven by this consumer) and Handler.Complete (a
// completion driven by the generic scheduling layer — handler.go), so both
// paths produce identical wire traffic instead of a second, divergent HIDE/
// DESTROY_BY_SOURCE shape (ruling 4). The destroy command is issued
// unconditionally per attack map, not gated on a monster tally: zero
// survivors is success (FR-P4), and a stale tally can only make the gate
// wrong in the direction of skipping a needed cleanup.
func emitCleanup(l logrus.FieldLogger, ctx context.Context, occurrenceId uuid.UUID, raw json.RawMessage) error {
	oc, err := DecodeOccurrenceContext(raw)
	if err != nil {
		return err
	}
	return message.Emit(l, ctx)(func(buf *message.Buffer) error {
		for _, am := range oc.AttackMaps {
			if err := buf.Put(monster.EnvCommandTopic, destroyBySourceCommandProvider(
				oc.WorldId, oc.ChannelId, am.MapId, occurrenceId,
			)); err != nil {
				return err
			}
			if err := buf.Put(event.EnvEventTopicEventVisual, hideVisualEventProvider(
				occurrenceId, oc.WorldId, oc.ChannelId, am.MapId,
				oc.Visual.Name, oc.Visual.HideState, oc.Visual.HideSubState,
			)); err != nil {
				return err
			}
		}
		return nil
	})
}
