package equipable

import (
	"atlas-character/asset"
	"atlas-character/equipable/statistics"
	"atlas-character/slottable"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func byInventoryProvider(db *gorm.DB) func(ctx context.Context) func(inventoryId uint32) model.Provider[[]Model] {
	return func(ctx context.Context) func(inventoryId uint32) model.Provider[[]Model] {
		return func(inventoryId uint32) model.Provider[[]Model] {
			t := tenant.MustFromContext(ctx)
			return model.SliceMap[entity, Model](makeModel)(getByInventory(t.Id(), inventoryId)(db))(model.ParallelMap())
		}
	}
}

func GetByInventory(l logrus.FieldLogger, db *gorm.DB, ctx context.Context) func(inventoryId uint32) ([]Model, error) {
	return func(inventoryId uint32) ([]Model, error) {
		return model.SliceMap(decorateWithStatistics(l, ctx))(byInventoryProvider(db)(ctx)(inventoryId))(model.ParallelMap())()
	}
}

func EquipmentProvider(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(inventoryId uint32) model.Provider[[]Model] {
	return func(db *gorm.DB) func(ctx context.Context) func(inventoryId uint32) model.Provider[[]Model] {
		return func(ctx context.Context) func(inventoryId uint32) model.Provider[[]Model] {
			return func(inventoryId uint32) model.Provider[[]Model] {
				fp := model.FilteredProvider[Model](byInventoryProvider(db)(ctx)(inventoryId), model.Filters(FilterOutInventory))
				return model.SliceMap(decorateWithStatistics(l, ctx))(fp)(model.ParallelMap())
			}
		}
	}
}

func InInventoryProvider(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(inventoryId uint32) model.Provider[[]Model] {
	return func(db *gorm.DB) func(ctx context.Context) func(inventoryId uint32) model.Provider[[]Model] {
		return func(ctx context.Context) func(inventoryId uint32) model.Provider[[]Model] {
			return func(inventoryId uint32) model.Provider[[]Model] {
				fp := model.FilteredProvider[Model](byInventoryProvider(db)(ctx)(inventoryId), model.Filters(FilterOutEquipment))
				return model.SliceMap(decorateWithStatistics(l, ctx))(fp)(model.ParallelMap())
			}
		}
	}
}

var ModelAssetMapper = model.Map(ToAsset)

func AssetBySlotProvider(db *gorm.DB) func(ctx context.Context) func(characterId uint32) func(slot int16) model.Provider[asset.Asset] {
	return func(ctx context.Context) func(characterId uint32) func(slot int16) model.Provider[asset.Asset] {
		return func(characterId uint32) func(slot int16) model.Provider[asset.Asset] {
			return func(slot int16) model.Provider[asset.Asset] {
				return ModelAssetMapper(BySlotProvider(db)(ctx)(characterId)(slot))
			}
		}
	}
}

func ToAsset(m Model) (asset.Asset, error) {
	return m, nil
}

var SlottableMapper = model.SliceMap(ToSlottable)

func ToSlottable(m Model) (asset.Slottable, error) {
	return m, nil
}

func BySlotProvider(db *gorm.DB) func(ctx context.Context) func(characterId uint32) func(slot int16) model.Provider[Model] {
	return func(ctx context.Context) func(characterId uint32) func(slot int16) model.Provider[Model] {
		return func(characterId uint32) func(slot int16) model.Provider[Model] {
			return func(slot int16) model.Provider[Model] {
				t := tenant.MustFromContext(ctx)
				return model.Map[entity, Model](makeModel)(getBySlot(t.Id(), characterId, slot)(db))
			}
		}
	}
}

func GetBySlot(db *gorm.DB) func(ctx context.Context) func(characterId uint32, slot int16) (Model, error) {
	return func(ctx context.Context) func(characterId uint32, slot int16) (Model, error) {
		return func(characterId uint32, slot int16) (Model, error) {
			return BySlotProvider(db)(ctx)(characterId)(slot)()
		}
	}
}

func FilterOutInventory(e Model) bool {
	return e.Slot() < 0
}

func FilterOutEquipment(e Model) bool {
	return e.Slot() > 0
}

func CreateItem(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(statCreator statistics.Creator) asset.CharacterAssetCreator {
	return func(db *gorm.DB) func(ctx context.Context) func(statCreator statistics.Creator) asset.CharacterAssetCreator {
		return func(ctx context.Context) func(statCreator statistics.Creator) asset.CharacterAssetCreator {
			return func(statCreator statistics.Creator) asset.CharacterAssetCreator {
				return func(characterId uint32) asset.InventoryAssetCreator {
					return func(inventoryId uint32, inventoryType int8) asset.ItemCreator {
						return func(itemId uint32) asset.Creator {
							return func(quantity uint32) model.Provider[asset.Asset] {
								l.Debugf("Creating equipable [%d] for character [%d].", itemId, characterId)
								slot, err := GetNextFreeSlot(l)(db)(ctx)(inventoryId)()
								if err != nil {
									l.WithError(err).Errorf("Unable to locate a free slot to create the item.")
									return model.ErrorProvider[asset.Asset](err)
								}
								l.Debugf("Found open slot [%d] in inventory [%d] of type [%d].", slot, inventoryId, itemId)
								l.Debugf("Generating new equipable statistics for item [%d].", itemId)

								sm, err := statCreator(itemId)()
								if err != nil {
									l.WithError(err).Errorf("Unable to generate equipment [%d] in equipable storage service for character [%d].", itemId, characterId)
									return model.ErrorProvider[asset.Asset](err)
								}

								t := tenant.MustFromContext(ctx)
								i, err := createItem(db, t.Id(), inventoryId, itemId, slot, sm.Id())
								if err != nil {
									return model.ErrorProvider[asset.Asset](err)
								}

								l.Debugf("Equipable [%d] created for character [%d].", sm.Id(), characterId)
								return model.Map(ToAsset)(model.Map[Model, Model](model.Decorate[Model](model.Decorators(statisticsDecorator(sm))))(model.FixedProvider[Model](i)))
							}
						}
					}
				}
			}
		}
	}
}

func GetNextFreeSlot(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(inventoryId uint32) model.Provider[int16] {
	return func(db *gorm.DB) func(ctx context.Context) func(inventoryId uint32) model.Provider[int16] {
		return func(ctx context.Context) func(inventoryId uint32) model.Provider[int16] {
			return func(inventoryId uint32) model.Provider[int16] {
				ms, err := GetByInventory(l, db, ctx)(inventoryId)
				if err != nil {
					return model.ErrorProvider[int16](err)
				}
				slot, err := slottable.GetNextFreeSlot(SlottableMapper(model.FixedProvider(ms))(model.ParallelMap()))
				if err != nil {
					return model.ErrorProvider[int16](err)
				}
				return model.FixedProvider(slot)
			}
		}
	}
}

func decorateWithStatistics(l logrus.FieldLogger, ctx context.Context) func(e Model) (Model, error) {
	return func(e Model) (Model, error) {
		sm, err := statistics.GetById(l, ctx)(e.ReferenceId())
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve generated equipment [%d] statistics.", e.Id())
			return e, nil
		}
		return statisticsDecorator(sm)(e), nil
	}
}

func statisticsDecorator(sm statistics.Model) model.Decorator[Model] {
	return func(m Model) Model {
		m.strength = sm.Strength()
		m.dexterity = sm.Dexterity()
		m.intelligence = sm.Intelligence()
		m.luck = sm.Luck()
		m.hp = sm.HP()
		m.mp = sm.MP()
		m.weaponAttack = sm.WeaponAttack()
		m.magicAttack = sm.MagicAttack()
		m.weaponDefense = sm.WeaponDefense()
		m.magicDefense = sm.MagicDefense()
		m.accuracy = sm.Accuracy()
		m.avoidability = sm.Avoidability()
		m.hands = sm.Hands()
		m.speed = sm.Speed()
		m.jump = sm.Jump()
		m.slots = sm.Slots()
		return m
	}
}

func UpdateSlot(db *gorm.DB) func(ctx context.Context) func(id uint32, slot int16) error {
	return func(ctx context.Context) func(id uint32, slot int16) error {
		return func(id uint32, slot int16) error {
			t := tenant.MustFromContext(ctx)
			return updateSlot(db, t.Id(), id, slot)
		}
	}
}

func DeleteByReferenceId(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) model.Operator[uint32] {
	return func(db *gorm.DB) func(ctx context.Context) model.Operator[uint32] {
		return func(ctx context.Context) model.Operator[uint32] {
			return func(referenceId uint32) error {
				l.Debugf("Attempting to delete equipment referencing [%d].", referenceId)
				err := statistics.Delete(l, ctx)(referenceId)
				if err != nil {
					return err
				}
				t := tenant.MustFromContext(ctx)
				return delete(db, t.Id(), referenceId)
			}
		}
	}
}

func DropByReferenceId(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(referenceId uint32) error {
	return func(db *gorm.DB) func(ctx context.Context) func(referenceId uint32) error {
		return func(ctx context.Context) func(referenceId uint32) error {
			return func(referenceId uint32) error {
				l.Debugf("Attempting to drop equipment referencing [%d].", referenceId)
				t := tenant.MustFromContext(ctx)
				return delete(db, t.Id(), referenceId)
			}
		}
	}
}
