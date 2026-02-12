package item

import (
	"atlas-data/database"
	"atlas-data/document"
	"atlas-data/xml"
	"context"
	"sync"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var isReg *document.Registry[string, StringRestModel]
var isOnce sync.Once

func GetStringModelRegistry() *document.Registry[string, StringRestModel] {
	isOnce.Do(func() {
		isReg = document.NewRegistry[string, StringRestModel]()
	})
	return isReg
}

func NewStringStorage(l logrus.FieldLogger, db *gorm.DB) *document.Storage[string, StringRestModel] {
	return document.NewStorage(l, db, GetStringModelRegistry(), "ITEM_STRING")
}

func InitStringFlat(db *gorm.DB) func(l logrus.FieldLogger) func(ctx context.Context) func(path string) error {
	return func(l logrus.FieldLogger) func(ctx context.Context) func(path string) error {
		return func(ctx context.Context) func(path string) error {
			return func(path string) error {
				exml, err := xml.Read(path)
				if err != nil {
					return err
				}

				return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
					s := NewStringStorage(l, tx)
					for _, mxml := range exml.ChildNodes {
						rm := StringRestModel{
							Id:   mxml.Name,
							Name: mxml.GetString("name", "MISSINGNO"),
						}
						_, err = s.Add(ctx)(rm)()
						if err != nil {
							return err
						}
					}
					return nil
				})
			}
		}
	}
}

func InitStringNested(db *gorm.DB) func(l logrus.FieldLogger) func(ctx context.Context) func(path string) error {
	return func(l logrus.FieldLogger) func(ctx context.Context) func(path string) error {
		return func(ctx context.Context) func(path string) error {
			return func(path string) error {
				exml, err := xml.Read(path)
				if err != nil {
					return err
				}

				return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
					s := NewStringStorage(l, tx)
					for _, cat := range exml.ChildNodes {
						for _, subCat := range cat.ChildNodes {
							for _, mxml := range subCat.ChildNodes {
								rm := StringRestModel{
									Id:   mxml.Name,
									Name: mxml.GetString("name", "MISSINGNO"),
								}
								_, err = s.Add(ctx)(rm)()
								if err != nil {
									return err
								}
							}
						}
					}
					return nil
				})
			}
		}
	}
}
