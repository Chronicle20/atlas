package asset

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Create creates a new asset entity in the database
func create(db *gorm.DB) func(tenantId uuid.UUID) func(compartmentId uuid.UUID) func(itemId uint32) (Entity, error) {
	return func(tenantId uuid.UUID) func(compartmentId uuid.UUID) func(itemId uint32) (Entity, error) {
		return func(compartmentId uuid.UUID) func(itemId uint32) (Entity, error) {
			return func(itemId uint32) (Entity, error) {
				entity := Entity{
					TenantId:      tenantId,
					CompartmentId: compartmentId,
					ItemId:        itemId,
				}

				if err := db.Create(&entity).Error; err != nil {
					return Entity{}, err
				}

				return entity, nil
			}
		}
	}
}

// deleteByItemId deletes an asset entity by tenant ID, compartment ID, and item ID
func deleteByItemId(db *gorm.DB, tenantId uuid.UUID, itemId uint32) error {
	return db.Where("tenant_id = ? AND item_id = ?", tenantId, itemId).Delete(&Entity{}).Error
}
