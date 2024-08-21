package item

import (
	"atlas-character/tenant"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func createItem(db *gorm.DB, t tenant.Model, inventoryId uint32, itemId uint32, quantity uint32, slot int16) (Model, error) {
	var im Model
	txError := db.Transaction(func(tx *gorm.DB) error {
		eii := &entity{
			TenantId:    t.Id,
			InventoryId: inventoryId,
			ItemId:      itemId,
			Quantity:    quantity,
			Slot:        slot,
		}
		err := db.Create(eii).Error
		if err != nil {
			return err
		}
		im, err = makeModel(*eii)
		if err != nil {
			return err
		}
		return nil
	})
	return im, txError
}

func deleteById(db *gorm.DB, tenantId uuid.UUID, id uint32) error {
	return db.Where(&entity{TenantId: tenantId, ID: id}).Delete(&entity{}).Error
}

func makeModel(e entity) (Model, error) {
	return Model{
		id:       e.ID,
		itemId:   e.ItemId,
		slot:     e.Slot,
		quantity: e.Quantity,
	}, nil
}

func remove(db *gorm.DB, inventoryId uint32, id uint32) error {
	return db.Delete(&entity{InventoryId: inventoryId, ID: id}).Error
}

func updateQuantity(db *gorm.DB, id uint32, amount uint32) error {
	return db.Model(&entity{ID: id}).Select("Quantity").Updates(&entity{Quantity: amount}).Error
}

func updateSlot(db *gorm.DB, id uint32, slot int16) error {
	return db.Model(&entity{ID: id}).Select("Slot").Updates(&entity{Slot: slot}).Error
}
