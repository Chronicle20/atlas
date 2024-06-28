package inventory

import "gorm.io/gorm"

func create(db *gorm.DB, characterId uint32, inventoryType int8, capacity uint32) (Model, error) {
	e := &entity{
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
