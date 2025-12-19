package asset

import (
	"atlas-cashshop/cashshop/item"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Migration sets up the database table for assets
func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

// Entity represents a cash shop inventory asset in the database
type Entity struct {
	Id            uuid.UUID `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	TenantId      uuid.UUID `gorm:"not null"`
	CompartmentId uuid.UUID `gorm:"not null"`
	ItemId        uint32    `gorm:"not null"`
}

// TableName returns the database table name for this entity
func (e Entity) TableName() string {
	return "cash_assets"
}

// Make converts an Entity to a Model
func Make(e Entity) (Model, error) {
	return NewBuilder(
		e.Id,
		e.CompartmentId,
		item.NewBuilder().SetId(e.ItemId).Build(),
	).Build(), nil
}
