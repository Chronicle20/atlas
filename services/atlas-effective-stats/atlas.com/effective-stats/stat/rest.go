package stat

import (
	"strconv"
)

// RestModel is the JSON:API representation of computed effective stats
type RestModel struct {
	Id            string           `json:"-"`
	Strength      uint32           `json:"strength"`
	Dexterity     uint32           `json:"dexterity"`
	Luck          uint32           `json:"luck"`
	Intelligence  uint32           `json:"intelligence"`
	MaxHP         uint32           `json:"maxHP"`
	MaxMP         uint32           `json:"maxMP"`
	WeaponAttack  uint32           `json:"weaponAttack"`
	WeaponDefense uint32           `json:"weaponDefense"`
	MagicAttack   uint32           `json:"magicAttack"`
	MagicDefense  uint32           `json:"magicDefense"`
	Accuracy      uint32           `json:"accuracy"`
	Avoidability  uint32           `json:"avoidability"`
	Speed         uint32           `json:"speed"`
	Jump          uint32           `json:"jump"`
	Bonuses       []BonusRestModel `json:"bonuses,omitempty"`
}

func (r RestModel) GetName() string {
	return "effective-stats"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// BonusRestModel is the JSON:API representation of a stat bonus
type BonusRestModel struct {
	Source     string  `json:"source"`
	StatType   string  `json:"statType"`
	Amount     int32   `json:"amount"`
	Multiplier float64 `json:"multiplier"`
}

// TransformBonus converts a Bonus to its REST model
func TransformBonus(b Bonus) BonusRestModel {
	return BonusRestModel{
		Source:     b.source,
		StatType:   string(b.statType),
		Amount:     b.amount,
		Multiplier: b.multiplier,
	}
}

// Transform converts Computed stats and bonuses to REST model
func Transform(characterId uint32, computed Computed, bonuses []Bonus) RestModel {
	bonusModels := make([]BonusRestModel, 0, len(bonuses))
	for _, b := range bonuses {
		bonusModels = append(bonusModels, TransformBonus(b))
	}

	return RestModel{
		Id:            strconv.FormatUint(uint64(characterId), 10),
		Strength:      computed.strength,
		Dexterity:     computed.dexterity,
		Luck:          computed.luck,
		Intelligence:  computed.intelligence,
		MaxHP:         computed.maxHP,
		MaxMP:         computed.maxMP,
		WeaponAttack:  computed.weaponAttack,
		WeaponDefense: computed.weaponDefense,
		MagicAttack:   computed.magicAttack,
		MagicDefense:  computed.magicDefense,
		Accuracy:      computed.accuracy,
		Avoidability:  computed.avoidability,
		Speed:         computed.speed,
		Jump:          computed.jump,
		Bonuses:       bonusModels,
	}
}
