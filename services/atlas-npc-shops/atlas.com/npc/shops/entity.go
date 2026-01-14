package shops

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Entity is the GORM entity for the shops Model
type Entity struct {
	gorm.Model
	Id        uuid.UUID `gorm:"type:uuid;primaryKey"`
	TenantId  uuid.UUID `gorm:"type:uuid;not null"`
	NpcId     uint32    `gorm:"not null"`
	Recharger bool      `gorm:"not null"`
}

func (e *Entity) TableName() string {
	return "shops"
}

// Make converts an Entity to a Model
func Make(entity Entity) (Model, error) {
	return NewBuilder(entity.NpcId).
		SetRecharger(entity.Recharger).
		Build()
}

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}
