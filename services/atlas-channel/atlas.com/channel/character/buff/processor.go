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

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	p := &Processor{
		l:   l,
		ctx: ctx,
	}
	return p
}

func (p *Processor) ByCharacterIdProvider(characterId uint32) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestById(characterId), Extract, model.Filters[Model]())
}

func (p *Processor) GetByCharacterId(characterId uint32) ([]Model, error) {
	return p.ByCharacterIdProvider(characterId)()
}

func (p *Processor) Apply(m _map.Model, fromId uint32, sourceId int32, duration int32, statups []statup.Model) model.Operator[uint32] {
	return func(characterId uint32) error {
		p.l.Debugf("Character [%d] applying effect from source [%d].", characterId, sourceId)
		return producer.ProviderImpl(p.l)(p.ctx)(buff2.EnvCommandTopic)(ApplyCommandProvider(m, characterId, fromId, sourceId, duration, statups))
	}
}

func (p *Processor) Cancel(m _map.Model, characterId uint32, sourceId int32) error {
	p.l.Debugf("Character [%d] cancelling effect from source [%d].", characterId, sourceId)
	return producer.ProviderImpl(p.l)(p.ctx)(buff2.EnvCommandTopic)(CancelCommandProvider(m, characterId, sourceId))
}
