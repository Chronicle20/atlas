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
	return NewBuilder().
		SetId(entity.Id).
		SetNpcId(entity.NpcId).
		SetTemplateId(entity.TemplateId).
		SetMesoPrice(entity.MesoPrice).
		SetDiscountRate(entity.DiscountRate).
		SetTokenTemplateId(entity.TokenTemplateId).
		SetTokenPrice(entity.TokenPrice).
		SetPeriod(entity.Period).
		SetLevelLimit(entity.LevelLimit).
		Build()
}

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}
