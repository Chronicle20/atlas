package inventory

import (
	"atlas-character/equipable"
	slot2 "atlas-character/equipment/slot"
	"atlas-character/equipment/slot/information"
	"atlas-character/inventory/item"
	"atlas-character/kafka/producer"
	"atlas-character/slottable"
	"atlas-character/tenant"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"math"
)

func byCharacterIdProvider(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(characterId uint32) model.Provider[Model] {
	return func(characterId uint32) model.Provider[Model] {
		return model.Fold[entity, Model](getByCharacter(tenant.Id, characterId)(db), supplier, foldInventory(l, db, span, tenant))
	}
}

func supplier() (Model, error) {
	return Model{
		equipable: EquipableModel{},
		useable:   ItemModel{mType: TypeValueUse},
		setup:     ItemModel{mType: TypeValueSetup},
		etc:       ItemModel{mType: TypeValueETC},
		cash:      ItemModel{mType: TypeValueCash},
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
			return model.Map(model.Fold(ip, NewItemModel(ent.ID, TypeValueUse, ent.Capacity), ItemFolder), foldProperty(ref.SetUseable))()
		case TypeValueSetup:
			ip := item.ByInventoryProvider(l, db, tenant)(ent.ID)
			return model.Map(model.Fold(ip, NewItemModel(ent.ID, TypeValueSetup, ent.Capacity), ItemFolder), foldProperty(ref.SetSetup))()
		case TypeValueETC:
			ip := item.ByInventoryProvider(l, db, tenant)(ent.ID)
			return model.Map(model.Fold(ip, NewItemModel(ent.ID, TypeValueETC, ent.Capacity), ItemFolder), foldProperty(ref.SetEtc))()
		case TypeValueCash:
			ip := item.ByInventoryProvider(l, db, tenant)(ent.ID)
			return model.Map(model.Fold(ip, NewItemModel(ent.ID, TypeValueCash, ent.Capacity), ItemFolder), foldProperty(ref.SetCash))()
		}
		return ref, errors.New("unknown inventory type")
	}
}

func GetInventories(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(characterId uint32) (Model, error) {
	return func(characterId uint32) (Model, error) {
		return byCharacterIdProvider(l, db, span, tenant)(characterId)()
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

		if quantity == 0 {
			quantity = 1
		}

		l.Debugf("Creating [%d] item [%d] for character [%d] in inventory [%d].", quantity, itemId, characterId, inventoryType)
		invLock := GetLockRegistry().GetById(characterId, inventoryType)
		invLock.Lock()
		defer invLock.Unlock()

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

			l.Debugf("Successfully finished processing create item for character [%d].", characterId)
			return nil
		})
		if err != nil {
			return err
		}
		for _, event := range events {
			_ = producer.ProviderImpl(l)(span)(EnvEventTopicItemGain)(itemGainEventProvider(tenant, characterId, event.ItemId(), event.ChangedQuantity(), event.Slot()))
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
						l.Debugf("Adding [%d] item [%d] in slot [%d] to make a new quantity of [%d] for character [%d].", changedQuantity, i.ItemId(), i.Slot(), newQuantity, characterId)
						err = item.UpdateQuantity(l, db, tenant)(i.Id(), newQuantity)
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
			l.Debugf("Creating [%d] item [%d] in slot [%d] for character [%d].", newQuantity, itemId, nes.slot, characterId)
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
		l.Debugf("Item created. Creating inventory [%d] adjustment of [%d] item [%d] in slot [%d].", inventoryType, quantity, itemId, i.Slot())
		return adjustment{mode: adjustmentModeCreate, itemId: itemId, inventoryType: inventoryType, changedQuantity: quantity, quantity: quantity, slot: i.Slot(), oldSlot: 0}, nil
	}
}

type itemCreator func(characterId uint32, inventoryId uint32, inventoryType int8, itemId uint32, quantity uint32) model.Provider[slottable.Slottable]

