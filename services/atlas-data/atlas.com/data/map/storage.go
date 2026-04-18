package _map

import (
	"context"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"atlas-data/document"
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
		doc: document.NewStorage[string, RestModel](l, db, GetModelRegistry(), "MAP"),
	}
}

func (s *Storage) Logger() logrus.FieldLogger {
	return s.l
}

func (s *Storage) ByIdProvider(ctx context.Context) func(id string) model.Provider[RestModel] {
	return s.doc.ByIdProvider(ctx)
}

func (s *Storage) GetById(ctx context.Context) func(id string) (RestModel, error) {
	return s.doc.GetById(ctx)
}

func (s *Storage) AllProvider(ctx context.Context) model.Provider[[]RestModel] {
	return s.doc.AllProvider(ctx)
}

func (s *Storage) GetAll(ctx context.Context) ([]RestModel, error) {
	return s.doc.GetAll(ctx)
}

func (s *Storage) Add(ctx context.Context) func(m RestModel) model.Provider[RestModel] {
	return func(m RestModel) model.Provider[RestModel] {
		t := tenant.MustFromContext(ctx)
		txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			scoped := document.NewStorage[string, RestModel](s.l, tx, GetModelRegistry(), "MAP")
			if _, err := scoped.Add(ctx)(m)(); err != nil {
				return err
			}
			ie := SearchIndexEntity{
				TenantId:   t.Id(),
				MapId:      uint32(m.Id),
				Name:       m.Name,
				StreetName: m.StreetName,
				UpdatedAt:  time.Now(),
			}
			return tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "tenant_id"}, {Name: "map_id"}},
				DoUpdates: clause.Assignments(map[string]interface{}{
					"name":        m.Name,
					"street_name": m.StreetName,
					"updated_at":  time.Now(),
				}),
			}).Create(&ie).Error
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
		if err := tx.Where("type = ?", "MAP").Delete(&document.Entity{}).Error; err != nil {
			return err
		}
		return tx.Where("tenant_id = ?", t.Id()).Delete(&SearchIndexEntity{}).Error
	})
}

func DeleteAllSearchIndex(ctx context.Context) func(db *gorm.DB) error {
	return func(db *gorm.DB) error {
		return db.WithContext(ctx).Where("1 = 1").Delete(&SearchIndexEntity{}).Error
	}
}
