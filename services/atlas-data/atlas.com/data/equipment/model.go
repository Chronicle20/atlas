package equipment

type Model struct {
	ItemId        uint32  `json:"item_id"`
	Strength      uint16  `json:"strength"`
	Dexterity     uint16  `json:"dexterity"`
	Intelligence  uint16  `json:"intelligence"`
	Luck          uint16  `json:"luck"`
	WeaponAttack  uint16  `json:"weapon_attack"`
	WeaponDefense uint16  `json:"weapon_defense"`
	MagicAttack   uint16  `json:"magic_attack"`
	MagicDefense  uint16  `json:"magic_defense"`
	Accuracy      uint16  `json:"accuracy"`
	Avoidability  uint16  `json:"avoidability"`
	Speed         uint16  `json:"speed"`
	Jump          uint16  `json:"jump"`
	HP            uint16  `json:"hp"`
	MP            uint16  `json:"mp"`
	Slots         uint16  `json:"slots"`
	Cash          bool    `json:"cash"`
	SlotName      string  `json:"slot_name"`
	SlotWz        string  `json:"slot_wz"`
	SlotIndex     []int16 `json:"slot_index"`
}

// Keep accessor method for itemId since the field is private
func (m Model) GetId() uint32 {
	return m.ItemId
}