func EquipItemForCharacter(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(characterId uint32, source int16, destination int16) {
	return func(characterId uint32, source int16, destination int16) {
		var e equipable.Model
		var err error

		l.Debugf("Received request to equip item at [%d] for character [%d]. Ideally placing in [%d]", source, characterId, destination)
		invLock := GetLockRegistry().GetById(characterId, TypeValueEquip)
		invLock.Lock()
		defer invLock.Unlock()

		err = db.Transaction(func(tx *gorm.DB) error {
			e, err = equipable.GetBySlot(l, tx, tenant)(characterId, source)
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve equipment in slot [%d].", source)
				return err
			}

			l.Debugf("Equipment [%d] is item [%d] for character [%d].", e.Id(), e.ItemId(), characterId)

			slots, err := information.GetById(l, span, tenant)(e.ItemId())
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve destination slots for item [%d].", e.ItemId())
				return err
			} else if len(slots) <= 0 {
				l.Errorf("Unable to retrieve destination slots for item [%d].", e.ItemId())
				return err
			}
			sh := slots[0]
			l.Debugf("Equipment [%d] to be equipped in slot [%d] for character [%d].", e.Id(), sh.Slot(), characterId)

			temporarySlot := int16(math.MinInt16)

			existingSlot := e.Slot()
			if equip, err := equipable.GetBySlot(l, tx, tenant)(characterId, sh.Slot()); err == nil && equip.Id() != 0 {
				l.Debugf("Equipment [%d] already exists in slot [%d], that item will be moved temporarily to [%d] for character [%d].", equip.Id(), sh.Slot(), temporarySlot, characterId)
				_ = equipable.UpdateSlot(l, tx, tenant)(equip.Id(), temporarySlot)
			}

			err = equipable.UpdateSlot(l, tx, tenant)(e.Id(), sh.Slot())
			if err != nil {
				return err
			}
			l.Debugf("Moved item [%d] from slot [%d] to [%d] for character [%d].", e.ItemId(), existingSlot, sh.Slot(), characterId)

			if equip, err := equipable.GetBySlot(l, tx, tenant)(characterId, temporarySlot); err == nil && equip.Id() != 0 {
				err := equipable.UpdateSlot(l, tx, tenant)(equip.Id(), existingSlot)
				if err != nil {
					return err
				}
				l.Debugf("Moved item from temporary location [%d] to slot [%d] for character [%d].", temporarySlot, existingSlot, characterId)
				_ = producer.ProviderImpl(l)(span)(EnvEventTopicEquipChanged)(itemUnequippedProvider(tenant, characterId, equip.ItemId()))
			}

			l.Debugf("Now verifying other inventory operations that may be necessary.")
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

			if e.ItemId()/10000 == 105 {
				l.Debugf("Item is an overall, we also need to unequip the bottom.")
				if equip, err := equipable.GetBySlot(l, tx, tenant)(characterId, int16(slot2.PositionBottom)); err == nil && equip.Id() != 0 {
					newSlot, err := equipable.GetNextFreeSlot(l, tx, span, tenant)(inv.Id())()
					if err != nil {
						l.WithError(err).Errorf("Unable to get next free equipment slot")
						return err
					}

					err = equipable.UpdateSlot(l, tx, tenant)(equip.Id(), newSlot)
					if err != nil {
						return err
					}
					l.Debugf("Moved bottom to slot [%d] for character [%d].", newSlot, characterId)
					_ = producer.ProviderImpl(l)(span)(EnvEventTopicEquipChanged)(itemUnequippedProvider(tenant, characterId, equip.ItemId()))
				} else {
					l.Debugf("No bottom to unequip.")
				}
			}
			if destination == int16(slot2.PositionBottom) {
				l.Debugf("Item is a bottom, need to unequip an overall if its in the top slot.")
				if equip, err := equipable.GetBySlot(l, tx, tenant)(characterId, int16(slot2.PositionBottom)); err == nil && equip.Id() != 0 && equip.ItemId()/10000 == 105 {
					newSlot, err := equipable.GetNextFreeSlot(l, tx, span, tenant)(inv.Id())()
					if err != nil {
						l.WithError(err).Errorf("Unable to get next free equipment slot")
						return err
					}

					err = equipable.UpdateSlot(l, tx, tenant)(equip.Id(), newSlot)
					if err != nil {
						return err
					}
					l.Debugf("Moved overall to slot [%d] for character [%d].", newSlot, characterId)
					_ = producer.ProviderImpl(l)(span)(EnvEventTopicEquipChanged)(itemUnequippedProvider(tenant, characterId, equip.ItemId()))
				}
			}
			return nil
		})
		if err != nil {
			l.WithError(err).Errorf("Unable to complete the equipment of item [%d] for character [%d].", e.Id(), characterId)
			return
		}

		_ = producer.ProviderImpl(l)(span)(EnvEventTopicEquipChanged)(itemEquippedProvider(tenant, characterId, e.ItemId()))
		//emitInventoryModificationEvent(l, span)(characterId, true, 2, e.ItemId(), TypeValueEquip, 1, slot, existingSlot)
	}
}

