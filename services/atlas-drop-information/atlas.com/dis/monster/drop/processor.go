package drop

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func allProvider(ctx context.Context) func(db *gorm.DB) model.Provider[[]Model] {
	t := tenant.MustFromContext(ctx)
	return func(db *gorm.DB) model.Provider[[]Model] {
		return model.SliceMap(makeDrop)(getAll(t.Id())(db))()
	}
}

func GetAll(_ logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) ([]Model, error) {
	return func(ctx context.Context) func(db *gorm.DB) ([]Model, error) {
		return func(db *gorm.DB) ([]Model, error) {
			return allProvider(ctx)(db)()
		}
	}
}

func forMonsterProvider(ctx context.Context) func(db *gorm.DB) func(monsterId uint32) model.Provider[[]Model] {
	t := tenant.MustFromContext(ctx)
	return func(db *gorm.DB) func(monsterId uint32) model.Provider[[]Model] {
		return func(monsterId uint32) model.Provider[[]Model] {
			return model.SliceMap(makeDrop)(getByMonsterId(t.Id(), monsterId)(db))()
		}
	}
}

func GetForMonster(_ logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(monsterId uint32) ([]Model, error) {
	return func(ctx context.Context) func(db *gorm.DB) func(monsterId uint32) ([]Model, error) {
		return func(db *gorm.DB) func(monsterId uint32) ([]Model, error) {
			return func(monsterId uint32) ([]Model, error) {
				return forMonsterProvider(ctx)(db)(monsterId)()
			}
		}
	}
}
