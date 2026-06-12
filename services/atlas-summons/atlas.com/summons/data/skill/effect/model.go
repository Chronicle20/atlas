package effect

// Model is atlas-summons' projection of a skill effect. It exposes only the
// attributes the summon lifecycle needs. Unlike the channel-side effect model
// it also surfaces weaponAttack/magicAttack (the summon damage ceiling and
// Beholder snapshots need them).
type Model struct {
	weaponAttack  int16
	magicAttack   int16
	hp            uint16
	duration      int32
	x             int16
	y             int16
	prop          float64
	monsterStatus map[string]uint32
	statups       []StatChange
}

// StatChange is one {stat-type, amount} buff delta as produced by atlas-data's
// skill-effect `statups` array (e.g. {WEAPON_DEFENSE, 60}). The Beholder hex
// snapshot sources its buff deltas from these. Kept local to the effect package
// (the summon package's StatChange mirrors it; the processor maps between them
// to avoid an import cycle, since effect is imported by summon).
type StatChange struct {
	Type   string
	Amount int32
}

// WeaponAttack returns the effect's `weaponAttack` attribute.
func (m Model) WeaponAttack() int16 { return m.weaponAttack }

// MagicAttack returns the effect's `magicAttack` attribute.
func (m Model) MagicAttack() int16 { return m.magicAttack }

// Hp returns the effect's `hp` attribute (the heal amount). For AURA_OF_BEHOLDER
// (1320008) this is the per-tick HP restored to the owner. atlas-data reads it
// from the WZ `hp` node (reader.go SetHp). Heal amounts top out well under the
// int16 range (300 at max level per Skill.wz/132.img), so the cast is safe.
func (m Model) Hp() int16 { return int16(m.hp) }

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

// Statups returns the buff stat deltas (the `statups` array) the effect grants.
// For HEX_OF_BEHOLDER (1320009) these are the periodic owner buff bonuses
// (WEAPON_DEFENSE/MAGIC_DEFENSE/ACCURACY per Skill.wz/132.img). atlas-data
// derives the array from the WZ pdd/mdd/acc/... nodes via produceBuffStatAmount.
func (m Model) Statups() []StatChange { return m.statups }
