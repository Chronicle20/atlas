package blacklist

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Entity is one blacklisted visitor name for one shop. Cosmic stores the
// blacklist by character name (the client sends/receives names), so the ban
// key is the name. Tenant-safe PK pattern: uuid surrogate + unique index on
// (tenant, shop, name).
type Entity struct {
	gorm.Model
	Id       uuid.UUID `gorm:"type:uuid;primaryKey"`
	TenantId uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_merchant_blacklists_tenant_shop_name"`
	ShopId   uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_merchant_blacklists_tenant_shop_name"`
	Name     string    `gorm:"not null;uniqueIndex:idx_merchant_blacklists_tenant_shop_name"`
}

func (e *Entity) TableName() string { return "merchant_blacklists" }

func Migration(db *gorm.DB) error { return db.AutoMigrate(&Entity{}) }
