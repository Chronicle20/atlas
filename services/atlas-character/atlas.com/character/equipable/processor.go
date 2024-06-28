package equipable

import (
	"atlas-character/database"
	"atlas-character/tenant"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func byInventoryProvider(_ logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(inventoryId uint32) model.SliceProvider[Model] {
	return func(inventoryId uint32) model.SliceProvider[Model] {
		return database.ModelSliceProvider[Model, entity](db)(getByInventory(tenant.Id(), inventoryId), makeModel)
	}
}

func GetByInventory(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(inventoryId uint32) ([]Model, error) {
	return func(inventoryId uint32) ([]Model, error) {
		return byInventoryProvider(l, db, tenant)(inventoryId)()
	}
}

func GetEquipment(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(inventoryId uint32) ([]Model, error) {
	return func(inventoryId uint32) ([]Model, error) {
		return model.FilteredProvider[Model](byInventoryProvider(l, db, tenant)(inventoryId), FilterOutInventory)()
	}
}

func GetInInventory(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(inventoryId uint32) ([]Model, error) {
	return func(inventoryId uint32) ([]Model, error) {
		return model.FilteredProvider[Model](byInventoryProvider(l, db, tenant)(inventoryId), FilterOutEquipment)()
	}
}

func FilterOutInventory(e Model) bool {
	return e.Slot() < 0
}

func FilterOutEquipment(e Model) bool {
	return e.Slot() > 0
}
