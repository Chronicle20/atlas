package consumable

import (
	"atlas-data/document"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func NewStorage(l logrus.FieldLogger, db *gorm.DB) *document.Storage[uint32, Model] {
	return document.NewStorage(l, db, GetModelRegistry(), "CONSUMABLE")
}

func Register(s *document.Storage[uint32, Model]) func(ctx context.Context) func(r model.Provider[[]Model]) error {
	return func(ctx context.Context) func(r model.Provider[[]Model]) error {
		return func(r model.Provider[[]Model]) error {
			ms, err := r()
			if err != nil {
				return err
			}
			for _, m := range ms {
				_, err = s.Add(ctx)(m)()
				if err != nil {
					return err
				}
			}
			return nil
		}
	}
}

// deprecated
func RegisterConsumable(db *gorm.DB) func(l logrus.FieldLogger) func(ctx context.Context) func(path string) {
	return func(l logrus.FieldLogger) func(ctx context.Context) func(path string) {
		return func(ctx context.Context) func(path string) {
			return func(path string) {
				_ = Register(NewStorage(l, db))(ctx)(ReadFromFile(l)(ctx)(path))
			}
		}
	}
}
