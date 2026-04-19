package _map

import (
	"context"
	"time"

	"atlas-data/document"
	"atlas-data/monster"
	"atlas-data/npc"
	"atlas-data/searchindex"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
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
			if err := searchindex.Upsert(tx, &ie,
				[]string{"tenant_id", "map_id"},
				[]string{"name", "street_name", "updated_at"},
			); err != nil {
				return err
			}

			if err := tx.Where("tenant_id = ? AND map_id = ?", t.Id(), uint32(m.Id)).
				Delete(&monster.SpawnIndexEntity{}).Error; err != nil {
				return err
			}

			counts := make(map[uint32]uint32)
			for _, mon := range m.Monsters {
				counts[mon.Template]++
			}
			if len(counts) > 0 {
				rows := make([]monster.SpawnIndexEntity, 0, len(counts))
				now := time.Now()
				for monsterId, count := range counts {
					rows = append(rows, monster.SpawnIndexEntity{
						TenantId:   t.Id(),
						MonsterId:  monsterId,
						MapId:      uint32(m.Id),
						Name:       m.Name,
						StreetName: m.StreetName,
						SpawnCount: count,
						UpdatedAt:  now,
					})
				}
				if err := tx.Create(&rows).Error; err != nil {
					return err
				}
				s.l.Debugf("monster_spawn_index: tenant=%s map=%d rows=%d", t.Id().String(), m.Id, len(rows))
			}

			if err := tx.Where("tenant_id = ? AND map_id = ?", t.Id(), uint32(m.Id)).
				Delete(&npc.SpawnIndexEntity{}).Error; err != nil {
				return err
			}

			npcCounts := make(map[uint32]uint32)
			for _, n := range m.NPCs {
				npcCounts[n.Template]++
			}
			if len(npcCounts) > 0 {
				npcRows := make([]npc.SpawnIndexEntity, 0, len(npcCounts))
				now := time.Now()
				for npcId, count := range npcCounts {
					npcRows = append(npcRows, npc.SpawnIndexEntity{
						TenantId:   t.Id(),
						NpcId:      npcId,
						MapId:      uint32(m.Id),
						Name:       m.Name,
						StreetName: m.StreetName,
						SpawnCount: count,
						UpdatedAt:  now,
					})
				}
				if err := tx.Create(&npcRows).Error; err != nil {
					return err
				}
				s.l.Debugf("npc_spawn_index: tenant=%s map=%d rows=%d", t.Id().String(), m.Id, len(npcRows))
			}
			return nil
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
		return searchindex.DeleteAllForTenant(tx, t.Id(), &SearchIndexEntity{})
	})
}

func DeleteAllSearchIndex(ctx context.Context) func(db *gorm.DB) error {
	return func(db *gorm.DB) error {
		return db.WithContext(ctx).Where("1 = 1").Delete(&SearchIndexEntity{}).Error
	}
}
