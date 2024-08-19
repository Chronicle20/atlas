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
		strength:      0,
		dexterity:     0,
		intelligence:  0,
		luck:          0,
		hp:            0,
		mp:            0,
		weaponAttack:  0,
		magicAttack:   0,
		weaponDefense: 0,
		magicDefense:  0,
		accuracy:      0,
		avoidability:  0,
		hands:         0,
		speed:         0,
		jump:          0,
		slots:         0,
	}, nil
}
