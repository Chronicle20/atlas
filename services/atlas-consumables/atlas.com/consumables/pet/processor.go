package pet

import (
	pet2 "atlas-consumables/kafka/message/pet"
	"atlas-consumables/kafka/producer"
	"context"
	"errors"
	"sort"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	ByIdProvider(petId uint64) model.Provider[Model]
	GetById(petId uint64) (Model, error)
	ByOwnerProvider(ownerId uint32) model.Provider[[]Model]
	GetByOwner(ownerId uint32) ([]Model, error)
	SpawnedByOwnerProvider(ownerId uint32) model.Provider[[]Model]
	HungryByOwnerProvider(ownerId uint32) model.Provider[[]Model]
	HungriestByOwnerProvider(ownerId uint32) model.Provider[Model]
	AwardFullness(actorId uint32, petId uint64, amount byte) error
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

func (p *ProcessorImpl) ByIdProvider(petId uint64) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(petId), Extract)
}

func (p *ProcessorImpl) GetById(petId uint64) (Model, error) {
	return p.ByIdProvider(petId)()
}

// ByOwnerProvider fetches every pet owned by a character. The upstream
// atlas-pets list is now paginated (task-117); callers here (e.g. finding
// the hungriest pet to feed) need the complete set, so this drains every
// page rather than fetching just the first.
func (p *ProcessorImpl) ByOwnerProvider(ownerId uint32) model.Provider[[]Model] {
	return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(byOwnerUrl(ownerId), 250, Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) GetByOwner(ownerId uint32) ([]Model, error) {
	return p.ByOwnerProvider(ownerId)()
}

func (p *ProcessorImpl) SpawnedByOwnerProvider(ownerId uint32) model.Provider[[]Model] {
	return model.FilteredProvider(p.ByOwnerProvider(ownerId), model.Filters[Model](Spawned))
}

func Spawned(m Model) bool {
	return m.Slot() >= 0
}

func (p *ProcessorImpl) HungryByOwnerProvider(ownerId uint32) model.Provider[[]Model] {
	return model.FilteredProvider(p.SpawnedByOwnerProvider(ownerId), model.Filters[Model](Hungry))
}

func Hungry(m Model) bool {
	return m.Fullness() < 100
}

func (p *ProcessorImpl) HungriestByOwnerProvider(ownerId uint32) model.Provider[Model] {
	return HungriestToOneProvider(p.HungryByOwnerProvider(ownerId))
}

func HungriestToOneProvider(p model.Provider[[]Model]) model.Provider[Model] {
	ps, err := p()
	if err != nil {
		return model.ErrorProvider[Model](err)
	}
	if len(ps) == 0 {
		return model.ErrorProvider[Model](errors.New("empty slice"))
	}
	sort.Slice(ps, func(i, j int) bool {
		return ps[i].Fullness() < ps[j].Fullness()
	})
	return model.FixedProvider(ps[0])
}

func IsTemplateFilter(templateIds ...uint32) model.Filter[Model] {
	return func(m Model) bool {
		for _, templateId := range templateIds {
			if m.TemplateId() == templateId {
				return true
			}
		}
		return false
	}
}

func (p *ProcessorImpl) AwardFullness(actorId uint32, petId uint64, amount byte) error {
	return producer.ProviderImpl(p.l)(p.ctx)(pet2.EnvCommandTopic)(awardFullnessCommandProvider(actorId, petId, amount))
}
