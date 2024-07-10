package inventory

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func create(db *gorm.DB, tenantId uuid.UUID, characterId uint32, inventoryType int8, capacity uint32) (Model, error) {
	e := &entity{
		TenantId:      tenantId,
		CharacterId:   characterId,
		InventoryType: inventoryType,
		Capacity:      capacity,
	}

	err := db.Create(e).Error
	if err != nil {
		return Model{}, err
	}
	return makeInventory(*e)
}

func delete(db *gorm.DB, tenantId uuid.UUID, id uint32) error {
	return db.Where(&entity{TenantId: tenantId, ID: id}).Delete(&entity{}).Error
}
