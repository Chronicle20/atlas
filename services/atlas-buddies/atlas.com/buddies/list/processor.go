package list

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func byCharacterIdProvider(_ logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32) model.Provider[Model] {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32) model.Provider[Model] {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32) model.Provider[Model] {
			return func(characterId uint32) model.Provider[Model] {
				return model.Map(Make)(byCharacterIdEntityProvider(t.Id(), characterId)(db))
			}
		}
	}
}

func GetByCharacterId(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32) (Model, error) {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32) (Model, error) {
		return func(db *gorm.DB) func(characterId uint32) (Model, error) {
			return func(characterId uint32) (Model, error) {
				return byCharacterIdProvider(l)(ctx)(db)(characterId)()
			}
		}
	}
}

func Create(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, capacity uint32) (Model, error) {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, capacity uint32) (Model, error) {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32, capacity uint32) (Model, error) {
			return func(characterId uint32, capacity uint32) (Model, error) {
				l.Debugf("Creating buddy list for character [%d] with a capacity of [%d].", characterId, capacity)
				m, err := create(db, t, characterId, capacity)
				if err != nil {
					l.WithError(err).Errorf("Unable to create initial buddy list for character [%d].", characterId)
					return Model{}, err
				}
				return m, nil
			}
		}
	}
}
