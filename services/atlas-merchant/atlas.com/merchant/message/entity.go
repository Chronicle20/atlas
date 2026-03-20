package message

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Entity struct {
	gorm.Model
	Id          uuid.UUID `gorm:"type:uuid;primaryKey"`
	TenantId    uuid.UUID `gorm:"type:uuid;not null"`
	ShopId      uuid.UUID `gorm:"type:uuid;not null;index"`
	CharacterId uint32    `gorm:"not null"`
	Content     string    `gorm:"type:text;not null"`
	SentAt      time.Time `gorm:"not null"`
}

func (e *Entity) TableName() string {
	return "messages"
}

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}
