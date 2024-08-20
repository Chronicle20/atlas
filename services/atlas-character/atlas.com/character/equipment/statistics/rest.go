package statistics

type RestModel struct {
	Id            string `json:"-"`
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
	Cash          bool   `json:"cash"`
}

func (r *RestModel) GetName() string {
	return "statistics"
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func Extract(m RestModel) (Model, error) {
	return Model{
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
		cash:          m.Cash,
	}, nil
}
