package information

type Model struct {
	hp             uint32
	mp             uint32
	boss           bool
	undead         bool
	resistances    map[string]string
	animationTimes map[string]uint32
	skills         []Skill
	revives        []uint32
	banish         Banish
}

type Skill struct {
	Id    uint32
	Level uint32
}

type Banish struct {
	Message    string
	MapId      uint32
	PortalName string
}

func (m Model) Hp() uint32 {
	return m.hp
}

func (m Model) Mp() uint32 {
	return m.mp
}

func (m Model) Boss() bool {
	return m.boss
}

func (m Model) Undead() bool {
	return m.undead
}

func (m Model) Resistances() map[string]string {
	return m.resistances
}

func (m Model) AnimationTimes() map[string]uint32 {
	return m.animationTimes
}

func (m Model) Skills() []Skill {
	return m.skills
}

func (m Model) Revives() []uint32 {
	return m.revives
}

func (m Model) Banish() Banish {
	return m.banish
}

// IsImmuneToElement checks if the monster is immune to a given element.
// Resistance values: "1"=immune, "2"=strong, "3"=normal, "4"=weak
func (m Model) IsImmuneToElement(element string) bool {
	if r, ok := m.resistances[element]; ok {
		return r == "1"
	}
	return false
}
