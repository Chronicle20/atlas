package inventory

import (
	"atlas-character/asset"
	"atlas-character/equipable"
	statistics2 "atlas-character/equipable/statistics"
	"atlas-character/equipment"
	slot2 "atlas-character/equipment/slot"
	"atlas-character/inventory/item"
	"atlas-character/kafka/producer"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"math"
)

func ByCharacterIdProvider(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(characterId uint32) model.Provider[Model] {
	return func(db *gorm.DB) func(ctx context.Context) func(characterId uint32) model.Provider[Model] {
		return func(ctx context.Context) func(characterId uint32) model.Provider[Model] {
			folder := foldInventory(l)(db)(ctx)
			return func(characterId uint32) model.Provider[Model] {
				t := tenant.MustFromContext(ctx)
				return model.Fold[entity, Model](getByCharacter(t.Id(), characterId)(db), supplier, folder)
			}
		}
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

func foldInventory(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(ref Model, ent entity) (Model, error) {
	return func(db *gorm.DB) func(ctx context.Context) func(ref Model, ent entity) (Model, error) {
		return func(ctx context.Context) func(ref Model, ent entity) (Model, error) {
			return func(ref Model, ent entity) (Model, error) {
				switch Type(ent.InventoryType) {
				case TypeValueEquip:
					ep := equipable.InInventoryProvider(l)(db)(ctx)(ent.ID)
					return model.Map(foldProperty(ref.SetEquipable))(model.Fold(ep, NewEquipableModel(ent.ID, ent.Capacity), EquipableFolder))()
				case TypeValueUse:
					ip := item.ByInventoryProvider(db)(ctx)(ent.ID)
					return model.Map(foldProperty(ref.SetUseable))(model.Fold(ip, NewItemModel(ent.ID, TypeValueUse, ent.Capacity), ItemFolder))()
				case TypeValueSetup:
					ip := item.ByInventoryProvider(db)(ctx)(ent.ID)
					return model.Map(foldProperty(ref.SetSetup))(model.Fold(ip, NewItemModel(ent.ID, TypeValueSetup, ent.Capacity), ItemFolder))()
				case TypeValueETC:
					ip := item.ByInventoryProvider(db)(ctx)(ent.ID)
					return model.Map(foldProperty(ref.SetEtc))(model.Fold(ip, NewItemModel(ent.ID, TypeValueETC, ent.Capacity), ItemFolder))()
				case TypeValueCash:
					ip := item.ByInventoryProvider(db)(ctx)(ent.ID)
					return model.Map(foldProperty(ref.SetCash))(model.Fold(ip, NewItemModel(ent.ID, TypeValueCash, ent.Capacity), ItemFolder))()
				}
				return ref, errors.New("unknown inventory type")
			}
		}
	}
}

func GetInventories(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(characterId uint32) (Model, error) {
	return func(db *gorm.DB) func(ctx context.Context) func(characterId uint32) (Model, error) {
		return func(ctx context.Context) func(characterId uint32) (Model, error) {
			return func(characterId uint32) (Model, error) {
				return ByCharacterIdProvider(l)(db)(ctx)(characterId)()
			}
		}
	}
}

func Create(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(characterId uint32, defaultCapacity uint32) (Model, error) {
	return func(db *gorm.DB) func(ctx context.Context) func(characterId uint32, defaultCapacity uint32) (Model, error) {
		return func(ctx context.Context) func(characterId uint32, defaultCapacity uint32) (Model, error) {
			return func(characterId uint32, defaultCapacity uint32) (Model, error) {
				tenant := tenant.MustFromContext(ctx)
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
				return GetInventories(l)(db)(ctx)(characterId)
			}
		}
	}
}

func CreateItem(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(eventProducer producer.Provider) func(characterId uint32, inventoryType Type, itemId uint32, quantity uint32) error {
	return func(db *gorm.DB) func(ctx context.Context) func(eventProducer producer.Provider) func(characterId uint32, inventoryType Type, itemId uint32, quantity uint32) error {
		return func(ctx context.Context) func(eventProducer producer.Provider) func(characterId uint32, inventoryType Type, itemId uint32, quantity uint32) error {
			return func(eventProducer producer.Provider) func(characterId uint32, inventoryType Type, itemId uint32, quantity uint32) error {
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

					var events = model.FixedProvider[[]kafka.Message]([]kafka.Message{})
					err := db.Transaction(func(tx *gorm.DB) error {
						invId, err := GetInventoryIdByType(tx)(ctx)(characterId, inventoryType)()
						if err != nil {
							l.WithError(err).Errorf("Unable to locate inventory [%d] for character [%d].", inventoryType, characterId)
							return err
						}

						iap := inventoryItemAddProvider(characterId)(itemId)
						iup := inventoryItemUpdateProvider(characterId)(itemId)
						var eap model.Provider[[]asset.Asset]
						var smp SlotMaxProvider
						var nac asset.Creator
						var aqu asset.QuantityUpdater

						if inventoryType == TypeValueEquip {
							eap = asset.NoOpSliceProvider
							smp = OfOneSlotMaxProvider
							nac = equipable.CreateItem(l)(tx)(ctx)(statistics2.Create(l)(ctx))(characterId)(invId, int8(inventoryType))(itemId)
							aqu = asset.NoOpQuantityUpdater
						} else {
							eap = item.AssetByItemIdProvider(tx)(ctx)(invId)(itemId)
							smp = func() (uint32, error) {
								// TODO properly look this up.
								return 200, nil
							}
							nac = item.CreateItem(tx)(ctx)(characterId)(invId, int8(inventoryType))(itemId)
							aqu = item.UpdateQuantity(tx)(ctx)
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
		}
	}
}

func GetInventoryIdByType(db *gorm.DB) func(ctx context.Context) func(characterId uint32, inventoryType Type) model.Provider[uint32] {
	return func(ctx context.Context) func(characterId uint32, inventoryType Type) model.Provider[uint32] {
		return func(characterId uint32, inventoryType Type) model.Provider[uint32] {
			t := tenant.MustFromContext(ctx)
			e, err := get(t.Id(), characterId, inventoryType)(db)()
			if err != nil {
				return model.ErrorProvider[uint32](err)
			}
			return model.FixedProvider(e.ID)
		}
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

func EquipItemForCharacter(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(freeSlotProvider func(db *gorm.DB) func(uint32) model.Provider[int16]) func(eventProducer producer.Provider) func(characterId uint32) func(source int16) func(destinationProvider equipment.DestinationProvider) {
	return func(db *gorm.DB) func(ctx context.Context) func(freeSlotProvider func(db *gorm.DB) func(uint32) model.Provider[int16]) func(eventProducer producer.Provider) func(characterId uint32) func(source int16) func(destinationProvider equipment.DestinationProvider) {
		return func(ctx context.Context) func(freeSlotProvider func(db *gorm.DB) func(uint32) model.Provider[int16]) func(eventProducer producer.Provider) func(characterId uint32) func(source int16) func(destinationProvider equipment.DestinationProvider) {
			return func(freeSlotProvider func(db *gorm.DB) func(uint32) model.Provider[int16]) func(eventProducer producer.Provider) func(characterId uint32) func(source int16) func(destinationProvider equipment.DestinationProvider) {
				return func(eventProducer producer.Provider) func(characterId uint32) func(source int16) func(destinationProvider equipment.DestinationProvider) {
					return func(characterId uint32) func(source int16) func(destinationProvider equipment.DestinationProvider) {
						characterInventoryMoveProvider := inventoryItemMoveProvider(characterId)
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
									inSlotProvider := equipable.AssetBySlotProvider(tx)(ctx)(characterId)
									slotUpdater := equipable.UpdateSlot(tx)(ctx)

									e, err = equipable.GetBySlot(tx)(ctx)(characterId, source)
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
									resp, _ := moveFromSlotToSlot(l)(inSlotProvider(actualDestination), temporarySlotProvider, slotUpdater, noOpInventoryItemMoveProvider)()
									events = model.MergeSliceProvider(events, model.FixedProvider(resp))

									l.Debugf("Attempting to move item that is being equipped to its final destination.")
									resp, _ = moveFromSlotToSlot(l)(inSlotProvider(source), model.FixedProvider(actualDestination), slotUpdater, characterInventoryMoveProvider(source))()
									events = model.MergeSliceProvider(events, model.FixedProvider(resp))

									l.Debugf("Attempting to move item that is in the temporary position to where the item that was just equipped was.")
									resp, _ = moveFromSlotToSlot(l)(inSlotProvider(temporarySlot()), model.FixedProvider(source), slotUpdater, noOpInventoryItemMoveProvider)()
									events = model.MergeSliceProvider(events, model.FixedProvider(resp))

									l.Debugf("Now verifying other inventory operations that may be necessary.")

									invId, err := GetInventoryIdByType(tx)(ctx)(characterId, TypeValueEquip)()
									if err != nil {
										l.WithError(err).Errorf("Unable to locate inventory [%d] for character [%d].", TypeValueEquip, characterId)
										return err
									}
									nextFreeSlotProvider := freeSlotProvider(tx)(invId)

									if e.ItemId()/10000 == 105 {
										l.Debugf("Item is an overall, we also need to unequip the bottom.")
										resp, err = moveFromSlotToSlot(l)(inSlotProvider(int16(slot2.PositionBottom)), nextFreeSlotProvider, slotUpdater, characterInventoryMoveProvider(int16(slot2.PositionBottom)))()
										if err != nil {
											l.WithError(err).Errorf("Unable to move bottom out of its slot.")
											return err
										}
										events = model.MergeSliceProvider(events, model.FixedProvider(resp))
									}
									if actualDestination == int16(slot2.PositionBottom) {
										l.Debugf("Item is a bottom, need to unequip an overall if its in the top slot.")
										ip := model.Map(IsOverall)(inSlotProvider(int16(slot2.PositionOverall)))
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

func UnequipItemForCharacter(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(freeSlotProvider func(db *gorm.DB) func(uint32) model.Provider[int16]) func(eventProducer producer.Provider) func(characterId uint32) func(oldSlot int16) {
	return func(db *gorm.DB) func(ctx context.Context) func(freeSlotProvider func(db *gorm.DB) func(uint32) model.Provider[int16]) func(eventProducer producer.Provider) func(characterId uint32) func(oldSlot int16) {
		return func(ctx context.Context) func(freeSlotProvider func(db *gorm.DB) func(uint32) model.Provider[int16]) func(eventProducer producer.Provider) func(characterId uint32) func(oldSlot int16) {
			return func(freeSlotProvider func(db *gorm.DB) func(uint32) model.Provider[int16]) func(eventProducer producer.Provider) func(characterId uint32) func(oldSlot int16) {
				return func(eventProducer producer.Provider) func(characterId uint32) func(oldSlot int16) {
					return func(characterId uint32) func(oldSlot int16) {
						return func(oldSlot int16) {
							l.Debugf("Received request to unequip item at [%d] for character [%d].", oldSlot, characterId)
							invLock := GetLockRegistry().GetById(characterId, TypeValueEquip)
							invLock.Lock()
							defer invLock.Unlock()

							var events = model.FixedProvider[[]kafka.Message]([]kafka.Message{})
							txErr := db.Transaction(func(tx *gorm.DB) error {
								inSlotProvider := equipable.AssetBySlotProvider(tx)(ctx)(characterId)
								slotUpdater := equipable.UpdateSlot(tx)(ctx)
								characterInventoryMoveProvider := inventoryItemMoveProvider(characterId)

								invId, err := GetInventoryIdByType(tx)(ctx)(characterId, TypeValueEquip)()
								if err != nil {
									l.WithError(err).Errorf("Unable to locate inventory [%d] for character [%d].", TypeValueEquip, characterId)
									return err
								}

								resp, err := moveFromSlotToSlot(l)(inSlotProvider(oldSlot), freeSlotProvider(tx)(invId), slotUpdater, characterInventoryMoveProvider(oldSlot))()
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
							err := eventProducer(EnvEventInventoryChanged)(events)
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

func DeleteInventory(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(characterId uint32, inventoryType Type, itemIdProvider model.Provider[[]uint32], itemDeleter func(db *gorm.DB) func(ctx context.Context) model.Operator[uint32]) error {
	return func(db *gorm.DB) func(ctx context.Context) func(characterId uint32, inventoryType Type, itemIdProvider model.Provider[[]uint32], itemDeleter func(db *gorm.DB) func(ctx context.Context) model.Operator[uint32]) error {
		return func(ctx context.Context) func(characterId uint32, inventoryType Type, itemIdProvider model.Provider[[]uint32], itemDeleter func(db *gorm.DB) func(ctx context.Context) model.Operator[uint32]) error {
			return func(characterId uint32, inventoryType Type, itemIdProvider model.Provider[[]uint32], itemDeleter func(db *gorm.DB) func(ctx context.Context) model.Operator[uint32]) error {
				invLock := GetLockRegistry().GetById(characterId, inventoryType)
				invLock.Lock()
				defer invLock.Unlock()
				return db.Transaction(func(tx *gorm.DB) error {
					err := model.ForEachSlice[uint32](itemIdProvider, itemDeleter(tx)(ctx))
					if err != nil {
						l.WithError(err).Errorf("Unable to delete items in inventory.")
						return err
					}
					t := tenant.MustFromContext(ctx)
					return deleteByType(tx, t.Id(), characterId, int8(inventoryType))
				})
			}
		}
	}
}

func DeleteEquipableInventory(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(characterId uint32, m EquipableModel) error {
	return func(db *gorm.DB) func(ctx context.Context) func(characterId uint32, m EquipableModel) error {
		return func(ctx context.Context) func(characterId uint32, m EquipableModel) error {
			return func(characterId uint32, m EquipableModel) error {
				idp := model.SliceMap(equipable.ReferenceId)(model.FixedProvider(m.Items()))(model.ParallelMap())
				return DeleteInventory(l)(db)(ctx)(characterId, TypeValueEquip, idp, equipable.DeleteByReferenceId(l))
			}
		}
	}
}

func DeleteItemInventory(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(characterId uint32, m ItemModel) error {
	return func(db *gorm.DB) func(ctx context.Context) func(characterId uint32, m ItemModel) error {
		return func(ctx context.Context) func(characterId uint32, m ItemModel) error {
			return func(characterId uint32, m ItemModel) error {
				idp := model.SliceMap(item.Id)(model.FixedProvider(m.Items()))(model.ParallelMap())
				return DeleteInventory(l)(db)(ctx)(characterId, m.mType, idp, item.DeleteById)
			}
		}
	}
}

type AssetMover func(characterId uint32) func(source int16) func(destination int16) error

func Move(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(eventProducer producer.Provider) func(inventoryType byte) AssetMover {
	return func(db *gorm.DB) func(ctx context.Context) func(eventProducer producer.Provider) func(inventoryType byte) AssetMover {
		return func(ctx context.Context) func(eventProducer producer.Provider) func(inventoryType byte) AssetMover {
			return func(eventProducer producer.Provider) func(inventoryType byte) AssetMover {
				return func(inventoryType byte) AssetMover {
					if inventoryType == 1 {
						return moveEquip(l)(db)(ctx)(eventProducer)
					} else {
						return moveItem(l)(db)(ctx)(eventProducer)(inventoryType)
					}
				}
			}
		}
	}
}

func moveItem(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(eventProducer producer.Provider) func(inventoryType byte) AssetMover {
	return func(db *gorm.DB) func(ctx context.Context) func(eventProducer producer.Provider) func(inventoryType byte) AssetMover {
		return func(ctx context.Context) func(eventProducer producer.Provider) func(inventoryType byte) AssetMover {
			return func(eventProducer producer.Provider) func(inventoryType byte) AssetMover {
				return func(inventoryType byte) AssetMover {
					return func(characterId uint32) func(source int16) func(destination int16) error {
						return func(source int16) func(destination int16) error {
							return func(destination int16) error {
								characterInventoryMoveProvider := inventoryItemMoveProvider(characterId)

								l.Debugf("Received request to move item at [%d] to [%d] for character [%d].", source, destination, characterId)
								invLock := GetLockRegistry().GetById(characterId, Type(inventoryType))
								invLock.Lock()
								defer invLock.Unlock()

								// TODO need to combine quantities if moving to the same item type.

								var events = model.FixedProvider[[]kafka.Message]([]kafka.Message{})
								txErr := db.Transaction(func(tx *gorm.DB) error {
									slotUpdater := item.UpdateSlot(tx)(ctx)

									invId, err := GetInventoryIdByType(tx)(ctx)(characterId, Type(inventoryType))()
									if err != nil {
										l.WithError(err).Errorf("Unable to locate inventory [%d] for character [%d].", inventoryType, characterId)
										return err
									}
									inSlotProvider := item.AssetBySlotProvider(tx)(ctx)(invId)

									l.Debugf("Attempting to move item that is currently occupying the destination to a temporary position.")
									resp, _ := moveFromSlotToSlot(l)(inSlotProvider(destination), temporarySlotProvider, slotUpdater, noOpInventoryItemMoveProvider)()
									events = model.MergeSliceProvider(events, model.FixedProvider(resp))

									l.Debugf("Attempting to move item that is being moved to its final destination.")
									resp, _ = moveFromSlotToSlot(l)(inSlotProvider(source), model.FixedProvider(destination), slotUpdater, characterInventoryMoveProvider(source))()
									events = model.MergeSliceProvider(events, model.FixedProvider(resp))

									l.Debugf("Attempting to move item that is in the temporary position to where the item that was just equipped was.")
									resp, _ = moveFromSlotToSlot(l)(inSlotProvider(temporarySlot()), model.FixedProvider(source), slotUpdater, noOpInventoryItemMoveProvider)()
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
					}
				}
			}
		}
	}
}

func temporarySlot() int16 {
	return int16(math.MinInt16)
}

var temporarySlotProvider = model.FixedProvider(temporarySlot())

func moveEquip(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(eventProducer producer.Provider) AssetMover {
	return func(db *gorm.DB) func(ctx context.Context) func(eventProducer producer.Provider) AssetMover {
		return func(ctx context.Context) func(eventProducer producer.Provider) AssetMover {
			return func(eventProducer producer.Provider) AssetMover {
				return func(characterId uint32) func(source int16) func(destination int16) error {
					return func(source int16) func(destination int16) error {
						return func(destination int16) error {
							l.Debugf("Received request to move item at [%d] to [%d] for character [%d].", source, destination, characterId)
							invLock := GetLockRegistry().GetById(characterId, TypeValueEquip)
							invLock.Lock()
							defer invLock.Unlock()

							var events = model.FixedProvider[[]kafka.Message]([]kafka.Message{})
							txErr := db.Transaction(func(tx *gorm.DB) error {
								inSlotProvider := equipable.AssetBySlotProvider(tx)(ctx)(characterId)
								slotUpdater := equipable.UpdateSlot(tx)(ctx)
								characterInventoryMoveProvider := inventoryItemMoveProvider(characterId)

								l.Debugf("Attempting to move item that is currently occupying the destination to a temporary position.")
								resp, _ := moveFromSlotToSlot(l)(inSlotProvider(destination), temporarySlotProvider, slotUpdater, noOpInventoryItemMoveProvider)()
								events = model.MergeSliceProvider(events, model.FixedProvider(resp))

								l.Debugf("Attempting to move item that is being moved to its final destination.")
								resp, _ = moveFromSlotToSlot(l)(inSlotProvider(source), model.FixedProvider(destination), slotUpdater, characterInventoryMoveProvider(source))()
								events = model.MergeSliceProvider(events, model.FixedProvider(resp))

								l.Debugf("Attempting to move item that is in the temporary position to where the item that was just equipped was.")
								resp, _ = moveFromSlotToSlot(l)(inSlotProvider(temporarySlot()), model.FixedProvider(source), slotUpdater, noOpInventoryItemMoveProvider)()
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
				}
			}
		}
	}
}

type AssetDropper func(characterId uint32) func(source int16) func(quantity int16) error

// Drop drops an asset from the designated inventory.
func Drop(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(inventoryType byte) AssetDropper {
	return func(db *gorm.DB) func(ctx context.Context) func(inventoryType byte) AssetDropper {
		return func(ctx context.Context) func(inventoryType byte) AssetDropper {
			return func(inventoryType byte) AssetDropper {
				if inventoryType == 1 {
					return dropEquip(l)(db)(ctx)
				} else {
					return dropItem(l)(db)(ctx)(inventoryType)
				}
			}
		}
	}
}

func dropItem(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(inventoryType byte) AssetDropper {
	return func(db *gorm.DB) func(ctx context.Context) func(inventoryType byte) AssetDropper {
		return func(ctx context.Context) func(inventoryType byte) AssetDropper {
			return func(inventoryType byte) AssetDropper {
				return func(characterId uint32) func(source int16) func(quantity int16) error {
					return func(source int16) func(quantity int16) error {
						return func(quantity int16) error {
							l.Debugf("Received request to drop item at [%d] for character [%d].", source, characterId)
							invLock := GetLockRegistry().GetById(characterId, Type(inventoryType))
							invLock.Lock()
							defer invLock.Unlock()

							var events = model.FixedProvider[[]kafka.Message]([]kafka.Message{})
							txErr := db.Transaction(func(tx *gorm.DB) error {
								invId, err := GetInventoryIdByType(tx)(ctx)(characterId, Type(inventoryType))()
								if err != nil {
									l.WithError(err).Errorf("Unable to locate inventory [%d] for character [%d].", inventoryType, characterId)
									return err
								}

								i, err := item.GetBySlot(tx)(ctx)(invId, source)
								if err != nil {
									l.WithError(err).Errorf("Unable to retrieve item in slot [%d].", source)
									return err
								}

								initialQuantity := i.Quantity()

								if initialQuantity <= uint32(quantity) {
									err = item.DeleteById(tx)(ctx)(i.Id())
									if err != nil {
										l.WithError(err).Errorf("Unable to drop item in slot [%d].", source)
										return err
									}
									events = model.MergeSliceProvider(events, inventoryItemRemoveProvider(characterId, i.ItemId(), i.Slot()))
									return nil
								}

								newQuantity := initialQuantity - uint32(quantity)
								err = item.UpdateQuantity(tx)(ctx)(i.Id(), newQuantity)
								if err != nil {
									l.WithError(err).Errorf("Unable to drop [%d] item in slot [%d].", quantity, source)
									return err
								}
								events = model.MergeSliceProvider(events, inventoryItemUpdateProvider(characterId)(i.ItemId())(newQuantity, i.Slot()))
								return nil
							})
							if txErr != nil {
								l.WithError(txErr).Errorf("Unable to complete dropping item for character [%d].", characterId)
								return txErr
							}
							err := producer.ProviderImpl(l)(ctx)(EnvEventInventoryChanged)(events)
							if err != nil {
								l.WithError(err).Errorf("Unable to convey inventory modifications to character [%d].", characterId)
							}
							return err
						}
					}
				}
			}
		}
	}
}

func dropEquip(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) AssetDropper {
	return func(db *gorm.DB) func(ctx context.Context) AssetDropper {
		return func(ctx context.Context) AssetDropper {
			return func(characterId uint32) func(source int16) func(quantity int16) error {
				return func(source int16) func(quantity int16) error {
					return func(quantity int16) error {
						l.Debugf("Received request to drop item at [%d] for character [%d].", source, characterId)
						invLock := GetLockRegistry().GetById(characterId, TypeValueEquip)
						invLock.Lock()
						defer invLock.Unlock()

						var events = model.FixedProvider[[]kafka.Message]([]kafka.Message{})
						txErr := db.Transaction(func(tx *gorm.DB) error {
							e, err := equipable.GetBySlot(tx)(ctx)(characterId, source)
							if err != nil {
								l.WithError(err).Errorf("Unable to retrieve equipment in slot [%d].", source)
								return err
							}
							err = equipable.DropByReferenceId(l)(tx)(ctx)(e.ReferenceId())
							if err != nil {
								l.WithError(err).Errorf("Unable to drop equipment in slot [%d].", source)
								return err
							}
							events = model.MergeSliceProvider(events, inventoryItemRemoveProvider(characterId, e.ItemId(), e.Slot()))
							return nil
						})
						if txErr != nil {
							l.WithError(txErr).Errorf("Unable to complete dropping item for character [%d].", characterId)
							return txErr
						}
						err := producer.ProviderImpl(l)(ctx)(EnvEventInventoryChanged)(events)
						if err != nil {
							l.WithError(err).Errorf("Unable to convey inventory modifications to character [%d].", characterId)
						}
						return err
					}
				}
			}
		}
	}
}
