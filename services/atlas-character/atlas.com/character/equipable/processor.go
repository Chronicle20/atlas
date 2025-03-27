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
	"time"
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
		m.ownerName = sm.OwnerName()
		m.locked = sm.Locked()
		m.spikes = sm.Spikes()
		m.karmaUsed = sm.KarmaUsed()
		m.cold = sm.Cold()
		m.canBeTraded = sm.CanBeTraded()
		m.levelType = sm.LevelType()
		m.level = sm.Level()
		m.expiration = sm.Expiration()
		m.hammersApplied = sm.HammersApplied()
		m.expiration = sm.Expiration()
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

type Updater func(m Model) Model
type StatisticUpdate func(stat int16) Updater

func AddStrength(amount int16) Updater {
	return func(m Model) Model {
		val := int32(m.Strength()) + int32(amount)
		if val < 0 {
			val = 0
		}
		return CloneFromModel(m).SetStrength(uint16(val)).Build()
	}
}

func AddDexterity(amount int16) Updater {
	return func(m Model) Model {
		val := int32(m.Dexterity()) + int32(amount)
		if val < 0 {
			val = 0
		}
		return CloneFromModel(m).SetDexterity(uint16(val)).Build()
	}
}

func AddIntelligence(amount int16) Updater {
	return func(m Model) Model {
		val := int32(m.Intelligence()) + int32(amount)
		if val < 0 {
			val = 0
		}
		return CloneFromModel(m).SetIntelligence(uint16(val)).Build()
	}
}

func AddLuck(amount int16) Updater {
	return func(m Model) Model {
		val := int32(m.Luck()) + int32(amount)
		if val < 0 {
			val = 0
		}
		return CloneFromModel(m).SetLuck(uint16(val)).Build()
	}
}

func AddHP(amount int16) Updater {
	return func(m Model) Model {
		val := int32(m.HP()) + int32(amount)
		if val < 0 {
			val = 0
		}
		return CloneFromModel(m).SetHP(uint16(val)).Build()
	}
}

func AddMP(amount int16) Updater {
	return func(m Model) Model {
		val := int32(m.MP()) + int32(amount)
		if val < 0 {
			val = 0
		}
		return CloneFromModel(m).SetMP(uint16(val)).Build()
	}
}

func AddWeaponAttack(amount int16) Updater {
	return func(m Model) Model {
		val := int32(m.WeaponAttack()) + int32(amount)
		if val < 0 {
			val = 0
		}
		return CloneFromModel(m).SetWeaponAttack(uint16(val)).Build()
	}
}

func AddMagicAttack(amount int16) Updater {
	return func(m Model) Model {
		val := int32(m.MagicAttack()) + int32(amount)
		if val < 0 {
			val = 0
		}
		return CloneFromModel(m).SetMagicAttack(uint16(val)).Build()
	}
}

func AddWeaponDefense(amount int16) Updater {
	return func(m Model) Model {
		val := int32(m.WeaponDefense()) + int32(amount)
		if val < 0 {
			val = 0
		}
		return CloneFromModel(m).SetWeaponDefense(uint16(val)).Build()
	}
}

func AddMagicDefense(amount int16) Updater {
	return func(m Model) Model {
		val := int32(m.MagicDefense()) + int32(amount)
		if val < 0 {
			val = 0
		}
		return CloneFromModel(m).SetMagicDefense(uint16(val)).Build()
	}
}

func AddAccuracy(amount int16) Updater {
	return func(m Model) Model {
		val := int32(m.Accuracy()) + int32(amount)
		if val < 0 {
			val = 0
		}
		return CloneFromModel(m).SetAccuracy(uint16(val)).Build()
	}
}

func AddAvoidability(amount int16) Updater {
	return func(m Model) Model {
		val := int32(m.Avoidability()) + int32(amount)
		if val < 0 {
			val = 0
		}
		return CloneFromModel(m).SetAvoidability(uint16(val)).Build()
	}
}

func AddHands(amount int16) Updater {
	return func(m Model) Model {
		val := int32(m.Hands()) + int32(amount)
		if val < 0 {
			val = 0
		}
		return CloneFromModel(m).SetHands(uint16(val)).Build()
	}
}

func AddSpeed(amount int16) Updater {
	return func(m Model) Model {
		val := int32(m.Speed()) + int32(amount)
		if val < 0 {
			val = 0
		}
		return CloneFromModel(m).SetSpeed(uint16(val)).Build()
	}
}

func AddJump(amount int16) Updater {
	return func(m Model) Model {
		val := int32(m.Jump()) + int32(amount)
		if val < 0 {
			val = 0
		}
		return CloneFromModel(m).SetJump(uint16(val)).Build()
	}
}

