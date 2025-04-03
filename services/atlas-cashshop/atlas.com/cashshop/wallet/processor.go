package wallet

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func byCharacterIdProvider(ctx context.Context) func(db *gorm.DB) func(characterId uint32) model.Provider[Model] {
	t := tenant.MustFromContext(ctx)
	return func(db *gorm.DB) func(characterId uint32) model.Provider[Model] {
		return func(characterId uint32) model.Provider[Model] {
			return model.Map(Make)(byCharacterIdEntityProvider(t.Id(), characterId)(db))
		}
	}
}

func GetByCharacterId(ctx context.Context) func(db *gorm.DB) func(characterId uint32) (Model, error) {
	return func(db *gorm.DB) func(characterId uint32) (Model, error) {
		return func(characterId uint32) (Model, error) {
			return byCharacterIdProvider(ctx)(db)(characterId)()
		}
	}
}

func Create(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, credit uint32, points uint32, prepaid uint32) (Model, error) {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, credit uint32, points uint32, prepaid uint32) (Model, error) {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32, credit uint32, points uint32, prepaid uint32) (Model, error) {
			return func(characterId uint32, credit uint32, points uint32, prepaid uint32) (Model, error) {
				l.Debugf("Initializing wallet information for character [%d]. Credit [%d], Points [%d], and Prepaid [%d].", characterId, credit, points, prepaid)
				c, err := createEntity(db, t, characterId, credit, points, prepaid)
				if err != nil {
					l.WithError(err).Errorf("Could not create wallet information for character [%d].", characterId)
					return Model{}, err
				}
				return c, err
			}
		}
	}
}

func Update(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, credit uint32, points uint32, prepaid uint32) (Model, error) {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, credit uint32, points uint32, prepaid uint32) (Model, error) {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32, credit uint32, points uint32, prepaid uint32) (Model, error) {
			return func(characterId uint32, credit uint32, points uint32, prepaid uint32) (Model, error) {
				l.Debugf("Updating wallet information for character [%d]. Credit [%d], Points [%d], and Prepaid [%d].", characterId, credit, points, prepaid)
				c, err := updateEntity(db, t, characterId, credit, points, prepaid)
				if err != nil {
					l.WithError(err).Errorf("Could not update wallet information for character [%d].", characterId)
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
				l.Debugf("Character [%d] was deleted. Cleaning up wallet information...", characterId)
				return deleteEntity(ctx)(db, t.Id(), characterId)
			}
		}
	}
}
