package fame

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

type Entity struct {
	TenantId    uuid.UUID `gorm:"not null"`
	Id          uuid.UUID `gorm:"type:uuid;primaryKey"`
	CharacterId uint32    `gorm:"not null"`
	TargetId    uint32    `gorm:"not null"`
	Amount      int8      `gorm:"not null"`
	CreatedAt   time.Time `gorm:"not null"`
}

func (e *Entity) BeforeCreate(_ *gorm.DB) error {
	if e.Id == uuid.Nil {
		e.Id = uuid.New()
	}
	return nil
}

func (e Entity) TableName() string {
	return "logs"
}

func Make(e Entity) (Model, error) {
	return Model{
		tenantId:    e.TenantId,
		id:          e.Id,
		characterId: e.CharacterId,
		targetId:    e.TargetId,
		amount:      e.Amount,
		createdAt:   e.CreatedAt,
	}, nil
}
