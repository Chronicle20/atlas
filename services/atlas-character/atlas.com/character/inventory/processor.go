package inventory

import (
	"atlas-character/asset"
	"atlas-character/equipable"
	statistics2 "atlas-character/equipable/statistics"
	"atlas-character/equipment"
	slot2 "atlas-character/equipment/slot"
	"atlas-character/inventory/item"
	"atlas-character/kafka/producer"
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
			ip := item.ByInventoryProvider(db, tenant)(ent.ID)
			return model.Map(model.Fold(ip, NewItemModel(ent.ID, TypeValueUse, ent.Capacity), ItemFolder), foldProperty(ref.SetUseable))()
		case TypeValueSetup:
			ip := item.ByInventoryProvider(db, tenant)(ent.ID)
			return model.Map(model.Fold(ip, NewItemModel(ent.ID, TypeValueSetup, ent.Capacity), ItemFolder), foldProperty(ref.SetSetup))()
		case TypeValueETC:
			ip := item.ByInventoryProvider(db, tenant)(ent.ID)
			return model.Map(model.Fold(ip, NewItemModel(ent.ID, TypeValueETC, ent.Capacity), ItemFolder), foldProperty(ref.SetEtc))()
		case TypeValueCash:
			ip := item.ByInventoryProvider(db, tenant)(ent.ID)
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

func CreateItem(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, eventProducer producer.Provider) func(tenant tenant.Model, characterId uint32, inventoryType Type, itemId uint32, quantity uint32) error {
	return func(tenant tenant.Model, characterId uint32, inventoryType Type, itemId uint32, quantity uint32) error {
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

		var events = model.FixedProvider[[]kafka.Message]([]kafka.Message{})
		err := db.Transaction(func(tx *gorm.DB) error {
			invId, err := GetInventoryIdByType(tx, tenant)(characterId, inventoryType)()
			if err != nil {
				l.WithError(err).Errorf("Unable to locate inventory [%d] for character [%d].", inventoryType, characterId)
				return err
			}

			iap := inventoryItemAddProvider(tenant)(characterId)(itemId)
			iup := inventoryItemUpdateProvider(tenant)(characterId)(itemId)
			var eap model.Provider[[]asset.Asset]
			var smp SlotMaxProvider
			var nac asset.Creator
			var aqu asset.QuantityUpdater

			if inventoryType == TypeValueEquip {
				eap = asset.NoOpSliceProvider
				smp = OfOneSlotMaxProvider
				nac = equipable.CreateItem(l, tx, span, tenant, statistics2.Create(l, span, tenant))(characterId)(invId, int8(inventoryType))(itemId)
				aqu = asset.NoOpQuantityUpdater
			} else {
				eap = model.SliceMap(item.ByItemIdProvider(tx)(tenant)(invId)(itemId), item.ToAsset)
				smp = func() (uint32, error) {
					// TODO properly look this up.
					return 200, nil
				}
				nac = item.CreateItem(tx, tenant)(characterId)(invId, int8(inventoryType))(itemId)
				aqu = item.UpdateQuantity(tx, tenant)
			}

			res, err := CreateAsset(l)(eap, smp, nac, aqu, iap, iup, quantity)()
			if err != nil {
				l.WithError(err).Errorf("Unable to create [%d] equipable [%d] for character [%d].", quantity, itemId, characterId)
				return err
			}
			events = model.MergeSliceProvider(events, model.FixedProvider(res))
			return err
		})
		if err != nil {
			return err
		}
		return eventProducer(EnvEventInventoryChanged)(events)
	}
}

func GetInventoryIdByType(db *gorm.DB, tenant tenant.Model) func(characterId uint32, inventoryType Type) model.Provider[uint32] {
	return func(characterId uint32, inventoryType Type) model.Provider[uint32] {
		e, err := get(tenant.Id, characterId, inventoryType)(db)()
		if err != nil {
			return model.ErrorProvider[uint32](err)
		}
		return model.FixedProvider(e.ID)
	}
}

type SlotMaxProvider model.Provider[uint32]

func OfOneSlotMaxProvider() (uint32, error) {
	return 1, nil
}

func CreateAsset(l logrus.FieldLogger) func(existingAssetProvider model.Provider[[]asset.Asset], slotMaxProvider SlotMaxProvider, newAssetCreator asset.Creator, assetQuantityUpdater asset.QuantityUpdater, addEventProvider ItemAddProvider, updateEventProvider ItemUpdateProvider, quantity uint32) model.Provider[[]kafka.Message] {
	return func(existingAssetProvider model.Provider[[]asset.Asset], slotMaxProvider SlotMaxProvider, newAssetCreator asset.Creator, assetQuantityUpdater asset.QuantityUpdater, addEventProvider ItemAddProvider, updateEventProvider ItemUpdateProvider, quantity uint32) model.Provider[[]kafka.Message] {
		runningQuantity := quantity
		slotMax, err := slotMaxProvider()
		if err != nil {
			return model.ErrorProvider[[]kafka.Message](err)
		}

		var result = model.FixedProvider[[]kafka.Message]([]kafka.Message{})

		existingItems, err := existingAssetProvider()
		if err != nil {
			l.WithError(err).Errorf("Unable to locate existing items in inventory for character.")
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
						l.Debugf("Updating existing asset [%d] of item [%d] in slot [%d] to have a quantity of [%d].", i.Id(), i.ItemId(), i.Slot(), i.Quantity())
						err = assetQuantityUpdater(i.Id(), newQuantity)
						if err != nil {
							l.WithError(err).Errorf("Updating the quantity of item [%d] to value [%d].", i.Id(), newQuantity)
						} else {
							result = model.MergeSliceProvider(result, updateEventProvider(newQuantity, i.Slot()))
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
			as, err := newAssetCreator(newQuantity)()
			if err != nil {
				return model.ErrorProvider[[]kafka.Message](err)
			}
			l.Debugf("Creating new asset [%d] of item [%d] in slot [%d] with quantity [%d].", as.Id(), as.ItemId(), as.Slot(), as.Quantity())
			result = model.MergeSliceProvider(result, addEventProvider(as.Quantity(), as.Slot()))
		}
		return result
	}
}

func EquipItemForCharacter(l logrus.FieldLogger) func(db *gorm.DB) func(tenant tenant.Model) func(freeSlotProvider func(db *gorm.DB) func(uint32) model.Provider[int16]) func(eventProducer producer.Provider) func(characterId uint32) func(source int16) func(destinationProvider equipment.DestinationProvider) {
	return func(db *gorm.DB) func(tenant tenant.Model) func(freeSlotProvider func(db *gorm.DB) func(uint32) model.Provider[int16]) func(eventProducer producer.Provider) func(characterId uint32) func(source int16) func(destinationProvider equipment.DestinationProvider) {
		return func(tenant tenant.Model) func(freeSlotProvider func(db *gorm.DB) func(uint32) model.Provider[int16]) func(eventProducer producer.Provider) func(characterId uint32) func(source int16) func(destinationProvider equipment.DestinationProvider) {
			return func(freeSlotProvider func(db *gorm.DB) func(uint32) model.Provider[int16]) func(eventProducer producer.Provider) func(characterId uint32) func(source int16) func(destinationProvider equipment.DestinationProvider) {
				return func(eventProducer producer.Provider) func(characterId uint32) func(source int16) func(destinationProvider equipment.DestinationProvider) {
					return func(characterId uint32) func(source int16) func(destinationProvider equipment.DestinationProvider) {
						characterInventoryMoveProvider := inventoryItemMoveProvider(tenant)(characterId)
						return func(source int16) func(destinationProvider equipment.DestinationProvider) {
							return func(destinationProvider equipment.DestinationProvider) {
								var e equipable.Model
								var err error

								l.Debugf("Received request to equip item at [%d] for character [%d].", source, characterId)
								invLock := GetLockRegistry().GetById(characterId, TypeValueEquip)
								invLock.Lock()
								defer invLock.Unlock()

								var events = model.FixedProvider[[]kafka.Message]([]kafka.Message{})

								err = db.Transaction(func(tx *gorm.DB) error {
									inSlotProvider := model.Flip(model.Flip(equipable.BySlotProvider)(tenant))(characterId)(tx)
									slotUpdater := model.Flip(equipable.UpdateSlot)(tenant)(tx)

									e, err = equipable.GetBySlot(tx, tenant)(characterId, source)
									if err != nil {
										l.WithError(err).Errorf("Unable to retrieve equipment in slot [%d].", source)
										return err
									}

									l.Debugf("Equipment [%d] is item [%d] for character [%d].", e.Id(), e.ItemId(), characterId)

									actualDestination, err := destinationProvider(e.ItemId())()
									if err != nil {
										l.WithError(err).Errorf("Unable to determine actual destination for item being equipped.")
										return err
									}

									l.Debugf("Equipment [%d] to be equipped in slot [%d] for character [%d].", e.Id(), actualDestination, characterId)

									l.Debugf("Attempting to move item that is currently occupying the destination to a temporary position.")
									resp, _ := moveFromSlotToSlot(l)(model.Map(inSlotProvider(actualDestination), equipable.ToAsset), temporarySlotProvider, slotUpdater, noOpInventoryItemMoveProvider)()
									events = model.MergeSliceProvider(events, model.FixedProvider(resp))

									l.Debugf("Attempting to move item that is being equipped to its final destination.")
									resp, _ = moveFromSlotToSlot(l)(model.Map(inSlotProvider(source), equipable.ToAsset), model.FixedProvider(actualDestination), slotUpdater, characterInventoryMoveProvider(source))()
									events = model.MergeSliceProvider(events, model.FixedProvider(resp))

									l.Debugf("Attempting to move item that is in the temporary position to where the item that was just equipped was.")
									resp, _ = moveFromSlotToSlot(l)(model.Map(inSlotProvider(temporarySlot()), equipable.ToAsset), model.FixedProvider(source), slotUpdater, noOpInventoryItemMoveProvider)()
									events = model.MergeSliceProvider(events, model.FixedProvider(resp))

									l.Debugf("Now verifying other inventory operations that may be necessary.")

									invId, err := GetInventoryIdByType(tx, tenant)(characterId, TypeValueEquip)()
									if err != nil {
										l.WithError(err).Errorf("Unable to locate inventory [%d] for character [%d].", TypeValueEquip, characterId)
										return err
									}
									nextFreeSlotProvider := freeSlotProvider(tx)(invId)

									if e.ItemId()/10000 == 105 {
										l.Debugf("Item is an overall, we also need to unequip the bottom.")
										resp, err = moveFromSlotToSlot(l)(model.Map(inSlotProvider(int16(slot2.PositionBottom)), equipable.ToAsset), nextFreeSlotProvider, slotUpdater, characterInventoryMoveProvider(int16(slot2.PositionBottom)))()
										if err != nil {
											l.WithError(err).Errorf("Unable to move bottom out of its slot.")
											return err
										}
										events = model.MergeSliceProvider(events, model.FixedProvider(resp))
									}
									if actualDestination == int16(slot2.PositionBottom) {
										l.Debugf("Item is a bottom, need to unequip an overall if its in the top slot.")
										ip := model.Map(model.Map(inSlotProvider(int16(slot2.PositionOverall)), equipable.ToAsset), IsOverall)
										resp, err = moveFromSlotToSlot(l)(ip, nextFreeSlotProvider, slotUpdater, characterInventoryMoveProvider(int16(slot2.PositionOverall)))()
										if err != nil && !errors.Is(err, notOverall) {
											l.WithError(err).Errorf("Unable to move overall out of its slot.")
											return err
										}
										events = model.MergeSliceProvider(events, model.FixedProvider(resp))
									}
									return nil
								})
								if err != nil {
									l.WithError(err).Errorf("Unable to complete the equipment of item [%d] for character [%d].", e.Id(), characterId)
									return
								}

								err = eventProducer(EnvEventInventoryChanged)(events)
								if err != nil {
									l.WithError(err).Errorf("Unable to convey inventory modifications to character [%d].", characterId)
								}
							}
						}
					}
				}
			}
		}
	}
}

var notOverall = errors.New("not an overall")

func IsOverall(m asset.Asset) (asset.Asset, error) {
	if m.ItemId()/10000 == 105 {
		return m, nil
	}
	return nil, notOverall
}

func moveFromSlotToSlot(l logrus.FieldLogger) func(modelProvider model.Provider[asset.Asset], newSlotProvider model.Provider[int16], slotUpdater func(id uint32, slot int16) error, moveEventProvider func(itemId uint32) func(slot int16) model.Provider[[]kafka.Message]) model.Provider[[]kafka.Message] {
	return func(modelProvider model.Provider[asset.Asset], newSlotProvider model.Provider[int16], slotUpdater func(id uint32, slot int16) error, moveEventProvider func(itemId uint32) func(slot int16) model.Provider[[]kafka.Message]) model.Provider[[]kafka.Message] {
		m, err := modelProvider()
		if err != nil {
			return model.ErrorProvider[[]kafka.Message](err)
		}
		if m.Id() == 0 {
			return model.ErrorProvider[[]kafka.Message](errors.New("item not found"))
		}
		newSlot, err := newSlotProvider()
		if err != nil {
			return model.ErrorProvider[[]kafka.Message](err)
		}
		err = slotUpdater(m.Id(), newSlot)
		if err != nil {
			return model.ErrorProvider[[]kafka.Message](err)
		}
		l.Debugf("Moved [%d] of template [%d] to slot [%d] from [%d].", m.Id(), m.ItemId(), newSlot, m.Slot())
		return moveEventProvider(m.ItemId())(newSlot)
	}
}

func UnequipItemForCharacter(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(characterId uint32, oldSlot int16) {
	return func(characterId uint32, oldSlot int16) {
		l.Debugf("Received request to unequip item at [%d] for character [%d].", oldSlot, characterId)
		invLock := GetLockRegistry().GetById(characterId, TypeValueEquip)
		invLock.Lock()
		defer invLock.Unlock()

		var events = model.FixedProvider[[]kafka.Message]([]kafka.Message{})
		txErr := db.Transaction(func(tx *gorm.DB) error {
			inSlotProvider := model.Flip(model.Flip(equipable.BySlotProvider)(tenant))(characterId)(tx)
			slotUpdater := model.Flip(equipable.UpdateSlot)(tenant)(tx)
			characterInventoryMoveProvider := inventoryItemMoveProvider(tenant)(characterId)

			invId, err := GetInventoryIdByType(tx, tenant)(characterId, TypeValueEquip)()
			if err != nil {
				l.WithError(err).Errorf("Unable to locate inventory [%d] for character [%d].", TypeValueEquip, characterId)
				return err
			}
			nextFreeSlotProvider := equipable.GetNextFreeSlot(l)(tx)(span)(tenant)(invId)

			resp, err := moveFromSlotToSlot(l)(model.Map(inSlotProvider(oldSlot), equipable.ToAsset), nextFreeSlotProvider, slotUpdater, characterInventoryMoveProvider(oldSlot))()
			if err != nil {
				l.WithError(err).Errorf("Unable to move overall out of its slot.")
				return err
			}
			events = model.MergeSliceProvider(events, model.FixedProvider(resp))
			return nil
		})
		if txErr != nil {
			l.WithError(txErr).Errorf("Unable to complete unequiping item at [%d] for character [%d].", oldSlot, characterId)
			return
		}
		err := producer.ProviderImpl(l)(span)(EnvEventInventoryChanged)(events)
		if err != nil {
			l.WithError(err).Errorf("Unable to convey inventory modifications to character [%d].", characterId)
		}
	}
}

func DeleteInventory(l logrus.FieldLogger, db *gorm.DB) func(tenant tenant.Model, characterId uint32, inventoryType Type, itemIdProvider model.Provider[[]uint32], itemDeleter func(db *gorm.DB) func(tenant tenant.Model) model.Operator[uint32]) error {
	return func(tenant tenant.Model, characterId uint32, inventoryType Type, itemIdProvider model.Provider[[]uint32], itemDeleter func(db *gorm.DB) func(tenant tenant.Model) model.Operator[uint32]) error {
		invLock := GetLockRegistry().GetById(characterId, inventoryType)
		invLock.Lock()
		defer invLock.Unlock()
		return db.Transaction(func(tx *gorm.DB) error {
			err := model.ForEachSlice[uint32](itemIdProvider, itemDeleter(tx)(tenant))
			if err != nil {
				l.WithError(err).Errorf("Unable to delete items in inventory.")
				return err
			}
			return deleteByType(tx, tenant.Id, characterId, int8(inventoryType))
		})
	}
}

func DeleteEquipableInventory(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(characterId uint32, m EquipableModel) error {
	return func(characterId uint32, m EquipableModel) error {
		idp := model.SliceMap(model.FixedProvider(m.Items()), equipable.ReferenceId)
		return DeleteInventory(l, db)(tenant, characterId, TypeValueEquip, idp, equipable.DeleteByReferenceId(l)(span))
	}
}

func DeleteItemInventory(l logrus.FieldLogger, db *gorm.DB, _ opentracing.Span, tenant tenant.Model) func(characterId uint32, m ItemModel) error {
	return func(characterId uint32, m ItemModel) error {
		idp := model.SliceMap(model.FixedProvider(m.Items()), item.Id)
		return DeleteInventory(l, db)(tenant, characterId, m.mType, idp, item.DeleteById)
	}
}

func Move(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, eventProducer producer.Provider) func(tenant tenant.Model, characterId uint32, inventoryType byte, source int16, destination int16) error {
	return func(tenant tenant.Model, characterId uint32, inventoryType byte, source int16, destination int16) error {
		if inventoryType == 1 {
			return moveEquip(l, db, span, eventProducer)(tenant, characterId, source, destination)
		} else {
			return moveItem(l, db, span, eventProducer)(tenant, characterId, inventoryType, source, destination)
		}
	}
}

func moveItem(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, eventProducer producer.Provider) func(tenant tenant.Model, characterId uint32, inventoryType byte, source int16, destination int16) error {
	return func(tenant tenant.Model, characterId uint32, inventoryType byte, source int16, destination int16) error {
		characterInventoryMoveProvider := inventoryItemMoveProvider(tenant)(characterId)

		l.Debugf("Received request to move item at [%d] to [%d] for character [%d].", source, destination, characterId)
		invLock := GetLockRegistry().GetById(characterId, Type(inventoryType))
		invLock.Lock()
		defer invLock.Unlock()

		// TODO need to combine quantities if moving to the same item type.

		var events = model.FixedProvider[[]kafka.Message]([]kafka.Message{})
		txErr := db.Transaction(func(tx *gorm.DB) error {
			slotUpdater := model.Flip(item.UpdateSlot)(tenant)(tx)

			invId, err := GetInventoryIdByType(tx, tenant)(characterId, Type(inventoryType))()
			if err != nil {
				l.WithError(err).Errorf("Unable to locate inventory [%d] for character [%d].", inventoryType, characterId)
				return err
			}
			inSlotProvider := model.Flip(model.Flip(item.BySlotProvider)(tenant))(invId)(tx)

			l.Debugf("Attempting to move item that is currently occupying the destination to a temporary position.")
			resp, _ := moveFromSlotToSlot(l)(model.Map(inSlotProvider(destination), item.ToAsset), temporarySlotProvider, slotUpdater, noOpInventoryItemMoveProvider)()
			events = model.MergeSliceProvider(events, model.FixedProvider(resp))

			l.Debugf("Attempting to move item that is being moved to its final destination.")
			resp, _ = moveFromSlotToSlot(l)(model.Map(inSlotProvider(source), item.ToAsset), model.FixedProvider(destination), slotUpdater, characterInventoryMoveProvider(source))()
			events = model.MergeSliceProvider(events, model.FixedProvider(resp))

			l.Debugf("Attempting to move item that is in the temporary position to where the item that was just equipped was.")
			resp, _ = moveFromSlotToSlot(l)(model.Map(inSlotProvider(temporarySlot()), item.ToAsset), model.FixedProvider(source), slotUpdater, noOpInventoryItemMoveProvider)()
			events = model.MergeSliceProvider(events, model.FixedProvider(resp))
			return nil
		})
		if txErr != nil {
			l.WithError(txErr).Errorf("Unable to complete moving item for character [%d].", characterId)
			return txErr
		}
		err := eventProducer(EnvEventInventoryChanged)(events)
		if err != nil {
			l.WithError(err).Errorf("Unable to convey inventory modifications to character [%d].", characterId)
		}
		return err
	}
}

func temporarySlot() int16 {
	return int16(math.MinInt16)
}

func temporarySlotProvider() (int16, error) {
	return temporarySlot(), nil
}

func moveEquip(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, eventProducer producer.Provider) func(tenant tenant.Model, characterId uint32, source int16, destination int16) error {
	return func(tenant tenant.Model, characterId uint32, source int16, destination int16) error {
		l.Debugf("Received request to move item at [%d] to [%d] for character [%d].", source, destination, characterId)
		invLock := GetLockRegistry().GetById(characterId, TypeValueEquip)
		invLock.Lock()
		defer invLock.Unlock()

		var events = model.FixedProvider[[]kafka.Message]([]kafka.Message{})
		txErr := db.Transaction(func(tx *gorm.DB) error {
			inSlotProvider := model.Flip(model.Flip(equipable.BySlotProvider)(tenant))(characterId)(tx)
			slotUpdater := model.Flip(equipable.UpdateSlot)(tenant)(tx)
			characterInventoryMoveProvider := inventoryItemMoveProvider(tenant)(characterId)

			l.Debugf("Attempting to move item that is currently occupying the destination to a temporary position.")
			resp, _ := moveFromSlotToSlot(l)(model.Map(inSlotProvider(destination), equipable.ToAsset), temporarySlotProvider, slotUpdater, noOpInventoryItemMoveProvider)()
			events = model.MergeSliceProvider(events, model.FixedProvider(resp))

			l.Debugf("Attempting to move item that is being moved to its final destination.")
			resp, _ = moveFromSlotToSlot(l)(model.Map(inSlotProvider(source), equipable.ToAsset), model.FixedProvider(destination), slotUpdater, characterInventoryMoveProvider(source))()
			events = model.MergeSliceProvider(events, model.FixedProvider(resp))

			l.Debugf("Attempting to move item that is in the temporary position to where the item that was just equipped was.")
			resp, _ = moveFromSlotToSlot(l)(model.Map(inSlotProvider(temporarySlot()), equipable.ToAsset), model.FixedProvider(source), slotUpdater, noOpInventoryItemMoveProvider)()
			events = model.MergeSliceProvider(events, model.FixedProvider(resp))
			return nil
		})
		if txErr != nil {
			l.WithError(txErr).Errorf("Unable to complete moving item for character [%d].", characterId)
			return txErr
		}
		err := eventProducer(EnvEventInventoryChanged)(events)
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
			invId, err := GetInventoryIdByType(tx, tenant)(characterId, Type(inventoryType))()
			if err != nil {
				l.WithError(err).Errorf("Unable to locate inventory [%d] for character [%d].", inventoryType, characterId)
				return err
			}

			i, err := item.GetBySlot(tx, tenant)(invId, source)
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve item in slot [%d].", source)
				return err
			}

			initialQuantity := i.Quantity()

			if initialQuantity <= uint32(quantity) {
				err = item.DeleteById(tx)(tenant)(i.Id())
				if err != nil {
					l.WithError(err).Errorf("Unable to drop item in slot [%d].", source)
					return err
				}
				events = model.MergeSliceProvider(events, inventoryItemRemoveProvider(tenant, characterId, i.ItemId(), i.Slot()))
				return nil
			}

			newQuantity := initialQuantity - uint32(quantity)
			err = item.UpdateQuantity(tx, tenant)(i.Id(), newQuantity)
			if err != nil {
				l.WithError(err).Errorf("Unable to drop [%d] item in slot [%d].", quantity, source)
				return err
			}
			events = model.MergeSliceProvider(events, inventoryItemUpdateProvider(tenant)(characterId)(i.ItemId())(newQuantity, i.Slot()))
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
			e, err := equipable.GetBySlot(tx, tenant)(characterId, source)
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve equipment in slot [%d].", source)
				return err
			}
			err = equipable.DropByReferenceId(l, tx, tenant)(e.ReferenceId())
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
