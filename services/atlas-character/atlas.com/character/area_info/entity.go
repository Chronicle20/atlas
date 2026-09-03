package area_info

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&entity{})
}

type entity struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;not null"`
	TenantId    uuid.UUID `gorm:"not null;uniqueIndex:idx_area_info_lookup,priority:1"`
	CharacterId uint32    `gorm:"not null;uniqueIndex:idx_area_info_lookup,priority:2"`
	Area        uint16    `gorm:"not null;uniqueIndex:idx_area_info_lookup,priority:3"`
	Info        string    `gorm:"not null"`
}

func (e entity) TableName() string {
	return "area_infos"
}
