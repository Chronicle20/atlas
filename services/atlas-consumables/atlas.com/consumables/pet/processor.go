package pet

import (
	"atlas-consumables/consumable"
	inventory2 "atlas-consumables/inventory"
	"atlas-consumables/kafka/producer"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/google/uuid"
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
			ps, err := HungryByOwnerProvider(l)(ctx)(ownerId)()
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
	}
}

func ConsumeItem(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, itemId uint32, slot int16, quantity uint32, transactionId uuid.UUID) error {
	return func(ctx context.Context) func(characterId uint32, itemId uint32, slot int16, quantity uint32, transactionId uuid.UUID) error {
		return func(characterId uint32, itemId uint32, slot int16, quantity uint32, transactionId uuid.UUID) error {
			p, err := HungriestByOwnerProvider(l)(ctx)(characterId)()
			if err != nil {
				_ = inventory2.CancelItemReservation(l)(ctx)(characterId, inventory.TypeValueUse, transactionId, slot)
				return err
			}

			ci, err := consumable.GetById(l)(ctx)(itemId)
			if err != nil {
				_ = inventory2.CancelItemReservation(l)(ctx)(characterId, inventory.TypeValueUse, transactionId, slot)
				return err
			}
			inc := byte(0)
			if val, ok := ci.GetSpec(consumable.SpecTypeInc); ok {
				inc = byte(val)
			}

			err = producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(awardFullnessCommandProvider(characterId, p.Id(), inc))
			if err != nil {
				return err
			}

			err = inventory2.ConsumeItem(l)(ctx)(characterId, inventory.TypeValueUse, transactionId, slot)
			if err != nil {
				return err
			}
			return nil
		}
	}
}
