package commodities

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Entity is the GORM entity for the commodities Model
type Entity struct {
	gorm.Model
	Id           uuid.UUID `gorm:"type:uuid;primaryKey"`
	TenantId     uuid.UUID `gorm:"type:uuid;not null"`
	NpcId        uint32    `gorm:"not null"`
	TemplateId   uint32    `gorm:"not null"`
	MesoPrice    uint32    `gorm:"not null"`
	DiscountRate byte      `gorm:"not null;default:0"`
	TokenTemplateId  uint32    `gorm:"not null;default:0"`
	TokenPrice   uint32    `gorm:"not null;default:0"`
	Period       uint32    `gorm:"not null;default:0"`
	LevelLimit   uint32    `gorm:"not null;default:0"`
}

func (e *Entity) TableName() string {
	return "commodities"
}

// Make converts an Entity to a Model
func Make(entity Entity) (Model, error) {
	return Model{
		id:           entity.Id,
		npcId:        entity.NpcId,
		templateId:   entity.TemplateId,
		mesoPrice:    entity.MesoPrice,
		discountRate: entity.DiscountRate,
		tokenTemplateId:  entity.TokenTemplateId,
		tokenPrice:   entity.TokenPrice,
		period:       entity.Period,
		levelLimit:   entity.LevelLimit,
	}, nil
}

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}
