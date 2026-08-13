package effect

import (
	"atlas-channel/data/skill/effect/statup"

	"github.com/Chronicle20/atlas/libs/atlas-constants/point"
)

type Model struct {
	weaponAttack         int16
	magicAttack          int16
	weaponDefense        int16
	magicDefense         int16
	accuracy             int16
	avoidability         int16
	speed                int16
	jump                 int16
	hp                   uint16
	mp                   uint16
	hpr                  float64
	mpr                  float64
	mhprRate             uint16
	mmprRate             uint16
	mobSkill             uint16
	mobSkillLevel        uint16
	mhpR                 byte
	mmpR                 byte
	hpCon                uint16
	mpCon                uint16
	duration             int32
	target               uint32
	barrier              int32
	mob                  uint32
	overtime             bool
	repeatEffect         bool
	moveTo               int32
	cp                   uint32
	nuffSkill            uint32
	skill                bool
	x                    int16
	y                    int16
	mobCount             uint32
	rangeValue           int32
	moneyCon             uint32
	cooldown             uint32
	morphId              uint32
	ghost                uint32
	fatigue              uint32
	berserk              uint32
	booster              uint32
	prop                 float64
	itemCon              uint32
	itemConNo            uint32
	damage               uint32
	attackCount          uint32
	fixDamage            int32
	dot                  int32
	dotInterval          int32
	dotTime              int32
	lt                   point.Model
	rb                   point.Model
	bulletCount          uint16
	bulletConsume        uint16
	mapProtection        byte
	cureAbnormalStatuses []string
	statups              []statup.Model
	monsterStatus        map[string]uint32
}

func (m Model) StatUps() []statup.Model {
	return m.statups
}

func (m Model) HPConsume() uint16 {
	return m.hpCon
}

func (m Model) MPConsume() uint16 {
	return m.mpCon
}

// Duration returns the effect duration in milliseconds. -1 is the
// "no duration" sentinel. Consumers should use
// time.Duration(d) * time.Millisecond. See task-054.
func (m Model) Duration() int32 {
	return m.duration
}

func (m Model) Cooldown() uint32 {
	return m.cooldown
}

func (m Model) ItemConsume() uint32 {
	return m.itemCon
}

func (m Model) ItemConsumeAmount() uint32 {
	return m.itemConNo
}

func (m Model) MonsterStatus() map[string]uint32 {
	return m.monsterStatus
}

func (m Model) CureAbnormalStatuses() []string {
	return m.cureAbnormalStatuses
}

// BulletConsume returns the WZ `bulletConsume` attribute — the one-time star
// batch charged when casting Shadow Stars (200 in reference data). Distinct from
// BulletCount (per-attack projectile count).
func (m Model) BulletConsume() uint16 {
	return m.bulletConsume
}

// BulletCount returns the WZ `bulletCount` attribute — the number of projectiles
// spent per attack (e.g. Lucky Seven 2, Triple Throw 3). Distinct from
// BulletConsume (the one-time Shadow Stars activation cost).
func (m Model) BulletCount() uint16 {
	return m.bulletCount
}

// HP exposes the skill's `hp` percentage attribute (used by Heal's
// amount formula).
func (m Model) HP() uint16 {
	return m.hp
}

// LT returns the skill effect's left-top rectangle corner relative to
// the caster's position. A zero-valued point.Model indicates the source
// data did not provide a rectangle; callers should treat that as a
// caster-only sentinel rather than infinite range.
func (m Model) LT() point.Model {
	return m.lt
}

// RB returns the skill effect's right-bottom rectangle corner.
func (m Model) RB() point.Model {
	return m.rb
}

// Dot returns the raw per-tick damage-over-time magnitude from WZ `dot`.
// It is zero on every provisioned version -- the node does not exist in any
// pre-v1.17 Skill.wz (task-200 design §2.1). Callers that need a DoT
// magnitude must not assume this is populated.
func (m Model) Dot() int32 {
	return m.dot
}

// DotInterval returns the DoT tick interval in MILLISECONDS (atlas-data
// converts from WZ seconds). Zero on every provisioned version.
func (m Model) DotInterval() int32 {
	return m.dotInterval
}

// DotTime returns the DoT lifetime in MILLISECONDS (atlas-data converts from
// WZ seconds). Zero on every provisioned version.
func (m Model) DotTime() int32 {
	return m.dotTime
}

// MobCount returns the cap on monsters affected by an AoE monster-buff
// skill (e.g., Priest Doom's 6-mob target ceiling). Zero means "no cap".
func (m Model) MobCount() uint32 {
	return m.mobCount
}

// Range returns the skill's WZ `range` attribute in map pixels. For Monster
// Magnet (which carries no lt/rb) this is the only WZ input to the server-side
// target region: the client selects candidates through
// CMobPool::CheckMobInTrapezoid out to this distance. Zero means the attribute
// is absent, in which case range-based selection must fall back to cap-only.
func (m Model) Range() int32 {
	return m.rangeValue
}

// Prop returns the proc-chance attribute (0.0–1.0). Used by passives like
// MP Eater to roll on each affected monster, and by monster-buff skills
// like Priest Doom as a per-target probability gate. Zero means "never
// apply"; values at or above 1 mean "always apply".
func (m Model) Prop() float64 {
	return m.prop
}

// X returns the integer X attribute (often used as a percent or
// multiplier; for MP Eater it is the absorb percent of monster MaxMp).
func (m Model) X() int16 {
	return m.x
}

// AttackCount returns the skill's attackCount attribute. For Meso Explosion
// (4211006) it is the maximum number of drops one attack may detonate
// (10–20 by level; task-150 design §3).
func (m Model) AttackCount() uint32 {
	return m.attackCount
}

// Y returns the integer Y attribute (for MP Recovery it is the percent of
// the HP loss returned as MP).
func (m Model) Y() int16 {
	return m.y
}
