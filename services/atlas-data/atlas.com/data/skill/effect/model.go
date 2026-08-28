package effect

import (
	"atlas-data/skill/effect/statup"

	"github.com/Chronicle20/atlas/libs/atlas-constants/point"
)

// Model is the immutable domain representation of a single skill effect
// (one `level` entry, or one node synthesized from `common` — see
// skill.getEffect, the single shared implementation for both read paths).
// It is produced by Builder.Build() and converted to the wire shape via
// Transform. Private fields + getters, per the project's immutable-model
// convention; construction goes exclusively through Builder in
// builder.go.
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
	overTime             bool
	repeatEffect         bool
	moveTo               int32
	cp                   uint32
	nuffSkill            uint32
	skill                bool
	x                    int16
	y                    int16
	mobCount             uint32
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
	lt                   point.Model
	rb                   point.Model
	bulletCount          uint16
	bulletConsume        uint16
	mapProtection        byte
	cureAbnormalStatuses []string
	statups              []statup.RestModel
	monsterStatus        map[string]uint32

	rangeValue        int32
	mastery           int32
	z                 int32
	dot               int32
	cr                int32
	dotInterval       int32
	dotTime           int32
	damR              int32
	criticaldamageMin int32
	v                 int32
	ignoreMobpdpR     int32
	epad              int32
	w                 int32
	u                 int32
	epdd              int32
	emdd              int32
	selfDestruction   int32
	asrR              int32
	t                 int32
	er                int32
	pddR              int32
	terR              int32
	madX              int32
	subProp           int32
	emhp              int32
	criticaldamageMax int32
	expR              int32
	emmp              int32
	consumeItemId     int32
	mddR              int32
	subTime           int32
	padX              int32
	mesoR             int32
}

func (m Model) WeaponAttack() int16  { return m.weaponAttack }
func (m Model) MagicAttack() int16   { return m.magicAttack }
func (m Model) WeaponDefense() int16 { return m.weaponDefense }
func (m Model) MagicDefense() int16  { return m.magicDefense }
func (m Model) Accuracy() int16      { return m.accuracy }
func (m Model) Avoidability() int16  { return m.avoidability }
func (m Model) Speed() int16         { return m.speed }
func (m Model) Jump() int16          { return m.jump }
func (m Model) Hp() uint16           { return m.hp }
func (m Model) Mp() uint16           { return m.mp }
func (m Model) HPR() float64         { return m.hpr }
func (m Model) MPR() float64         { return m.mpr }
func (m Model) MHPRRate() uint16     { return m.mhprRate }
func (m Model) MMPRRate() uint16     { return m.mmprRate }
func (m Model) MobSkill() uint16     { return m.mobSkill }
func (m Model) MobSkillLevel() uint16 {
	return m.mobSkillLevel
}
func (m Model) MHPR() byte                { return m.mhpR }
func (m Model) MMPR() byte                { return m.mmpR }
func (m Model) HPConsume() uint16         { return m.hpCon }
func (m Model) MPConsume() uint16         { return m.mpCon }
func (m Model) Duration() int32           { return m.duration }
func (m Model) Target() uint32            { return m.target }
func (m Model) Barrier() int32            { return m.barrier }
func (m Model) Mob() uint32               { return m.mob }
func (m Model) OverTime() bool            { return m.overTime }
func (m Model) RepeatEffect() bool        { return m.repeatEffect }
func (m Model) MoveTo() int32             { return m.moveTo }
func (m Model) CP() uint32                { return m.cp }
func (m Model) NuffSkill() uint32         { return m.nuffSkill }
func (m Model) Skill() bool               { return m.skill }
func (m Model) X() int16                  { return m.x }
func (m Model) Y() int16                  { return m.y }
func (m Model) MobCount() uint32          { return m.mobCount }
func (m Model) MoneyConsume() uint32      { return m.moneyCon }
func (m Model) Cooldown() uint32          { return m.cooldown }
func (m Model) MorphId() uint32           { return m.morphId }
func (m Model) Ghost() uint32             { return m.ghost }
func (m Model) Fatigue() uint32           { return m.fatigue }
func (m Model) Berserk() uint32           { return m.berserk }
func (m Model) Booster() uint32           { return m.booster }
func (m Model) Prop() float64             { return m.prop }
func (m Model) ItemConsume() uint32       { return m.itemCon }
func (m Model) ItemConsumeAmount() uint32 { return m.itemConNo }
func (m Model) Damage() uint32            { return m.damage }
func (m Model) AttackCount() uint32       { return m.attackCount }
func (m Model) FixDamage() int32          { return m.fixDamage }
func (m Model) LT() point.Model           { return m.lt }
func (m Model) RB() point.Model           { return m.rb }
func (m Model) BulletCount() uint16       { return m.bulletCount }
func (m Model) BulletConsume() uint16     { return m.bulletConsume }
func (m Model) MapProtection() byte       { return m.mapProtection }
func (m Model) CureAbnormalStatuses() []string {
	return m.cureAbnormalStatuses
}
func (m Model) Statups() []statup.RestModel { return m.statups }
func (m Model) MonsterStatus() map[string]uint32 {
	return m.monsterStatus
}

func (m Model) Range() int32             { return m.rangeValue }
func (m Model) Mastery() int32           { return m.mastery }
func (m Model) Z() int32                 { return m.z }
func (m Model) Dot() int32               { return m.dot }
func (m Model) Cr() int32                { return m.cr }
func (m Model) DotInterval() int32       { return m.dotInterval }
func (m Model) DotTime() int32           { return m.dotTime }
func (m Model) DamR() int32              { return m.damR }
func (m Model) CriticaldamageMin() int32 { return m.criticaldamageMin }
func (m Model) V() int32                 { return m.v }
func (m Model) IgnoreMobpdpR() int32     { return m.ignoreMobpdpR }
func (m Model) Epad() int32              { return m.epad }
func (m Model) W() int32                 { return m.w }
func (m Model) U() int32                 { return m.u }
func (m Model) Epdd() int32              { return m.epdd }
func (m Model) Emdd() int32              { return m.emdd }
func (m Model) SelfDestruction() int32   { return m.selfDestruction }
func (m Model) AsrR() int32              { return m.asrR }
func (m Model) T() int32                 { return m.t }
func (m Model) Er() int32                { return m.er }
func (m Model) PddR() int32              { return m.pddR }
func (m Model) TerR() int32              { return m.terR }
func (m Model) MadX() int32              { return m.madX }
func (m Model) SubProp() int32           { return m.subProp }
func (m Model) Emhp() int32              { return m.emhp }
func (m Model) CriticaldamageMax() int32 { return m.criticaldamageMax }
func (m Model) ExpR() int32              { return m.expR }
func (m Model) Emmp() int32              { return m.emmp }
func (m Model) ConsumeItemId() int32     { return m.consumeItemId }
func (m Model) MddR() int32              { return m.mddR }
func (m Model) SubTime() int32           { return m.subTime }
func (m Model) PadX() int32              { return m.padX }
func (m Model) MesoR() int32             { return m.mesoR }
