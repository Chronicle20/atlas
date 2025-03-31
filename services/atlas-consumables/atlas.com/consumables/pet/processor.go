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

func ByIdProvider(l logrus.FieldLogger) func(ctx context.Context) func(petId uint64) model.Provider[Model] {
	return func(ctx context.Context) func(petId uint64) model.Provider[Model] {
		return func(petId uint64) model.Provider[Model] {
			return requests.Provider[RestModel, Model](l, ctx)(requestById(petId), Extract)
		}
	}
}

func GetById(l logrus.FieldLogger) func(ctx context.Context) func(petId uint64) (Model, error) {
	return func(ctx context.Context) func(petId uint64) (Model, error) {
		return func(petId uint64) (Model, error) {
			return ByIdProvider(l)(ctx)(petId)()
		}
	}
}

func ByOwnerProvider(l logrus.FieldLogger) func(ctx context.Context) func(ownerId uint32) model.Provider[[]Model] {
	return func(ctx context.Context) func(ownerId uint32) model.Provider[[]Model] {
		return func(ownerId uint32) model.Provider[[]Model] {
			return requests.SliceProvider[RestModel, Model](l, ctx)(requestByOwnerId(ownerId), Extract, model.Filters[Model]())
		}
	}
}

func GetByOwner(l logrus.FieldLogger) func(ctx context.Context) func(ownerId uint32) ([]Model, error) {
	return func(ctx context.Context) func(ownerId uint32) ([]Model, error) {
		return func(ownerId uint32) ([]Model, error) {
			return ByOwnerProvider(l)(ctx)(ownerId)()
		}
	}
}

func SpawnedByOwnerProvider(l logrus.FieldLogger) func(ctx context.Context) func(ownerId uint32) model.Provider[[]Model] {
	return func(ctx context.Context) func(ownerId uint32) model.Provider[[]Model] {
		return func(ownerId uint32) model.Provider[[]Model] {
			return model.FilteredProvider(ByOwnerProvider(l)(ctx)(ownerId), model.Filters[Model](Spawned))
		}
	}
}

func Spawned(m Model) bool {
	return m.Slot() >= 0
}

func HungryByOwnerProvider(l logrus.FieldLogger) func(ctx context.Context) func(ownerId uint32) model.Provider[[]Model] {
	return func(ctx context.Context) func(ownerId uint32) model.Provider[[]Model] {
		return func(ownerId uint32) model.Provider[[]Model] {
			return model.FilteredProvider(SpawnedByOwnerProvider(l)(ctx)(ownerId), model.Filters[Model](Hungry))
		}
	}
}

func Hungry(m Model) bool {
	return m.Fullness() < 100
}

func HungriestByOwnerProvider(l logrus.FieldLogger) func(ctx context.Context) func(ownerId uint32) model.Provider[Model] {
	return func(ctx context.Context) func(ownerId uint32) model.Provider[Model] {
		return func(ownerId uint32) model.Provider[Model] {
			return HungriestToOneProvider(HungryByOwnerProvider(l)(ctx)(ownerId))
		}
	}
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

func AwardFullness(l logrus.FieldLogger) func(ctx context.Context) func(actorId uint32, petId uint64, amount byte) error {
	return func(ctx context.Context) func(actorId uint32, petId uint64, amount byte) error {
		return func(actorId uint32, petId uint64, amount byte) error {
			return producer.ProviderImpl(l)(ctx)(pet2.EnvCommandTopic)(awardFullnessCommandProvider(actorId, petId, amount))
		}
	}
}
