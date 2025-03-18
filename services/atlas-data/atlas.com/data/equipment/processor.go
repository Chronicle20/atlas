package equipment

import (
	"atlas-data/document"
	"atlas-data/xml"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func NewStorage(l logrus.FieldLogger, db *gorm.DB) *document.Storage[uint32, RestModel] {
	return document.NewStorage(l, db, GetModelRegistry(), "EQUIPMENT")
}

func Register(s *document.Storage[uint32, RestModel]) func(ctx context.Context) func(r model.Provider[RestModel]) error {
	return func(ctx context.Context) func(r model.Provider[RestModel]) error {
		return func(r model.Provider[RestModel]) error {
			m, err := r()
			if err != nil {
				return err
			}
			_, err = s.Add(ctx)(m)()
			if err != nil {
				return err
			}
			return nil
		}
	}
}

// deprecated
func RegisterEquipment(db *gorm.DB) func(l logrus.FieldLogger) func(ctx context.Context) func(path string) {
	return func(l logrus.FieldLogger) func(ctx context.Context) func(path string) {
		return func(ctx context.Context) func(path string) {
			return func(path string) {
				_ = Register(NewStorage(l, db))(ctx)(Read(l)(xml.FromPathProvider(path)))
			}
		}
	}
}
