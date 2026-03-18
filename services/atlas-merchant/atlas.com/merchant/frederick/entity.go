package frederick

import (
	"atlas-merchant/kafka/message/asset"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ItemEntity struct {
	gorm.Model
	Id           uuid.UUID       `gorm:"type:uuid;primaryKey"`
	TenantId     uuid.UUID       `gorm:"type:uuid;not null"`
	CharacterId  uint32          `gorm:"not null;index"`
	ItemId       uint32          `gorm:"not null"`
	ItemType     byte            `gorm:"not null"`
	Quantity     uint16          `gorm:"not null"`
	ItemSnapshot asset.AssetData `gorm:"type:jsonb"`
	StoredAt     time.Time       `gorm:"not null"`
	LastNotified *time.Time
}

func (e *ItemEntity) TableName() string {
	return "frederick_items"
}

type MesoEntity struct {
	gorm.Model
	Id          uuid.UUID `gorm:"type:uuid;primaryKey"`
	TenantId    uuid.UUID `gorm:"type:uuid;not null"`
	CharacterId uint32    `gorm:"not null;index"`
	Amount      uint32    `gorm:"not null"`
	StoredAt    time.Time `gorm:"not null"`
}

func (e *MesoEntity) TableName() string {
	return "frederick_mesos"
}

func Migration(db *gorm.DB) error {
	err := db.AutoMigrate(&ItemEntity{})
	if err != nil {
		return err
	}
	err = db.AutoMigrate(&MesoEntity{})
	if err != nil {
		return err
	}
	return db.AutoMigrate(&NotificationEntity{})
}
