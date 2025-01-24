package monster

type Model struct {
	id                 uint32
	name               string
	hp                 uint32
	mp                 uint32
	experience         uint32
	level              uint32
	weaponAttack       uint32
	weaponDefense      uint32
	magicAttack        uint32
	magicDefense       uint32
	friendly           bool
	removeAfter        uint32
	boss               bool
	explosiveReward    bool
	ffaLoot            bool
	undead             bool
	buffToGive         uint32
	cp                 uint32
	removeOnMiss       bool
	changeable         bool
	animationTimes     map[string]uint32
	resistances        map[string]string
	loseItems          []LoseItem
	skills             []Skill
	revives            []uint32
	tagColor           byte
	tagBackgroundColor byte
	fixedStance        uint32
	firstAttack        bool
	banish             *Banish
	dropPeriod         uint32
	selfDestruction    *SelfDestruction
	coolDamage         *CoolDamage
}

func (m Model) Id() uint32 {
	return m.id
}

type Skill struct {
	id    uint32
	level uint32
}

type Banish struct {
	message    string
	mapId      uint32
	portalName string
}

type SelfDestruction struct {
	action      byte
	removeAfter int32
	hp          int32
}

type CoolDamage struct {
	damage      uint32
	probability uint32
}

type LoseItem struct {
	itemId uint32
	chance byte
	x      byte
}
