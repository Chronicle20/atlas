package stackable

import (
	"context"

	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func ByCompartmentIdProvider(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(compartmentId uuid.UUID) model.Provider[[]Model] {
	return func(ctx context.Context) func(db *gorm.DB) func(compartmentId uuid.UUID) model.Provider[[]Model] {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(compartmentId uuid.UUID) model.Provider[[]Model] {
			return func(compartmentId uuid.UUID) model.Provider[[]Model] {
				return model.SliceMap(Make)(getByCompartmentId(t.Id(), compartmentId)(db))(model.ParallelMap())
			}
		}
	}
}

func Identity(m Model) uint32 {
	return m.id
}

func This(m Model) Model {
	return m
}

func Delete(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(id uint32) error {
	return func(ctx context.Context) func(db *gorm.DB) func(id uint32) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(id uint32) error {
			return func(id uint32) error {
				l.Debugf("Attempting to delete stackable item [%d].", id)
				return deleteById(db, t.Id(), id)
			}
		}
	}
}
