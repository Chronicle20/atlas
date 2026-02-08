package asset

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

type Entity struct {
	Id            uint32         `gorm:"primaryKey;autoIncrement:true"`
	TenantId      uuid.UUID      `gorm:"not null"`
	CompartmentId uuid.UUID      `gorm:"not null"`
	CashId        int64          `gorm:"not null"`
	TemplateId    uint32         `gorm:"not null"`
	CommodityId   uint32         `gorm:"not null;default:0"`
	Quantity      uint32         `gorm:"not null"`
	Flag          uint16         `gorm:"not null"`
	PurchasedBy   uint32         `gorm:"not null"`
	Expiration    time.Time      `gorm:"not null"`
	CreatedAt     time.Time      `gorm:"not null"`
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

func (e Entity) TableName() string {
	return "cash_assets"
}

func Make(e Entity) (Model, error) {
	return NewBuilder(e.CompartmentId, e.TemplateId).
		SetId(e.Id).
		SetCashId(e.CashId).
		SetCommodityId(e.CommodityId).
		SetQuantity(e.Quantity).
		SetFlag(e.Flag).
		SetPurchasedBy(e.PurchasedBy).
		SetExpiration(e.Expiration).
		SetCreatedAt(e.CreatedAt).
		Build(), nil
}
