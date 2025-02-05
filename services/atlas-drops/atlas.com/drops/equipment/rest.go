package equipment

import "strconv"

type RestModel struct {
	Id            uint32 `json:"-"`
	ItemId        uint32 `json:"itemId"`
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

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:            rm.Id,
		itemId:        rm.ItemId,
		strength:      rm.Strength,
		dexterity:     rm.Dexterity,
		intelligence:  rm.Intelligence,
		luck:          rm.Luck,
		hp:            rm.HP,
		mp:            rm.MP,
		weaponAttack:  rm.WeaponAttack,
		magicAttack:   rm.MagicAttack,
		weaponDefense: rm.WeaponDefense,
		magicDefense:  rm.MagicDefense,
		accuracy:      rm.Accuracy,
		avoidability:  rm.Avoidability,
		hands:         rm.Hands,
		speed:         rm.Speed,
		jump:          rm.Jump,
		slots:         rm.Slots,
	}, nil
}
