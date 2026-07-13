package warp

import (
	"context"

	"atlas-maps/character/location"
	"atlas-maps/kafka/message"
	characterKafka "atlas-maps/kafka/message/character"
	mapsproducer "atlas-maps/kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	_map "atlas-maps/map"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// mapTransitioner is the narrow slice of _map.Processor that warp needs. It
// keeps the processor unit-testable without standing up the full map processor
// (which makes external calls). _map.Processor satisfies it.
type mapTransitioner interface {
	TransitionMapAndEmit(transactionId uuid.UUID, newField field.Model, characterId uint32, oldField field.Model) error
}

// Processor is the single authoritative character warp implementation. Both the
// CHANGE_MAP Kafka consumer and the PATCH /characters/{id}/location REST handler
// call ChangeMap so the two paths cannot diverge.
type Processor interface {
	// ChangeMap persists dest as the character's location, emits the canonical
	// MAP_CHANGED status event, and transitions the per-map registries. dest
	// must be a fully-formed field (world, channel, map, instance). The current
	// row is read internally for the MAP_CHANGED "old" side; if absent, oldField
	// defaults to dest (parity with the pre-task-087 consumer). Returns an error
	// only when the durable Set fails; emit/transition failures are logged and
	// the call still succeeds (parity with the consumer).
	//
	// When useTargetPosition is true the emitted MAP_CHANGED carries an exact
	// (targetX, targetY) landing instead of a named portal — atlas-channel then
	// uses the SET_FIELD chase mechanism to place the avatar there (Mystic Door).
	ChangeMap(transactionId uuid.UUID, characterId uint32, worldId world.Id, dest field.Model, portalId uint32, useTargetPosition bool, targetX int16, targetY int16) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	lp  location.Processor
	pp  producer.Provider
	mp  mapTransitioner
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	pp := producer.ProviderImpl(l)(ctx)
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		lp:  location.NewProcessor(l, ctx, db),
		pp:  pp,
		mp:  _map.NewProcessor(l, ctx, pp, db),
	}
}

var _ Processor = (*ProcessorImpl)(nil)

// newProcessorWithDeps is the unit-test seam (mirrors location's
// newProcessorWithInfo). It is not exported and not a *_testhelpers.go file.
func newProcessorWithDeps(l logrus.FieldLogger, ctx context.Context, lp location.Processor, pp producer.Provider, mp mapTransitioner) *ProcessorImpl {
	return &ProcessorImpl{l: l, ctx: ctx, lp: lp, pp: pp, mp: mp}
}

func (p *ProcessorImpl) ChangeMap(transactionId uuid.UUID, characterId uint32, worldId world.Id, dest field.Model, portalId uint32, useTargetPosition bool, targetX int16, targetY int16) error {
	oldField := dest
	if old, err := p.lp.GetById(characterId); err == nil {
		oldField = old.Field()
	}

	if _, err := p.lp.Set(characterId, dest); err != nil {
		return err
	}

	if err := message.Emit(p.pp)(func(buf *message.Buffer) error {
		return buf.Put(characterKafka.EnvEventTopicCharacterStatus,
			mapsproducer.MapChangedStatusProvider(transactionId, characterId, worldId, oldField, dest, portalId, useTargetPosition, targetX, targetY))
	}); err != nil {
		p.l.WithError(err).Errorf("ChangeMap: failed to emit MAP_CHANGED status for character [%d].", characterId)
	}

	if err := p.mp.TransitionMapAndEmit(transactionId, dest, characterId, oldField); err != nil {
		p.l.WithError(err).Warnf("ChangeMap: TransitionMapAndEmit failed for character [%d].", characterId)
	}

	return nil
}
