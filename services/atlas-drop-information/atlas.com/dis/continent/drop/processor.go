package drop

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func GetAll(_ logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func() ([]Model, error) {
	return func(ctx context.Context) func(db *gorm.DB) func() ([]Model, error) {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func() ([]Model, error) {
			return func() ([]Model, error) {
				return model.SliceMap(makeDrop)(getAll(t.Id())(db))()()
			}
		}
	}
}
