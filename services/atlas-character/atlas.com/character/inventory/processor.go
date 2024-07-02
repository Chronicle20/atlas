package inventory

import (
	"atlas-character/database"
	"atlas-character/equipable"
	"atlas-character/inventory/item"
	"atlas-character/tenant"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func byCharacterIdProvider(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(characterId uint32) model.Provider[Model] {
	return func(characterId uint32) model.Provider[Model] {
		return model.Fold[entity, Model](getByCharacter(tenant.Id(), characterId)(db), supplier, foldInventory(l, db, tenant))
	}
}

func supplier() (Model, error) {
	return Model{
		equipable: EquipableModel{},
		useable:   ItemModel{},
		setup:     ItemModel{},
		etc:       ItemModel{},
		cash:      ItemModel{},
	}, nil
}

func foldInventory(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(ref Model, ent entity) (Model, error) {
	return func(ref Model, ent entity) (Model, error) {
		switch Type(ent.InventoryType) {
		case TypeValueEquip:
			equipables, err := equipable.GetInInventory(l, db, tenant)(ent.ID)
			if err != nil {
				return ref, err
			}
			ref.equipable = ref.Equipable().SetItems(equipables).SetId(ent.ID).SetCapacity(ent.Capacity)
			return ref, nil
		case TypeValueUse:
			items, err := item.GetByInventory(l, db, tenant)(ent.ID)
			if err != nil {
				return ref, err
			}
			ref.useable = ref.Useable().SetItems(items).SetId(ent.ID).SetCapacity(ent.Capacity)
			return ref, nil
		case TypeValueSetup:
			items, err := item.GetByInventory(l, db, tenant)(ent.ID)
			if err != nil {
				return ref, err
			}
			ref.setup = ref.Setup().SetItems(items).SetId(ent.ID).SetCapacity(ent.Capacity)
			return ref, nil
		case TypeValueETC:
			items, err := item.GetByInventory(l, db, tenant)(ent.ID)
			if err != nil {
				return ref, err
			}
			ref.etc = ref.ETC().SetItems(items).SetId(ent.ID).SetCapacity(ent.Capacity)
			return ref, nil
		case TypeValueCash:
			items, err := item.GetByInventory(l, db, tenant)(ent.ID)
			if err != nil {
				return ref, err
			}
			ref.cash = ref.Cash().SetItems(items).SetId(ent.ID).SetCapacity(ent.Capacity)
			return ref, nil
		}
		return ref, errors.New("unknown inventory type")
	}
}

func GetInventories(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(characterId uint32) (Model, error) {
	return func(characterId uint32) (Model, error) {
		return byCharacterIdProvider(l, db, tenant)(characterId)()
	}
}

func GetInventory(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(characterId uint32, inventoryType string, filters ...ItemFilter) (Model, error) {
	return func(characterId uint32, inventoryType string, filters ...ItemFilter) (Model, error) {
		//if it, ok := GetTypeFromName(inventoryType); ok {
		//	return GetInventoryByTypeVal(l, db, tenant)(characterId, it, filters...)
		//}
		//return nil, errors.New("invalid inventory type")
		return Model{}, nil
	}
}

func GetInventoryByTypeVal(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(characterId uint32, inventoryType Type, filters ...ItemFilter) (Model, error) {
	return func(characterId uint32, inventoryType Type, filters ...ItemFilter) (Model, error) {
		inv, err := database.ModelProvider[Model, entity](db)(get(tenant.Id(), characterId, inventoryType), makeInventory)()
		if err != nil {
			return Model{}, err
		}

		//items, err := item.GetByInventory(l, db, tenant)(inv.Id())
		//if err != nil {
		//	return Model{}, err
		//}
		//for _, i := range items {
		//	ok := true
		//	for _, filter := range filters {
		//		if !filter(i) {
		//			ok = false
		//			break
		//		}
		//	}
		//	if ok {
		//		//inv = inv.AddItem(i)
		//	}
		//}

		return inv, nil
	}
}

type ItemFilter func(i item.Model) bool

func FilterSlot(slot int16) ItemFilter {
	return func(i item.Model) bool {
		return i.Slot() == slot
	}
}

func FilterItemId(l logrus.FieldLogger, db *gorm.DB, _ opentracing.Span, tenant tenant.Model) func(itemId uint32) ItemFilter {
	return func(itemId uint32) ItemFilter {
		return func(i item.Model) bool {
			ii, err := item.GetById(l, db, tenant)(i.Id())
			if err != nil {
				return false
			}
			return ii.ItemId() == itemId
		}
	}
}

func Create(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(characterId uint32, defaultCapacity uint32) (Model, error) {
	return func(characterId uint32, defaultCapacity uint32) (Model, error) {
		err := db.Transaction(func(tx *gorm.DB) error {
			for _, t := range Types {
				_, err := create(db, tenant.Id(), characterId, int8(t), defaultCapacity)
				if err != nil {
					l.WithError(err).Errorf("Unable to create inventory [%d] for character [%d].", t, characterId)
					return err
				}
			}
			return nil
		})
		if err != nil {
			l.WithError(err).Errorf("Unable to create inventory for character [%d]", characterId)
			return Model{}, err
		}

		return GetInventories(l, db, tenant)(characterId)
	}
}

func CreateItem(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(characterId uint32, inventoryType Type, itemId uint32, quantity uint32) error {
	return func(characterId uint32, inventoryType Type, itemId uint32, quantity uint32) error {
		m, err := GetInventories(l, db, tenant)(characterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to locate inventories for character [%d].", characterId)
			return err
		}
		_, err = m.GetHolderByType(inventoryType)
		if err != nil {
			l.WithError(err).Errorf("Unable to locate inventory [%d] for character [%d].", inventoryType, characterId)
			return err
		}

		return nil
	}
}
