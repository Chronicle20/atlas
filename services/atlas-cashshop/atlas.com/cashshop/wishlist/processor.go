package wishlist

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func byCharacterIdProvider(ctx context.Context) func(db *gorm.DB) func(characterId uint32) model.Provider[[]Model] {
	t := tenant.MustFromContext(ctx)
	return func(db *gorm.DB) func(characterId uint32) model.Provider[[]Model] {
		return func(characterId uint32) model.Provider[[]Model] {
			return model.SliceMap(Make)(byCharacterIdEntityProvider(t.Id(), characterId)(db))(model.ParallelMap())
		}
	}
}

func GetByCharacterId(ctx context.Context) func(db *gorm.DB) func(characterId uint32) ([]Model, error) {
	return func(db *gorm.DB) func(characterId uint32) ([]Model, error) {
		return func(characterId uint32) ([]Model, error) {
			return byCharacterIdProvider(ctx)(db)(characterId)()
		}
	}
}

func Add(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, serialNumber uint32) (Model, error) {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, serialNumber uint32) (Model, error) {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32, serialNumber uint32) (Model, error) {
			return func(characterId uint32, serialNumber uint32) (Model, error) {
				l.Debugf("Character [%d] adding [%d] to their wishlist.", characterId, serialNumber)
				return createEntity(db, t, characterId, serialNumber)
			}
		}
	}
}

func Delete(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, itemId uuid.UUID) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, itemId uuid.UUID) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32, itemId uuid.UUID) error {
			return func(characterId uint32, itemId uuid.UUID) error {
				l.Debugf("Deleting wish list item [%s] for character [%d].", itemId, characterId)
				return deleteEntity(ctx)(db, t.Id(), characterId, itemId)
			}
		}
	}
}

func DeleteAll(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32) error {
			return func(characterId uint32) error {
				l.Debugf("Deleting wish list for character [%d].", characterId)
				return deleteEntityForCharacter(ctx)(db, t.Id(), characterId)
			}
		}
	}
}
