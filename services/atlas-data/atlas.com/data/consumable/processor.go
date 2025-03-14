package consumable

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func RegisterConsumable(l logrus.FieldLogger) func(ctx context.Context) func(path string) {
	return func(ctx context.Context) func(path string) {
		t := tenant.MustFromContext(ctx)
		return func(path string) {
			ms, err := ReadFromFile(l)(ctx)(path)()
			if err != nil {
				return
			}
			for _, m := range ms {
				l.Debugf("Processed consumable [%d].", m.Id())
				_ = GetConsumableModelRegistry().Add(t, m)
			}
		}
	}
}

func allProvider(ctx context.Context) model.Provider[[]Model] {
	t := tenant.MustFromContext(ctx)
	return func() ([]Model, error) {
		m, err := GetConsumableModelRegistry().GetAll(t)
		if err == nil {
			return m, nil
		}
		nt, err := tenant.Create(uuid.Nil, t.Region(), t.MajorVersion(), t.MinorVersion())
		if err != nil {
			return []Model{}, err
		}
		return GetConsumableModelRegistry().GetAll(nt)
	}
}

func GetAll(ctx context.Context) func() ([]Model, error) {
	return func() ([]Model, error) {
		return allProvider(ctx)()
	}
}

func byIdProvider(ctx context.Context) func(consumableId uint32) model.Provider[Model] {
	t := tenant.MustFromContext(ctx)
	return func(consumableId uint32) model.Provider[Model] {
		return func() (Model, error) {
			m, err := GetConsumableModelRegistry().Get(t, consumableId)
			if err == nil {
				return m, nil
			}
			nt, err := tenant.Create(uuid.Nil, t.Region(), t.MajorVersion(), t.MinorVersion())
			if err != nil {
				return Model{}, err
			}
			return GetConsumableModelRegistry().Get(nt, consumableId)
		}
	}
}

func GetById(ctx context.Context) func(consumableId uint32) (Model, error) {
	return func(consumableId uint32) (Model, error) {
		return byIdProvider(ctx)(consumableId)()
	}
}
