package inventory

import (
	"atlas-character/asset"
	"atlas-character/data/consumable"
	"atlas-character/drop"
	"atlas-character/equipable"
	statistics2 "atlas-character/equipable/statistics"
	"atlas-character/equipment"
	slot2 "atlas-character/equipment/slot"
	"atlas-character/inventory/item"
	"atlas-character/kafka/producer"
	"context"
	"errors"
	item2 "github.com/Chronicle20/atlas-constants/item"
	"math"
	"time"

	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func ByCharacterIdProvider(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(characterId uint32) model.Provider[Model] {
	return func(db *gorm.DB) func(ctx context.Context) func(characterId uint32) model.Provider[Model] {
		return func(ctx context.Context) func(characterId uint32) model.Provider[Model] {
			folder := foldInventory(l)(db)(ctx)
			return func(characterId uint32) model.Provider[Model] {
				t := tenant.MustFromContext(ctx)
				return model.Fold(getByCharacter(t.Id(), characterId)(db), supplier, folder)
			}
		}
	}
}

func supplier() (Model, error) {
	return Model{
		equipable: EquipableModel{},
		useable:   ItemModel{mType: inventory.TypeValueUse},
		setup:     ItemModel{mType: inventory.TypeValueSetup},
		etc:       ItemModel{mType: inventory.TypeValueETC},
		cash:      ItemModel{mType: inventory.TypeValueCash},
	}, nil
}

func EquipableFolder(m EquipableModel, em equipable.Model) (EquipableModel, error) {
	if em.Slot() <= 0 {
		return m, nil
	}
	m.items = append(m.items, em)
	return m, nil
}

func FoldProperty[M any, N any](setter func(sm N) M) model.Transformer[N, M] {
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
				switch inventory.Type(ent.InventoryType) {
				case inventory.TypeValueEquip:
					ep := equipable.InInventoryProvider(l)(db)(ctx)(ent.ID)
					return model.Map(FoldProperty(ref.SetEquipable))(model.Fold(ep, NewEquipableModel(ent.ID, ent.Capacity), EquipableFolder))()
				case inventory.TypeValueUse:
					ip := item.ByInventoryProvider(db)(ctx)(ent.ID)
					return model.Map(FoldProperty(ref.SetUseable))(model.Fold(ip, NewItemModel(ent.ID, inventory.TypeValueUse, ent.Capacity), ItemFolder))()
				case inventory.TypeValueSetup:
					ip := item.ByInventoryProvider(db)(ctx)(ent.ID)
					return model.Map(FoldProperty(ref.SetSetup))(model.Fold(ip, NewItemModel(ent.ID, inventory.TypeValueSetup, ent.Capacity), ItemFolder))()
				case inventory.TypeValueETC:
					ip := item.ByInventoryProvider(db)(ctx)(ent.ID)
					return model.Map(FoldProperty(ref.SetEtc))(model.Fold(ip, NewItemModel(ent.ID, inventory.TypeValueETC, ent.Capacity), ItemFolder))()
				case inventory.TypeValueCash:
					ip := item.ByInventoryProvider(db)(ctx)(ent.ID)
					return model.Map(FoldProperty(ref.SetCash))(model.Fold(ip, NewItemModel(ent.ID, inventory.TypeValueCash, ent.Capacity), ItemFolder))()
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
					for _, t := range TypeValues {
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

func CreateItem(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(eventProducer producer.Provider) func(characterId uint32, inventoryType inventory.Type, itemId uint32, quantity uint32) error {
	return func(db *gorm.DB) func(ctx context.Context) func(eventProducer producer.Provider) func(characterId uint32, inventoryType inventory.Type, itemId uint32, quantity uint32) error {
		return func(ctx context.Context) func(eventProducer producer.Provider) func(characterId uint32, inventoryType inventory.Type, itemId uint32, quantity uint32) error {
			return func(eventProducer producer.Provider) func(characterId uint32, inventoryType inventory.Type, itemId uint32, quantity uint32) error {
				return func(characterId uint32, inventoryType inventory.Type, itemId uint32, quantity uint32) error {
					expectedInventoryType, ok := inventory.TypeFromItemId(itemId)
					if !ok || expectedInventoryType != inventoryType {
						l.Errorf("Provided inventoryType [%d] does not match expected one [%d] for itemId [%d].", inventoryType, expectedInventoryType, itemId)
						return errors.New("invalid inventory type")
					}

					if quantity == 0 {
						quantity = 1
					}

					l.Debugf("Creating [%d] item [%d] for character [%d] in inventory [%d].", quantity, itemId, characterId, inventoryType)
					invLock := GetLockRegistry().GetById(characterId, inventoryType)
					invLock.Lock()
					defer invLock.Unlock()

					var events = model.FixedProvider([]kafka.Message{})
					err := db.Transaction(func(tx *gorm.DB) error {
						inv, err := GetInventoryByType(l)(tx)(ctx)(characterId, inventoryType)()
						if err != nil {
							l.WithError(err).Errorf("Unable to locate inventory [%d] for character [%d].", inventoryType, characterId)
							return err
						}

						iap := inventoryItemAddProvider(characterId)(inventoryType)(itemId)
						iup := inventoryItemQuantityUpdateProvider(characterId)(inventoryType)(itemId)
						var eap model.Provider[[]asset.Asset]
						var smp SlotMaxProvider
						var nac asset.Creator
						var aqu asset.QuantityUpdater

						if inventoryType == inventory.TypeValueEquip {
							eap = asset.NoOpSliceProvider
							smp = ItemSlotMaxProvider(l)(ctx)(inventoryType, itemId)
							nac = equipable.CreateItem(l)(tx)(ctx)(statistics2.Create(l)(ctx))(characterId)(inv.Id(), int8(inventoryType), inv.Capacity())(itemId)
							aqu = asset.NoOpQuantityUpdater
						} else {
							eap = item.AssetByItemIdProvider(tx)(ctx)(inv.Id())(itemId)
							smp = ItemSlotMaxProvider(l)(ctx)(inventoryType, itemId)
							nac = item.CreateItem(tx)(ctx)(characterId)(inv.Id(), int8(inventoryType), inv.Capacity())(itemId)
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

func GetInventoryIdByType(db *gorm.DB) func(ctx context.Context) func(characterId uint32, inventoryType inventory.Type) model.Provider[uint32] {
	return func(ctx context.Context) func(characterId uint32, inventoryType inventory.Type) model.Provider[uint32] {
		t := tenant.MustFromContext(ctx)
		return func(characterId uint32, inventoryType inventory.Type) model.Provider[uint32] {
			e, err := get(t.Id(), characterId, inventoryType)(db)()
			if err != nil {
				return model.ErrorProvider[uint32](err)
			}
			return model.FixedProvider(e.ID)
		}
	}
}

func GetInventoryByType(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(characterId uint32, inventoryType inventory.Type) model.Provider[ItemHolder] {
	return func(db *gorm.DB) func(ctx context.Context) func(characterId uint32, inventoryType inventory.Type) model.Provider[ItemHolder] {
		return func(ctx context.Context) func(characterId uint32, inventoryType inventory.Type) model.Provider[ItemHolder] {
			t := tenant.MustFromContext(ctx)
			return func(characterId uint32, inventoryType inventory.Type) model.Provider[ItemHolder] {
				e, err := get(t.Id(), characterId, inventoryType)(db)()
				if err != nil {
					return model.ErrorProvider[ItemHolder](err)
				}
				var ei ItemHolder
				if inventoryType == inventory.TypeValueEquip {
					ei, err = model.Fold(equipable.InInventoryProvider(l)(db)(ctx)(e.ID), NewEquipableModel(e.ID, e.Capacity), EquipableFolder)()
				} else {
					ei, err = model.Fold(item.ByInventoryProvider(db)(ctx)(e.ID), NewItemModel(e.ID, inventory.TypeValueCash, e.Capacity), ItemFolder)()
				}
				if err != nil {
					return model.ErrorProvider[ItemHolder](err)
				}
				return model.FixedProvider(ei)
			}
		}
	}
}

type SlotMaxProvider model.Provider[uint32]

func ItemSlotMaxProvider(l logrus.FieldLogger) func(ctx context.Context) func(inventoryType inventory.Type, itemId uint32) SlotMaxProvider {
	return func(ctx context.Context) func(inventoryType inventory.Type, itemId uint32) SlotMaxProvider {
		return func(inventoryType inventory.Type, itemId uint32) SlotMaxProvider {
			defaultMax := uint32(100)
			if inventoryType == inventory.TypeValueEquip {
				return SlotMaxProvider(model.FixedProvider[uint32](1))
			}
			if inventoryType == inventory.TypeValueUse {
				cd, err := consumable.GetById(l)(ctx)(itemId)
				if err != nil {
					return SlotMaxProvider(model.FixedProvider[uint32](defaultMax))
				}
				if cd.SlotMax() != 0 {
					return SlotMaxProvider(model.FixedProvider[uint32](cd.SlotMax()))
				}
				return SlotMaxProvider(model.FixedProvider[uint32](defaultMax))
			}
			// TODO query Set-Up, ETC, and Cash values.
			return SlotMaxProvider(model.FixedProvider[uint32](defaultMax))
		}
	}
}

func CreateAsset(l logrus.FieldLogger) func(existingAssetProvider model.Provider[[]asset.Asset], slotMaxProvider SlotMaxProvider, newAssetCreator asset.Creator, assetQuantityUpdater asset.QuantityUpdater, addEventProvider ItemAddProvider, updateEventProvider ItemUpdateProvider, quantity uint32) model.Provider[[]kafka.Message] {
	return func(existingAssetProvider model.Provider[[]asset.Asset], slotMaxProvider SlotMaxProvider, newAssetCreator asset.Creator, assetQuantityUpdater asset.QuantityUpdater, addEventProvider ItemAddProvider, updateEventProvider ItemUpdateProvider, quantity uint32) model.Provider[[]kafka.Message] {
		runningQuantity := quantity
		slotMax, err := slotMaxProvider()
		if err != nil {
			return model.ErrorProvider[[]kafka.Message](err)
		}

		var result = model.FixedProvider([]kafka.Message{})

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
						characterInventoryMoveProvider := inventoryItemMoveProvider(characterId)(inventory.TypeValueEquip)
						return func(source int16) func(destinationProvider equipment.DestinationProvider) {
							return func(destinationProvider equipment.DestinationProvider) {
								var e equipable.Model

								l.Debugf("Received request to equip item at [%d] for character [%d].", source, characterId)
								invLock := GetLockRegistry().GetById(characterId, inventory.TypeValueEquip)
								invLock.Lock()
								defer invLock.Unlock()

								var events = model.FixedProvider([]kafka.Message{})

								txErr := db.Transaction(func(tx *gorm.DB) error {
									inSlotProvider := equipable.AssetBySlotProvider(tx)(ctx)(characterId)
									slotUpdater := equipable.UpdateSlot(tx)(ctx)

									var err error
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

									invId, err := GetInventoryIdByType(tx)(ctx)(characterId, inventory.TypeValueEquip)()
									if err != nil {
										l.WithError(err).Errorf("Unable to locate inventory [%d] for character [%d].", inventory.TypeValueEquip, characterId)
										return err
									}
									nextFreeSlotProvider := freeSlotProvider(tx)(invId)

									ts, err := slot2.GetSlotByType("top")
									if err != nil {
										l.WithError(err).Errorf("Unable to find top slot")
										return err
									}
									ps, err := slot2.GetSlotByType("pants")
									if err != nil {
										l.WithError(err).Errorf("Unable to find pants slot")
										return err
									}

									if item2.GetClassification(item2.Id(e.ItemId())) == item2.ClassificationOverall {
										l.Debugf("Item is an overall, we also need to unequip the bottom.")
										resp, err = moveFromSlotToSlot(l)(inSlotProvider(int16(ps.Position)), nextFreeSlotProvider, slotUpdater, characterInventoryMoveProvider(int16(ps.Position)))()
										if err != nil && !errors.Is(err, ErrItemNotFound) {
											l.WithError(err).Errorf("Unable to move bottom out of its slot.")
											return err
										}
										events = model.MergeSliceProvider(events, model.FixedProvider(resp))
									}
									if actualDestination == int16(ps.Position) {
										l.Debugf("Item is a bottom, need to unequip an overall if its in the top slot.")
										ip := model.Map(IsOverall)(inSlotProvider(int16(ts.Position)))
										resp, err = moveFromSlotToSlot(l)(ip, nextFreeSlotProvider, slotUpdater, characterInventoryMoveProvider(int16(ts.Position)))()
										if err != nil && !errors.Is(err, ErrNotOverall) {
											l.WithError(err).Errorf("Unable to move overall out of its slot.")
											return err
										}
										events = model.MergeSliceProvider(events, model.FixedProvider(resp))
									}
									return nil
								})
								if txErr != nil {
									l.WithError(txErr).Errorf("Unable to complete the equipment of item [%d] for character [%d].", e.Id(), characterId)
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
}

var ErrNotOverall = errors.New("not an overall")

func IsOverall(m asset.Asset) (asset.Asset, error) {
	if item2.GetClassification(item2.Id(m.ItemId())) == item2.ClassificationOverall {
		return m, nil
	}
	return nil, ErrNotOverall
}

var ErrItemNotFound = errors.New("item not found")

func moveFromSlotToSlot(l logrus.FieldLogger) func(modelProvider model.Provider[asset.Asset], newSlotProvider model.Provider[int16], slotUpdater func(id uint32, slot int16) error, moveEventProvider func(itemId uint32) func(slot int16) model.Provider[[]kafka.Message]) model.Provider[[]kafka.Message] {
	return func(modelProvider model.Provider[asset.Asset], newSlotProvider model.Provider[int16], slotUpdater func(id uint32, slot int16) error, moveEventProvider func(itemId uint32) func(slot int16) model.Provider[[]kafka.Message]) model.Provider[[]kafka.Message] {
		m, err := modelProvider()
		if err != nil {
			return model.ErrorProvider[[]kafka.Message](ErrItemNotFound)
		}
		if m.Id() == 0 {
			return model.ErrorProvider[[]kafka.Message](ErrItemNotFound)
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
							invLock := GetLockRegistry().GetById(characterId, inventory.TypeValueEquip)
							invLock.Lock()
							defer invLock.Unlock()

							var events = model.FixedProvider([]kafka.Message{})
							txErr := db.Transaction(func(tx *gorm.DB) error {
								inSlotProvider := equipable.AssetBySlotProvider(tx)(ctx)(characterId)
								slotUpdater := equipable.UpdateSlot(tx)(ctx)
								characterInventoryMoveProvider := inventoryItemMoveProvider(characterId)(inventory.TypeValueEquip)

								invId, err := GetInventoryIdByType(tx)(ctx)(characterId, inventory.TypeValueEquip)()
								if err != nil {
									l.WithError(err).Errorf("Unable to locate inventory [%d] for character [%d].", inventory.TypeValueEquip, characterId)
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

func DeleteInventory(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(characterId uint32, inventoryType inventory.Type, itemIdProvider model.Provider[[]uint32], itemDeleter func(db *gorm.DB) func(ctx context.Context) model.Operator[uint32]) error {
	return func(db *gorm.DB) func(ctx context.Context) func(characterId uint32, inventoryType inventory.Type, itemIdProvider model.Provider[[]uint32], itemDeleter func(db *gorm.DB) func(ctx context.Context) model.Operator[uint32]) error {
		return func(ctx context.Context) func(characterId uint32, inventoryType inventory.Type, itemIdProvider model.Provider[[]uint32], itemDeleter func(db *gorm.DB) func(ctx context.Context) model.Operator[uint32]) error {
			return func(characterId uint32, inventoryType inventory.Type, itemIdProvider model.Provider[[]uint32], itemDeleter func(db *gorm.DB) func(ctx context.Context) model.Operator[uint32]) error {
				invLock := GetLockRegistry().GetById(characterId, inventoryType)
				invLock.Lock()
				defer invLock.Unlock()
				return db.Transaction(func(tx *gorm.DB) error {
					err := model.ForEachSlice(itemIdProvider, itemDeleter(tx)(ctx))
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
				return DeleteInventory(l)(db)(ctx)(characterId, inventory.TypeValueEquip, idp, equipable.DeleteByReferenceId(l))
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

func Move(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(eventProducer producer.Provider) func(inventoryType inventory.Type) AssetMover {
	return func(db *gorm.DB) func(ctx context.Context) func(eventProducer producer.Provider) func(inventoryType inventory.Type) AssetMover {
		return func(ctx context.Context) func(eventProducer producer.Provider) func(inventoryType inventory.Type) AssetMover {
			return func(eventProducer producer.Provider) func(inventoryType inventory.Type) AssetMover {
				return func(inventoryType inventory.Type) AssetMover {
					if inventoryType == inventory.TypeValueEquip {
						return moveEquip(l)(db)(ctx)(eventProducer)
					} else {
						return moveItem(l)(db)(ctx)(eventProducer)(inventoryType)
					}
				}
			}
		}
	}
}

func moveItem(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(eventProducer producer.Provider) func(inventoryType inventory.Type) AssetMover {
	return func(db *gorm.DB) func(ctx context.Context) func(eventProducer producer.Provider) func(inventoryType inventory.Type) AssetMover {
		return func(ctx context.Context) func(eventProducer producer.Provider) func(inventoryType inventory.Type) AssetMover {
			t := tenant.MustFromContext(ctx)
			return func(eventProducer producer.Provider) func(inventoryType inventory.Type) AssetMover {
				return func(inventoryType inventory.Type) AssetMover {
					return func(characterId uint32) func(source int16) func(destination int16) error {
						return func(source int16) func(destination int16) error {
							return func(destination int16) error {
								l.Debugf("Received request to move item at [%d] to [%d] for character [%d].", source, destination, characterId)
								invLock := GetLockRegistry().GetById(characterId, inventoryType)
								invLock.Lock()
								defer invLock.Unlock()

								// TODO need to combine quantities if moving to the same item type.

								var events = model.FixedProvider([]kafka.Message{})
								txErr := db.Transaction(func(tx *gorm.DB) error {
									slotUpdater := item.UpdateSlot(tx)(ctx)

									invId, err := GetInventoryIdByType(tx)(ctx)(characterId, inventoryType)()
									if err != nil {
										l.WithError(err).Errorf("Unable to locate inventory [%d] for character [%d].", inventoryType, characterId)
										return err
									}
									inSlotProvider := item.AssetBySlotProvider(tx)(ctx)(invId)

									l.Debugf("Attempting to move item that is currently occupying the destination to a temporary position.")
									resp, _ := moveFromSlotToSlot(l)(inSlotProvider(destination), temporarySlotProvider, slotUpdater, noOpInventoryItemMoveProvider)()
									events = model.MergeSliceProvider(events, model.FixedProvider(resp))

									l.Debugf("Attempting to move item that is being moved to its final destination.")
									resp, _ = moveFromSlotToSlot(l)(inSlotProvider(source), model.FixedProvider(destination), slotUpdater, inventoryItemMoveProvider(characterId)(inventoryType)(source))()
									events = model.MergeSliceProvider(events, model.FixedProvider(resp))

									l.Debugf("Attempting to move item that is in the temporary position to where the item that was just equipped was.")
									resp, _ = moveFromSlotToSlot(l)(inSlotProvider(temporarySlot()), model.FixedProvider(source), slotUpdater, noOpInventoryItemMoveProvider)()
									events = model.MergeSliceProvider(events, model.FixedProvider(resp))

									GetReservationRegistry().SwapReservation(t, characterId, inventoryType, source, destination)
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
							invLock := GetLockRegistry().GetById(characterId, inventory.TypeValueEquip)
							invLock.Lock()
							defer invLock.Unlock()

							var events = model.FixedProvider([]kafka.Message{})
							txErr := db.Transaction(func(tx *gorm.DB) error {
								inSlotProvider := equipable.AssetBySlotProvider(tx)(ctx)(characterId)
								slotUpdater := equipable.UpdateSlot(tx)(ctx)
								characterInventoryMoveProvider := inventoryItemMoveProvider(characterId)(inventory.TypeValueEquip)

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

type AssetDropper func(worldId byte, channelId byte, mapId uint32, characterId uint32, x int16, y int16, source int16, quantity int16) error

// Drop drops an asset from the designated inventory.
func Drop(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(inventoryType inventory.Type) AssetDropper {
	return func(db *gorm.DB) func(ctx context.Context) func(inventoryType inventory.Type) AssetDropper {
		return func(ctx context.Context) func(inventoryType inventory.Type) AssetDropper {
			return func(inventoryType inventory.Type) AssetDropper {
				if inventoryType == inventory.TypeValueEquip {
					return dropEquip(l)(db)(ctx)
				} else {
					return dropItem(l)(db)(ctx)(inventoryType)
				}
			}
		}
	}
}

func dropItem(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(inventoryType inventory.Type) AssetDropper {
	return func(db *gorm.DB) func(ctx context.Context) func(inventoryType inventory.Type) AssetDropper {
		return func(ctx context.Context) func(inventoryType inventory.Type) AssetDropper {
			t := tenant.MustFromContext(ctx)
			return func(inventoryType inventory.Type) AssetDropper {
				return func(worldId byte, channelId byte, mapId uint32, characterId uint32, x int16, y int16, source int16, quantity int16) error {
					l.Debugf("Received request to drop item at [%d] for character [%d].", source, characterId)
					invLock := GetLockRegistry().GetById(characterId, inventoryType)
					invLock.Lock()
					defer invLock.Unlock()

					var i item.Model

					var events = model.FixedProvider([]kafka.Message{})
					txErr := db.Transaction(func(tx *gorm.DB) error {
						invId, err := GetInventoryIdByType(tx)(ctx)(characterId, inventoryType)()
						if err != nil {
							l.WithError(err).Errorf("Unable to locate inventory [%d] for character [%d].", inventoryType, characterId)
							return err
						}

						i, err = item.GetBySlot(tx)(ctx)(invId, source)
						if err != nil {
							l.WithError(err).Errorf("Unable to retrieve item in slot [%d].", source)
							return err
						}

						reservedQty := GetReservationRegistry().GetReservedQuantity(t, characterId, inventoryType, source)

						initialQuantity := i.Quantity() - reservedQty

						if initialQuantity <= uint32(quantity) {
							err = item.DeleteById(tx)(ctx)(i.Id())
							if err != nil {
								l.WithError(err).Errorf("Unable to drop item in slot [%d].", source)
								return err
							}
							events = model.MergeSliceProvider(events, inventoryItemRemoveProvider(characterId)(inventoryType)(i.ItemId(), i.Slot()))
							return nil
						}

						newQuantity := initialQuantity - uint32(quantity)
						err = item.UpdateQuantity(tx)(ctx)(i.Id(), newQuantity)
						if err != nil {
							l.WithError(err).Errorf("Unable to drop [%d] item in slot [%d].", quantity, source)
							return err
						}
						events = model.MergeSliceProvider(events, inventoryItemQuantityUpdateProvider(characterId)(inventoryType)(i.ItemId())(newQuantity, i.Slot()))
						return nil
					})
					if txErr != nil {
						l.WithError(txErr).Errorf("Unable to complete dropping item for character [%d].", characterId)
						return txErr
					}

					// TODO determine appropriate drop type and mod
					_ = drop.CreateForItem(l)(ctx)(worldId, channelId, mapId, i.ItemId(), uint32(math.Abs(float64(quantity))), 2, x, y, characterId)

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

func dropEquip(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) AssetDropper {
	return func(db *gorm.DB) func(ctx context.Context) AssetDropper {
		return func(ctx context.Context) AssetDropper {
			return func(worldId byte, channelId byte, mapId uint32, characterId uint32, x int16, y int16, source int16, quantity int16) error {
				l.Debugf("Received request to drop item at [%d] for character [%d].", source, characterId)
				invLock := GetLockRegistry().GetById(characterId, inventory.TypeValueEquip)
				invLock.Lock()
				defer invLock.Unlock()

				var e equipable.Model

				var events = model.FixedProvider([]kafka.Message{})
				txErr := db.Transaction(func(tx *gorm.DB) error {
					var err error
					e, err = equipable.GetBySlot(tx)(ctx)(characterId, source)
					if err != nil {
						l.WithError(err).Errorf("Unable to retrieve equipment in slot [%d].", source)
						return err
					}
					err = equipable.DropByReferenceId(l)(tx)(ctx)(e.ReferenceId())
					if err != nil {
						l.WithError(err).Errorf("Unable to drop equipment in slot [%d].", source)
						return err
					}
					events = model.MergeSliceProvider(events, inventoryItemRemoveProvider(characterId)(inventory.TypeValueEquip)(e.ItemId(), e.Slot()))
					return nil
				})
				if txErr != nil {
					l.WithError(txErr).Errorf("Unable to complete dropping item for character [%d].", characterId)
					return txErr
				}

				// TODO determine appropriate drop type and mod
				_ = drop.CreateForEquipment(l)(ctx)(worldId, channelId, mapId, e.ItemId(), e.ReferenceId(), 2, x, y, characterId)

				err := producer.ProviderImpl(l)(ctx)(EnvEventInventoryChanged)(events)
				if err != nil {
					l.WithError(err).Errorf("Unable to convey inventory modifications to character [%d].", characterId)
				}
				return err
			}
		}
	}
}

func AttemptItemPickUp(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, dropId uint32, itemId uint32, quantity uint32) error {
	return func(db *gorm.DB) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, dropId uint32, itemId uint32, quantity uint32) error {
		return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, dropId uint32, itemId uint32, quantity uint32) error {
			return func(worldId byte, channelId byte, mapId uint32, characterId uint32, dropId uint32, itemId uint32, quantity uint32) error {
				inventoryType, ok := inventory.TypeFromItemId(itemId)
				if !ok {
					return errors.New("invalid itemId")
				}
				if quantity == 0 {
					quantity = 1
				}

				l.Debugf("Creating [%d] item [%d] for character [%d] in inventory [%d].", quantity, itemId, characterId, inventoryType)
				invLock := GetLockRegistry().GetById(characterId, inventoryType)
				invLock.Lock()
				defer invLock.Unlock()

				var events = model.FixedProvider([]kafka.Message{})
				txErr := db.Transaction(func(tx *gorm.DB) error {
					inv, err := GetInventoryByType(l)(tx)(ctx)(characterId, inventoryType)()
					if err != nil {
						l.WithError(err).Errorf("Unable to locate inventory [%d] for character [%d].", inventoryType, characterId)
						return err
					}

					iap := inventoryItemAddProvider(characterId)(inventoryType)(itemId)
					iup := inventoryItemQuantityUpdateProvider(characterId)(inventoryType)(itemId)
					eap := item.AssetByItemIdProvider(tx)(ctx)(inv.Id())(itemId)
					smp := ItemSlotMaxProvider(l)(ctx)(inventoryType, itemId)
					nac := item.CreateItem(tx)(ctx)(characterId)(inv.Id(), int8(inventoryType), inv.Capacity())(itemId)
					aqu := item.UpdateQuantity(tx)(ctx)

					res, err := CreateAsset(l)(eap, smp, nac, aqu, iap, iup, quantity)()
					if err != nil {
						l.WithError(err).Errorf("Unable to create [%d] item [%d] for character [%d].", quantity, itemId, characterId)
						return err
					}
					events = model.MergeSliceProvider(events, model.FixedProvider(res))
					return err
				})
				if txErr != nil {
					_ = drop.CancelReservation(l)(ctx)(worldId, channelId, mapId, dropId, characterId)
					return txErr
				}
				_ = producer.ProviderImpl(l)(ctx)(EnvEventInventoryChanged)(events)
				_ = drop.RequestPickUp(l)(ctx)(worldId, channelId, mapId, dropId, characterId)
				return nil
			}
		}
	}
}

func AttemptEquipmentPickUp(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, dropId uint32, itemId uint32, equipmentId uint32) error {
	return func(db *gorm.DB) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, dropId uint32, itemId uint32, equipmentId uint32) error {
		return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, dropId uint32, itemId uint32, equipmentId uint32) error {
			return func(worldId byte, channelId byte, mapId uint32, characterId uint32, dropId uint32, itemId uint32, equipmentId uint32) error {
				inventoryType, ok := inventory.TypeFromItemId(itemId)
				if !ok {
					return errors.New("invalid inventory item")
				}

				if inventoryType != inventory.TypeValueEquip {
					l.Errorf("Provided inventoryType [%d] does not match expected one [%d] for itemId [%d].", inventoryType, 1, itemId)
					return errors.New("invalid inventory type")
				}

				l.Debugf("Gaining [%d] item [%d] for character [%d] in inventory [%d].", 1, itemId, characterId, inventoryType)
				invLock := GetLockRegistry().GetById(characterId, inventoryType)
				invLock.Lock()
				defer invLock.Unlock()

				var events = model.FixedProvider([]kafka.Message{})
				txErr := db.Transaction(func(tx *gorm.DB) error {
					inv, err := GetInventoryByType(l)(tx)(ctx)(characterId, inventoryType)()
					if err != nil {
						l.WithError(err).Errorf("Unable to locate inventory [%d] for character [%d].", inventoryType, characterId)
						return err
					}

					iap := inventoryItemAddProvider(characterId)(inventoryType)(itemId)
					iup := inventoryItemQuantityUpdateProvider(characterId)(inventoryType)(itemId)
					eap := asset.NoOpSliceProvider
					smp := ItemSlotMaxProvider(l)(ctx)(inventoryType, itemId)
					escf := statistics2.Existing(l)(ctx)(equipmentId)
					nac := equipable.CreateItem(l)(tx)(ctx)(escf)(characterId)(inv.Id(), int8(inventoryType), inv.Capacity())(itemId)
					aqu := asset.NoOpQuantityUpdater

					res, err := CreateAsset(l)(eap, smp, nac, aqu, iap, iup, 1)()
					if err != nil {
						l.WithError(err).Errorf("Unable to create [%d] equipable [%d] for character [%d].", 1, itemId, characterId)
						return err
					}
					events = model.MergeSliceProvider(events, model.FixedProvider(res))
					return err
				})
				if txErr != nil {
					_ = drop.CancelReservation(l)(ctx)(worldId, channelId, mapId, dropId, characterId)
					return txErr
				}
				_ = producer.ProviderImpl(l)(ctx)(EnvEventInventoryChanged)(events)
				_ = drop.RequestPickUp(l)(ctx)(worldId, channelId, mapId, dropId, characterId)
				return nil
			}
		}
	}
}

type Reserve struct {
	Slot     int16
	ItemId   uint32
	Quantity int16
}

func RequestReserve(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, inventoryType inventory.Type, reserves []Reserve, transactionId uuid.UUID) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, inventoryType inventory.Type, reserves []Reserve, transactionId uuid.UUID) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32, inventoryType inventory.Type, reserves []Reserve, transactionId uuid.UUID) error {
			return func(characterId uint32, inventoryType inventory.Type, reserves []Reserve, transactionId uuid.UUID) error {
				var events = model.FixedProvider([]kafka.Message{})
				txErr := db.Transaction(func(tx *gorm.DB) error {
					invLock := GetLockRegistry().GetById(characterId, inventoryType)
					invLock.Lock()
					defer invLock.Unlock()

					invId, err := GetInventoryIdByType(tx)(ctx)(characterId, inventoryType)()
					if err != nil {
						l.WithError(err).Errorf("Unable to locate inventory [%d] for character [%d].", inventoryType, characterId)
						return err
					}

					for _, res := range reserves {
						i, err := item.GetBySlot(tx)(ctx)(invId, res.Slot)
						if err != nil {
							return err
						}
						if i.ItemId() != res.ItemId {
							return errors.New("item id does not match")
						}
						reservedQuantity := GetReservationRegistry().GetReservedQuantity(t, characterId, inventoryType, res.Slot)

						if i.Quantity()-reservedQuantity < uint32(res.Quantity) {
							return errors.New("not enough available quantity")
						}
						_, err = GetReservationRegistry().AddReservation(t, transactionId, characterId, inventoryType, res.Slot, res.ItemId, uint32(res.Quantity), time.Second*time.Duration(30))
						if err != nil {
							return err
						}
						events = model.MergeSliceProvider(events, inventoryItemReserveProvider(characterId)(inventoryType)(res.ItemId)(uint32(res.Quantity), res.Slot, transactionId))
					}
					return nil
				})
				if txErr != nil {
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

func ConsumeItem(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, inventoryType inventory.Type, transactionId uuid.UUID, slot int16) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, inventoryType inventory.Type, transactionId uuid.UUID, slot int16) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32, inventoryType inventory.Type, transactionId uuid.UUID, slot int16) error {
			return func(characterId uint32, inventoryType inventory.Type, transactionId uuid.UUID, slot int16) error {
				res, err := GetReservationRegistry().RemoveReservation(t, transactionId, characterId, inventoryType, slot)
				if err != nil {
					return nil
				}

				l.Debugf("Received request to consume item at [%d] for character [%d].", slot, characterId)
				invLock := GetLockRegistry().GetById(characterId, inventoryType)
				invLock.Lock()
				defer invLock.Unlock()

				var i item.Model

				var events = model.FixedProvider([]kafka.Message{})

				txErr := db.Transaction(func(tx *gorm.DB) error {
					invId, err := GetInventoryIdByType(tx)(ctx)(characterId, inventoryType)()
					if err != nil {
						l.WithError(err).Errorf("Unable to locate inventory [%d] for character [%d].", inventoryType, characterId)
						return err
					}

					i, err = item.GetBySlot(tx)(ctx)(invId, slot)
					if err != nil {
						l.WithError(err).Errorf("Unable to retrieve item in slot [%d].", slot)
						return err
					}

					reservedQty := GetReservationRegistry().GetReservedQuantity(t, characterId, inventoryType, slot)

					initialQuantity := i.Quantity() - reservedQty

					if initialQuantity <= uint32(1) {
						err = item.DeleteById(tx)(ctx)(i.Id())
						if err != nil {
							l.WithError(err).Errorf("Unable to consume item in slot [%d].", slot)
							return err
						}
						events = model.MergeSliceProvider(events, inventoryItemRemoveProvider(characterId)(inventoryType)(i.ItemId(), i.Slot()))
						return nil
					}

					newQuantity := initialQuantity - res.Quantity()
					err = item.UpdateQuantity(tx)(ctx)(i.Id(), newQuantity)
					if err != nil {
						l.WithError(err).Errorf("Unable to consume [%d] item in slot [%d].", res.Quantity(), slot)
						return err
					}
					events = model.MergeSliceProvider(events, inventoryItemQuantityUpdateProvider(characterId)(inventoryType)(i.ItemId())(newQuantity, i.Slot()))
					return nil
				})
				if txErr != nil {
					return txErr
				}
				err = producer.ProviderImpl(l)(ctx)(EnvEventInventoryChanged)(events)
				if err != nil {
					l.WithError(err).Errorf("Unable to convey inventory modifications to character [%d].", characterId)
					return err
				}
				return nil
			}
		}
	}
}

func DestroyItem(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, inventoryType inventory.Type, slot int16) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, inventoryType inventory.Type, slot int16) error {
		return func(db *gorm.DB) func(characterId uint32, inventoryType inventory.Type, slot int16) error {
			return func(characterId uint32, inventoryType inventory.Type, slot int16) error {
				l.Debugf("Received request to destroy item at [%d] for character [%d].", slot, characterId)
				invLock := GetLockRegistry().GetById(characterId, inventoryType)
				invLock.Lock()
				defer invLock.Unlock()

				var events = model.FixedProvider([]kafka.Message{})

				txErr := db.Transaction(func(tx *gorm.DB) error {
					invId, err := GetInventoryIdByType(tx)(ctx)(characterId, inventoryType)()
					if err != nil {
						l.WithError(err).Errorf("Unable to locate inventory [%d] for character [%d].", inventoryType, characterId)
						return err
					}

					if inventoryType == inventory.TypeValueEquip {
						e, err := equipable.GetBySlot(tx)(ctx)(invId, slot)
						if err != nil {
							l.WithError(err).Errorf("Unable to retrieve item in slot [%d].", slot)
							return err
						}

						err = equipable.DeleteByReferenceId(l)(tx)(ctx)(e.ReferenceId())
						if err != nil {
							l.WithError(err).Errorf("Unable to destroy item in slot [%d].", slot)
							return err
						}
						events = model.MergeSliceProvider(events, inventoryItemRemoveProvider(characterId)(inventoryType)(e.ItemId(), e.Slot()))
					} else {
						i, err := item.GetBySlot(tx)(ctx)(invId, slot)
						if err != nil {
							l.WithError(err).Errorf("Unable to retrieve item in slot [%d].", slot)
							return err
						}

						err = item.DeleteById(tx)(ctx)(i.Id())
						if err != nil {
							l.WithError(err).Errorf("Unable to destroy item in slot [%d].", slot)
							return err
						}
						events = model.MergeSliceProvider(events, inventoryItemRemoveProvider(characterId)(inventoryType)(i.ItemId(), i.Slot()))
					}
					return nil
				})
				if txErr != nil {
					return txErr
				}
				err := producer.ProviderImpl(l)(ctx)(EnvEventInventoryChanged)(events)
				if err != nil {
					l.WithError(err).Errorf("Unable to convey inventory modifications to character [%d].", characterId)
					return err
				}
				return nil
			}
		}
	}
}

func CancelReservation(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, inventoryType inventory.Type, transactionId uuid.UUID, slot int16) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, inventoryType inventory.Type, transactionId uuid.UUID, slot int16) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32, inventoryType inventory.Type, transactionId uuid.UUID, slot int16) error {
			return func(characterId uint32, inventoryType inventory.Type, transactionId uuid.UUID, slot int16) error {
				res, err := GetReservationRegistry().RemoveReservation(t, transactionId, characterId, inventoryType, slot)
				if err != nil {
					return nil
				}
				err = producer.ProviderImpl(l)(ctx)(EnvEventInventoryChanged)(inventoryItemCancelReservationProvider(characterId)(inventoryType)(res.ItemId())(res.Quantity(), slot))
				if err != nil {
					l.WithError(err).Errorf("Unable to convey inventory modifications to character [%d].", characterId)
				}
				return nil
			}
		}
	}
}

func UpdateEquip(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, slot int16, updates ...equipable.Updater) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, slot int16, updates ...equipable.Updater) error {
		return func(db *gorm.DB) func(characterId uint32, slot int16, updates ...equipable.Updater) error {
			return func(characterId uint32, slot int16, updates ...equipable.Updater) error {
				e, err := equipable.Update(l)(db)(ctx)(characterId, slot, updates...)
				if err != nil {
					return err
				}
				err = producer.ProviderImpl(l)(ctx)(EnvEventInventoryChanged)(inventoryItemAttributeUpdateProvider(characterId)(e))
				if err != nil {
					l.WithError(err).Errorf("Unable to convey inventory modifications to character [%d].", characterId)
					return err
				}
				return nil
			}
		}
	}
}
