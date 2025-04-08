package asset

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func updateSlot(db *gorm.DB, tenantId uuid.UUID, id uint32, slot int16) error {
	return db.Model(&Entity{TenantId: tenantId, Id: id}).Select("Slot").Updates(&Entity{Slot: slot}).Error
}

func deleteById(db *gorm.DB, tenantId uuid.UUID, id uint32) error {
	return db.Where(&Entity{TenantId: tenantId, Id: id}).Delete(&Entity{}).Error
}
