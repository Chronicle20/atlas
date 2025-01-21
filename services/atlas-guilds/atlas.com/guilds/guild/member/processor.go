package member

import (
	"context"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func AddMember(_ logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(guildId uint32, characterId uint32, name string, jobId uint16, level byte, rank byte) (Model, error) {
	return func(ctx context.Context) func(db *gorm.DB) func(guildId uint32, characterId uint32, name string, jobId uint16, level byte, rank byte) (Model, error) {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(guildId uint32, characterId uint32, name string, jobId uint16, level byte, rank byte) (Model, error) {
			return func(guildId uint32, characterId uint32, name string, jobId uint16, level byte, rank byte) (Model, error) {
				var m Model
				var txErr error
				txErr = db.Transaction(func(tx *gorm.DB) error {
					var err error
					m, err = create(tx, t, guildId, characterId, name, jobId, level, rank)
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
