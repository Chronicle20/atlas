package pet

import (
	"atlas-data/document"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var DocType = "PET"

func RegisterPet(db *gorm.DB) func(l logrus.FieldLogger) func(ctx context.Context) func(path string) {
	return func(l logrus.FieldLogger) func(ctx context.Context) func(path string) {
		return func(ctx context.Context) func(path string) {
			return func(path string) {
				m, err := ReadFromFile(l)(ctx)(path)()
				if err != nil {
					return
				}
				err = document.Create(ctx)(db)(DocType, m.GetId(), &m)
				if err != nil {
					return
				}
				l.Debugf("Processed pet [%d].", m.GetId())
			}
		}
	}
}

func allProvider(ctx context.Context) func(db *gorm.DB) model.Provider[[]Model] {
	return func(db *gorm.DB) model.Provider[[]Model] {
		t := tenant.MustFromContext(ctx)
		return func() ([]Model, error) {
			ms, err := GetModelRegistry().GetAll(t)
			if err == nil {
				return ms, nil
			}
			ms, err = document.GetAll[RestModel, Model](ctx)(db)(Extract)(DocType)
			if err == nil {
				for _, m := range ms {
					_ = GetModelRegistry().Add(t, m)
				}
				return ms, nil
			}

			nt, err := tenant.Create(uuid.Nil, t.Region(), t.MajorVersion(), t.MinorVersion())
			ms, err = GetModelRegistry().GetAll(nt)
			if err == nil {
				return ms, nil
			}

			nctx := tenant.WithContext(ctx, nt)
			ms, err = document.GetAll[RestModel, Model](nctx)(db)(Extract)(DocType)
			if err == nil {
				for _, m := range ms {
					_ = GetModelRegistry().Add(t, m)
				}
				return ms, nil
			}
			return nil, err
		}
	}
}

func GetAll(ctx context.Context) func(db *gorm.DB) ([]Model, error) {
	return func(db *gorm.DB) ([]Model, error) {
		return allProvider(ctx)(db)()
	}
}

func byIdProvider(ctx context.Context) func(db *gorm.DB) func(id uint32) model.Provider[Model] {
	return func(db *gorm.DB) func(id uint32) model.Provider[Model] {
		t := tenant.MustFromContext(ctx)
		return func(id uint32) model.Provider[Model] {
			return func() (Model, error) {
				m, err := GetModelRegistry().Get(t, id)
				if err == nil {
					return m, nil
				}
				m, err = document.Get[Model](ctx)(db)(DocType, id)
				if err == nil {
					_ = GetModelRegistry().Add(t, m)
					return m, nil
				}
				nt, err := tenant.Create(uuid.Nil, t.Region(), t.MajorVersion(), t.MinorVersion())
				m, err = GetModelRegistry().Get(nt, id)
				if err == nil {
					return m, nil
				}
				nctx := tenant.WithContext(ctx, nt)
				m, err = document.Get[Model](nctx)(db)(DocType, id)
				if err == nil {
					_ = GetModelRegistry().Add(nt, m)
					return m, nil
				}
				return Model{}, err
			}
		}
	}
}

func GetById(ctx context.Context) func(db *gorm.DB) func(id uint32) (Model, error) {
	return func(db *gorm.DB) func(id uint32) (Model, error) {
		return func(id uint32) (Model, error) {
			return byIdProvider(ctx)(db)(id)()
		}
	}
}
