package drop

import (
	drop2 "atlas-channel/kafka/message/drop"
	"atlas-channel/kafka/producer"
	"context"

	"github.com/Chronicle20/atlas-constants/field"
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

func (p *Processor) InMapModelProvider(f field.Model) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestInMap(f), Extract, model.Filters[Model]())
}

func (p *Processor) ForEachInMap(f field.Model, o model.Operator[Model]) error {
	return model.ForEachSlice(p.InMapModelProvider(f), o, model.ParallelExecute())
}

func (p *Processor) RequestReservation(f field.Model, dropId uint32, characterId uint32, partyId uint32, characterX int16, characterY int16, petSlot int8) error {
	return producer.ProviderImpl(p.l)(p.ctx)(drop2.EnvCommandTopic)(RequestReservationCommandProvider(f, dropId, characterId, partyId, characterX, characterY, petSlot))
}
