package pet

import (
	pet2 "atlas-consumables/kafka/message/pet"
	"atlas-consumables/kafka/producer"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
	"sort"
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

func (p *Processor) ByIdProvider(petId uint64) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(petId), Extract)
}

func (p *Processor) GetById(petId uint64) (Model, error) {
	return p.ByIdProvider(petId)()
}

func (p *Processor) ByOwnerProvider(ownerId uint32) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByOwnerId(ownerId), Extract, model.Filters[Model]())
}

func (p *Processor) GetByOwner(ownerId uint32) ([]Model, error) {
	return p.ByOwnerProvider(ownerId)()
}

func (p *Processor) SpawnedByOwnerProvider(ownerId uint32) model.Provider[[]Model] {
	return model.FilteredProvider(p.ByOwnerProvider(ownerId), model.Filters[Model](Spawned))
}

func Spawned(m Model) bool {
	return m.Slot() >= 0
}

func (p *Processor) HungryByOwnerProvider(ownerId uint32) model.Provider[[]Model] {
	return model.FilteredProvider(p.SpawnedByOwnerProvider(ownerId), model.Filters[Model](Hungry))
}

func Hungry(m Model) bool {
	return m.Fullness() < 100
}

func (p *Processor) HungriestByOwnerProvider(ownerId uint32) model.Provider[Model] {
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

func (p *Processor) AwardFullness(actorId uint32, petId uint64, amount byte) error {
	return producer.ProviderImpl(p.l)(p.ctx)(pet2.EnvCommandTopic)(awardFullnessCommandProvider(actorId, petId, amount))
}
