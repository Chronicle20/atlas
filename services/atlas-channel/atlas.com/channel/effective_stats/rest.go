package effective_stats

// RestModel mirrors the atlas-effective-stats service's character stats
// response. atlas-channel reads it for snapshot computations that depend
// on Luck / MagicAttack / etc. (e.g., player VENOM damage-per-tick).
type RestModel struct {
	Id            string `json:"-"`
	Strength      uint32 `json:"strength"`
	Dexterity     uint32 `json:"dexterity"`
	Luck          uint32 `json:"luck"`
	Intelligence  uint32 `json:"intelligence"`
	MaxHp         uint32 `json:"maxHP"`
	MaxMp         uint32 `json:"maxMP"`
	WeaponAttack  uint32 `json:"weaponAttack"`
	WeaponDefense uint32 `json:"weaponDefense"`
	MagicAttack   uint32 `json:"magicAttack"`
	MagicDefense  uint32 `json:"magicDefense"`
	Accuracy      uint32 `json:"accuracy"`
	Avoidability  uint32 `json:"avoidability"`
	Speed         uint32 `json:"speed"`
	Jump          uint32 `json:"jump"`
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
