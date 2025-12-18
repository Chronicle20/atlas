package compartment

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Migration sets up the database table for compartments
func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

// Entity represents a cash shop inventory compartment in the database
type Entity struct {
	Id        uuid.UUID `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	TenantId  uuid.UUID `gorm:"not null"`
	AccountId uint32    `gorm:"not null"`
	Type      byte      `gorm:"not null"`
	Capacity  uint32    `gorm:"not null;default:55"`
}

// TableName returns the database table name for this entity
func (e Entity) TableName() string {
	return "cash_compartments"
}

// Make converts an Entity to a Model
func Make(e Entity) (Model, error) {
	return NewBuilder(
		e.Id,
		e.AccountId,
		CompartmentType(e.Type),
		e.Capacity,
	).Build(), nil
}
