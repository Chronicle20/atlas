package stackable

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func deleteById(db *gorm.DB, tenantId uuid.UUID, id uint32) error {
	return db.Where(&Entity{TenantId: tenantId, Id: id}).Delete(&Entity{}).Error
}
