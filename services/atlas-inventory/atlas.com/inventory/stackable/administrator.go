package stackable

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func updateQuantity(db *gorm.DB, tenantId uuid.UUID, id uint32, quantity uint32) error {
	return db.Model(&Entity{TenantId: tenantId, Id: id}).Select("Quantity").Updates(&Entity{Quantity: quantity}).Error
}

func deleteById(db *gorm.DB, tenantId uuid.UUID, id uint32) error {
	return db.Where(&Entity{TenantId: tenantId, Id: id}).Delete(&Entity{}).Error
}
