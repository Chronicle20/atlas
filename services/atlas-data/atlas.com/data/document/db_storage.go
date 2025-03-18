package document

import (
	"context"
	"encoding/json"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"strconv"
)

type Identifier[I uint32] interface {
	GetId() I
}

type DbStorage[I uint32, M Identifier[I]] struct {
	l       logrus.FieldLogger
	db      *gorm.DB
	docType string
}

func NewDbStorage[I uint32, M Identifier[I]](l logrus.FieldLogger, db *gorm.DB, docType string) *DbStorage[I, M] {
	return &DbStorage[I, M]{
		l:       l,
		db:      db,
		docType: docType,
	}
}

func (s *DbStorage[I, M]) All(ctx context.Context) model.Provider[[]M] {
	t := tenant.MustFromContext(ctx)
	results := make([]M, 0)
	docs := make([]Entity, 0)
	err := s.db.Where(&Entity{TenantId: t.Id(), Type: s.docType}).Find(&docs).Error
	if err != nil {
		return model.ErrorProvider[[]M](err)
	}

	for _, doc := range docs {
		var rm M
		err = json.Unmarshal(doc.Content, &rm)
		if err != nil {
			return model.ErrorProvider[[]M](err)
		}
		results = append(results, rm)
	}
	return model.FixedProvider[[]M](results)
}

func (s *DbStorage[I, M]) ById(ctx context.Context) func(id I) model.Provider[M] {
	t := tenant.MustFromContext(ctx)
	return func(id I) model.Provider[M] {
		var res M
		doc := Entity{}
		err := s.db.
			Where("tenant_id = ? AND type = ? AND document_id = ?", t.Id(), s.docType, id).
			First(&doc).Error
		if err != nil {
			return model.ErrorProvider[M](err)
		}

		err = json.Unmarshal(doc.Content, &res)
		if err != nil {
			return model.ErrorProvider[M](err)
		}
		return model.FixedProvider[M](res)
	}
}

func (s *DbStorage[I, M]) Add(ctx context.Context) func(m M) model.Provider[M] {
	t := tenant.MustFromContext(ctx)
	return func(m M) model.Provider[M] {
		data, err := json.Marshal(m)
		if err != nil {
			return model.ErrorProvider[M](err)
		}

		txErr := s.db.Transaction(func(tx *gorm.DB) error {
			e := Entity{
				TenantId:   t.Id(),
				Type:       s.docType,
				DocumentId: uint32(m.GetId()),
				Content:    data,
			}
			if err = tx.Create(&e).Error; err != nil {
				return err
			}
			return nil
		})
		if txErr != nil {
			return model.ErrorProvider[M](txErr)
		}
		return model.FixedProvider[M](m)
	}
}

func (s *DbStorage[I, M]) Clear(ctx context.Context) error {
	t := tenant.MustFromContext(ctx)
	return s.db.Where(&Entity{TenantId: t.Id()}).Delete(&Entity{}).Error
}

// deprecated
func DeleteAll(ctx context.Context) func(db *gorm.DB) error {
	t := tenant.MustFromContext(ctx)
	return func(db *gorm.DB) error {
		return db.Where(&Entity{TenantId: t.Id()}).Delete(&Entity{}).Error
	}
}

// deprecated
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

// deprecated
func Get[M any](ctx context.Context) func(db *gorm.DB) func(docType string, docId uint32) (M, error) {
	t := tenant.MustFromContext(ctx)
	return func(db *gorm.DB) func(docType string, docId uint32) (M, error) {
		return func(docType string, docId uint32) (M, error) {
			var res M
			doc := Entity{}
			err := db.
				Where("tenant_id = ? AND type = ? AND document_id = ?", t.Id(), docType, docId).
				First(&doc).Error
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

// deprecated
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
