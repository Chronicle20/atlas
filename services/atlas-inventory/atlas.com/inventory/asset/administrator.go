package asset

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func create(db *gorm.DB, tenantId uuid.UUID, compartmentId uuid.UUID, templateId uint32, slot int16, expiration time.Time, referenceId uint32, referenceType ReferenceType) (Model[any], error) {
	e := &Entity{
		TenantId:      tenantId,
		CompartmentId: compartmentId,
		Slot:          slot,
		TemplateId:    templateId,
		Expiration:    expiration,
		ReferenceId:   referenceId,
		ReferenceType: string(referenceType),
	}

	err := db.Create(e).Error
	if err != nil {
		return Model[any]{}, err
	}
	return Make(*e)
}

func updateSlot(db *gorm.DB, tenantId uuid.UUID, id uint32, slot int16) error {
	return db.Model(&Entity{TenantId: tenantId, Id: id}).Select("Slot").Updates(&Entity{Slot: slot}).Error
}

func deleteById(db *gorm.DB, tenantId uuid.UUID, id uint32) error {
	return db.Where(&Entity{TenantId: tenantId, Id: id}).Delete(&Entity{}).Error
}
