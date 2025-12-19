package buff

import (
	"atlas-channel/data/skill/effect/statup"
	buff2 "atlas-channel/kafka/message/buff"
	"atlas-channel/kafka/producer"
	"context"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

// Processor interface defines the operations for buff processing
type Processor interface {
	ByCharacterIdProvider(characterId uint32) model.Provider[[]Model]
	GetByCharacterId(characterId uint32) ([]Model, error)
	Apply(m _map.Model, fromId uint32, sourceId int32, duration int32, statups []statup.Model) model.Operator[uint32]
	Cancel(m _map.Model, characterId uint32, sourceId int32) error
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

func (p *ProcessorImpl) ByCharacterIdProvider(characterId uint32) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestById(characterId), Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) GetByCharacterId(characterId uint32) ([]Model, error) {
	return p.ByCharacterIdProvider(characterId)()
}

func (p *ProcessorImpl) Apply(m _map.Model, fromId uint32, sourceId int32, duration int32, statups []statup.Model) model.Operator[uint32] {
	return func(characterId uint32) error {
		p.l.Debugf("Character [%d] applying effect from source [%d].", characterId, sourceId)
		return producer.ProviderImpl(p.l)(p.ctx)(buff2.EnvCommandTopic)(ApplyCommandProvider(m, characterId, fromId, sourceId, duration, statups))
	}
}

func (p *ProcessorImpl) Cancel(m _map.Model, characterId uint32, sourceId int32) error {
	p.l.Debugf("Character [%d] cancelling effect from source [%d].", characterId, sourceId)
	return producer.ProviderImpl(p.l)(p.ctx)(buff2.EnvCommandTopic)(CancelCommandProvider(m, characterId, sourceId))
}
