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
	Id            uint32    `gorm:"primaryKey;autoIncrement:true"`
	TenantId      uuid.UUID `gorm:"not null"`
	CompartmentId uuid.UUID `gorm:"not null"`
	CashId        int64     `gorm:"not null"`
	TemplateId    uint32    `gorm:"not null"`
	CommodityId   uint32    `gorm:"not null;default:0"`
	// Currency is the wallet bucket (wallet.Model.Balance's convention: 1 =
	// credit/NX, 2 = Maple Points, anything else = prepaid) this asset was
	// purchased with -- recorded so a locker rebate (task-240 task 11) knows
	// which bucket to credit back, instead of guessing. 0 is NOT "unknown":
	// it is the explicit default-bucket convention (controller correction
	// C2) for both (a) every asset that predates this column and (b) every
	// asset created on a gift/reward/surprise path that was never bought
	// with currency at all (C3) -- a rebate treats 0 as the ordinary
	// credit/NX bucket, same as ordinary Purchase's ordinary arm.
	Currency    uint32         `gorm:"not null;default:0"`
	Quantity    uint32         `gorm:"not null"`
	Flag        uint16         `gorm:"not null"`
	PetId       uint32         `gorm:"not null;default:0"`
	PurchasedBy uint32         `gorm:"not null"`
	Expiration  time.Time      `gorm:"not null"`
	CreatedAt   time.Time      `gorm:"not null"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (e Entity) TableName() string {
	return "cash_assets"
}

func Make(e Entity) (Model, error) {
	return NewBuilder(e.CompartmentId, e.TemplateId).
		SetId(e.Id).
		SetCashId(e.CashId).
		SetCommodityId(e.CommodityId).
		SetCurrency(e.Currency).
		SetQuantity(e.Quantity).
		SetFlag(e.Flag).
		SetPetId(e.PetId).
		SetPurchasedBy(e.PurchasedBy).
		SetExpiration(e.Expiration).
		SetCreatedAt(e.CreatedAt).
		Build(), nil
}
