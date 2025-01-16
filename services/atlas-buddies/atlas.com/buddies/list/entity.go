package buddy

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&entity{})
}

type entity struct {
	TenantId    uuid.UUID `gorm:"not null"`
	CharacterId uint32    `gorm:"primaryKey;autoIncrement:false;not null"`
	Key         int32     `gorm:"primaryKey;autoIncrement:false;not null"`
	Type        int8      `gorm:"not null"`
	Action      int32     `gorm:"not null"`
}

func (e entity) TableName() string {
	return "buddies"
}
