package inventory

import (
	"atlas-character/equipable"
	slot2 "atlas-character/equipment/slot"
	"atlas-character/equipment/slot/information"
	"atlas-character/equipment/statistics"
	"atlas-character/inventory/item"
	"atlas-character/kafka/producer"
	"atlas-character/slottable"
	"atlas-character/tenant"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/opentracing/opentracing-go"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"math"
)

func ByCharacterIdProvider(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(characterId uint32) model.Provider[Model] {
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
		return ByCharacterIdProvider(l, db, span, tenant)(characterId)()
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

		var events model.Provider[[]kafka.Message]
		err := db.Transaction(func(tx *gorm.DB) error {
			inv, err := GetInventoryByType(l, tx, span, tenant)(characterId, inventoryType)()
			if err != nil {
				l.WithError(err).Errorf("Unable to locate inventory [%d] for character [%d].", inventoryType, characterId)
				return err
			}

			if inventoryType == TypeValueEquip {
				events = createEquipable(l, tx, span, tenant)(characterId, inv.Id(), inventoryType, itemId)
				if err != nil {
					l.WithError(err).Errorf("Unable to create [%d] equipable [%d] for character [%d].", quantity, itemId, characterId)
					return err
				}
			} else {
				events = createItem(l, tx, span, tenant)(characterId, inv.Id(), inventoryType, itemId, quantity)
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
		return producer.ProviderImpl(l)(span)(EnvEventInventoryChanged)(events)
	}
}

func GetInventoryByType(l logrus.FieldLogger, tx *gorm.DB, span opentracing.Span, tenant tenant.Model) func(characterId uint32, inventoryType Type) model.Provider[ItemHolder] {
	return func(characterId uint32, inventoryType Type) model.Provider[ItemHolder] {
		return model.Map(ByCharacterIdProvider(l, tx, span, tenant)(characterId), getInventoryByType(inventoryType))
	}
}

func getInventoryByType(inventoryType Type) model.Transformer[Model, ItemHolder] {
	return func(m Model) (ItemHolder, error) {
		return m.GetHolderByType(inventoryType)
	}
}

func createEquipable(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(characterId uint32, inventoryId uint32, inventoryType Type, itemId uint32) model.Provider[[]kafka.Message] {
	var creator itemCreator = equipable.CreateItem(l, db, span, tenant)
	return func(characterId uint32, inventoryId uint32, inventoryType Type, itemId uint32) model.Provider[[]kafka.Message] {
		return createNewItem(l, tenant)(creator, characterId, inventoryId, inventoryType, itemId, 1)
	}
}

func createItem(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(characterId uint32, inventoryId uint32, inventoryType Type, itemId uint32, quantity uint32) model.Provider[[]kafka.Message] {
	var creator itemCreator = item.CreateItem(l, db, tenant)
	return func(characterId uint32, inventoryId uint32, inventoryType Type, itemId uint32, quantity uint32) model.Provider[[]kafka.Message] {
		runningQuantity := quantity
		slotMax := item.MaxInSlot()
		var result = model.FixedProvider[[]kafka.Message]([]kafka.Message{})

		existingItems, err := item.GetByItemId(l, db, tenant)(inventoryId, itemId)
		if err != nil {
			l.WithError(err).Errorf("Unable to locate items [%d] in inventory [%d] for character [%d].", itemId, inventoryType, characterId)
			return model.ErrorProvider[[]kafka.Message](err)
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
							result = model.MergeSliceProvider(result, inventoryItemUpdateProvider(tenant, characterId, itemId, newQuantity, i.Slot()))
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
			nes, err := createNewItem(l, tenant)(creator, characterId, inventoryId, inventoryType, itemId, newQuantity)()
			if err != nil {
				return model.ErrorProvider[[]kafka.Message](err)
			}
			result = model.MergeSliceProvider(result, model.FixedProvider[[]kafka.Message](nes))
		}
		return result
	}
}

func createNewItem(l logrus.FieldLogger, tenant tenant.Model) func(creator itemCreator, characterId uint32, inventoryId uint32, inventoryType Type, itemId uint32, quantity uint32) model.Provider[[]kafka.Message] {
	return func(creator itemCreator, characterId uint32, inventoryId uint32, inventoryType Type, itemId uint32, quantity uint32) model.Provider[[]kafka.Message] {
		i, err := creator(characterId, inventoryId, int8(inventoryType), itemId, quantity)()
		if err != nil {
			l.WithError(err).Errorf("Unable to create item [%d] for character [%d].", itemId, characterId)
			return model.ErrorProvider[[]kafka.Message](err)
		}
		l.Debugf("Item created. Creating inventory [%d] adjustment of [%d] item [%d] in slot [%d].", inventoryType, quantity, itemId, i.Slot())
		return inventoryItemAddProvider(tenant, characterId, itemId, quantity, i.Slot())
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

		var events = model.FixedProvider[[]kafka.Message]([]kafka.Message{})

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
			is, err := statistics.GetById(l, span, tenant)(e.ItemId())
			if err != nil {
				return err
			}

			actualDestination := int16(0)
			if is.Cash() {
				actualDestination = slots[0].Slot() - 100
			} else {
				actualDestination = slots[0].Slot()
			}

			l.Debugf("Equipment [%d] to be equipped in slot [%d] for character [%d].", e.Id(), actualDestination, characterId)

			temporarySlot := int16(math.MinInt16)

			existingSlot := e.Slot()
			if equip, err := equipable.GetBySlot(l, tx, tenant)(characterId, actualDestination); err == nil && equip.Id() != 0 {
				l.Debugf("Equipment [%d] already exists in slot [%d], that item will be moved temporarily to [%d] for character [%d].", equip.Id(), actualDestination, temporarySlot, characterId)
				_ = equipable.UpdateSlot(l, tx, tenant)(equip.Id(), temporarySlot)
			}

			err = equipable.UpdateSlot(l, tx, tenant)(e.Id(), actualDestination)
			if err != nil {
				return err
			}
			l.Debugf("Moved item [%d] from slot [%d] to [%d] for character [%d].", e.ItemId(), existingSlot, actualDestination, characterId)

			if equip, err := equipable.GetBySlot(l, tx, tenant)(characterId, temporarySlot); err == nil && equip.Id() != 0 {
				err := equipable.UpdateSlot(l, tx, tenant)(equip.Id(), existingSlot)
				if err != nil {
					return err
				}
				l.Debugf("Moved item from temporary location [%d] to slot [%d] for character [%d].", temporarySlot, existingSlot, characterId)
				events = model.MergeSliceProvider(events, inventoryItemMoveProvider(tenant, characterId, equip.ItemId(), existingSlot, equip.Slot()))
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
					events = model.MergeSliceProvider(events, inventoryItemMoveProvider(tenant, characterId, e.ItemId(), newSlot, int16(slot2.PositionBottom)))
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
					events = model.MergeSliceProvider(events, inventoryItemMoveProvider(tenant, characterId, e.ItemId(), newSlot, int16(slot2.PositionOverall)))
				}
			}
			return nil
		})
		if err != nil {
			l.WithError(err).Errorf("Unable to complete the equipment of item [%d] for character [%d].", e.Id(), characterId)
			return
		}

		events = model.MergeSliceProvider(events, inventoryItemMoveProvider(tenant, characterId, e.ItemId(), destination, source))

		err = producer.ProviderImpl(l)(span)(EnvEventInventoryChanged)(events)
		if err != nil {
			l.WithError(err).Errorf("Unable to convey inventory modifications to character [%d].", characterId)
		}
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

		var events = model.FixedProvider[[]kafka.Message]([]kafka.Message{})
		txErr := db.Transaction(func(tx *gorm.DB) error {
			e, err = equipable.GetBySlot(l, tx, tenant)(characterId, oldSlot)
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve equipment in slot [%d].", oldSlot)
				return err
			}

			inv, err := GetInventoryByType(l, tx, span, tenant)(characterId, TypeValueEquip)()
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
			events = model.MergeSliceProvider(events, inventoryItemMoveProvider(tenant, characterId, e.ItemId(), newSlot, oldSlot))
			return nil
		})
		if txErr != nil {
			l.WithError(txErr).Errorf("Unable to complete unequiping item at [%d] for character [%d].", oldSlot, characterId)
			return
		}
		err = producer.ProviderImpl(l)(span)(EnvEventInventoryChanged)(events)
		if err != nil {
			l.WithError(err).Errorf("Unable to convey inventory modifications to character [%d].", characterId)
		}
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

type SlotGetter[E any] func(inventoryId uint32, source int16) (E, error)

func Move(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(characterId uint32, inventoryType byte, source int16, destination int16) error {
	return func(characterId uint32, inventoryType byte, source int16, destination int16) error {
		if inventoryType == 1 {
			return moveEquip(l, db, span, tenant)(characterId, source, destination)
		} else {
			return moveItem(l, db, span, tenant)(characterId, inventoryType, source, destination)
		}
	}
}

func moveItem(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(characterId uint32, inventoryType byte, source int16, destination int16) error {
	return func(characterId uint32, inventoryType byte, source int16, destination int16) error {
		l.Debugf("Received request to move item at [%d] to [%d] for character [%d].", source, destination, characterId)
		invLock := GetLockRegistry().GetById(characterId, Type(inventoryType))
		invLock.Lock()
		defer invLock.Unlock()

		// TODO need to combine quantities if moving to the same item type.

		var events = model.FixedProvider[[]kafka.Message]([]kafka.Message{})
		txErr := db.Transaction(func(tx *gorm.DB) error {
			inv, err := GetInventoryByType(l, tx, span, tenant)(characterId, Type(inventoryType))()
			if err != nil {
				l.WithError(err).Errorf("Unable to locate inventory [%d] for character [%d].", inventoryType, characterId)
				return err
			}

			movingItem, err := item.GetBySlot(l, tx, tenant)(inv.Id(), source)
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve item in slot [%d].", source)
				return err
			}
			l.Debugf("Item [%d] is item [%d] for character [%d].", movingItem.Id(), movingItem.ItemId(), characterId)

			temporarySlot := int16(math.MinInt16)
			if otherItem, err := item.GetBySlot(l, tx, tenant)(inv.Id(), destination); err == nil && otherItem.Id() != 0 {
				l.Debugf("Item [%d] already exists in slot [%d], that item will be moved temporarily to [%d] for character [%d].", otherItem.Id(), destination, temporarySlot, characterId)
				_ = item.UpdateSlot(l, tx, tenant)(otherItem.Id(), temporarySlot)
			}

			err = item.UpdateSlot(l, tx, tenant)(movingItem.Id(), destination)
			if err != nil {
				return err
			}
			l.Debugf("Moved item [%d] from slot [%d] to [%d] for character [%d].", movingItem.ItemId(), source, destination, characterId)

			if otherItem, err := item.GetBySlot(l, tx, tenant)(characterId, temporarySlot); err == nil && otherItem.Id() != 0 {
				err := item.UpdateSlot(l, tx, tenant)(otherItem.Id(), source)
				if err != nil {
					return err
				}
				l.Debugf("Moved item from temporary location [%d] to slot [%d] for character [%d].", temporarySlot, source, characterId)
				events = model.MergeSliceProvider(events, inventoryItemMoveProvider(tenant, characterId, otherItem.ItemId(), source, otherItem.Slot()))
			}
			return nil
		})
		if txErr != nil {
			l.WithError(txErr).Errorf("Unable to complete moving item for character [%d].", characterId)
			return txErr
		}
		err := producer.ProviderImpl(l)(span)(EnvEventInventoryChanged)(events)
		if err != nil {
			l.WithError(err).Errorf("Unable to convey inventory modifications to character [%d].", characterId)
		}
		return err
	}
}

func moveEquip(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(characterId uint32, source int16, destination int16) error {
	return func(characterId uint32, source int16, destination int16) error {
		l.Debugf("Received request to move item at [%d] to [%d] for character [%d].", source, destination, characterId)
		invLock := GetLockRegistry().GetById(characterId, TypeValueEquip)
		invLock.Lock()
		defer invLock.Unlock()

		var events = model.FixedProvider[[]kafka.Message]([]kafka.Message{})
		txErr := db.Transaction(func(tx *gorm.DB) error {
			e, err := equipable.GetBySlot(l, tx, tenant)(characterId, source)
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve equipment in slot [%d].", source)
				return err
			}
			l.Debugf("Equipment [%d] is item [%d] for character [%d].", e.Id(), e.ItemId(), characterId)

			temporarySlot := int16(math.MinInt16)
			if equip, err := equipable.GetBySlot(l, tx, tenant)(characterId, destination); err == nil && equip.Id() != 0 {
				l.Debugf("Equipment [%d] already exists in slot [%d], that item will be moved temporarily to [%d] for character [%d].", equip.Id(), destination, temporarySlot, characterId)
				_ = equipable.UpdateSlot(l, tx, tenant)(equip.Id(), temporarySlot)
			}

			err = equipable.UpdateSlot(l, tx, tenant)(e.Id(), destination)
			if err != nil {
				return err
			}
			l.Debugf("Moved item [%d] from slot [%d] to [%d] for character [%d].", e.ItemId(), source, destination, characterId)

			if equip, err := equipable.GetBySlot(l, tx, tenant)(characterId, temporarySlot); err == nil && equip.Id() != 0 {
				err := equipable.UpdateSlot(l, tx, tenant)(equip.Id(), source)
				if err != nil {
					return err
				}
				l.Debugf("Moved item from temporary location [%d] to slot [%d] for character [%d].", temporarySlot, source, characterId)
				events = model.MergeSliceProvider(events, inventoryItemMoveProvider(tenant, characterId, equip.ItemId(), source, equip.Slot()))
			}
			return nil
		})
		if txErr != nil {
			l.WithError(txErr).Errorf("Unable to complete moving item for character [%d].", characterId)
			return txErr
		}
		err := producer.ProviderImpl(l)(span)(EnvEventInventoryChanged)(events)
		if err != nil {
			l.WithError(err).Errorf("Unable to convey inventory modifications to character [%d].", characterId)
		}
		return err
	}
}

func Drop(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(characterId uint32, inventoryType byte, source int16, quantity int16) error {
	return func(characterId uint32, inventoryType byte, source int16, quantity int16) error {
		if inventoryType == 1 {
			return dropEquip(l, db, span, tenant)(characterId, source, quantity)
		} else {
			return dropItem(l, db, span, tenant)(characterId, inventoryType, source, quantity)
		}
	}
}

func dropItem(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(characterId uint32, inventoryType byte, source int16, quantity int16) error {
	return func(characterId uint32, inventoryType byte, source int16, quantity int16) error {
		l.Debugf("Received request to drop item at [%d] for character [%d].", source, characterId)
		invLock := GetLockRegistry().GetById(characterId, Type(inventoryType))
		invLock.Lock()
		defer invLock.Unlock()

		var events = model.FixedProvider[[]kafka.Message]([]kafka.Message{})
		txErr := db.Transaction(func(tx *gorm.DB) error {
			inv, err := GetInventoryByType(l, tx, span, tenant)(characterId, Type(inventoryType))()
			if err != nil {
				l.WithError(err).Errorf("Unable to locate inventory [%d] for character [%d].", inventoryType, characterId)
				return err
			}

			i, err := item.GetBySlot(l, tx, tenant)(inv.Id(), source)
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve item in slot [%d].", source)
				return err
			}

			initialQuantity := i.Quantity()

			if initialQuantity <= uint32(quantity) {
				err = item.DeleteBySlot(l, db, tenant)(inv.Id(), source)
				if err != nil {
					l.WithError(err).Errorf("Unable to drop item in slot [%d].", source)
					return err
				}
				events = model.MergeSliceProvider(events, inventoryItemRemoveProvider(tenant, characterId, i.ItemId(), i.Slot()))
				return nil
			}

			newQuantity := initialQuantity - uint32(quantity)
			err = item.UpdateQuantity(l, db, tenant)(inv.Id(), newQuantity)
			if err != nil {
				l.WithError(err).Errorf("Unable to drop item in slot [%d].", source)
				return err
			}
			events = model.MergeSliceProvider(events, inventoryItemUpdateProvider(tenant, characterId, i.ItemId(), newQuantity, i.Slot()))
			return nil
		})
		if txErr != nil {
			l.WithError(txErr).Errorf("Unable to complete dropping item for character [%d].", characterId)
			return txErr
		}
		err := producer.ProviderImpl(l)(span)(EnvEventInventoryChanged)(events)
		if err != nil {
			l.WithError(err).Errorf("Unable to convey inventory modifications to character [%d].", characterId)
		}
		return err
	}
}

func dropEquip(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(characterId uint32, source int16, quantity int16) error {
	return func(characterId uint32, source int16, quantity int16) error {
		l.Debugf("Received request to drop item at [%d] for character [%d].", source, characterId)
		invLock := GetLockRegistry().GetById(characterId, TypeValueEquip)
		invLock.Lock()
		defer invLock.Unlock()

		var events = model.FixedProvider[[]kafka.Message]([]kafka.Message{})
		txErr := db.Transaction(func(tx *gorm.DB) error {
			e, err := equipable.GetBySlot(l, tx, tenant)(characterId, source)
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve equipment in slot [%d].", source)
				return err
			}
			err = equipable.DropByReferenceId(l, db, tenant)(e.ReferenceId())
			if err != nil {
				l.WithError(err).Errorf("Unable to drop equipment in slot [%d].", source)
				return err
			}
			events = model.MergeSliceProvider(events, inventoryItemRemoveProvider(tenant, characterId, e.ItemId(), e.Slot()))
			return nil
		})
		if txErr != nil {
			l.WithError(txErr).Errorf("Unable to complete dropping item for character [%d].", characterId)
			return txErr
		}
		err := producer.ProviderImpl(l)(span)(EnvEventInventoryChanged)(events)
		if err != nil {
			l.WithError(err).Errorf("Unable to convey inventory modifications to character [%d].", characterId)
		}
		return err
	}
}
