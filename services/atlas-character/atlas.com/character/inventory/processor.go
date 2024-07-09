package inventory

import (
	"atlas-character/database"
	"atlas-character/equipable"
	"atlas-character/equipment/slot/information"
	"atlas-character/inventory/item"
	"atlas-character/slottable"
	"atlas-character/tenant"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"math"
)

type ItemProvider[M any] func(inventoryId uint32) model.SliceProvider[M]

func byCharacterIdProvider(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(characterId uint32) model.Provider[Model] {
	return func(characterId uint32) model.Provider[Model] {
		return model.Fold[entity, Model](getByCharacter(tenant.Id, characterId)(db), supplier, foldInventory(l, db, span, tenant))
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

func EquipableFolder(m EquipableModel, em equipable.Model) (EquipableModel, error) {
	if em.Slot() <= 0 {
		return m, nil
	}
	m.items = append(m.items, em)
	return m, nil
}

func foldProperty[M any, N any](setter func(sm N) M) model.Transformer[N, M] {
	return func(n N) (M, error) {
		return setter(n), nil
	}
}

func ItemFolder(m ItemModel, em item.Model) (ItemModel, error) {
	m.items = append(m.items, em)
	return m, nil
}

func foldInventory(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(ref Model, ent entity) (Model, error) {
	return func(ref Model, ent entity) (Model, error) {
		switch Type(ent.InventoryType) {
		case TypeValueEquip:
			ep := equipable.InInventoryProvider(l, db, span, tenant)(ent.ID)
			return model.Map(model.Fold(ep, NewEquipableModel(ent.ID, ent.Capacity), EquipableFolder), foldProperty(ref.SetEquipable))()
		case TypeValueUse:
			ip := item.ByInventoryProvider(l, db, tenant)(ent.ID)
			return model.Map(model.Fold(ip, NewItemModel(ent.ID, ent.Capacity), ItemFolder), foldProperty(ref.SetUseable))()
		case TypeValueSetup:
			ip := item.ByInventoryProvider(l, db, tenant)(ent.ID)
			return model.Map(model.Fold(ip, NewItemModel(ent.ID, ent.Capacity), ItemFolder), foldProperty(ref.SetSetup))()
		case TypeValueETC:
			ip := item.ByInventoryProvider(l, db, tenant)(ent.ID)
			return model.Map(model.Fold(ip, NewItemModel(ent.ID, ent.Capacity), ItemFolder), foldProperty(ref.SetEtc))()
		case TypeValueCash:
			ip := item.ByInventoryProvider(l, db, tenant)(ent.ID)
			return model.Map(model.Fold(ip, NewItemModel(ent.ID, ent.Capacity), ItemFolder), foldProperty(ref.SetCash))()
		}
		return ref, errors.New("unknown inventory type")
	}
}

func GetInventories(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(characterId uint32) (Model, error) {
	return func(characterId uint32) (Model, error) {
		return byCharacterIdProvider(l, db, span, tenant)(characterId)()
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
		inv, err := database.ModelProvider[Model, entity](db)(get(tenant.Id, characterId, inventoryType), makeInventory)()
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

func Create(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(characterId uint32, defaultCapacity uint32) (Model, error) {
	return func(characterId uint32, defaultCapacity uint32) (Model, error) {
		err := db.Transaction(func(tx *gorm.DB) error {
			for _, t := range Types {
				_, err := create(db, tenant.Id, characterId, int8(t), defaultCapacity)
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

		return GetInventories(l, db, span, tenant)(characterId)
	}
}

type adjustmentMode byte

const (
	adjustmentModeCreate adjustmentMode = 0
	adjustmentModeUpdate adjustmentMode = 1
)

type adjustment struct {
	mode            adjustmentMode
	itemId          uint32
	inventoryType   Type
	changedQuantity uint32
	quantity        uint32
	slot            int16
	oldSlot         int16
}

func (i adjustment) Mode() adjustmentMode {
	return i.mode
}

func (i adjustment) ItemId() uint32 {
	return i.itemId
}

func (i adjustment) InventoryType() Type {
	return i.inventoryType
}

func (i adjustment) Quantity() uint32 {
	return i.quantity
}

func (i adjustment) ChangedQuantity() uint32 {
	return i.changedQuantity
}

func (i adjustment) Slot() int16 {
	return i.slot
}

func (i adjustment) OldSlot() int16 {
	return i.oldSlot
}

func CreateItem(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(characterId uint32, inventoryType Type, itemId uint32, quantity uint32) error {
	return func(characterId uint32, inventoryType Type, itemId uint32, quantity uint32) error {
		expectedInventoryType := math.Floor(float64(itemId) / 1000000)
		if expectedInventoryType != float64(inventoryType) {
			l.Errorf("Provided inventoryType [%d] does not match expected one [%d] for itemId [%d].", inventoryType, uint32(expectedInventoryType), itemId)
			return errors.New("invalid inventory type")
		}

		var events = make([]adjustment, 0)
		err := db.Transaction(func(tx *gorm.DB) error {
			m, err := GetInventories(l, tx, span, tenant)(characterId)
			if err != nil {
				l.WithError(err).Errorf("Unable to locate inventories for character [%d].", characterId)
				return err
			}
			inv, err := m.GetHolderByType(inventoryType)
			if err != nil {
				l.WithError(err).Errorf("Unable to locate inventory [%d] for character [%d].", inventoryType, characterId)
				return err
			}

			if inventoryType == TypeValueEquip {
				events, err = createEquipable(l, tx, span, tenant)(characterId, inv.Id(), inventoryType, itemId)
				if err != nil {
					l.WithError(err).Errorf("Unable to create [%d] equipable [%d] for character [%d].", quantity, itemId, characterId)
					return err
				}
			} else {
				events, err = createItem(l, tx, span, tenant)(characterId, inv.Id(), inventoryType, itemId, quantity)
				if err != nil {
					l.WithError(err).Errorf("Unable to create [%d] items [%d] for character [%d].", quantity, itemId, characterId)
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
		for _, event := range events {
			emitItemGainEvent(l, span, tenant)(characterId, event.ItemId(), event.ChangedQuantity())
			//emitInventoryModificationEvent(l, span)(characterId, true, e.Mode(), e.ItemId(), e.InventoryType(), e.Quantity(), e.Slot(), e.OldSlot())
		}
		return err
	}
}

func createEquipable(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(characterId uint32, inventoryId uint32, inventoryType Type, itemId uint32) ([]adjustment, error) {
	var creator itemCreator = equipable.CreateItem(l, db, span, tenant)
	return func(characterId uint32, inventoryId uint32, inventoryType Type, itemId uint32) ([]adjustment, error) {
		event, err := createNewItem(l)(creator, characterId, inventoryId, inventoryType, itemId, 1)
		if err != nil {
			return make([]adjustment, 0), nil
		}
		return []adjustment{event}, nil
	}
}

func createItem(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(characterId uint32, inventoryId uint32, inventoryType Type, itemId uint32, quantity uint32) ([]adjustment, error) {
	var creator itemCreator = item.CreateItem(l, db, tenant)
	return func(characterId uint32, inventoryId uint32, inventoryType Type, itemId uint32, quantity uint32) ([]adjustment, error) {
		runningQuantity := quantity
		slotMax := item.MaxInSlot()
		var events = make([]adjustment, 0)

		existingItems, err := item.GetByItemId(l, db, tenant)(inventoryId, itemId)
		if err != nil {
			l.WithError(err).Errorf("Unable to locate items [%d] in inventory [%d] for character [%d].", itemId, inventoryType, characterId)
			return events, err
		}
		if len(existingItems) > 0 {
			index := 0
			for runningQuantity > 0 {
				if index < len(existingItems) {
					i := existingItems[index]
					oldQuantity := i.Quantity()

					if oldQuantity < slotMax {
						newQuantity := uint32(math.Min(float64(oldQuantity+runningQuantity), float64(slotMax)))
						changedQuantity := newQuantity - oldQuantity
						runningQuantity = runningQuantity - changedQuantity
						err := item.UpdateQuantity(l, db, tenant)(i.Id(), newQuantity)
						if err != nil {
							l.WithError(err).Errorf("Updating the quantity of item [%d] to value [%d].", i.Id(), newQuantity)
						} else {
							events = append(events, adjustment{mode: adjustmentModeUpdate, itemId: itemId, inventoryType: inventoryType, changedQuantity: changedQuantity, quantity: newQuantity, slot: i.Slot(), oldSlot: 0})
						}
					}
					index++
				} else {
					break
				}
			}
		}
		for runningQuantity > 0 {
			newQuantity := uint32(math.Min(float64(runningQuantity), float64(slotMax)))
			runningQuantity = runningQuantity - newQuantity
			nes, err := createNewItem(l)(creator, characterId, inventoryId, inventoryType, itemId, newQuantity)
			if err != nil {
				return events, err
			}
			events = append(events, nes)
		}
		return events, nil
	}
}

func createNewItem(l logrus.FieldLogger) func(creator itemCreator, characterId uint32, inventoryId uint32, inventoryType Type, itemId uint32, quantity uint32) (adjustment, error) {
	return func(creator itemCreator, characterId uint32, inventoryId uint32, inventoryType Type, itemId uint32, quantity uint32) (adjustment, error) {
		i, err := creator(characterId, inventoryId, int8(inventoryType), itemId, quantity)()
		if err != nil {
			l.WithError(err).Errorf("Unable to create item [%d] for character [%d].", itemId, characterId)
			return adjustment{}, err
		}
		return adjustment{mode: adjustmentModeCreate, itemId: itemId, inventoryType: inventoryType, changedQuantity: quantity, quantity: quantity, slot: i.Slot(), oldSlot: 0}, nil
	}
}

type itemCreator func(characterId uint32, inventoryId uint32, inventoryType int8, itemId uint32, quantity uint32) model.Provider[slottable.Slottable]

func EquipItemForCharacter(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(characterId uint32, source int16, destination int16) {
	return func(characterId uint32, source int16, destination int16) {
		l.Debugf("Received request to equip item at [%d] for character [%d]. Ideally placing in [%d]", source, characterId, destination)
		e, err := equipable.GetBySlot(l, db, tenant)(characterId, source)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve equipment in slot [%d].", source)
			return
		}

		l.Debugf("Equipment [%d] is item [%d] for character [%d].", e.Id(), e.ItemId(), characterId)

		slots, err := information.GetById(l, span, tenant)(e.ItemId())
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve destination slots for item [%d].", e.ItemId())
			return
		} else if len(slots) <= 0 {
			l.Errorf("Unable to retrieve destination slots for item [%d].", e.ItemId())
			return
		}
		slot := slots[0]
		l.Debugf("Equipment [%d] to be equipped in slot [%d] for character [%d].", e.Id(), slot.Slot(), characterId)

		temporarySlot := int16(math.MinInt16)

		existingSlot := e.Slot()
		err = db.Transaction(func(tx *gorm.DB) error {
			if equip, err := equipable.GetBySlot(l, tx, tenant)(characterId, slot.Slot()); err == nil && equip.Id() != 0 {
				l.Debugf("Equipment [%d] already exists in slot [%d], that item will be moved temporarily to [%d] for character [%d].", equip.Id(), slot.Slot(), temporarySlot, characterId)
				_ = equipable.UpdateSlot(l, tx, tenant)(equip.Id(), temporarySlot)
			}

			err = equipable.UpdateSlot(l, tx, tenant)(e.Id(), slot.Slot())
			if err != nil {
				return err
			}
			l.Debugf("Moved item [%d] from slot [%d] to [%d] for character [%d].", e.ItemId(), existingSlot, slot.Slot(), characterId)

			if equip, err := equipable.GetBySlot(l, tx, tenant)(characterId, temporarySlot); err == nil && equip.Id() != 0 {
				err := equipable.UpdateSlot(l, tx, tenant)(equip.Id(), existingSlot)
				if err != nil {
					return err
				}
				l.Debugf("Moved item from temporary location [%d] to slot [%d] for character [%d].", temporarySlot, existingSlot, characterId)
			}
			return nil
		})
		if err != nil {
			l.WithError(err).Errorf("Unable to complete the equipment of item [%d] for character [%d].", e.Id(), characterId)
			return
		}

		//emitItemEquipped(l, span)(characterId)
		//emitInventoryModificationEvent(l, span)(characterId, true, 2, e.ItemId(), TypeValueEquip, 1, slot, existingSlot)
	}
}

func UnequipItemForCharacter(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(characterId uint32, oldSlot int16) {
	return func(characterId uint32, oldSlot int16) {
		l.Debugf("Received request to unequip item at [%d] for character [%d].", oldSlot, characterId)

		txErr := db.Transaction(func(tx *gorm.DB) error {
			e, err := equipable.GetBySlot(l, tx, tenant)(characterId, oldSlot)
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve equipment in slot [%d].", oldSlot)
				return err
			}

			ci, err := GetInventories(l, tx, span, tenant)(characterId)
			if err != nil {
				l.WithError(err).Errorf("Unable to locate inventories for character [%d].", characterId)
				return err
			}
			inv, err := ci.GetHolderByType(TypeValueEquip)
			if err != nil {
				l.WithError(err).Errorf("Unable to locate inventory [%d] for character [%d].", TypeValueEquip, characterId)
				return err
			}
			newSlot, err := equipable.GetNextFreeSlot(l, db, span, tenant)(inv.Id())()
			if err != nil {
				l.WithError(err).Errorf("Unable to get next free equipment slot")
				return err
			}

			err = equipable.UpdateSlot(l, tx, tenant)(e.Id(), newSlot)
			if err != nil {
				return err
			}

			l.Debugf("Unequipped %d for character %d and place it in slot [%d], from [%d].", e.Id(), characterId, newSlot, oldSlot)
			return nil
		})
		if txErr != nil {
			l.WithError(txErr).Errorf("Unable to complete unequiping item at [%d] for character [%d].", oldSlot, characterId)
			return
		}
		//emitItemUnequipped(l, span)(characterId)
		//emitInventoryModificationEvent(l, span)(characterId, true, 2, itemId, TypeValueEquip, 1, newSlot, oldSlot)
	}
}
