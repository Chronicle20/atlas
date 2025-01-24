package equipment

import (
	"strconv"
)

type RestModel struct {
	Id            uint32 `json:"-"`
	Strength      uint16 `json:"strength"`
	Dexterity     uint16 `json:"dexterity"`
	Intelligence  uint16 `json:"intelligence"`
	Luck          uint16 `json:"luck"`
	HP            uint16 `json:"hp"`
	MP            uint16 `json:"mp"`
	WeaponAttack  uint16 `json:"weaponAttack"`
	MagicAttack   uint16 `json:"magicAttack"`
	WeaponDefense uint16 `json:"weaponDefense"`
	MagicDefense  uint16 `json:"magicDefense"`
	Accuracy      uint16 `json:"accuracy"`
	Avoidability  uint16 `json:"avoidability"`
	Speed         uint16 `json:"speed"`
	Jump          uint16 `json:"jump"`
	Slots         uint16 `json:"slots"`
	Cash          bool   `json:"cash"`
}

func (r RestModel) GetName() string {
	return "statistics"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:            m.itemId,
		Strength:      m.strength,
		Dexterity:     m.dexterity,
		Intelligence:  m.intelligence,
		Luck:          m.luck,
		HP:            m.hp,
		MP:            m.mp,
		WeaponAttack:  m.weaponAttack,
		MagicAttack:   m.magicAttack,
		WeaponDefense: m.weaponDefense,
		MagicDefense:  m.magicDefense,
		Accuracy:      m.accuracy,
		Avoidability:  m.avoidability,
		Speed:         m.speed,
		Jump:          m.jump,
		Slots:         m.slots,
		Cash:          m.cash,
	}, nil
}

type SlotRestModel struct {
	Id   uint32 `json:"-"`
	Name string `json:"name"`
	WZ   string `json:"WZ"`
	Slot int16  `json:"slot"`
}

func (r SlotRestModel) GetName() string {
	return "slots"
}

func (r SlotRestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func TransformSlot(m Model) ([]SlotRestModel, error) {
	var results = make([]SlotRestModel, 0)
	for _, s := range m.slotIndex {
		rm := SlotRestModel{
			Id:   m.itemId,
			Name: m.slotName,
			WZ:   m.slotWz,
			Slot: s,
		}
		results = append(results, rm)
	}
	return results, nil
}
