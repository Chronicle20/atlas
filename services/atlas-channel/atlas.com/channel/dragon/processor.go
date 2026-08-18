package dragon

import (
	dragonmsg "atlas-channel/kafka/message/dragon"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

type Processor interface {
	InMapModelProvider(f field.Model) model.Provider[[]Model]
	ForEachInMap(f field.Model, o model.Operator[Model]) error
	Create(f field.Model, characterId uint32) error
	Destroy(f field.Model, characterId uint32) error
	Move(f field.Model, characterId uint32, startX int16, startY int16, stance byte, rawMovement []byte) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

// InMapModelProvider drains every page of the field-scoped dragon list. A
// truncated list means some existing dragons silently fail to replay to an
// entering character.
func (p *ProcessorImpl) InMapModelProvider(f field.Model) model.Provider[[]Model] {
	return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(inMapUrl(f), 250, Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) ForEachInMap(f field.Model, o model.Operator[Model]) error {
	return model.ForEachSlice(p.InMapModelProvider(f), o, model.ParallelExecute())
}

func (p *ProcessorImpl) Create(f field.Model, characterId uint32) error {
	p.l.Debugf("Requesting dragon create for character [%d] in map [%d].", characterId, f.MapId())
	return producer.ProviderImpl(p.l)(p.ctx)(dragonmsg.EnvCommandTopic)(CreateCommandProvider(f, characterId))
}

func (p *ProcessorImpl) Destroy(f field.Model, characterId uint32) error {
	p.l.Debugf("Requesting dragon destroy for character [%d].", characterId)
	return producer.ProviderImpl(p.l)(p.ctx)(dragonmsg.EnvCommandTopic)(DestroyCommandProvider(f, characterId))
}

func (p *ProcessorImpl) Move(f field.Model, characterId uint32, startX int16, startY int16, stance byte, rawMovement []byte) error {
	return producer.ProviderImpl(p.l)(p.ctx)(dragonmsg.EnvCommandTopic)(MoveCommandProvider(f, characterId, startX, startY, stance, rawMovement))
}
