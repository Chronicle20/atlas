package compartment

import (
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// createEntity creates a new compartment entity
func createEntity(db *gorm.DB, t tenant.Model, accountId uint32, type_ CompartmentType, capacity uint32) (Model, error) {
	// Create compartment entity
	entity := Entity{
		TenantId:  t.Id(),
		AccountId: accountId,
		Type:      byte(type_),
		Capacity:  capacity,
	}

	if err := db.Create(&entity).Error; err != nil {
		return Model{}, err
	}

	// Convert entity to model
	return Make(entity)
}

// deleteEntity deletes a compartment entity by ID
func deleteEntity(db *gorm.DB, tenantId uuid.UUID, id uuid.UUID) error {
	return db.Where("tenant_id = ? AND id = ?", tenantId, id).Delete(&Entity{}).Error
}

// deleteAllByAccountId deletes all compartments for an account
func deleteAllByAccountId(db *gorm.DB, tenantId uuid.UUID, accountId uint32) error {
	return db.Where("tenant_id = ? AND account_id = ?", tenantId, accountId).Delete(&Entity{}).Error
}

// updateCapacity updates the capacity of a compartment
func updateCapacity(db *gorm.DB, tenantId uuid.UUID, id uuid.UUID, capacity uint32) (Model, error) {
	var entity Entity

	// Find the entity
	if err := db.Where("tenant_id = ? AND id = ?", tenantId, id).First(&entity).Error; err != nil {
		return Model{}, err
	}

	// Update the capacity
	entity.Capacity = capacity

	// Save the entity
	if err := db.Save(&entity).Error; err != nil {
		return Model{}, err
	}

	// Convert entity to model
	return Make(entity)
}
