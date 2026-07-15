package buff

import (
	"atlas-channel/data/skill/effect/statup"
	buff2 "atlas-channel/kafka/message/buff"
	"context"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

// Processor interface defines the operations for buff processing
type Processor interface {
	ByCharacterIdProvider(characterId uint32) model.Provider[[]Model]
	GetByCharacterId(characterId uint32) ([]Model, error)
	Apply(f field.Model, fromId uint32, sourceId int32, level byte, duration int32, statups []statup.Model) model.Operator[uint32]
	Cancel(f field.Model, characterId uint32, sourceId int32) error
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
func (p *ProcessorImpl) ByCharacterIdProvider(characterId uint32) model.Provider[[]Model] {
	return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(characterBuffsUrl(characterId), 250, Extract, model.Filters[Model]())
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

func (p *ProcessorImpl) Cancel(f field.Model, characterId uint32, sourceId int32) error {
	p.l.Debugf("Character [%d] cancelling effect from source [%d].", characterId, sourceId)
	return producer.ProviderImpl(p.l)(p.ctx)(buff2.EnvCommandTopic)(CancelCommandProvider(f, characterId, sourceId))
}
