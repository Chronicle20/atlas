package macro

import (
	"atlas-skills/database"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func byCharacterIdProvider(ctx context.Context) func(db *gorm.DB) func(characterId uint32) model.Provider[[]Model] {
	t := tenant.MustFromContext(ctx)
	return func(db *gorm.DB) func(characterId uint32) model.Provider[[]Model] {
		return func(characterId uint32) model.Provider[[]Model] {
			return model.SliceMap(Make)(getByCharacterId(t.Id(), characterId)(db))(model.ParallelMap())
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

func Update(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, macros []Model) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, macros []Model) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32, macros []Model) error {
			return func(characterId uint32, macros []Model) error {
				l.Debugf("Updating skill macros for character [%d].", characterId)
				txErr := database.ExecuteTransaction(db, func(tx *gorm.DB) error {
					err := deleteByCharacter(tx, t, characterId)
					if err != nil {
						return err
					}
					for _, macro := range macros {
						_, err = create(tx, t.Id(), macro.Id(), characterId, macro.Name(), macro.Shout(), uint32(macro.SkillId1()), uint32(macro.SkillId2()), uint32(macro.SkillId3()))()
						if err != nil {
							return err
						}
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

func Delete(ctx context.Context) func(db *gorm.DB) func(characterId uint32) error {
	t := tenant.MustFromContext(ctx)
	return func(db *gorm.DB) func(characterId uint32) error {
		return func(characterId uint32) error {
			return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
				return deleteByCharacter(tx, t, characterId)
			})
		}
	}
}
