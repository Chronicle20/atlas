package title

import (
	"context"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func CreateDefaults(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(guildId uint32) ([]Model, error) {
	return func(ctx context.Context) func(db *gorm.DB) func(guildId uint32) ([]Model, error) {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(guildId uint32) ([]Model, error) {
			return func(guildId uint32) ([]Model, error) {
				var results []Model
				var txErr error
				txErr = db.Transaction(func(tx *gorm.DB) error {
					var err error
					results, err = createDefault(db, t, guildId)
					if err != nil {
						return err
					}
					return nil
				})
				return results, txErr
			}
		}
	}

}
