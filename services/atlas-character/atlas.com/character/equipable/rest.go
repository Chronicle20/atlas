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

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
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

func Extract(m RestModel) (Model, error) {
	return Model{
		id:            m.Id,
		itemId:        m.ItemId,
		slot:          m.Slot,
		strength:      m.Strength,
		dexterity:     m.Dexterity,
		intelligence:  m.Intelligence,
		luck:          m.Luck,
		hp:            m.HP,
		mp:            m.MP,
		weaponAttack:  m.WeaponAttack,
		magicAttack:   m.MagicAttack,
		weaponDefense: m.WeaponDefense,
		magicDefense:  m.MagicDefense,
		accuracy:      m.Accuracy,
		avoidability:  m.Avoidability,
		hands:         m.Hands,
		speed:         m.Speed,
		jump:          m.Jump,
		slots:         m.Slots,
	}, nil
}
