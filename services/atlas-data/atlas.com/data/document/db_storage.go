package document

import (
	"atlas-data/database"
	"context"
	"encoding/json"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"strconv"
)

type Identifier[I string] interface {
	GetID() I
}

type DbStorage[I string, M Identifier[I]] struct {
	l       logrus.FieldLogger
	db      *gorm.DB
	docType string
}

func NewDbStorage[I string, M Identifier[I]](l logrus.FieldLogger, db *gorm.DB, docType string) *DbStorage[I, M] {
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
		err = jsonapi.Unmarshal(doc.Content, &rm)
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

		err = jsonapi.Unmarshal(doc.Content, &res)
		if err != nil {
			return model.ErrorProvider[M](err)
		}
		return model.FixedProvider[M](res)
	}
}

type Server struct {
	baseUrl string
	prefix  string
}

func (s Server) GetBaseURL() string {
	return s.baseUrl
}

func (s Server) GetPrefix() string {
	return s.prefix
}

func getServer() Server {
	return Server{
		baseUrl: "",
		prefix:  "/api/",
	}
}

func (s *DbStorage[I, M]) Add(ctx context.Context) func(m M) model.Provider[M] {
	t := tenant.MustFromContext(ctx)
	return func(m M) model.Provider[M] {
		d, err := jsonapi.MarshalToStruct(m, getServer())
		if err != nil {
			return model.ErrorProvider[M](err)
		}
		data, err := json.Marshal(d)
		if err != nil {
			return model.ErrorProvider[M](err)
		}

		txErr := database.ExecuteTransaction(s.db, func(tx *gorm.DB) error {
			docId, err := strconv.Atoi(string(m.GetID()))
			if err != nil {
				return err
			}

			e := Entity{
				TenantId:   t.Id(),
				Type:       s.docType,
				DocumentId: uint32(docId),
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
	return s.db.Where(&Entity{TenantId: t.Id(), Type: s.docType}).Delete(&Entity{}).Error
}

func DeleteAll(ctx context.Context) func(db *gorm.DB) error {
	t := tenant.MustFromContext(ctx)
	return func(db *gorm.DB) error {
		return db.Where(&Entity{TenantId: t.Id()}).Delete(&Entity{}).Error
	}
}