func AddSlots(amount int16) Updater {
	return func(m Model) Model {
		val := int32(m.Slots()) + int32(amount)
		if val < 0 {
			val = 0
		}
		return CloneFromModel(m).SetSlots(uint16(val)).Build()
	}
}

func SetOwnerName(name string) Updater {
	return func(m Model) Model {
		return CloneFromModel(m).SetOwnerName(name).Build()
	}
}

func SetLocked(locked bool) Updater {
	return func(m Model) Model {
		return CloneFromModel(m).SetLocked(locked).Build()
	}
}

func SetSpikes(spikes bool) Updater {
	return func(m Model) Model {
		return CloneFromModel(m).SetSpikes(spikes).Build()
	}
}

func SetKarmaUsed(karmaUsed bool) Updater {
	return func(m Model) Model {
		return CloneFromModel(m).SetKarmaUsed(karmaUsed).Build()
	}
}

func SetCold(cold bool) Updater {
	return func(m Model) Model {
		return CloneFromModel(m).SetCold(cold).Build()
	}
}

func SetCanBeTraded(canBeTraded bool) Updater {
	return func(m Model) Model {
		return CloneFromModel(m).SetCanBeTraded(canBeTraded).Build()
	}
}

func AddLevelType(levelType int8) Updater {
	return func(m Model) Model {
		val := int16(m.LevelType()) + int16(levelType)
		if val < 0 {
			val = 0
		}
		return CloneFromModel(m).SetLevelType(byte(val)).Build()
	}
}

func AddLevel(level int8) Updater {
	return func(m Model) Model {
		val := int16(m.Level()) + int16(level)
		if val < 0 {
			val = 0
		}
		return CloneFromModel(m).SetLevel(byte(val)).Build()
	}
}

func AddExperience(experience int32) Updater {
	return func(m Model) Model {
		val := int64(m.Experience()) + int64(experience)
		if val < 0 {
			val = 0
		}
		return CloneFromModel(m).SetExperience(uint32(experience)).Build()
	}
}

func AddHammersApplied(hammersApplied int32) Updater {
	return func(m Model) Model {
		val := int64(m.HammersApplied()) + int64(hammersApplied)
		if val < 0 {
			val = 0
		}
		return CloneFromModel(m).SetHammersApplied(uint32(hammersApplied)).Build()
	}
}

func SetExpiration(expiration time.Time) Updater {
	return func(m Model) Model {
		return CloneFromModel(m).SetExpiration(expiration).Build()
	}
}

func Update(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(characterId uint32, slot int16, updates ...Updater) (Model, error) {
	return func(db *gorm.DB) func(ctx context.Context) func(characterId uint32, slot int16, updates ...Updater) (Model, error) {
		return func(ctx context.Context) func(characterId uint32, slot int16, updates ...Updater) (Model, error) {
			return func(characterId uint32, slot int16, updates ...Updater) (Model, error) {
				p := BySlotProvider(db)(ctx)(characterId)(slot)
				e, err := model.Map(decorateWithStatistics(l, ctx))(p)()
				if err != nil {
					return Model{}, err
				}
				for _, update := range updates {
					e = update(e)
				}

				is := statistics.RestModel{
					Id:             e.ReferenceId(),
					ItemId:         e.ItemId(),
					Strength:       e.Strength(),
					Dexterity:      e.Dexterity(),
					Intelligence:   e.Intelligence(),
					Luck:           e.Luck(),
					HP:             e.HP(),
					MP:             e.MP(),
					WeaponAttack:   e.WeaponAttack(),
					MagicAttack:    e.MagicAttack(),
					WeaponDefense:  e.MagicDefense(),
					MagicDefense:   e.MagicDefense(),
					Accuracy:       e.Accuracy(),
					Avoidability:   e.Avoidability(),
					Hands:          e.Hands(),
					Speed:          e.Speed(),
					Jump:           e.Jump(),
					Slots:          e.Slots(),
					OwnerName:      e.OwnerName(),
					Locked:         e.Locked(),
					Spikes:         e.Spikes(),
					KarmaUsed:      e.KarmaUsed(),
					Cold:           e.Cold(),
					CanBeTraded:    e.CanBeTraded(),
					LevelType:      e.LevelType(),
					Level:          e.Level(),
					Experience:     e.Experience(),
					HammersApplied: e.HammersApplied(),
					Expiration:     e.Expiration(),
				}

				_, err = statistics.UpdateById(l, ctx)(e.ReferenceId(), is)
				if err != nil {
					return Model{}, err
				}
				return e, nil
			}
		}
	}
}
