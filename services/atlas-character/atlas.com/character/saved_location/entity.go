package saved_location

import (
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&entity{})
}

type entity struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;not null"`
	TenantId     uuid.UUID `gorm:"not null;uniqueIndex:idx_saved_location_lookup,priority:1"`
	CharacterId  uint32    `gorm:"not null;uniqueIndex:idx_saved_location_lookup,priority:2"`
	LocationType string    `gorm:"not null;uniqueIndex:idx_saved_location_lookup,priority:3"`
	MapId        _map.Id   `gorm:"not null"`
	PortalId     uint32    `gorm:"not null"`
}

func (e entity) TableName() string {
	return "saved_locations"
}
