package chalkboard

import (
	chalkboard2 "atlas-channel/kafka/message/chalkboard"
	"atlas-channel/kafka/producer"
	chalkboard3 "atlas-channel/kafka/producer/chalkboard"
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

func (p *Processor) InMapModelProvider(m _map.Model) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestInMap(m), Extract, model.Filters[Model]())
}

func (p *Processor) ForEachInMap(m _map.Model, f model.Operator[Model]) error {
	return model.ForEachSlice(p.InMapModelProvider(m), f, model.ParallelExecute())
}

func (p *Processor) AttemptUse(m _map.Model, characterId uint32, message string) error {
	p.l.Debugf("Character [%d] attempting to set a chalkboard message [%s].", characterId, message)
	return producer.ProviderImpl(p.l)(p.ctx)(chalkboard2.EnvCommandTopic)(chalkboard3.SetCommandProvider(m, characterId, message))
}

func (p *Processor) Close(m _map.Model, characterId uint32) error {
	p.l.Debugf("Character [%d] attempting to close chalkboard.", characterId)
	return producer.ProviderImpl(p.l)(p.ctx)(chalkboard2.EnvCommandTopic)(chalkboard3.ClearCommandProvider(m, characterId))
}
