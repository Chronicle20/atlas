package crimsonbalrog

import (
	"atlas-events/event/occurrence"
	"atlas-events/event/transition"
	"atlas-events/kafka/message"
	event "atlas-events/kafka/message/event"
	monsterstatus "atlas-events/kafka/message/monsterstatus"
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// monsterSourceEvent is the SpawnSourceType this occurrence's own spawns
// carry (producer.go's spawnFieldCommandProvider, Task 25) — the provenance
// tag OnMonsterStatus matches an incoming status event against (FR-P3).
const monsterSourceEvent = "EVENT"

// MonsterProcessor maintains an occurrence's monster SET from the provenance
// echoed on atlas-monsters' status envelope, and completes the occurrence
// once every spawned monster is accounted for and none is alive (FR-B18).
type MonsterProcessor interface {
	OnMonsterStatus(e monsterstatus.StatusEvent[json.RawMessage]) error
}

// MonsterProcessorImpl is the CRIMSON_BALROG MonsterProcessor.
type MonsterProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
}

// NewMonsterProcessor constructs a MonsterProcessor.
func NewMonsterProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) *MonsterProcessorImpl {
	return &MonsterProcessorImpl{l: l, ctx: ctx, db: db}
}

var _ MonsterProcessor = (*MonsterProcessorImpl)(nil)

// OnMonsterStatus maintains the occurrence's monster SET from the provenance
// echoed on the status envelope (design §8.2). Only CREATED, KILLED and
// DESTROYED are consumed; every other type on the topic is ignored. A
// monster with someone else's provenance, or none, is ignored entirely
// (FR-P3) — this is the whole correctness story for ownership.
//
// CREATED is insert-if-absent, not upsert (occurrence.ObserveMonsterSpawned):
// a KILLED that arrived first already wrote a dead row, and the late CREATED
// must not resurrect it. The two events share a topic but have no ordering
// guarantee across partitions (design §9.5).
func (p *MonsterProcessorImpl) OnMonsterStatus(e monsterstatus.StatusEvent[json.RawMessage]) error {
	if e.SpawnSourceType != monsterSourceEvent {
		return nil
	}
	occurrenceId, err := uuid.Parse(e.SpawnSourceId)
	if err != nil {
		return nil // not one of ours
	}

	op := occurrence.NewProcessor(p.l, p.ctx, p.db)
	o, err := op.GetById(occurrenceId)
	if err != nil {
		return nil // completed and swept, or another service's id space
	}
	if o.Type() != TypeName || o.State() != occurrence.StateActive {
		return nil
	}

	switch e.Type {
	case monsterstatus.EventMonsterStatusCreated:
		return op.ObserveMonsterSpawned(occurrenceId, e.UniqueId, e.MonsterId)
	case monsterstatus.EventMonsterStatusKilled, monsterstatus.EventMonsterStatusDestroyed:
		if err := op.ObserveMonsterGone(occurrenceId, e.UniqueId, e.MonsterId); err != nil {
			return err
		}
		return p.completeIfEliminated(o)
	}
	return nil
}

// completeIfEliminated fires only when the FULL spawn set is accounted for
// and none is alive. Without the total check a completion could fire in the
// window after the first spawn's CREATED but before the second's (FR-B18).
//
// The completion itself is the guarded UPDATE occurrence.Complete already
// provides (WHERE state = 'ACTIVE'): won reports whether THIS call is the one
// that transitioned the row. A losing caller — another goroutine's KILLED
// racing this one past the tally check — gets won=false and skips cleanup,
// because the winner already ran it (FR-B20); running it twice would emit a
// second HIDE.
func (p *MonsterProcessorImpl) completeIfEliminated(o occurrence.Model) error {
	oc, err := DecodeOccurrenceContext(o.Context())
	if err != nil {
		return err
	}

	op := occurrence.NewProcessor(p.l, p.ctx, p.db)
	total, alive, err := op.MonsterTally(o.Id())
	if err != nil {
		return err
	}
	want := int(oc.MonsterCount) * len(oc.AttackMaps)
	if total < want || alive > 0 {
		return nil
	}

	won, err := op.Complete(o.Id(), occurrence.ReasonMonstersEliminated, transition.TriggerTypeMonsterKilled, "")
	if err != nil {
		return err
	}
	if !won {
		return nil
	}
	return p.hideVisuals(o, oc)
}

// hideVisuals emits HIDE with the configured hide pair for each attack map.
// It does NOT emit DESTROY_BY_SOURCE — on this path nothing is left, every
// spawned monster is already dead — and it does not restore the BGM (design
// §15.4: atlas-data exposes no Map.wz info/bgm default to restore).
func (p *MonsterProcessorImpl) hideVisuals(o occurrence.Model, oc OccurrenceContext) error {
	return message.Emit(p.l, p.ctx)(func(buf *message.Buffer) error {
		for _, am := range oc.AttackMaps {
			if err := buf.Put(event.EnvEventTopicEventVisual, hideVisualEventProvider(
				o.Id(), oc.WorldId, oc.ChannelId, am.MapId,
				oc.Visual.Name, oc.Visual.HideState, oc.Visual.HideSubState,
			)); err != nil {
				return err
			}
		}
		return nil
	})
}
