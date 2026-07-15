package document

import (
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"context"
	"encoding/json"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	results := make([]M, 0)
	docs := make([]Entity, 0)
	err := s.db.WithContext(ctx).Where("type = ?", s.docType).Find(&docs).Error
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

// AllPaged pages this document type's rows over the documents table,
// ordered by document_id (with the schema primary key appended as a
// tie-break by database.PagedQuery), scoped to the context tenant via the
// same "type = ?" filter All uses.
func (s *DbStorage[I, M]) AllPaged(ctx context.Context) func(page model.Page) model.Provider[model.Paged[M]] {
	return func(page model.Page) model.Provider[model.Paged[M]] {
		return func() (model.Paged[M], error) {
			scoped := s.db.WithContext(ctx).Where("type = ?", s.docType).Order("document_id")
			pe, err := database.PagedQuery[Entity](scoped, page)()
			if err != nil {
				return model.Paged[M]{}, err
			}
			ms := make([]M, 0, len(pe.Items))
			for _, doc := range pe.Items {
				var rm M
				if err := jsonapi.Unmarshal(doc.Content, &rm); err != nil {
					return model.Paged[M]{}, err
				}
				ms = append(ms, rm)
			}
			return model.Paged[M]{Items: ms, Total: pe.Total, Page: pe.Page}, nil
		}
	}
}

func (s *DbStorage[I, M]) ById(ctx context.Context) func(id I) model.Provider[M] {
	return func(id I) model.Provider[M] {
		var res M
		doc := Entity{}
		err := s.db.WithContext(ctx).
			Where("type = ? AND document_id = ?", s.docType, id).
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

		txErr := database.ExecuteTransaction(s.db.WithContext(ctx), func(tx *gorm.DB) error {
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
			if err = tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "type"}, {Name: "document_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"content", "updated_at"}),
			}).Create(&e).Error; err != nil {
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
	return s.db.WithContext(ctx).Where("type = ?", s.docType).Delete(&Entity{}).Error
}

func DeleteAll(ctx context.Context) func(db *gorm.DB) error {
	return func(db *gorm.DB) error {
		return db.WithContext(ctx).Where("1 = 1").Delete(&Entity{}).Error
	}
}
