package inventory

import (
	"atlas-inventory/compartment"
	"context"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func ByCharacterIdProvider(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32) model.Provider[Model] {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32) model.Provider[Model] {
		return func(db *gorm.DB) func(characterId uint32) model.Provider[Model] {
			return func(characterId uint32) model.Provider[Model] {
				b, err := model.Fold(compartment.ByCharacterIdProvider(l)(ctx)(db)(characterId), BuilderSupplier(characterId), FoldCompartment)()
				if err != nil {
					return model.ErrorProvider[Model](err)
				}
				return model.FixedProvider(b.Build())
			}
		}
	}
}

func GetByCharacterId(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32) (Model, error) {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32) (Model, error) {
		return func(db *gorm.DB) func(characterId uint32) (Model, error) {
			return func(characterId uint32) (Model, error) {
				return ByCharacterIdProvider(l)(ctx)(db)(characterId)()
			}
		}
	}
}
