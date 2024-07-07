package equipable

import (
	"atlas-character/tenant"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func createItem(db *gorm.DB, t tenant.Model, inventoryId uint32, itemId uint32, slot int16, referenceId uint32) (Model, error) {
	var im Model
	txError := db.Transaction(func(tx *gorm.DB) error {
		eii := &entity{
			TenantId:    t.Id,
			InventoryId: inventoryId,
			ItemId:      itemId,
			Slot:        slot,
			ReferenceId: referenceId,
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

func makeModel(e entity) (Model, error) {
	return Model{
		id:     e.ID,
		itemId: e.ItemId,
		slot:   e.Slot,
	}, nil
}

func updateSlot(db *gorm.DB, tenantId uuid.UUID, id uint32, slot int16) error {
	return db.Model(&entity{TenantId: tenantId, ID: id}).Select("Slot").Updates(&entity{Slot: slot}).Error
}
