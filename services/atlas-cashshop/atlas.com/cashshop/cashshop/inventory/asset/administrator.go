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
