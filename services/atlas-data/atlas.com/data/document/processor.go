package document

import (
	"context"
	"encoding/json"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/jtumidanski/api2go/jsonapi"
	"gorm.io/gorm"
	"strconv"
)

func DeleteAll(ctx context.Context) func(db *gorm.DB) error {
	t := tenant.MustFromContext(ctx)
	return func(db *gorm.DB) error {
		return db.Where(&Entity{TenantId: t.Id()}).Delete(&Entity{}).Error
	}
}

func Create(ctx context.Context) func(db *gorm.DB) func(docType string, docId uint32, object interface{}) error {
	t := tenant.MustFromContext(ctx)
	return func(db *gorm.DB) func(docType string, docId uint32, object interface{}) error {
		return func(docType string, docId uint32, object interface{}) error {
			data, err := json.Marshal(object)
			if err != nil {
				return err
			}

			return db.Transaction(func(tx *gorm.DB) error {
				e := Entity{
					TenantId:   t.Id(),
					Type:       docType,
					DocumentId: docId,
					Content:    data,
				}
				if err = tx.Create(&e).Error; err != nil {
					return err
				}
				return nil
			})
		}
	}
}

func Get[M any](ctx context.Context) func(db *gorm.DB) func(docType string, docId uint32) (M, error) {
	t := tenant.MustFromContext(ctx)
	return func(db *gorm.DB) func(docType string, docId uint32) (M, error) {
		return func(docType string, docId uint32) (M, error) {
			var res M
			doc := Entity{}
			err := db.Where(&Entity{TenantId: t.Id(), Type: docType, DocumentId: docId}).First(&doc).Error
			if err != nil {
				return res, err
			}

			err = json.Unmarshal(doc.Content, &res)
			if err != nil {
				return res, err
			}
			return res, nil
		}
	}
}

func GetAll[R any, M any](ctx context.Context) func(db *gorm.DB) func(tf model.Transformer[R, M]) func(docType string) ([]M, error) {
	t := tenant.MustFromContext(ctx)
	return func(db *gorm.DB) func(tf model.Transformer[R, M]) func(docType string) ([]M, error) {
		return func(tf model.Transformer[R, M]) func(docType string) ([]M, error) {
			return func(docType string) ([]M, error) {
				results := make([]M, 0)
				docs := make([]Entity, 0)
				err := db.Where(&Entity{TenantId: t.Id(), Type: docType}).Find(&docs).Error
				if err != nil {
					return results, err
				}

				for _, doc := range docs {
					var rm R
					err = jsonapi.Unmarshal(doc.Content, &rm)
					if err != nil {
						return results, err
					}
					var obj interface{}
					obj = &rm
					if val, ok := obj.(jsonapi.UnmarshalIdentifier); ok {
						err = val.SetID(strconv.Itoa(int(doc.DocumentId)))
						if err != nil {
							return results, err
						}
					}

					var res M
					res, err = tf(rm)
					if err != nil {
						return results, err
					}
					results = append(results, res)
				}
				return results, nil
			}
		}
	}
}
