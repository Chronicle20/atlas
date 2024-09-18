package equipable

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func createItem(db *gorm.DB, tenantId uuid.UUID, inventoryId uint32, itemId uint32, slot int16, referenceId uint32) (Model, error) {
	eii := &entity{
		TenantId:    tenantId,
		InventoryId: inventoryId,
		ItemId:      itemId,
		Slot:        slot,
		ReferenceId: referenceId,
	}
	err := db.Create(eii).Error
	if err != nil {
		return Model{}, err
	}
	return makeModel(*eii)
}

func makeModel(e entity) (Model, error) {
	return Model{
		id:          e.ID,
		itemId:      e.ItemId,
		slot:        e.Slot,
		referenceId: e.ReferenceId,
	}, nil
}

func updateSlot(db *gorm.DB, tenantId uuid.UUID, id uint32, slot int16) error {
	return db.Model(&entity{TenantId: tenantId, ID: id}).Select("Slot").Updates(&entity{Slot: slot}).Error
}

func delete(db *gorm.DB, tenantId uuid.UUID, referenceId uint32) error {
	return db.Where(&entity{TenantId: tenantId, ReferenceId: referenceId}).Delete(&entity{}).Error
}
