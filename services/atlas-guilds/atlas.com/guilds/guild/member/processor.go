package member

import (
	"atlas-guilds/guild/character"
	"context"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func AddMember(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(guildId uint32, characterId uint32, name string, jobId uint16, level byte, title byte) (Model, error) {
	return func(ctx context.Context) func(db *gorm.DB) func(guildId uint32, characterId uint32, name string, jobId uint16, level byte, title byte) (Model, error) {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(guildId uint32, characterId uint32, name string, jobId uint16, level byte, title byte) (Model, error) {
			return func(guildId uint32, characterId uint32, name string, jobId uint16, level byte, title byte) (Model, error) {
				var m Model
				var txErr error
				txErr = db.Transaction(func(tx *gorm.DB) error {
					var err error
					m, err = create(tx, t, guildId, characterId, name, jobId, level, title)
					if err != nil {
						return err
					}

					err = character.SetGuild(l)(ctx)(tx)(characterId, guildId)
					if err != nil {
						return err
					}

					return nil
				})
				return m, txErr
			}
		}
	}
}

func RemoveMember(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(guildId uint32, characterId uint32) error {
	return func(ctx context.Context) func(db *gorm.DB) func(guildId uint32, characterId uint32) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(guildId uint32, characterId uint32) error {
			return func(guildId uint32, characterId uint32) error {
				return db.Transaction(func(tx *gorm.DB) error {
					err := tx.Where("tenant_id = ? AND guild_id = ? AND character_id = ?", t.Id(), guildId, characterId).Delete(&Entity{}).Error
					if err != nil {
						return err
					}

					err = character.SetGuild(l)(ctx)(tx)(characterId, 0)
					if err != nil {
						return err
					}
					return nil
				})
			}
		}
	}
}
