package equipable

import (
	"atlas-character/asset"
	"atlas-character/database"
	"atlas-character/equipable/statistics"
	"atlas-character/slottable"
	"atlas-character/tenant"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func byInventoryProvider(db *gorm.DB, tenant tenant.Model) func(inventoryId uint32) model.Provider[[]Model] {
	return func(inventoryId uint32) model.Provider[[]Model] {
		return database.ModelSliceProvider[Model, entity](db)(getByInventory(tenant.Id, inventoryId), makeModel)
	}
}

func GetByInventory(l logrus.FieldLogger, db *gorm.DB, ctx context.Context, tenant tenant.Model) func(inventoryId uint32) ([]Model, error) {
	return func(inventoryId uint32) ([]Model, error) {
		return model.SliceMap(byInventoryProvider(db, tenant)(inventoryId), decorateWithStatistics(l, ctx, tenant), model.ParallelMap())()
	}
}

func EquipmentProvider(l logrus.FieldLogger, db *gorm.DB, ctx context.Context, tenant tenant.Model) func(inventoryId uint32) model.Provider[[]Model] {
	return func(inventoryId uint32) model.Provider[[]Model] {
		fp := model.FilteredProvider[Model](byInventoryProvider(db, tenant)(inventoryId), FilterOutInventory)
		return model.SliceMap(fp, decorateWithStatistics(l, ctx, tenant), model.ParallelMap())
	}
}

func InInventoryProvider(l logrus.FieldLogger, db *gorm.DB, ctx context.Context, tenant tenant.Model) func(inventoryId uint32) model.Provider[[]Model] {
	return func(inventoryId uint32) model.Provider[[]Model] {
		fp := model.FilteredProvider[Model](byInventoryProvider(db, tenant)(inventoryId), FilterOutEquipment)
		return model.SliceMap(fp, decorateWithStatistics(l, ctx, tenant), model.ParallelMap())
	}
}

func ToAsset(m Model) (asset.Asset, error) {
	return m, nil
}

func ToSlottable(m Model) (asset.Slottable, error) {
	return m, nil
}

func BySlotProvider(db *gorm.DB) func(tenant tenant.Model) func(characterId uint32) func(slot int16) model.Provider[Model] {
	return func(tenant tenant.Model) func(characterId uint32) func(slot int16) model.Provider[Model] {
		return func(characterId uint32) func(slot int16) model.Provider[Model] {
			return func(slot int16) model.Provider[Model] {
				return database.ModelProvider[Model, entity](db)(getBySlot(tenant.Id, characterId, slot), makeModel)
			}
		}
	}
}

func GetBySlot(db *gorm.DB, tenant tenant.Model) func(characterId uint32, slot int16) (Model, error) {
	return func(characterId uint32, slot int16) (Model, error) {
		return BySlotProvider(db)(tenant)(characterId)(slot)()
	}
}

func FilterOutInventory(e Model) bool {
	return e.Slot() < 0
}

func FilterOutEquipment(e Model) bool {
	return e.Slot() > 0
}

func CreateItem(l logrus.FieldLogger, db *gorm.DB, ctx context.Context, tenant tenant.Model, statCreator statistics.Creator) asset.CharacterAssetCreator {
	return func(characterId uint32) asset.InventoryAssetCreator {
		return func(inventoryId uint32, inventoryType int8) asset.ItemCreator {
			return func(itemId uint32) asset.Creator {
				return func(quantity uint32) model.Provider[asset.Asset] {
					l.Debugf("Creating equipable [%d] for character [%d].", itemId, characterId)
					slot, err := GetNextFreeSlot(l)(db)(ctx)(tenant)(inventoryId)()
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

					i, err := createItem(db, tenant, inventoryId, itemId, slot, sm.Id())
					if err != nil {
						return model.ErrorProvider[asset.Asset](err)
					}

					l.Debugf("Equipable [%d] created for character [%d].", sm.Id(), characterId)
					rmp := model.Map[Model, Model](model.FixedProvider[Model](i), model.Decorate[Model](statisticsDecorator(sm)))
					return model.Map(rmp, ToAsset)
				}
			}
		}
	}
}

func GetNextFreeSlot(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(tenant tenant.Model) func(inventoryId uint32) model.Provider[int16] {
	return func(db *gorm.DB) func(ctx context.Context) func(tenant tenant.Model) func(inventoryId uint32) model.Provider[int16] {
		return func(ctx context.Context) func(tenant tenant.Model) func(inventoryId uint32) model.Provider[int16] {
			return func(tenant tenant.Model) func(inventoryId uint32) model.Provider[int16] {
				return func(inventoryId uint32) model.Provider[int16] {
					ms, err := GetByInventory(l, db, ctx, tenant)(inventoryId)
					if err != nil {
						return model.ErrorProvider[int16](err)
					}
					slot, err := slottable.GetNextFreeSlot(model.SliceMap(model.FixedProvider(ms), ToSlottable))
					if err != nil {
						return model.ErrorProvider[int16](err)
					}
					return model.FixedProvider(slot)
				}
			}
		}
	}
}

func decorateWithStatistics(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(e Model) (Model, error) {
	return func(e Model) (Model, error) {
		sm, err := statistics.GetById(l, ctx, tenant)(e.ReferenceId())
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

func UpdateSlot(db *gorm.DB) func(tenant tenant.Model) func(id uint32, slot int16) error {
	return func(tenant tenant.Model) func(id uint32, slot int16) error {
		return func(id uint32, slot int16) error {
			return updateSlot(db, tenant.Id, id, slot)
		}
	}
}

func DeleteByReferenceId(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(tenant tenant.Model) model.Operator[uint32] {
	return func(ctx context.Context) func(db *gorm.DB) func(tenant tenant.Model) model.Operator[uint32] {
		return func(db *gorm.DB) func(tenant tenant.Model) model.Operator[uint32] {
			return func(tenant tenant.Model) model.Operator[uint32] {
				return func(referenceId uint32) error {
					l.Debugf("Attempting to delete equipment referencing [%d].", referenceId)
					err := statistics.Delete(l, ctx, tenant)(referenceId)
					if err != nil {
						return err
					}
					return delete(db, tenant.Id, referenceId)
				}
			}
		}
	}
}

func DropByReferenceId(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(referenceId uint32) error {
	return func(referenceId uint32) error {
		l.Debugf("Attempting to drop equipment referencing [%d].", referenceId)
		return delete(db, tenant.Id, referenceId)
	}
}
