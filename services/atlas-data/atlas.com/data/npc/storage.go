package npc

import (
	"atlas-data/document"
	"atlas-data/searchindex"
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Storage struct {
	l   logrus.FieldLogger
	db  *gorm.DB
	doc *document.Storage[string, RestModel]
}

func NewStorage(l logrus.FieldLogger, db *gorm.DB) *Storage {
	return &Storage{
		l:   l,
		db:  db,
		doc: document.NewStorage[string, RestModel](l, db, GetModelRegistry(), "NPC"),
	}
}

func (s *Storage) Logger() logrus.FieldLogger { return s.l }

func (s *Storage) ByIdProvider(ctx context.Context) func(id string) model.Provider[RestModel] {
	return s.doc.ByIdProvider(ctx)
}

func (s *Storage) GetById(ctx context.Context) func(id string) (RestModel, error) {
	return s.doc.GetById(ctx)
}

func (s *Storage) AllPagedProvider(ctx context.Context) func(page model.Page) model.Provider[model.Paged[RestModel]] {
	return s.doc.AllPagedProvider(ctx)
}

func (s *Storage) Add(ctx context.Context) func(m RestModel) model.Provider[RestModel] {
	return func(m RestModel) model.Provider[RestModel] {
		t := tenant.MustFromContext(ctx)
		txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			scoped := document.NewStorage[string, RestModel](s.l, tx, GetModelRegistry(), "NPC")
			if _, err := scoped.Add(ctx)(m)(); err != nil {
				return err
			}
			ie := SearchIndexEntity{
				TenantId:  t.Id(),
				NpcId:     m.Id,
				Name:      m.Name,
				Storebank: m.Storebank,
				UpdatedAt: time.Now(),
			}
			return searchindex.Upsert(tx, &ie,
				[]string{"tenant_id", "npc_id"},
				[]string{"name", "storebank", "updated_at"},
			)
		})
		if txErr != nil {
			return model.ErrorProvider[RestModel](txErr)
		}
		return model.FixedProvider(m)
	}
}

func (s *Storage) Clear(ctx context.Context) error {
	t := tenant.MustFromContext(ctx)
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("type = ?", "NPC").Delete(&document.Entity{}).Error; err != nil {
			return err
		}
		return searchindex.DeleteAllForTenant(tx, t.Id(), &SearchIndexEntity{})
	})
}
