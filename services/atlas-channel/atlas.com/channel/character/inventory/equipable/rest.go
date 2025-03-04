package equipable

import "strconv"

type RestModel struct {
	Id            uint32 `json:"id"`
	ItemId        uint32 `json:"itemId"`
	Slot          int16  `json:"slot"`
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
	Hands         uint16 `json:"hands"`
	Speed         uint16 `json:"speed"`
	Jump          uint16 `json:"jump"`
	Slots         uint16 `json:"slots"`
}

func (r RestModel) GetName() string {
	return "equipables"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func Transform(m Model) (RestModel, error) {
	rm := RestModel{
		Id:            m.id,
		ItemId:        m.itemId,
		Slot:          m.slot,
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
		Hands:         m.hands,
		Speed:         m.speed,
		Jump:          m.jump,
		Slots:         m.slots,
	}
	return rm, nil
}

func Extract(model RestModel) (Model, error) {
	return Model{
		id:            model.Id,
		itemId:        model.ItemId,
		slot:          model.Slot,
		strength:      model.Strength,
		dexterity:     model.Dexterity,
		intelligence:  model.Intelligence,
		luck:          model.Luck,
		hp:            model.HP,
		mp:            model.MP,
		weaponAttack:  model.WeaponAttack,
		magicAttack:   model.MagicAttack,
		weaponDefense: model.WeaponDefense,
		magicDefense:  model.MagicDefense,
		accuracy:      model.Accuracy,
		avoidability:  model.Avoidability,
		hands:         model.Hands,
		speed:         model.Speed,
		jump:          model.Jump,
		slots:         model.Slots,
	}, nil
}
