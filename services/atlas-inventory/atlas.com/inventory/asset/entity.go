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
	TenantId      uuid.UUID `gorm:"not null"`
	Id            uint32    `gorm:"primaryKey;autoIncrement;not null"`
	CompartmentId uuid.UUID `gorm:"not null"`
	Slot          int16     `gorm:"not null"`
	TemplateId    uint32    `gorm:"not null"`
	Expiration    time.Time `gorm:"not null"`
	ReferenceId   uint32    `gorm:"not null"`
	ReferenceType string    `gorm:"not null"`
}

func (e Entity) TableName() string {
	return "assets"
}

func Make(e Entity) (Model[any], error) {
	return Model[any]{
		id:            e.Id,
		compartmentId: e.CompartmentId,
		slot:          e.Slot,
		templateId:    e.TemplateId,
		expiration:    e.Expiration,
		referenceId:   e.ReferenceId,
		referenceType: ReferenceType(e.ReferenceType),
	}, nil
}
