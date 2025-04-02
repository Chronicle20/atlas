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

func updateEntity(db *gorm.DB, tenantId uuid.UUID, characterId uint32, inventoryType int8, capacity uint32) (Model, error) {
	var e entity

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

	return makeInventory(e)
}

func deleteByType(db *gorm.DB, tenantId uuid.UUID, characterId uint32, inventoryType int8) error {
	return db.Where(&entity{TenantId: tenantId, CharacterId: characterId, InventoryType: inventoryType}).Delete(&entity{}).Error
}

func deleteById(db *gorm.DB, tenantId uuid.UUID, id uint32) error {
	return db.Where(&entity{TenantId: tenantId, ID: id}).Delete(&entity{}).Error
}
