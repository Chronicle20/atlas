package compartment

import (
	"atlas-inventory/asset"
	"context"

	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func ByCharacterIdProvider(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32) model.Provider[[]Model] {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32) model.Provider[[]Model] {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32) model.Provider[[]Model] {
			return func(characterId uint32) model.Provider[[]Model] {
				p := model.SliceMap(Make)(getByCharacter(t.Id(), characterId)(db))(model.ParallelMap())
				cs, err := p()
				if err != nil {
					return model.ErrorProvider[[]Model](err)
				}
				return model.SliceMap(DecorateAsset(l)(ctx)(db))(model.FixedProvider(cs))(model.ParallelMap())
			}
		}
	}
}

func GetByCharacterId(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32) ([]Model, error) {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32) ([]Model, error) {
		return func(db *gorm.DB) func(characterId uint32) ([]Model, error) {
			return func(characterId uint32) ([]Model, error) {
				return ByCharacterIdProvider(l)(ctx)(db)(characterId)()
			}
		}
	}
}

func DecorateAsset(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(m Model) (Model, error) {
	return func(ctx context.Context) func(db *gorm.DB) func(m Model) (Model, error) {
		return func(db *gorm.DB) func(m Model) (Model, error) {
			return func(m Model) (Model, error) {
				as, err := asset.ByCompartmentIdProvider(l)(ctx)(db)(m.Id(), m.Type())()
				if err != nil {
					return Model{}, err
				}
				return Clone(m).SetAssets(as).Build(), nil
			}
		}
	}
}
