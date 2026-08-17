package drop

import (
	drop2 "atlas-channel/kafka/message/drop"
	"context"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

type Processor interface {
	InMapModelProvider(f field.Model) model.Provider[[]Model]
	ForEachInMap(f field.Model, o model.Operator[Model]) error
	RequestReservation(f field.Model, dropId uint32, characterId uint32, partyId uint32, characterX int16, characterY int16, petSlot int8) error
	SpawnMeso(f field.Model, mesos uint32, x int16, y int16, ownerId uint32, dropperId uint32, dropperX int16, dropperY int16) error
	ConsumeAll(f field.Model, dropIds []uint32) error
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

// InMapModelProvider fetches every drop currently in one map instance. This
// is a hot-path consumer (drop spawn state on every channel spawn
// broadcast, ForEachInMap for reservation logic); the upstream atlas-drops
// list is now paginated (task-117), so this drains every page rather than
// fetching just the first -- a truncated list here means drops silently
// vanish from the client's view.
func (p *ProcessorImpl) InMapModelProvider(f field.Model) model.Provider[[]Model] {
	url, err := inMapUrl(p.ctx, f)
	if err != nil {
		return model.ErrorProvider[[]Model](err)
	}
	return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(url, 250, Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) ForEachInMap(f field.Model, o model.Operator[Model]) error {
	return model.ForEachSlice(p.InMapModelProvider(f), o, model.ParallelExecute())
}

func (p *ProcessorImpl) RequestReservation(f field.Model, dropId uint32, characterId uint32, partyId uint32, characterX int16, characterY int16, petSlot int8) error {
	return producer.ProviderImpl(p.l)(p.ctx)(drop2.EnvCommandTopic)(RequestReservationCommandProvider(f, dropId, characterId, partyId, characterX, characterY, petSlot))
}

func (p *ProcessorImpl) SpawnMeso(f field.Model, mesos uint32, x int16, y int16, ownerId uint32, dropperId uint32, dropperX int16, dropperY int16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(drop2.EnvCommandTopic)(SpawnMesoCommandProvider(f, mesos, x, y, ownerId, dropperId, dropperX, dropperY))
}

// ConsumeAll emits one drop CONSUME command per exploded meso drop in a
// single produce call, carrying the attacker's field in the envelope
// (task-150 FR-8). atlas-drops removes each drop and emits CONSUMED; the
// drop consumer then announces the explode animation to the field.
func (p *ProcessorImpl) ConsumeAll(f field.Model, dropIds []uint32) error {
	if len(dropIds) == 0 {
		return nil
	}
	return producer.ProviderImpl(p.l)(p.ctx)(drop2.EnvCommandTopic)(ConsumeAllCommandProvider(uuid.New(), f, dropIds))
}
