package npc

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
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

func SpawnIndexMigration(db *gorm.DB) error {
	if err := db.AutoMigrate(&SpawnIndexEntity{}); err != nil {
		return err
	}
	return db.Exec("CREATE INDEX IF NOT EXISTS idx_npc_spawn_index_lookup ON npc_spawn_index (tenant_id, npc_id, spawn_count DESC)").Error
}
