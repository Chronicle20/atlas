package pet

import (
	"github.com/Chronicle20/atlas-tenant"
	"gorm.io/gorm"
)

func create(db *gorm.DB) func(t tenant.Model, ownerId uint32, m Model) (Model, error) {
	return func(t tenant.Model, ownerId uint32, m Model) (Model, error) {
		e := &Entity{
			TenantId:        t.Id(),
			OwnerId:         ownerId,
			InventoryItemId: m.InventoryItemId(),
			TemplateId:      m.TemplateId(),
			Name:            m.Name(),
			Level:           m.Level(),
			Tameness:        m.Tameness(),
			Fullness:        m.Fullness(),
			Expiration:      m.Expiration(),
		}

		err := db.Create(e).Error
		if err != nil {
			return Model{}, err
		}
		return modelFromEntity(*e)
	}
}

func deleteByInventoryItemId(t tenant.Model, inventoryItemId uint32) func(db *gorm.DB) error {
	return func(db *gorm.DB) error {
		return db.Where(&Entity{TenantId: t.Id(), InventoryItemId: inventoryItemId}).Delete(&Entity{}).Error
	}
}

func deleteForCharacter(t tenant.Model, ownerId uint32) func(db *gorm.DB) error {
	return func(db *gorm.DB) error {
		return db.Where(&Entity{TenantId: t.Id(), OwnerId: ownerId}).Delete(&Entity{}).Error
	}
}

func modelFromEntity(e Entity) (Model, error) {
	return NewModelBuilder(e.Id, e.InventoryItemId, e.TemplateId, e.Name, e.OwnerId).
		SetLevel(e.Level).
		SetTameness(e.Tameness).
		SetFullness(e.Fullness).
		SetExpiration(e.Expiration).
		SetSlot(e.Slot).
		Build(), nil
}
