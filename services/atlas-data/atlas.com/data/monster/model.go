package monster

type Model struct {
	Id                 uint32            `json:"id"`
	Name               string            `json:"name"`
	HP                 uint32            `json:"hp"`
	MP                 uint32            `json:"mp"`
	Experience         uint32            `json:"experience"`
	Level              uint32            `json:"level"`
	WeaponAttack       uint32            `json:"weapon_attack"`
	WeaponDefense      uint32            `json:"weapon_defense"`
	MagicAttack        uint32            `json:"magic_attack"`
	MagicDefense       uint32            `json:"magic_defense"`
	Friendly           bool              `json:"friendly"`
	RemoveAfter        uint32            `json:"remove_after"`
	Boss               bool              `json:"boss"`
	ExplosiveReward    bool              `json:"explosive_reward"`
	FFALoot            bool              `json:"ffa_loot"`
	Undead             bool              `json:"undead"`
	BuffToGive         uint32            `json:"buff_to_give"`
	CP                 uint32            `json:"cp"`
	RemoveOnMiss       bool              `json:"remove_on_miss"`
	Changeable         bool              `json:"changeable"`
	AnimationTimes     map[string]uint32 `json:"animation_times"`
	Resistances        map[string]string `json:"resistances"`
	LoseItems          []LoseItem        `json:"lose_items"`
	Skills             []Skill           `json:"skills"`
	Revives            []uint32          `json:"revives"`
	TagColor           byte              `json:"tag_color"`
	TagBackgroundColor byte              `json:"tag_background_color"`
	FixedStance        uint32            `json:"fixed_stance"`
	FirstAttack        bool              `json:"first_attack"`
	Banish             *Banish           `json:"banish,omitempty"`
	DropPeriod         uint32            `json:"drop_period"`
	SelfDestruction    *SelfDestruction  `json:"self_destruction,omitempty"`
	CoolDamage         *CoolDamage       `json:"cool_damage,omitempty"`
}

func (m Model) GetId() uint32 {
	return m.Id
}

type Skill struct {
	Id    uint32 `json:"id"`
	Level uint32 `json:"level"`
}

type Banish struct {
	Message    string `json:"message"`
	MapId      uint32 `json:"map_id"`
	PortalName string `json:"portal_name"`
}

type SelfDestruction struct {
	Action      byte  `json:"action"`
	RemoveAfter int32 `json:"remove_after"`
	HP          int32 `json:"hp"`
}

type CoolDamage struct {
	Damage      uint32 `json:"damage"`
	Probability uint32 `json:"probability"`
}

type LoseItem struct {
	ItemId uint32 `json:"item_id"`
	Chance byte   `json:"chance"`
	X      byte   `json:"x"`
}
