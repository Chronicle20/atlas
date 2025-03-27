package equipment

import (
	"atlas-equipables/equipment/statistics"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"math"
	"math/rand"
)

var entityModelMapper = model.Map(makeEquipment)

func ByIdModelProvider(db *gorm.DB) func(ctx context.Context) func(id uint32) model.Provider[Model] {
	return func(ctx context.Context) func(id uint32) model.Provider[Model] {
		return func(id uint32) model.Provider[Model] {
			t := tenant.MustFromContext(ctx)
			return entityModelMapper(byIdEntityProvider(t.Id(), id)(db))
		}
	}
}

func GetById(_ logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(id uint32) (Model, error) {
	return func(db *gorm.DB) func(ctx context.Context) func(id uint32) (Model, error) {
		return func(ctx context.Context) func(id uint32) (Model, error) {
			return func(id uint32) (Model, error) {
				return ByIdModelProvider(db)(ctx)(id)()
			}
		}
	}
}

func Create(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(i Model) (Model, error) {
	return func(db *gorm.DB) func(ctx context.Context) func(i Model) (Model, error) {
		return func(ctx context.Context) func(i Model) (Model, error) {
			return func(i Model) (Model, error) {
				l.Debugf("Creating equipable for item [%d].", i.ItemId())
				t := tenant.MustFromContext(ctx)
				if i.Strength() == 0 && i.Dexterity() == 0 && i.Intelligence() == 0 && i.Luck() == 0 && i.HP() == 0 && i.MP() == 0 && i.WeaponAttack() == 0 && i.WeaponDefense() == 0 &&
					i.MagicAttack() == 0 && i.MagicDefense() == 0 && i.Accuracy() == 0 && i.Avoidability() == 0 && i.Hands() == 0 && i.Speed() == 0 && i.Jump() == 0 &&
					i.Slots() == 0 {
					ea, err := statistics.GetById(l, ctx)(i.ItemId())
					if err != nil {
						l.WithError(err).Errorf("Unable to get equipment information for %d.", i.ItemId())
						return Model{}, err
					} else {
						return create(db, t.Id(), i.ItemId(), ea.Strength(), ea.Dexterity(), ea.Intelligence(), ea.Luck(),
							ea.HP(), ea.MP(), ea.WeaponAttack(), ea.MagicAttack(), ea.WeaponDefense(), ea.MagicDefense(), ea.Accuracy(),
							ea.Avoidability(), ea.Hands(), ea.Speed(), ea.Jump(), ea.Slots())
					}
				} else {
					return create(db, t.Id(), i.ItemId(), i.Strength(), i.Dexterity(), i.Intelligence(), i.Luck(), i.HP(), i.MP(), i.WeaponAttack(),
						i.MagicAttack(), i.WeaponDefense(), i.MagicDefense(), i.Accuracy(), i.Avoidability(), i.Hands(), i.Speed(), i.Jump(), i.Slots())
				}
			}
		}
	}
}

func CreateRandom(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(itemId uint32) (Model, error) {
	return func(db *gorm.DB) func(ctx context.Context) func(itemId uint32) (Model, error) {
		return func(ctx context.Context) func(itemId uint32) (Model, error) {
			return func(itemId uint32) (Model, error) {

				l.Debugf("Creating equipable for item [%d].", itemId)
				ea, err := statistics.GetById(l, ctx)(itemId)
				if err != nil {
					l.WithError(err).Errorf("Unable to get equipment information for %d.", itemId)
					return Model{}, err
				} else {
					strength := getRandomStat(ea.Strength(), 5)
					dexterity := getRandomStat(ea.Dexterity(), 5)
					intelligence := getRandomStat(ea.Intelligence(), 5)
					luck := getRandomStat(ea.Luck(), 5)
					hp := getRandomStat(ea.HP(), 10)
					mp := getRandomStat(ea.MP(), 10)
					weaponAttack := getRandomStat(ea.WeaponAttack(), 5)
					magicAttack := getRandomStat(ea.MagicAttack(), 5)
					weaponDefense := getRandomStat(ea.WeaponDefense(), 10)
					magicDefense := getRandomStat(ea.MagicDefense(), 10)
					accuracy := getRandomStat(ea.Accuracy(), 5)
					avoidability := getRandomStat(ea.Avoidability(), 5)
					hands := getRandomStat(ea.Hands(), 5)
					speed := getRandomStat(ea.Speed(), 5)
					jump := getRandomStat(ea.Jump(), 5)
					slots := ea.Slots()
					t := tenant.MustFromContext(ctx)
					return create(db, t.Id(), itemId, strength, dexterity, intelligence, luck, hp, mp, weaponAttack, magicAttack, weaponDefense, magicDefense, accuracy, avoidability, hands, speed, jump, slots)
				}
			}
		}
	}
}

func getRandomStat(defaultValue uint16, max uint16) uint16 {
	if defaultValue == 0 {
		return 0
	}
	maxRange := math.Min(math.Ceil(float64(defaultValue)*0.1), float64(max))
	return uint16(float64(defaultValue)-maxRange) + uint16(math.Floor(rand.Float64()*(maxRange*2.0+1.0)))
}

func UpdateById(l logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(i Model) (Model, error) {
	return func(db *gorm.DB) func(ctx context.Context) func(i Model) (Model, error) {
		return func(ctx context.Context) func(i Model) (Model, error) {
			t := tenant.MustFromContext(ctx)
			return func(i Model) (Model, error) {
				var um Model
				l.Debugf("Updating equipable [%d].", i.Id())
				txErr := db.Transaction(func(tx *gorm.DB) error {
					c, err := GetById(l)(tx)(ctx)(i.Id())
					if err != nil {
						return err
					}
					updates := make(map[string]interface{})
					if i.Strength() != c.Strength() {
						updates["strength"] = i.Strength()
					}
					if i.Dexterity() != c.Dexterity() {
						updates["dexterity"] = i.Dexterity()
					}
					if i.Intelligence() != c.Intelligence() {
						updates["intelligence"] = i.Intelligence()
					}
					if i.Luck() != c.Luck() {
						updates["luck"] = i.Luck()
					}
					if i.HP() != c.HP() {
						updates["hp"] = i.HP()
					}
					if i.MP() != c.MP() {
						updates["mp"] = i.MP()
					}
					if i.WeaponAttack() != c.WeaponAttack() {
						updates["weapon_attack"] = i.WeaponAttack()
					}
					if i.MagicAttack() != c.MagicAttack() {
						updates["magic_attack"] = i.MagicAttack()
					}
					if i.WeaponDefense() != c.WeaponDefense() {
						updates["weapon_defense"] = i.WeaponDefense()
					}
					if i.MagicDefense() != c.MagicDefense() {
						updates["magic_defense"] = i.MagicDefense()
					}
					if i.Accuracy() != c.Accuracy() {
						updates["accuracy"] = i.Accuracy()
					}
					if i.Avoidability() != c.Avoidability() {
						updates["avoidability"] = i.Avoidability()
					}
					if i.Hands() != c.Hands() {
						updates["hands"] = i.Hands()
					}
					if i.Speed() != c.Speed() {
						updates["speed"] = i.Speed()
					}
					if i.Jump() != c.Jump() {
						updates["jump"] = i.Jump()
					}
					if i.Slots() != c.Slots() {
						updates["slots"] = i.Slots()
					}
					if i.OwnerName() != c.OwnerName() {
						updates["owner_name"] = i.OwnerName()
					}
					if i.Locked() != c.Locked() {
						updates["locked"] = i.Locked()
					}
					if i.Spikes() != c.Spikes() {
						updates["spikes"] = i.Spikes()
					}
					if i.KarmaUsed() != c.KarmaUsed() {
						updates["karma_used"] = i.KarmaUsed()
					}
					if i.Cold() != c.Cold() {
						updates["cold"] = i.Cold()
					}
					if i.CanBeTraded() != c.CanBeTraded() {
						updates["can_be_traded"] = i.CanBeTraded()
					}
					if i.LevelType() != c.LevelType() {
						updates["level_type"] = i.LevelType()
					}
					if i.Level() != c.Level() {
						updates["level"] = i.Level()
					}
					if i.Experience() != c.Experience() {
						updates["experience"] = i.Experience()
					}
					if i.HammersApplied() != c.HammersApplied() {
						updates["hammers_applied"] = i.HammersApplied()
					}
					if i.Expiration() != c.Expiration() {
						updates["expiration"] = i.Expiration()
					}
					um, err = update(tx, t.Id(), i.Id(), updates)
					return err
				})
				if txErr != nil {
					return Model{}, txErr
				}
				return um, nil
			}
		}
	}
}

func DeleteById(_ logrus.FieldLogger) func(db *gorm.DB) func(ctx context.Context) func(equipmentId uint32) error {
	return func(db *gorm.DB) func(ctx context.Context) func(equipmentId uint32) error {
		return func(ctx context.Context) func(equipmentId uint32) error {
			return func(equipmentId uint32) error {
				t := tenant.MustFromContext(ctx)
				return delete(db, t.Id(), equipmentId)
			}
		}
	}
}
