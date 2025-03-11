package pet

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func RegisterPet(l logrus.FieldLogger) func(ctx context.Context) func(path string) {
	return func(ctx context.Context) func(path string) {
		t := tenant.MustFromContext(ctx)
		return func(path string) {
			m, err := ReadFromFile(l)(ctx)(path)()
			if err == nil {
				l.Debugf("Processed pet [%d].", m.Id())
				_ = GetPetModelRegistry().Add(t, m)
			}
		}
	}
}

func allProvider(ctx context.Context) model.Provider[[]Model] {
	t := tenant.MustFromContext(ctx)
	return func() ([]Model, error) {
		m, err := GetPetModelRegistry().GetAll(t)
		if err == nil {
			return m, nil
		}
		nt, err := tenant.Create(uuid.Nil, t.Region(), t.MajorVersion(), t.MinorVersion())
		if err != nil {
			return []Model{}, err
		}
		return GetPetModelRegistry().GetAll(nt)
	}
}

func GetAll(ctx context.Context) func() ([]Model, error) {
	return func() ([]Model, error) {
		return allProvider(ctx)()
	}
}

func byIdProvider(ctx context.Context) func(petId uint64) model.Provider[Model] {
	t := tenant.MustFromContext(ctx)
	return func(petId uint64) model.Provider[Model] {
		return func() (Model, error) {
			m, err := GetPetModelRegistry().Get(t, petId)
			if err == nil {
				return m, nil
			}
			nt, err := tenant.Create(uuid.Nil, t.Region(), t.MajorVersion(), t.MinorVersion())
			if err != nil {
				return Model{}, err
			}
			return GetPetModelRegistry().Get(nt, petId)
		}
	}
}

func GetById(ctx context.Context) func(petId uint64) (Model, error) {
	return func(petId uint64) (Model, error) {
		return byIdProvider(ctx)(petId)()
	}
}
