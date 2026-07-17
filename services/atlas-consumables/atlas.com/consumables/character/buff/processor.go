package buff

import (
	"atlas-consumables/character/buff/stat"
	buff2 "atlas-consumables/kafka/message/character/buff"
	"context"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type Processor interface {
	Apply(f field.Model, fromId uint32, sourceId int32, level byte, duration int32, statups []stat.Model) model.Operator[uint32]
	Cancel(f field.Model, characterId uint32, sourceId int32) error
	CancelByTypes(f field.Model, characterId uint32, types []string) error
}

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

func (p *ProcessorImpl) Apply(f field.Model, fromId uint32, sourceId int32, level byte, duration int32, statups []stat.Model) model.Operator[uint32] {
	return func(characterId uint32) error {
		return producer.ProviderImpl(p.l)(p.ctx)(buff2.EnvCommandTopic)(applyCommandProvider(f, characterId, fromId, sourceId, level, duration, statups))
	}
}

func (p *ProcessorImpl) Cancel(f field.Model, characterId uint32, sourceId int32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(buff2.EnvCommandTopic)(cancelCommandProvider(f, characterId, sourceId))
}

func (p *ProcessorImpl) CancelByTypes(f field.Model, characterId uint32, types []string) error {
	return producer.ProviderImpl(p.l)(p.ctx)(buff2.EnvCommandTopic)(cancelByTypesCommandProvider(f, characterId, types))
}
