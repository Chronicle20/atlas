package compartment

import (
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func create(db *gorm.DB, tenantId uuid.UUID, characterId uint32, inventoryType inventory.Type, capacity uint32) (Model, error) {
	e := &Entity{
		TenantId:      tenantId,
		CharacterId:   characterId,
		InventoryType: inventoryType,
		Capacity:      capacity,
	}

	err := db.Create(e).Error
	if err != nil {
		return Model{}, err
	}
	return Make(*e)
}

func updateCapacity(db *gorm.DB, tenantId uuid.UUID, characterId uint32, inventoryType int8, capacity uint32) (Model, error) {
	var e Entity

	err := db.
		Where("tenant_id = ? AND character_id = ? AND inventory_type = ?", tenantId, characterId, inventoryType).
		First(&e).Error
	if err != nil {
		return Model{}, err
	}

	e.Capacity = capacity

	err = db.Save(&e).Error
	if err != nil {
		return Model{}, err
	}

	return Make(e)
}

func deleteById(db *gorm.DB, tenantId uuid.UUID, id uuid.UUID) error {
	return db.Where(&Entity{TenantId: tenantId, Id: id}).Delete(&Entity{}).Error
}
