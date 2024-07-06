package equipment

import (
	"atlas-equipables/database"
	"atlas-equipables/equipment/statistics"
	"atlas-equipables/tenant"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"math"
	"math/rand"
)

func ByIdModelProvider(_ logrus.FieldLogger, db *gorm.DB, _ opentracing.Span, tenant tenant.Model) func(id uint32) model.Provider[Model] {
	return func(id uint32) model.Provider[Model] {
		return database.ModelProvider[Model, entity](db)(byIdEntityProvider(tenant.Id(), id), makeEquipment)
	}
}

func GetById(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(id uint32) (Model, error) {
	return func(id uint32) (Model, error) {
		return ByIdModelProvider(l, db, span, tenant)(id)()
	}
}

func Create(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(itemId uint32, strength uint16, dexterity uint16, intelligence uint16, luck uint16,
	hp uint16, mp uint16, weaponAttack uint16, magicAttack uint16, weaponDefense uint16, magicDefense uint16,
	accuracy uint16, avoidability uint16, hands uint16, speed uint16, jump uint16, slots uint16) (Model, error) {
	return func(itemId uint32, strength uint16, dexterity uint16, intelligence uint16, luck uint16, hp uint16, mp uint16, weaponAttack uint16, magicAttack uint16, weaponDefense uint16, magicDefense uint16, accuracy uint16, avoidability uint16, hands uint16, speed uint16, jump uint16, slots uint16) (Model, error) {
		if strength == 0 && dexterity == 0 && intelligence == 0 && luck == 0 && hp == 0 && mp == 0 && weaponAttack == 0 && weaponDefense == 0 &&
			magicAttack == 0 && magicDefense == 0 && accuracy == 0 && avoidability == 0 && hands == 0 && speed == 0 && jump == 0 &&
			slots == 0 {
			ea, err := statistics.GetById(l, span, tenant)(itemId)
			if err != nil {
				l.WithError(err).Errorf("Unable to get equipment information for %d.", itemId)
				return Model{}, err
			} else {
				return create(db, tenant.Id(), itemId, ea.Strength(), ea.Dexterity(), ea.Intelligence(), ea.Luck(),
					ea.HP(), ea.MP(), ea.WeaponAttack(), ea.MagicAttack(), ea.WeaponDefense(), ea.MagicDefense(), ea.Accuracy(),
					ea.Avoidability(), ea.Hands(), ea.Speed(), ea.Jump(), ea.Slots())
			}
		} else {
			return create(db, tenant.Id(), itemId, strength, dexterity, intelligence, luck, hp, mp, weaponAttack,
				magicAttack, weaponDefense, magicDefense, accuracy, avoidability, hands, speed, jump, slots)
		}
	}
}

func CreateRandom(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(itemId uint32) (Model, error) {
	return func(itemId uint32) (Model, error) {
		ea, err := statistics.GetById(l, span, tenant)(itemId)
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
			return create(db, tenant.Id(), itemId, strength, dexterity, intelligence, luck, hp, mp, weaponAttack, magicAttack, weaponDefense, magicDefense, accuracy, avoidability, hands, speed, jump, slots)
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

func DeleteById(_ logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(equipmentId uint32) error {
	return func(equipmentId uint32) error {
		return delete(db, tenant.Id(), equipmentId)
	}
}
