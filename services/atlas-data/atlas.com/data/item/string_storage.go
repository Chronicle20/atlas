package item

import (
	"context"
	"strconv"
	"time"

	"atlas-data/document"
	"atlas-data/searchindex"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type StringStorage struct {
	l   logrus.FieldLogger
	db  *gorm.DB
	doc *document.Storage[string, StringRestModel]
}

func NewStringStorage(l logrus.FieldLogger, db *gorm.DB) *StringStorage {
	return &StringStorage{
		l:   l,
		db:  db,
		doc: document.NewStorage[string, StringRestModel](l, db, GetStringModelRegistry(), "ITEM_STRING"),
	}
}

func (s *StringStorage) Logger() logrus.FieldLogger { return s.l }

func (s *StringStorage) ByIdProvider(ctx context.Context) func(id string) model.Provider[StringRestModel] {
	return s.doc.ByIdProvider(ctx)
}

func (s *StringStorage) GetById(ctx context.Context) func(id string) (StringRestModel, error) {
	return s.doc.GetById(ctx)
}

func (s *StringStorage) AllProvider(ctx context.Context) model.Provider[[]StringRestModel] {
	return s.doc.AllProvider(ctx)
}

func (s *StringStorage) GetAll(ctx context.Context) ([]StringRestModel, error) {
	return s.doc.GetAll(ctx)
}

func (s *StringStorage) Add(ctx context.Context) func(m StringRestModel) model.Provider[StringRestModel] {
	return func(m StringRestModel) model.Provider[StringRestModel] {
		t := tenant.MustFromContext(ctx)
		txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			scoped := document.NewStorage[string, StringRestModel](s.l, tx, GetStringModelRegistry(), "ITEM_STRING")
			if _, err := scoped.Add(ctx)(m)(); err != nil {
				return err
			}
			itemId, err := strconv.Atoi(m.Id)
			if err != nil {
				return err
			}
			compartment, subcategory := Classify(uint32(itemId))
			ie := StringSearchIndexEntity{
				TenantId:    t.Id(),
				ItemId:      uint32(itemId),
				Name:        m.Name,
				Compartment: compartment,
				Subcategory: subcategory,
				UpdatedAt:   time.Now(),
			}
			return searchindex.Upsert(tx, &ie,
				[]string{"tenant_id", "item_id"},
				[]string{"name", "compartment", "subcategory", "updated_at"},
			)
		})
		if txErr != nil {
			return model.ErrorProvider[StringRestModel](txErr)
		}
		return model.FixedProvider(m)
	}
}

func (s *StringStorage) Clear(ctx context.Context) error {
	t := tenant.MustFromContext(ctx)
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("type = ?", "ITEM_STRING").Delete(&document.Entity{}).Error; err != nil {
			return err
		}
		return searchindex.DeleteAllForTenant(tx, t.Id(), &StringSearchIndexEntity{})
	})
}
