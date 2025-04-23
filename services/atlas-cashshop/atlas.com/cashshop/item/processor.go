package item

import (
	"context"

	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"gorm.io/gorm"
)

func byIdProvider(ctx context.Context) func(db *gorm.DB) func(itemId uint32) model.Provider[[]Model] {
	t := tenant.MustFromContext(ctx)
	return func(db *gorm.DB) func(itemId uint32) model.Provider[[]Model] {
		return func(itemId uint32) model.Provider[[]Model] {
			return model.SliceMap(Make)(byIdEntityProvider(t.Id(), itemId)(db))(model.ParallelMap())
		}
	}
}

func GetById(ctx context.Context) func(db *gorm.DB) func(itemId uint32) ([]Model, error) {
	return func(db *gorm.DB) func(itemId uint32) ([]Model, error) {
		return func(itemId uint32) ([]Model, error) {
			return byIdProvider(ctx)(db)(itemId)()
		}
	}
}
