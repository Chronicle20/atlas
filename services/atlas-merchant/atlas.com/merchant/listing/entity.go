package listing

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Entity struct {
	gorm.Model
	Id               uuid.UUID `gorm:"type:uuid;primaryKey"`
	TenantId         uuid.UUID `gorm:"type:uuid;not null"`
	ShopId           uuid.UUID `gorm:"type:uuid;not null;index"`
	ItemId           uint32    `gorm:"not null"`
	ItemType         byte      `gorm:"not null"`
	Quantity         uint16    `gorm:"not null"`
	BundleSize       uint16    `gorm:"not null"`
	BundlesRemaining uint16    `gorm:"not null"`
	PricePerBundle   uint32    `gorm:"not null"`
	ItemSnapshot     []byte    `gorm:"type:jsonb"`
	TransactionId    uuid.UUID `gorm:"type:uuid"`
	DisplayOrder     uint16    `gorm:"not null;default:0"`
	Version          uint32    `gorm:"not null;default:1"`
	ListedAt         time.Time `gorm:"not null"`
}

func (e *Entity) TableName() string {
	return "listings"
}

func Make(entity Entity) (Model, error) {
	return NewBuilder().
		SetId(entity.Id).
		SetShopId(entity.ShopId).
		SetItemId(entity.ItemId).
		SetItemType(entity.ItemType).
		SetQuantity(entity.Quantity).
		SetBundleSize(entity.BundleSize).
		SetBundlesRemaining(entity.BundlesRemaining).
		SetPricePerBundle(entity.PricePerBundle).
		SetItemSnapshot(entity.ItemSnapshot).
		SetDisplayOrder(entity.DisplayOrder).
		SetListedAt(entity.ListedAt).
		Build()
}

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}
