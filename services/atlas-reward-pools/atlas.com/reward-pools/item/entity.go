package item

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&entity{})
}

type entity struct {
	TenantId   uuid.UUID `gorm:"not null"`
	ID         uint32    `gorm:"primaryKey;autoIncrement;not null"`
	GachaponId string    `gorm:"not null;index:idx_gachapon_items_tier"`
	ItemId     uint32    `gorm:"not null"`
	Quantity   uint32    `gorm:"not null;default:1"`
	Tier       string    `gorm:"not null;index:idx_gachapon_items_tier"`
	// Weight is an optional explicit roll weight for this item, used by
	// weighted (e.g. incubator) reward pools. `default:0` backfills
	// pre-existing rows when AutoMigrate adds this column; the existing
	// tier-based roll does not consume it.
	Weight uint32 `gorm:"not null;default:0"`
	// CommodityId is the cash shop commodity (serial number) awarded by a
	// cash-surprise pool entry. The commodity catalog owns the reward's
	// itemId/count/period, so rolling a commodity guarantees a
	// self-consistent locker entry (task-207). `default:0` backfills
	// pre-existing rows when AutoMigrate adds this column; gachapon and
	// incubator entries leave it 0 and keep using ItemId.
	CommodityId uint32 `gorm:"not null;default:0"`
}

func (e entity) TableName() string {
	return "gachapon_items"
}
