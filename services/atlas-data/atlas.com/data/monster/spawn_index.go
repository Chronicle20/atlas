package monster

import (
	"atlas-data/searchindex"
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
)

type SpawnIndexEntity struct {
	TenantId   uuid.UUID `gorm:"type:uuid;primaryKey"`
	MonsterId  uint32    `gorm:"primaryKey"`
	MapId      uint32    `gorm:"primaryKey"`
	Name       string    `gorm:"not null"`
	StreetName string    `gorm:"not null"`
	SpawnCount uint32    `gorm:"not null"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime"`
}

func (SpawnIndexEntity) TableName() string {
	return "monster_spawn_index"
}

// SpawnMapsFor returns every monster_spawn_index row for monsterId, ordered
// most-spawned first, from the tenant partition this request should read.
//
// monster_spawn_index rows are derived from MAP documents by map.Storage.Add,
// so they land in whichever partition ran the ingest — the version-scoped
// canonical tenant for shared content, not the requesting tenant (issue #1213).
// The partition is resolved against this table specifically, then the read
// bypasses the automatic tenant filter so the GORM tenant callback cannot
// re-inject the request tenant and contradict it. tenant_id remains explicitly
// constrained to the resolved partition, which is only ever the request
// tenant's own id or the canonical id derived from its own region/version.
func SpawnMapsFor(db *gorm.DB, ctx context.Context, monsterId uint32) ([]SpawnIndexEntity, error) {
	partition, err := searchindex.ResolvePartitionTenantId[SpawnIndexEntity](db, ctx)
	if err != nil {
		return nil, err
	}

	var rows []SpawnIndexEntity
	if err := db.WithContext(database.WithoutTenantFilter(ctx)).
		Where("tenant_id = ? AND monster_id = ?", partition, monsterId).
		Order("spawn_count DESC, name ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func SpawnIndexMigration(db *gorm.DB) error {
	if err := db.AutoMigrate(&SpawnIndexEntity{}); err != nil {
		return err
	}
	return db.Exec("CREATE INDEX IF NOT EXISTS idx_monster_spawn_index_lookup ON monster_spawn_index (tenant_id, monster_id, spawn_count DESC)").Error
}
