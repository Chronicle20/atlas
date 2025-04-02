package cash

import (
	"atlas-cashshop/cash/wishlist"
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

func Create(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32) (Model, error) {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32) (Model, error) {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32) (Model, error) {
			return func(characterId uint32) (Model, error) {
				l.Debugf("Character [%d] was created. Initializing cash shop information...", characterId)
				c, err := createEntity(db, t, characterId, 0, 0, 0)
				if err != nil {
					l.WithError(err).Error("Could not create cash shop information for character [%d].", characterId)
					return Model{}, err
				}
				return c, err
			}
		}
	}
}

func Delete(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32) error {
			return func(characterId uint32) error {
				l.Debugf("Character [%d] was deleted. Cleaning up cash shop information...", characterId)
				txErr := db.Transaction(func(tx *gorm.DB) error {
					err := wishlist.DeleteAll(l)(ctx)(tx)(characterId)
					if err != nil {
						return err
					}
					err = deleteEntity(ctx)(tx, t.Id(), characterId)
					if err != nil {
						return err
					}
					return nil
				})
				if txErr != nil {
					return txErr
				}
				return nil
			}
		}
	}
}
