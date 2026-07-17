package visit

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Entity is a per-shop visitor tally: how many times a named character has
// entered the shop. Powers the hired-merchant visit-list (mode 0x2E), which
// shows name + visit count. Tenant-safe PK: uuid surrogate + unique index on
// (tenant, shop, name).
type Entity struct {
	gorm.Model
	Id       uuid.UUID `gorm:"type:uuid;primaryKey"`
	TenantId uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_merchant_visits_tenant_shop_name"`
	ShopId   uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_merchant_visits_tenant_shop_name"`
	Name     string    `gorm:"not null;uniqueIndex:idx_merchant_visits_tenant_shop_name"`
	Count    uint32    `gorm:"not null;default:0"`
}

func (e *Entity) TableName() string { return "merchant_visits" }

func Migration(db *gorm.DB) error { return db.AutoMigrate(&Entity{}) }
