package _map

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SearchIndexEntity struct {
	TenantId   uuid.UUID `gorm:"type:uuid;primaryKey"`
	MapId      uint32    `gorm:"primaryKey"`
	Name       string    `gorm:"not null"`
	StreetName string    `gorm:"not null"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime"`
}

func (SearchIndexEntity) TableName() string {
	return "map_search_index"
}

func Migration(db *gorm.DB) error {
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS pg_trgm").Error; err != nil {
		return err
	}
	if err := db.AutoMigrate(&SearchIndexEntity{}); err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_map_search_index_name_trgm ON map_search_index USING GIN (LOWER(name) gin_trgm_ops)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_map_search_index_street_trgm ON map_search_index USING GIN (LOWER(street_name) gin_trgm_ops)").Error; err != nil {
		return err
	}
	return nil
}
