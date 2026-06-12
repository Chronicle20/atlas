package effect

// Model is atlas-summons' projection of a skill effect. It exposes only the
// attributes the summon lifecycle needs. Unlike the channel-side effect model
// it also surfaces weaponAttack/magicAttack (the summon damage ceiling and
// Beholder snapshots need them).
type Model struct {
	weaponAttack  int16
	magicAttack   int16
	duration      int32
	x             int16
	y             int16
	prop          float64
	monsterStatus map[string]uint32
}

// WeaponAttack returns the effect's `weaponAttack` attribute.
func (m Model) WeaponAttack() int16 { return m.weaponAttack }

// MagicAttack returns the effect's `magicAttack` attribute.
func (m Model) MagicAttack() int16 { return m.magicAttack }

// Duration returns the effect duration in milliseconds. -1 is the "no duration"
// sentinel. Consumers should use time.Duration(d) * time.Millisecond.
func (m Model) Duration() int32 { return m.duration }

// X returns the integer X attribute. For summon puppets it is the puppet HP.
func (m Model) X() int16 { return m.x }

// Y returns the integer Y attribute.
func (m Model) Y() int16 { return m.y }

// Prop returns the proc-chance attribute (0.0-1.0).
func (m Model) Prop() float64 { return m.prop }

// MonsterStatus returns the monster status-effect map applied by the skill
// (e.g. stun/freeze keyed by status name).
func (m Model) MonsterStatus() map[string]uint32 { return m.monsterStatus }
