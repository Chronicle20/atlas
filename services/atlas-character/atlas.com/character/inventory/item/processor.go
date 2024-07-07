package item

import (
	"atlas-character/database"
	"atlas-character/slottable"
	"atlas-character/tenant"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var characterCreationItems = []uint32{
	1302000, 1312004, 1322005, 1442079, // weapons
	1040002, 1040006, 1040010, 1041002, 1041006, 1041010, 1041011, 1042167, // top
	1060002, 1060006, 1061002, 1061008, 1062115, // bottom
	1072001, 1072005, 1072037, 1072038, 1072383, // shoes
	30000, 30010, 30020, 30030, 31000, 31040, 31050, // hair
	20000, 20001, 20002, 21000, 21001, 21002, 21201, 20401, 20402, 21700, 20100, //face
}

func invalidCharacterCreationItem(itemId uint32) bool {
	for _, v := range characterCreationItems {
		if itemId == v {
			return false
		}
	}
	return true
}

//func CreateEquipment(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(characterId uint32, inventoryId uint32, itemId uint32, equipmentId uint32, characterCreation bool) (EquipmentModel, error) {
//	return func(characterId uint32, inventoryId uint32, itemId uint32, equipmentId uint32, characterCreation bool) (EquipmentModel, error) {
//		if characterCreation {
//			if invalidCharacterCreationItem(itemId) {
//				l.Errorf("Received a request to create an item %d for character %d which is not valid for character creation. This is usually a hack.")
//				return EquipmentModel{}, errors.New("not valid item for character creation")
//			}
//		}
//
//		nextOpen, err := GetNextFreeSlot(l, db, tenant)(inventoryId)
//		if err != nil {
//			nextOpen = 1
//		}
//
//		eq, err := createEquipment(db, inventoryId, itemId, nextOpen, equipmentId)
//		if err != nil {
//			l.Errorf("Persisting equipment %d association for character %d in Slot %d.", equipmentId, characterId, nextOpen)
//			return EquipmentModel{}, err
//		}
//		return eq, nil
//	}
//}
//
//func CreateItem(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(characterId uint32, inventoryId uint32, inventoryType int8, itemId uint32, quantity uint32) (Model, error) {
//	return func(characterId uint32, inventoryId uint32, inventoryType int8, itemId uint32, quantity uint32) (Model, error) {
//		slot, err := GetNextFreeSlot(l, db, tenant)(inventoryId)
//		if err != nil {
//			return Model{}, err
//		}
//		return createItem(db, inventoryId, itemId, quantity, slot)
//	}
//}

//func GetEquipment(_ logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(inventoryId uint32) ([]EquipmentModel, error) {
//	return func(inventoryId uint32) ([]EquipmentModel, error) {
//		return database.ModelSliceProvider[EquipmentModel, entityInventoryItem](db)(getByInventory(tenant.Id(), inventoryId), makeEquipment)()
//	}
//}

func ByInventoryProvider(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(inventoryId uint32) model.SliceProvider[Model] {
	return func(inventoryId uint32) model.SliceProvider[Model] {
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

//func GetEquippedItemBySlot(_ logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(inventoryId uint32, slot int16) (EquipmentModel, error) {
//	return func(inventoryId uint32, slot int16) (EquipmentModel, error) {
//		return database.ModelProvider[EquipmentModel, entityInventoryItem](db)(getBySlot(tenant.Id(), inventoryId, slot), makeEquipment)()
//	}
//}

func GetById(_ logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(id uint32) (Model, error) {
	return func(id uint32) (Model, error) {
		return database.ModelProvider[Model, entity](db)(getById(tenant.Id, id), makeModel)()
	}
}

//func GetByEquipmentId(_ logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(id uint32) (EquipmentModel, error) {
//	return func(id uint32) (EquipmentModel, error) {
//		return database.ModelProvider[EquipmentModel, entityInventoryItem](db)(getByEquipmentId(tenant.Id(), id), makeEquipment)()
//	}
//}

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
		slot, err := slottable.GetNextFreeSlot(model.SliceMap(model.FixedSliceProvider(ms), slottableTransformer))
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
