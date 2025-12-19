package item

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

type Entity struct {
	Id          uint32    `gorm:"primaryKey;autoIncrement:true"`
	TenantId    uuid.UUID `gorm:"not null"`
	CashId      int64     `gorm:"not null"`
	TemplateId  uint32    `gorm:"not null"`
	Quantity    uint32    `gorm:"not null"`
	Flag        uint16    `gorm:"not null"`
	PurchasedBy uint32    `gorm:"not null"`
	Expiration  time.Time `gorm:"not null"`
}

func (e Entity) TableName() string {
	return "items"
}

func Make(e Entity) (Model, error) {
	return Model{
		id:          e.Id,
		cashId:      e.CashId,
		templateId:  e.TemplateId,
		quantity:    e.Quantity,
		flag:        e.Flag,
		purchasedBy: e.PurchasedBy,
		expiration:  e.Expiration,
	}, nil
}
