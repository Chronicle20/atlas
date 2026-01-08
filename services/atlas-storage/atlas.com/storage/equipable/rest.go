package equipable

import (
	"strconv"
	"time"
)

type RestModel struct {
	Id             string    `json:"-"`
	OwnerId        uint32    `json:"ownerId"`
	Strength       uint16    `json:"strength"`
	Dexterity      uint16    `json:"dexterity"`
	Intelligence   uint16    `json:"intelligence"`
	Luck           uint16    `json:"luck"`
	Hp             uint16    `json:"hp"`
	Mp             uint16    `json:"mp"`
	WeaponAttack   uint16    `json:"weaponAttack"`
	MagicAttack    uint16    `json:"magicAttack"`
	WeaponDefense  uint16    `json:"weaponDefense"`
	MagicDefense   uint16    `json:"magicDefense"`
	Accuracy       uint16    `json:"accuracy"`
	Avoidability   uint16    `json:"avoidability"`
	Hands          uint16    `json:"hands"`
	Speed          uint16    `json:"speed"`
	Jump           uint16    `json:"jump"`
	Slots          uint16    `json:"slots"`
	Locked         bool      `json:"locked"`
	Spikes         bool      `json:"spikes"`
	KarmaUsed      bool      `json:"karmaUsed"`
	Cold           bool      `json:"cold"`
	CanBeTraded    bool      `json:"canBeTraded"`
	LevelType      byte      `json:"levelType"`
	Level          byte      `json:"level"`
	Experience     uint32    `json:"experience"`
	HammersApplied uint32    `json:"hammersApplied"`
	Expiration     time.Time `json:"expiration"`
}

func (r RestModel) GetName() string {
	return "equipables"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func Extract(rm RestModel) (Model, error) {
	id, err := strconv.ParseUint(rm.Id, 10, 32)
	if err != nil {
		return Model{}, err
	}

	return Model{
		id:             uint32(id),
		ownerId:        rm.OwnerId,
		strength:       rm.Strength,
		dexterity:      rm.Dexterity,
		intelligence:   rm.Intelligence,
		luck:           rm.Luck,
		hp:             rm.Hp,
		mp:             rm.Mp,
		weaponAttack:   rm.WeaponAttack,
		magicAttack:    rm.MagicAttack,
		weaponDefense:  rm.WeaponDefense,
		magicDefense:   rm.MagicDefense,
		accuracy:       rm.Accuracy,
		avoidability:   rm.Avoidability,
		hands:          rm.Hands,
		speed:          rm.Speed,
		jump:           rm.Jump,
		slots:          rm.Slots,
		locked:         rm.Locked,
		spikes:         rm.Spikes,
		karmaUsed:      rm.KarmaUsed,
		cold:           rm.Cold,
		canBeTraded:    rm.CanBeTraded,
		levelType:      rm.LevelType,
		level:          rm.Level,
		experience:     rm.Experience,
		hammersApplied: rm.HammersApplied,
		expiration:     rm.Expiration,
	}, nil
}
