package item

import (
	"atlas-character/database"
	"atlas-character/slottable"
	"atlas-character/tenant"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func ByInventoryProvider(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(inventoryId uint32) model.Provider[[]Model] {
	return func(inventoryId uint32) model.Provider[[]Model] {
		return database.ModelSliceProvider[Model, entity](db)(getByInventory(tenant.Id, inventoryId), makeModel)
	}
}

func GetByInventory(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(inventoryId uint32) ([]Model, error) {
	return func(inventoryId uint32) ([]Model, error) {
		return ByInventoryProvider(l, db, tenant)(inventoryId)()
	}
}

func GetBySlot(_ logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(inventoryId uint32, slot int16) (Model, error) {
	return func(inventoryId uint32, slot int16) (Model, error) {
		return database.ModelProvider[Model, entity](db)(getBySlot(tenant.Id, inventoryId, slot), makeModel)()
	}
}

func GetById(_ logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(id uint32) (Model, error) {
	return func(id uint32) (Model, error) {
		return database.ModelProvider[Model, entity](db)(getById(tenant.Id, id), makeModel)()
	}
}

func GetByItemId(_ logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(inventoryId uint32, itemId uint32) ([]Model, error) {
	return func(inventoryId uint32, itemId uint32) ([]Model, error) {
		return database.ModelSliceProvider[Model, entity](db)(getForCharacter(tenant.Id, inventoryId, itemId), makeModel)()
	}
}

func UpdateSlot(_ logrus.FieldLogger, db *gorm.DB) func(id uint32, slot int16) error {
	return func(id uint32, slot int16) error {
		return updateSlot(db, id, slot)
	}
}

func UpdateQuantity(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(id uint32, quantity uint32) error {
	return func(id uint32, quantity uint32) error {
		i, err := GetById(l, db, tenant)(id)
		if err != nil {
			return err
		}
		return updateQuantity(db, i.Id(), quantity)
	}
}

func MaxInSlot() uint32 {
	//TODO make this more sophisticated
	return 200
}

func CreateItem(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(characterId uint32, inventoryId uint32, inventoryType int8, itemId uint32, quantity uint32) model.Provider[slottable.Slottable] {
	return func(characterId uint32, inventoryId uint32, inventoryType int8, itemId uint32, quantity uint32) model.Provider[slottable.Slottable] {
		ms, err := GetByInventory(l, db, tenant)(inventoryId)
		if err != nil {
			return model.ErrorProvider[slottable.Slottable](err)
		}
		slot, err := slottable.GetNextFreeSlot(model.SliceMap(model.FixedProvider(ms), slottableTransformer))
		if err != nil {
			return model.ErrorProvider[slottable.Slottable](err)
		}
		i, err := createItem(db, tenant, inventoryId, itemId, quantity, slot)
		if err != nil {
			return model.ErrorProvider[slottable.Slottable](err)
		}
		return model.FixedProvider[slottable.Slottable](i)
	}
}

func slottableTransformer(m Model) (slottable.Slottable, error) {
	return m, nil
}

func RemoveItem(_ logrus.FieldLogger, db *gorm.DB) func(characterId uint32, id uint32) error {
	return func(characterId uint32, id uint32) error {
		return remove(db, characterId, id)
	}
}

func DeleteBySlot(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(inventoryId uint32, slot int16) error {
	return func(inventoryId uint32, slot int16) error {
		return deleteBySlot(db, tenant.Id, inventoryId, slot)
	}
}
