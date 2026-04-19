package monster

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
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

func SpawnIndexMigration(db *gorm.DB) error {
	if err := db.AutoMigrate(&SpawnIndexEntity{}); err != nil {
		return err
	}
	return db.Exec("CREATE INDEX IF NOT EXISTS idx_monster_spawn_index_lookup ON monster_spawn_index (tenant_id, monster_id, spawn_count DESC)").Error
}
