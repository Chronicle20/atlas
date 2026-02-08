package statistics

import "strconv"

type RestModel struct {
	Id            uint32 `json:"-"`
	Strength      uint16 `json:"strength"`
	Dexterity     uint16 `json:"dexterity"`
	Intelligence  uint16 `json:"intelligence"`
	Luck          uint16 `json:"luck"`
	Hp            uint16 `json:"hp"`
	Mp            uint16 `json:"mp"`
	WeaponAttack  uint16 `json:"weaponAttack"`
	MagicAttack   uint16 `json:"magicAttack"`
	WeaponDefense uint16 `json:"weaponDefense"`
	MagicDefense  uint16 `json:"magicDefense"`
	Accuracy      uint16 `json:"accuracy"`
	Avoidability  uint16 `json:"avoidability"`
	Speed         uint16 `json:"speed"`
	Jump          uint16 `json:"jump"`
	Slots         uint16 `json:"slots"`
}

func (r RestModel) GetName() string {
	return "statistics"
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

func Extract(m RestModel) (Model, error) {
	return Model{
		strength:      m.Strength,
		dexterity:     m.Dexterity,
		intelligence:  m.Intelligence,
		luck:          m.Luck,
		hp:            m.Hp,
		mp:            m.Mp,
		weaponAttack:  m.WeaponAttack,
		magicAttack:   m.MagicAttack,
		weaponDefense: m.WeaponDefense,
		magicDefense:  m.MagicDefense,
		accuracy:      m.Accuracy,
		avoidability:  m.Avoidability,
		speed:         m.Speed,
		jump:          m.Jump,
		slots:         m.Slots,
	}, nil
}
