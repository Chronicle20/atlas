package information

type Model struct {
	hp              uint32
	mp              uint32
	boss            bool
	undead          bool
	friendly        bool
	firstAttack     bool
	weaponAttack    uint32
	dropPeriod      uint32
	resistances     map[string]string
	animationTimes  map[string]uint32
	skills          []Skill
	revives         []uint32
	banish          Banish
	attacks         []AttackInfo
	selfDestruction SelfDestruction
	hpRecovery      uint32
	mpRecovery      uint32
}

// SelfDestruction is a monster's WZ `selfDestruction` block. Present is
// carried explicitly because hp == -1 is BOTH "no HP threshold" and the
// absent-block default (task-253 design §2.6): without it, a timer-driven
// mob and an ordinary monster are indistinguishable.
type SelfDestruction struct {
	present     bool
	action      byte
	removeAfter int32
	hp          int32
}

func NewSelfDestruction(present bool, action byte, removeAfter int32, hp int32) SelfDestruction {
	return SelfDestruction{present: present, action: action, removeAfter: removeAfter, hp: hp}
}

func (s SelfDestruction) Present() bool      { return s.present }
func (s SelfDestruction) Action() byte       { return s.action }
func (s SelfDestruction) RemoveAfter() int32 { return s.removeAfter }
func (s SelfDestruction) Hp() int32          { return s.hp }

// OnHpThreshold reports the HP-driven mechanic: the mob detonates once its
// post-damage HP falls to or below Hp().
func (s SelfDestruction) OnHpThreshold() bool { return s.present && s.hp > -1 }

// OnTimer reports the timer-driven mechanic: the mob detonates RemoveAfter
// seconds after spawn, with no HP predicate.
func (s SelfDestruction) OnTimer() bool { return s.present && s.hp <= -1 }

func (m Model) SelfDestruction() SelfDestruction {
	return m.selfDestruction
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

type AttackInfo struct {
	Pos         uint8
	ConMP       int32
	AttackAfter int32
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

func (m Model) Attacks() []AttackInfo {
	return m.attacks
}

func (m Model) Revives() []uint32 {
	return m.revives
}

func (m Model) Banish() Banish {
	return m.banish
}

// IsImmuneToElement checks if the monster is immune to a given element.
// Resistance values: "1"=immune, "2"=strong, "3"=normal, "4"=weak
func (m Model) Friendly() bool {
	return m.friendly
}

func (m Model) WeaponAttack() uint32 {
	return m.weaponAttack
}

func (m Model) DropPeriod() uint32 {
	return m.dropPeriod
}

// IsImmuneToElement checks if the monster is immune to a given element.
// Resistance values: "1"=immune, "2"=strong, "3"=normal, "4"=weak
func (m Model) IsImmuneToElement(element string) bool {
	if r, ok := m.resistances[element]; ok {
		return r == "1"
	}
	return false
}

func (m Model) HpRecovery() uint32 {
	return m.hpRecovery
}

func (m Model) MpRecovery() uint32 {
	return m.mpRecovery
}

// FirstAttack reports whether the template is aggressive — Mob/<id>.img/info/firstAttack.
// This is the gate that separates a genuinely aggressive mob from one that
// merely picks up drops: CMob::ApplyControl fires for bPickUpDrop templates too.
func (m Model) FirstAttack() bool {
	return m.firstAttack
}
