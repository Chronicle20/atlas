package buff

import (
	"atlas-channel/data/skill/effect/statup"
	buff2 "atlas-channel/kafka/message/buff"
	"context"
	"errors"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Processor interface defines the operations for buff processing
type Processor interface {
	ByCharacterIdProvider(characterId uint32) model.Provider[[]Model]
	GetByCharacterId(characterId uint32) ([]Model, error)
	Apply(f field.Model, fromId uint32, sourceId int32, level byte, duration int32, statups []statup.Model) model.Operator[uint32]
	ApplyNoExpiry(f field.Model, fromId uint32, sourceId int32, level byte, statups []statup.Model) model.Operator[uint32]
	Cancel(f field.Model, characterId uint32, sourceId int32) error
	UpdateStatValue(f field.Model, characterId uint32, u StatValueUpdate) error
	CancelByTypes(f field.Model, characterId uint32, types []string) error
	Expire(f field.Model, characterId uint32) error
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
	return p
}

var _ Processor = (*ProcessorImpl)(nil)

// ByCharacterIdProvider fetches every buff for a character. The upstream
// atlas-buffs list is now paginated (task-117); callers here need the
// complete set (e.g. cancelling every buff invalidated by a map/mount
// change, or syncing buff state on session events), so this drains every
// page rather than fetching just the first.
//
// A 404 is normalized to the empty set rather than propagated. atlas-buffs
// materializes a character's buff registry entry lazily, so GET
// /characters/{id}/buffs replies 404 until something applies a first buff --
// that is "this character has no buffs", not a failure. Callers here all ask
// the same question and several skip a character on error, which silently
// dropped exactly the buffless players (observed as Echo of Hero's map-wide
// fan-out logging fetch_failures / applied:0 for a fresh recipient).
func (p *ProcessorImpl) ByCharacterIdProvider(characterId uint32) model.Provider[[]Model] {
	url, err := characterBuffsUrl(p.ctx, characterId)
	if err != nil {
		return model.ErrorProvider[[]Model](err)
	}
	drain := requests.DrainProvider[RestModel, Model](p.l, p.ctx)(url, 250, Extract, model.Filters[Model]())
	return func() ([]Model, error) {
		ms, err := drain()
		if errors.Is(err, requests.ErrNotFound) {
			return []Model{}, nil
		}
		return ms, err
	}
}

func (p *ProcessorImpl) GetByCharacterId(characterId uint32) ([]Model, error) {
	return p.ByCharacterIdProvider(characterId)()
}

func (p *ProcessorImpl) Apply(f field.Model, fromId uint32, sourceId int32, level byte, duration int32, statups []statup.Model) model.Operator[uint32] {
	return func(characterId uint32) error {
		p.l.Debugf("Character [%d] applying effect from source [%d].", characterId, sourceId)
		return producer.ProviderImpl(p.l)(p.ctx)(buff2.EnvCommandTopic)(ApplyCommandProvider(f, characterId, fromId, sourceId, level, duration, statups))
	}
}

func (p *ProcessorImpl) ApplyNoExpiry(f field.Model, fromId uint32, sourceId int32, level byte, statups []statup.Model) model.Operator[uint32] {
	return func(characterId uint32) error {
		p.l.Debugf("Character [%d] applying no-expiry effect from source [%d].", characterId, sourceId)
		return producer.ProviderImpl(p.l)(p.ctx)(buff2.EnvCommandTopic)(ApplyNoExpiryCommandProvider(f, characterId, fromId, sourceId, level, statups))
	}
}

func (p *ProcessorImpl) Cancel(f field.Model, characterId uint32, sourceId int32) error {
	p.l.Debugf("Character [%d] cancelling effect from source [%d].", characterId, sourceId)
	return producer.ProviderImpl(p.l)(p.ctx)(buff2.EnvCommandTopic)(CancelCommandProvider(f, characterId, sourceId))
}

func (p *ProcessorImpl) UpdateStatValue(f field.Model, characterId uint32, u StatValueUpdate) error {
	p.l.Debugf("Character [%d] updating stat [%s] on buff [%d]: %s %d (cap %d, createIfMissing %t).", characterId, u.StatType, u.SourceId, u.Operation, u.Amount, u.Cap, u.CreateIfMissing)
	return producer.ProviderImpl(p.l)(p.ctx)(buff2.EnvCommandTopic)(UpdateStatValueCommandProvider(f, characterId, u))
}

func (p *ProcessorImpl) CancelByTypes(f field.Model, characterId uint32, types []string) error {
	p.l.Debugf("Character [%d] cancelling buffs by types %v.", characterId, types)
	return producer.ProviderImpl(p.l)(p.ctx)(buff2.EnvCommandTopic)(CancelByTypesCommandProvider(f, characterId, types))
}

// Expire asks atlas-buffs to re-evaluate this character's buffs and announce
// whatever has lapsed. Triggered by the client's CANCEL_DEBUFF nudge, which
// carries no payload — so this cannot and must not cancel by name. A sweep
// that finds nothing lapsed emits nothing (FR-2.9 / NFR-2.1). (task-190)
func (p *ProcessorImpl) Expire(f field.Model, characterId uint32) error {
	p.l.Debugf("Character [%d] requesting buff expiry sweep.", characterId)
	return producer.ProviderImpl(p.l)(p.ctx)(buff2.EnvCommandTopic)(ExpireCommandProvider(f, characterId))
}
