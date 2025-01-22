package character

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func byIdProvider(_ logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32) model.Provider[Model] {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32) model.Provider[Model] {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32) model.Provider[Model] {
			return func(characterId uint32) model.Provider[Model] {
				return model.Map(Make)(getById(t.Id(), characterId)(db))
			}
		}
	}
}

func GetById(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32) (Model, error) {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32) (Model, error) {
		return func(db *gorm.DB) func(characterId uint32) (Model, error) {
			return func(characterId uint32) (Model, error) {
				return byIdProvider(l)(ctx)(db)(characterId)()
			}
		}
	}
}

func SetGuild(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, guildId uint32) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, guildId uint32) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32, guildId uint32) error {
			return func(characterId uint32, guildId uint32) error {
				return db.Transaction(func(tx *gorm.DB) error {
					c, _ := getById(t.Id(), characterId)(tx)()
					if c.GuildId != 0 {
						c.GuildId = guildId
						return tx.Save(&c).Error
					}
					c = Entity{
						TenantId:    t.Id(),
						CharacterId: characterId,
						GuildId:     guildId,
					}
					return tx.Save(&c).Error
				})
			}
		}
	}
}
