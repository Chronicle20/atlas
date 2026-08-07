package effect

import (
	"atlas-data/skill/effect/statup"

	"github.com/Chronicle20/atlas/libs/atlas-constants/point"
)

func NewModelBuilder() *ModelBuilder {
	return &ModelBuilder{}
}

type ModelBuilder struct {
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

func (b *ModelBuilder) SetDuration(duration int32) *ModelBuilder {
	b.duration = duration
	return b
}

func (b *ModelBuilder) SetHp(hp uint16) *ModelBuilder {
	b.hp = hp
	return b
}

func (b *ModelBuilder) SetHPRecovery(hpr float64) *ModelBuilder {
	b.hpr = hpr
	return b
}

func (b *ModelBuilder) SetMp(mp uint16) *ModelBuilder {
	b.mp = mp
	return b
}

func (b *ModelBuilder) SetMPRecovery(mpr float64) *ModelBuilder {
	b.mpr = mpr
	return b
}

func (b *ModelBuilder) SetHPCon(hpCon uint16) *ModelBuilder {
	b.hpCon = hpCon
	return b
}

func (b *ModelBuilder) SetMPCon(mpCon uint16) *ModelBuilder {
	b.mpCon = mpCon
	return b
}

func (b *ModelBuilder) SetProp(prop float64) *ModelBuilder {
	b.prop = prop
	return b
}

func (b *ModelBuilder) SetCP(cp uint32) *ModelBuilder {
	b.cp = cp
	return b
}

func (b *ModelBuilder) SetCureAbnormalStatuses(statuses []string) *ModelBuilder {
	b.cureAbnormalStatuses = statuses
	return b
}

func (b *ModelBuilder) SetNuffSkill(nuffSkill uint32) *ModelBuilder {
	b.nuffSkill = nuffSkill
	return b
}

func (b *ModelBuilder) SetMobCount(mobCount uint32) *ModelBuilder {
	b.mobCount = mobCount
	return b
}

func (b *ModelBuilder) SetCooldown(cooldown uint32) *ModelBuilder {
	b.cooldown = cooldown
	return b
}

func (b *ModelBuilder) SetMorphId(morphId uint32) *ModelBuilder {
	b.morphId = morphId
	return b
}

func (b *ModelBuilder) SetGhost(ghost uint32) *ModelBuilder {
	b.ghost = ghost
	return b
}

func (b *ModelBuilder) SetFatigue(fatigue uint32) *ModelBuilder {
	b.fatigue = fatigue
	return b
}

func (b *ModelBuilder) SetRepeatEffect(repeatEffect bool) *ModelBuilder {
	b.repeatEffect = repeatEffect
	return b
}

func (b *ModelBuilder) SetMob(mob uint32) *ModelBuilder {
	b.mob = mob
	return b
}

func (b *ModelBuilder) SetSkill(skill bool) *ModelBuilder {
	b.skill = skill
	return b
}

// Duration returns the effect duration in milliseconds. -1 is the
// "no duration" sentinel (the wz `time` attribute was missing).
// Positive values are ms counts converted from raw wz seconds at
// read time. Consumers should use time.Duration(d) * time.Millisecond.
// See task-054.
func (b *ModelBuilder) Duration() int32 {
	return b.duration
}

func (b *ModelBuilder) SetOverTime(overTime bool) *ModelBuilder {
	b.overTime = overTime
	return b
}

func (b *ModelBuilder) SetWeaponAttack(weaponAttack int16) *ModelBuilder {
	b.weaponAttack = weaponAttack
	return b
}

func (b *ModelBuilder) SetWeaponDefense(weaponDefense int16) *ModelBuilder {
	b.weaponDefense = weaponDefense
	return b
}

func (b *ModelBuilder) SetMagicAttack(magicAttack int16) *ModelBuilder {
	b.magicAttack = magicAttack
	return b
}

func (b *ModelBuilder) SetMagicDefense(magicDefense int16) *ModelBuilder {
	b.magicDefense = magicDefense
	return b
}

func (b *ModelBuilder) SetAccuracy(accuracy int16) *ModelBuilder {
	b.accuracy = accuracy
	return b
}

func (b *ModelBuilder) SetAvoidability(avoidability int16) *ModelBuilder {
	b.avoidability = avoidability
	return b
}

func (b *ModelBuilder) SetSpeed(speed int16) *ModelBuilder {
	b.speed = speed
	return b
}

func (b *ModelBuilder) SetJump(jump int16) *ModelBuilder {
	b.jump = jump
	return b
}

func (b *ModelBuilder) SetBarrier(barrier int32) *ModelBuilder {
	b.barrier = barrier
	return b
}

func (b *ModelBuilder) Barrier() int32 {
	return b.barrier
}

func (b *ModelBuilder) MapProtection() byte {
	return b.mapProtection
}

func (b *ModelBuilder) SetMapProtection(protection byte) *ModelBuilder {
	b.mapProtection = protection
	return b
}

func (b *ModelBuilder) OverTime() bool {
	return b.overTime
}

func (b *ModelBuilder) WeaponAttack() int16 {
	return b.weaponAttack
}

func (b *ModelBuilder) WeaponDefense() int16 {
	return b.weaponDefense
}

func (b *ModelBuilder) MagicAttack() int16 {
	return b.magicAttack
}

func (b *ModelBuilder) MagicDefense() int16 {
	return b.magicDefense
}

func (b *ModelBuilder) Accuracy() int16 {
	return b.accuracy
}

func (b *ModelBuilder) Avoidability() int16 {
	return b.avoidability
}

func (b *ModelBuilder) Speed() int16 {
	return b.speed
}

func (b *ModelBuilder) Jump() int16 {
	return b.jump
}

func (b *ModelBuilder) SetX(x int16) *ModelBuilder {
	b.x = x
	return b
}

func (b *ModelBuilder) SetY(y int16) *ModelBuilder {
	b.y = y
	return b
}

func (b *ModelBuilder) SetLT(p point.Model) *ModelBuilder {
	b.lt = p
	return b
}

func (b *ModelBuilder) SetRB(p point.Model) *ModelBuilder {
	b.rb = p
	return b
}

func (b *ModelBuilder) LT() point.Model {
	return b.lt
}

func (b *ModelBuilder) RB() point.Model {
	return b.rb
}

func (b *ModelBuilder) SetDamage(damage uint32) *ModelBuilder {
	b.damage = damage
	return b
}

func (b *ModelBuilder) SetFixDamage(damage int32) *ModelBuilder {
	b.fixDamage = damage
	return b
}

func (b *ModelBuilder) SetAttackCount(count uint32) *ModelBuilder {
	b.attackCount = count
	return b
}

func (b *ModelBuilder) SetBulletCount(count uint16) *ModelBuilder {
	b.bulletCount = count
	return b
}

func (b *ModelBuilder) SetBulletConsume(consume uint16) *ModelBuilder {
	b.bulletConsume = consume
	return b
}

func (b *ModelBuilder) SetMoneyConsume(consume uint32) *ModelBuilder {
	b.moneyCon = consume
	return b
}

func (b *ModelBuilder) SetItemConsume(consume uint32) *ModelBuilder {
	b.itemCon = consume
	return b
}

func (b *ModelBuilder) SetItemConsumeNumber(number uint32) *ModelBuilder {
	b.itemConNo = number
	return b
}

func (b *ModelBuilder) SetMoveTo(moveTo int32) *ModelBuilder {
	b.moveTo = moveTo
	return b
}

func (b *ModelBuilder) X() int16 {
	return b.x
}

func (b *ModelBuilder) Damage() uint32 {
	return b.damage
}

func (b *ModelBuilder) Y() int16 {
	return b.y
}

func (b *ModelBuilder) Prop() float64 {
	return b.prop
}

func (b *ModelBuilder) MorphId() uint32 {
	return b.morphId
}

func (b *ModelBuilder) SetMonsterStatus(ms map[string]uint32) *ModelBuilder {
	b.monsterStatus = ms
	return b
}

func (b *ModelBuilder) SetStatups(statups []statup.RestModel) *ModelBuilder {
	b.statups = statups
	return b
}

func (b *ModelBuilder) Build() RestModel {
	var ltPtr *PointRestModel
	if b.lt.X() != 0 || b.lt.Y() != 0 {
		ltPtr = &PointRestModel{X: int16(b.lt.X()), Y: int16(b.lt.Y())}
	}
	var rbPtr *PointRestModel
	if b.rb.X() != 0 || b.rb.Y() != 0 {
		rbPtr = &PointRestModel{X: int16(b.rb.X()), Y: int16(b.rb.Y())}
	}
	return RestModel{
		WeaponAttack:         b.weaponAttack,
		MagicAttack:          b.magicAttack,
		WeaponDefense:        b.weaponDefense,
		MagicDefense:         b.magicDefense,
		Accuracy:             b.accuracy,
		Avoidability:         b.avoidability,
		Speed:                b.speed,
		Jump:                 b.jump,
		Hp:                   b.hp,
		Mp:                   b.mp,
		HPR:                  b.hpr,
		MPR:                  b.mpr,
		MHPRRate:             b.mhprRate,
		MMPRRate:             b.mmprRate,
		MobSkill:             b.mobSkill,
		MobSkillLevel:        b.mobSkillLevel,
		MHPR:                 b.mhpR,
		MMPR:                 b.mmpR,
		HPConsume:            b.hpCon,
		MPConsume:            b.mpCon,
		Duration:             b.duration,
		Target:               b.target,
		Barrier:              b.barrier,
		Mob:                  b.mob,
		OverTime:             b.overTime, // Kept lowercase `b.overTime` as per request
		RepeatEffect:         b.repeatEffect,
		MoveTo:               b.moveTo,
		CP:                   b.cp,
		NuffSkill:            b.nuffSkill,
		Skill:                b.skill,
		X:                    b.x,
		Y:                    b.y,
		MobCount:             b.mobCount,
		MoneyConsume:         b.moneyCon,
		Cooldown:             b.cooldown,
		MorphId:              b.morphId,
		Ghost:                b.ghost,
		Fatigue:              b.fatigue,
		Berserk:              b.berserk,
		Booster:              b.booster,
		Prop:                 b.prop,
		ItemConsume:          b.itemCon,
		ItemConsumeAmount:    b.itemConNo,
		Damage:               b.damage,
		AttackCount:          b.attackCount,
		FixDamage:            b.fixDamage,
		BulletCount:          b.bulletCount,
		BulletConsume:        b.bulletConsume,
		MapProtection:        b.mapProtection,
		CureAbnormalStatuses: b.cureAbnormalStatuses,
		Statups:              b.statups,
		MonsterStatus:        b.monsterStatus,
		LT:                   ltPtr,
		RB:                   rbPtr,
		Range:                b.rangeValue,
		Mastery:              b.mastery,
		Z:                    b.z,
		Dot:                  b.dot,
		Cr:                   b.cr,
		DotInterval:          b.dotInterval,
		DotTime:              b.dotTime,
		DamR:                 b.damR,
		CriticaldamageMin:    b.criticaldamageMin,
		V:                    b.v,
		IgnoreMobpdpR:        b.ignoreMobpdpR,
		Epad:                 b.epad,
		W:                    b.w,
		U:                    b.u,
		Epdd:                 b.epdd,
		Emdd:                 b.emdd,
		SelfDestruction:      b.selfDestruction,
		AsrR:                 b.asrR,
		T:                    b.t,
		Er:                   b.er,
		PddR:                 b.pddR,
		TerR:                 b.terR,
		MadX:                 b.madX,
		SubProp:              b.subProp,
		Emhp:                 b.emhp,
		CriticaldamageMax:    b.criticaldamageMax,
		ExpR:                 b.expR,
		Emmp:                 b.emmp,
		ConsumeItemId:        b.consumeItemId,
		MddR:                 b.mddR,
		SubTime:              b.subTime,
		PadX:                 b.padX,
		MesoR:                b.mesoR,
	}
}

func (b *ModelBuilder) SetMobSkill(mobSkill uint16) *ModelBuilder {
	b.mobSkill = mobSkill
	return b
}

func (b *ModelBuilder) SetMobSkillLevel(skillLevel uint16) *ModelBuilder {
	b.mobSkillLevel = skillLevel
	return b
}

func (b *ModelBuilder) SetTarget(target uint32) *ModelBuilder {
	b.target = target
	return b
}

func (b *ModelBuilder) SetRange(v int32) *ModelBuilder             { b.rangeValue = v; return b }
func (b *ModelBuilder) SetMastery(v int32) *ModelBuilder           { b.mastery = v; return b }
func (b *ModelBuilder) SetZ(v int32) *ModelBuilder                 { b.z = v; return b }
func (b *ModelBuilder) SetDot(v int32) *ModelBuilder               { b.dot = v; return b }
func (b *ModelBuilder) SetCr(v int32) *ModelBuilder                { b.cr = v; return b }
func (b *ModelBuilder) SetDotInterval(v int32) *ModelBuilder       { b.dotInterval = v; return b }
func (b *ModelBuilder) SetDotTime(v int32) *ModelBuilder           { b.dotTime = v; return b }
func (b *ModelBuilder) SetDamR(v int32) *ModelBuilder              { b.damR = v; return b }
func (b *ModelBuilder) SetCriticaldamageMin(v int32) *ModelBuilder { b.criticaldamageMin = v; return b }
func (b *ModelBuilder) SetMHPRRate(v uint16) *ModelBuilder         { b.mhprRate = v; return b }
func (b *ModelBuilder) SetV(v int32) *ModelBuilder                 { b.v = v; return b }
func (b *ModelBuilder) SetIgnoreMobpdpR(v int32) *ModelBuilder     { b.ignoreMobpdpR = v; return b }
func (b *ModelBuilder) SetEpad(v int32) *ModelBuilder              { b.epad = v; return b }
func (b *ModelBuilder) SetW(v int32) *ModelBuilder                 { b.w = v; return b }
func (b *ModelBuilder) SetU(v int32) *ModelBuilder                 { b.u = v; return b }
func (b *ModelBuilder) SetEpdd(v int32) *ModelBuilder              { b.epdd = v; return b }
func (b *ModelBuilder) SetEmdd(v int32) *ModelBuilder              { b.emdd = v; return b }
func (b *ModelBuilder) SetSelfDestruction(v int32) *ModelBuilder   { b.selfDestruction = v; return b }
func (b *ModelBuilder) SetAsrR(v int32) *ModelBuilder              { b.asrR = v; return b }
func (b *ModelBuilder) SetMMPRRate(v uint16) *ModelBuilder         { b.mmprRate = v; return b }
func (b *ModelBuilder) SetT(v int32) *ModelBuilder                 { b.t = v; return b }
func (b *ModelBuilder) SetEr(v int32) *ModelBuilder                { b.er = v; return b }
func (b *ModelBuilder) SetPddR(v int32) *ModelBuilder              { b.pddR = v; return b }
func (b *ModelBuilder) SetTerR(v int32) *ModelBuilder              { b.terR = v; return b }
func (b *ModelBuilder) SetMadX(v int32) *ModelBuilder              { b.madX = v; return b }
func (b *ModelBuilder) SetSubProp(v int32) *ModelBuilder           { b.subProp = v; return b }
func (b *ModelBuilder) SetEmhp(v int32) *ModelBuilder              { b.emhp = v; return b }
func (b *ModelBuilder) SetCriticaldamageMax(v int32) *ModelBuilder { b.criticaldamageMax = v; return b }
func (b *ModelBuilder) SetExpR(v int32) *ModelBuilder              { b.expR = v; return b }
func (b *ModelBuilder) SetEmmp(v int32) *ModelBuilder              { b.emmp = v; return b }

// SetConsumeItemId sets wz `common/itemConsume`. See RestModel.ConsumeItemId:
// this is NOT the same key as `itemCon`, which SetItemConsume carries.
func (b *ModelBuilder) SetConsumeItemId(v int32) *ModelBuilder { b.consumeItemId = v; return b }

func (b *ModelBuilder) SetMddR(v int32) *ModelBuilder    { b.mddR = v; return b }
func (b *ModelBuilder) SetSubTime(v int32) *ModelBuilder { b.subTime = v; return b }
func (b *ModelBuilder) SetPadX(v int32) *ModelBuilder    { b.padX = v; return b }
func (b *ModelBuilder) SetMesoR(v int32) *ModelBuilder   { b.mesoR = v; return b }