func UnequipItemForCharacter(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(characterId uint32, oldSlot int16) {
	return func(characterId uint32, oldSlot int16) {
		var e equipable.Model
		var err error

		l.Debugf("Received request to unequip item at [%d] for character [%d].", oldSlot, characterId)
		invLock := GetLockRegistry().GetById(characterId, TypeValueEquip)
		invLock.Lock()
		defer invLock.Unlock()

		txErr := db.Transaction(func(tx *gorm.DB) error {
			e, err = equipable.GetBySlot(l, tx, tenant)(characterId, oldSlot)
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
			newSlot, err := equipable.GetNextFreeSlot(l, tx, span, tenant)(inv.Id())()
			if err != nil {
				l.WithError(err).Errorf("Unable to get next free equipment slot")
				return err
			}

			err = equipable.UpdateSlot(l, tx, tenant)(e.Id(), newSlot)
			if err != nil {
				return err
			}

			l.Debugf("Unequipped [%d] for character [%d] and place it in slot [%d], from [%d].", e.Id(), characterId, newSlot, oldSlot)
			return nil
		})
		if txErr != nil {
			l.WithError(txErr).Errorf("Unable to complete unequiping item at [%d] for character [%d].", oldSlot, characterId)
			return
		}
		_ = producer.ProviderImpl(l)(span)(EnvEventTopicEquipChanged)(itemUnequippedProvider(tenant, characterId, e.ItemId()))
		//emitInventoryModificationEvent(l, span)(characterId, true, 2, itemId, TypeValueEquip, 1, newSlot, oldSlot)
	}
}

func DeleteEquipableInventory(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(characterId uint32, m EquipableModel) error {
	return func(characterId uint32, m EquipableModel) error {
		invLock := GetLockRegistry().GetById(characterId, TypeValueEquip)
		invLock.Lock()
		defer invLock.Unlock()

		return db.Transaction(func(tx *gorm.DB) error {
			for _, e := range m.Items() {
				err := equipable.DeleteByReferenceId(l, db, span, tenant)(e.ReferenceId())
				if err != nil {
					l.WithError(err).Errorf("Unable to delete equipable in inventory [%d] slot [%d].", m.Id(), e.Slot())
					return err
				}
			}
			return delete(tx, tenant.Id, m.Id())
		})
	}
}

func DeleteItemInventory(l logrus.FieldLogger, db *gorm.DB, _ opentracing.Span, tenant tenant.Model) func(characterId uint32, m ItemModel) error {
	return func(characterId uint32, m ItemModel) error {
		invLock := GetLockRegistry().GetById(characterId, m.Type())
		invLock.Lock()
		defer invLock.Unlock()

		return db.Transaction(func(tx *gorm.DB) error {
			for _, i := range m.Items() {
				err := item.DeleteBySlot(l, tx, tenant)(m.Id(), i.Slot())
				if err != nil {
					l.WithError(err).Errorf("Unable to delete item in inventory [%d] slot [%d].", m.Id(), i.Slot())
					return err
				}

			}
			return delete(tx, tenant.Id, m.Id())
		})
	}
}
