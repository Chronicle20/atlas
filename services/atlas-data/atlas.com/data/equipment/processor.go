package equipment

import (
	"context"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"atlas-data/document"
	"atlas-data/item"
	"atlas-data/xml"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func NewStorage(l logrus.FieldLogger, db *gorm.DB) *document.Storage[string, RestModel] {
	return document.NewStorage(l, db, GetModelRegistry(), "EQUIPMENT")
}

func Register(tx *gorm.DB, s *document.Storage[string, RestModel]) func(ctx context.Context) func(r model.Provider[RestModel]) error {
	return func(ctx context.Context) func(r model.Provider[RestModel]) error {
		return func(r model.Provider[RestModel]) error {
			m, err := r()
			if err != nil {
				return err
			}
			if _, err = s.Add(ctx)(m)(); err != nil {
				return err
			}
			slotWZ := ""
			if len(m.EquipSlots) > 0 {
				slotWZ = m.EquipSlots[0].WZ
			}
			return item.UpdateEquipmentClassification(tx, ctx, m.Id, slotWZ, m.ReqJob)
		}
	}
}

func RegisterEquipment(db *gorm.DB) func(l logrus.FieldLogger) func(ctx context.Context) func(path string) error {
	return func(l logrus.FieldLogger) func(ctx context.Context) func(path string) error {
		return func(ctx context.Context) func(path string) error {
			return func(path string) error {
				return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
					return Register(tx, NewStorage(l, tx))(ctx)(Read(l)(xml.FromPathProvider(path)))
				})
			}
		}
	}
}
