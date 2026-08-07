package npc

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
	NpcId      uint32    `gorm:"primaryKey"`
	MapId      uint32    `gorm:"primaryKey"`
	Name       string    `gorm:"not null"`
	StreetName string    `gorm:"not null"`
	SpawnCount uint32    `gorm:"not null"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime"`
}

func (SpawnIndexEntity) TableName() string {
	return "npc_spawn_index"
}

// SpawnMapsFor returns every npc_spawn_index row for npcId, ordered
// most-spawned first, from the tenant partition this request should read.
// See monster.SpawnMapsFor for why the partition is resolved rather than taken
// from the request tenant, and why the read bypasses the automatic tenant
// filter (issue #1213).
func SpawnMapsFor(db *gorm.DB, ctx context.Context, npcId uint32) ([]SpawnIndexEntity, error) {
	partition, err := searchindex.ResolvePartitionTenantId[SpawnIndexEntity](db, ctx)
	if err != nil {
		return nil, err
	}

	var rows []SpawnIndexEntity
	if err := db.WithContext(database.WithoutTenantFilter(ctx)).
		Where("tenant_id = ? AND npc_id = ?", partition, npcId).
		Order("spawn_count DESC, map_id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func SpawnIndexMigration(db *gorm.DB) error {
	if err := db.AutoMigrate(&SpawnIndexEntity{}); err != nil {
		return err
	}
	return db.Exec("CREATE INDEX IF NOT EXISTS idx_npc_spawn_index_lookup ON npc_spawn_index (tenant_id, npc_id, spawn_count DESC)").Error
}
