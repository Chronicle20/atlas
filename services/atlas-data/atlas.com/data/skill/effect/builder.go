package effect

import (
	"atlas-data/skill/effect/statup"

	"github.com/Chronicle20/atlas/libs/atlas-constants/point"
)

func NewBuilder() *Builder {
	return &Builder{}
}

type Builder struct {
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

func (b *Builder) SetDuration(duration int32) *Builder {
	b.duration = duration
	return b
}

func (b *Builder) SetHp(hp uint16) *Builder {
	b.hp = hp
	return b
}

func (b *Builder) SetHPRecovery(hpr float64) *Builder {
	b.hpr = hpr
	return b
}

func (b *Builder) SetMp(mp uint16) *Builder {
	b.mp = mp
	return b
}

func (b *Builder) SetMPRecovery(mpr float64) *Builder {
	b.mpr = mpr
	return b
}

func (b *Builder) SetHPCon(hpCon uint16) *Builder {
	b.hpCon = hpCon
	return b
}

func (b *Builder) SetMPCon(mpCon uint16) *Builder {
	b.mpCon = mpCon
	return b
}

func (b *Builder) SetProp(prop float64) *Builder {
	b.prop = prop
	return b
}

func (b *Builder) SetCP(cp uint32) *Builder {
	b.cp = cp
	return b
}

func (b *Builder) SetCureAbnormalStatuses(statuses []string) *Builder {
	b.cureAbnormalStatuses = statuses
	return b
}

func (b *Builder) SetNuffSkill(nuffSkill uint32) *Builder {
	b.nuffSkill = nuffSkill
	return b
}

func (b *Builder) SetMobCount(mobCount uint32) *Builder {
	b.mobCount = mobCount
	return b
}

func (b *Builder) SetCooldown(cooldown uint32) *Builder {
	b.cooldown = cooldown
	return b
}

func (b *Builder) SetMorphId(morphId uint32) *Builder {
	b.morphId = morphId
	return b
}

func (b *Builder) SetGhost(ghost uint32) *Builder {
	b.ghost = ghost
	return b
}

func (b *Builder) SetFatigue(fatigue uint32) *Builder {
	b.fatigue = fatigue
	return b
}

func (b *Builder) SetRepeatEffect(repeatEffect bool) *Builder {
	b.repeatEffect = repeatEffect
	return b
}

func (b *Builder) SetMob(mob uint32) *Builder {
	b.mob = mob
	return b
}

func (b *Builder) SetSkill(skill bool) *Builder {
	b.skill = skill
	return b
}

// Duration returns the effect duration in milliseconds. -1 is the
// "no duration" sentinel (the wz `time` attribute was missing).
// Positive values are ms counts converted from raw wz seconds at
// read time. Consumers should use time.Duration(d) * time.Millisecond.
// See task-054.
func (b *Builder) Duration() int32 {
	return b.duration
}

func (b *Builder) SetOverTime(overTime bool) *Builder {
	b.overTime = overTime
	return b
}

func (b *Builder) SetWeaponAttack(weaponAttack int16) *Builder {
	b.weaponAttack = weaponAttack
	return b
}

func (b *Builder) SetWeaponDefense(weaponDefense int16) *Builder {
	b.weaponDefense = weaponDefense
	return b
}

func (b *Builder) SetMagicAttack(magicAttack int16) *Builder {
	b.magicAttack = magicAttack
	return b
}

func (b *Builder) SetMagicDefense(magicDefense int16) *Builder {
	b.magicDefense = magicDefense
	return b
}

func (b *Builder) SetAccuracy(accuracy int16) *Builder {
	b.accuracy = accuracy
	return b
}

func (b *Builder) SetAvoidability(avoidability int16) *Builder {
	b.avoidability = avoidability
	return b
}

func (b *Builder) SetSpeed(speed int16) *Builder {
	b.speed = speed
	return b
}

func (b *Builder) SetJump(jump int16) *Builder {
	b.jump = jump
	return b
}

func (b *Builder) SetBarrier(barrier int32) *Builder {
	b.barrier = barrier
	return b
}

func (b *Builder) Barrier() int32 {
	return b.barrier
}

func (b *Builder) MapProtection() byte {
	return b.mapProtection
}

func (b *Builder) SetMapProtection(protection byte) *Builder {
	b.mapProtection = protection
	return b
}

func (b *Builder) OverTime() bool {
	return b.overTime
}

func (b *Builder) WeaponAttack() int16 {
	return b.weaponAttack
}

func (b *Builder) WeaponDefense() int16 {
	return b.weaponDefense
}

func (b *Builder) MagicAttack() int16 {
	return b.magicAttack
}

func (b *Builder) MagicDefense() int16 {
	return b.magicDefense
}

func (b *Builder) Accuracy() int16 {
	return b.accuracy
}

func (b *Builder) Avoidability() int16 {
	return b.avoidability
}

func (b *Builder) Speed() int16 {
	return b.speed
}

func (b *Builder) Jump() int16 {
	return b.jump
}

func (b *Builder) SetX(x int16) *Builder {
	b.x = x
	return b
}

func (b *Builder) SetY(y int16) *Builder {
	b.y = y
	return b
}

func (b *Builder) SetLT(p point.Model) *Builder {
	b.lt = p
	return b
}

func (b *Builder) SetRB(p point.Model) *Builder {
	b.rb = p
	return b
}

func (b *Builder) LT() point.Model {
	return b.lt
}

func (b *Builder) RB() point.Model {
	return b.rb
}

func (b *Builder) SetDamage(damage uint32) *Builder {
	b.damage = damage
	return b
}

func (b *Builder) SetFixDamage(damage int32) *Builder {
	b.fixDamage = damage
	return b
}

func (b *Builder) SetAttackCount(count uint32) *Builder {
	b.attackCount = count
	return b
}

func (b *Builder) SetBulletCount(count uint16) *Builder {
	b.bulletCount = count
	return b
}

func (b *Builder) SetBulletConsume(consume uint16) *Builder {
	b.bulletConsume = consume
	return b
}

func (b *Builder) SetMoneyConsume(consume uint32) *Builder {
	b.moneyCon = consume
	return b
}

func (b *Builder) SetItemConsume(consume uint32) *Builder {
	b.itemCon = consume
	return b
}

func (b *Builder) SetItemConsumeNumber(number uint32) *Builder {
	b.itemConNo = number
	return b
}

func (b *Builder) SetMoveTo(moveTo int32) *Builder {
	b.moveTo = moveTo
	return b
}

func (b *Builder) X() int16 {
	return b.x
}

func (b *Builder) Damage() uint32 {
	return b.damage
}

func (b *Builder) Y() int16 {
	return b.y
}

func (b *Builder) Prop() float64 {
	return b.prop
}

func (b *Builder) MorphId() uint32 {
	return b.morphId
}

func (b *Builder) SetMonsterStatus(ms map[string]uint32) *Builder {
	b.monsterStatus = ms
	return b
}

func (b *Builder) SetStatups(statups []statup.RestModel) *Builder {
	b.statups = statups
	return b
}

// Build materializes the immutable domain Model from the builder's
// accumulated state. Wire-shape concerns (RestModel, the LT/RB
// nil-when-zero pointer rule) live in Transform, not here.
func (b *Builder) Build() Model {
	return Model{
		weaponAttack:         b.weaponAttack,
		magicAttack:          b.magicAttack,
		weaponDefense:        b.weaponDefense,
		magicDefense:         b.magicDefense,
		accuracy:             b.accuracy,
		avoidability:         b.avoidability,
		speed:                b.speed,
		jump:                 b.jump,
		hp:                   b.hp,
		mp:                   b.mp,
		hpr:                  b.hpr,
		mpr:                  b.mpr,
		mhprRate:             b.mhprRate,
		mmprRate:             b.mmprRate,
		mobSkill:             b.mobSkill,
		mobSkillLevel:        b.mobSkillLevel,
		mhpR:                 b.mhpR,
		mmpR:                 b.mmpR,
		hpCon:                b.hpCon,
		mpCon:                b.mpCon,
		duration:             b.duration,
		target:               b.target,
		barrier:              b.barrier,
		mob:                  b.mob,
		overTime:             b.overTime,
		repeatEffect:         b.repeatEffect,
		moveTo:               b.moveTo,
		cp:                   b.cp,
		nuffSkill:            b.nuffSkill,
		skill:                b.skill,
		x:                    b.x,
		y:                    b.y,
		mobCount:             b.mobCount,
		moneyCon:             b.moneyCon,
		cooldown:             b.cooldown,
		morphId:              b.morphId,
		ghost:                b.ghost,
		fatigue:              b.fatigue,
		berserk:              b.berserk,
		booster:              b.booster,
		prop:                 b.prop,
		itemCon:              b.itemCon,
		itemConNo:            b.itemConNo,
		damage:               b.damage,
		attackCount:          b.attackCount,
		fixDamage:            b.fixDamage,
		lt:                   b.lt,
		rb:                   b.rb,
		bulletCount:          b.bulletCount,
		bulletConsume:        b.bulletConsume,
		mapProtection:        b.mapProtection,
		cureAbnormalStatuses: b.cureAbnormalStatuses,
		statups:              b.statups,
		monsterStatus:        b.monsterStatus,
		rangeValue:           b.rangeValue,
		mastery:              b.mastery,
		z:                    b.z,
		dot:                  b.dot,
		cr:                   b.cr,
		dotInterval:          b.dotInterval,
		dotTime:              b.dotTime,
		damR:                 b.damR,
		criticaldamageMin:    b.criticaldamageMin,
		v:                    b.v,
		ignoreMobpdpR:        b.ignoreMobpdpR,
		epad:                 b.epad,
		w:                    b.w,
		u:                    b.u,
		epdd:                 b.epdd,
		emdd:                 b.emdd,
		selfDestruction:      b.selfDestruction,
		asrR:                 b.asrR,
		t:                    b.t,
		er:                   b.er,
		pddR:                 b.pddR,
		terR:                 b.terR,
		madX:                 b.madX,
		subProp:              b.subProp,
		emhp:                 b.emhp,
		criticaldamageMax:    b.criticaldamageMax,
		expR:                 b.expR,
		emmp:                 b.emmp,
		consumeItemId:        b.consumeItemId,
		mddR:                 b.mddR,
		subTime:              b.subTime,
		padX:                 b.padX,
		mesoR:                b.mesoR,
	}
}

func (b *Builder) SetMobSkill(mobSkill uint16) *Builder {
	b.mobSkill = mobSkill
	return b
}

func (b *Builder) SetMobSkillLevel(skillLevel uint16) *Builder {
	b.mobSkillLevel = skillLevel
	return b
}

func (b *Builder) SetTarget(target uint32) *Builder {
	b.target = target
	return b
}

func (b *Builder) SetRange(v int32) *Builder             { b.rangeValue = v; return b }
func (b *Builder) SetMastery(v int32) *Builder           { b.mastery = v; return b }
func (b *Builder) SetZ(v int32) *Builder                 { b.z = v; return b }
func (b *Builder) SetDot(v int32) *Builder               { b.dot = v; return b }
func (b *Builder) SetCr(v int32) *Builder                { b.cr = v; return b }
func (b *Builder) SetDotInterval(v int32) *Builder       { b.dotInterval = v; return b }
func (b *Builder) SetDotTime(v int32) *Builder           { b.dotTime = v; return b }
func (b *Builder) SetDamR(v int32) *Builder              { b.damR = v; return b }
func (b *Builder) SetCriticaldamageMin(v int32) *Builder { b.criticaldamageMin = v; return b }
func (b *Builder) SetMHPRRate(v uint16) *Builder         { b.mhprRate = v; return b }
func (b *Builder) SetV(v int32) *Builder                 { b.v = v; return b }
func (b *Builder) SetIgnoreMobpdpR(v int32) *Builder     { b.ignoreMobpdpR = v; return b }
func (b *Builder) SetEpad(v int32) *Builder              { b.epad = v; return b }
func (b *Builder) SetW(v int32) *Builder                 { b.w = v; return b }
func (b *Builder) SetU(v int32) *Builder                 { b.u = v; return b }
func (b *Builder) SetEpdd(v int32) *Builder              { b.epdd = v; return b }
func (b *Builder) SetEmdd(v int32) *Builder              { b.emdd = v; return b }
func (b *Builder) SetSelfDestruction(v int32) *Builder   { b.selfDestruction = v; return b }
func (b *Builder) SetAsrR(v int32) *Builder              { b.asrR = v; return b }
func (b *Builder) SetMMPRRate(v uint16) *Builder         { b.mmprRate = v; return b }
func (b *Builder) SetT(v int32) *Builder                 { b.t = v; return b }
func (b *Builder) SetEr(v int32) *Builder                { b.er = v; return b }
func (b *Builder) SetPddR(v int32) *Builder              { b.pddR = v; return b }
func (b *Builder) SetTerR(v int32) *Builder              { b.terR = v; return b }
func (b *Builder) SetMadX(v int32) *Builder              { b.madX = v; return b }
func (b *Builder) SetSubProp(v int32) *Builder           { b.subProp = v; return b }
func (b *Builder) SetEmhp(v int32) *Builder              { b.emhp = v; return b }
func (b *Builder) SetCriticaldamageMax(v int32) *Builder { b.criticaldamageMax = v; return b }
func (b *Builder) SetExpR(v int32) *Builder              { b.expR = v; return b }
func (b *Builder) SetEmmp(v int32) *Builder              { b.emmp = v; return b }

// SetConsumeItemId sets wz `common/itemConsume`. See RestModel.ConsumeItemId:
// this is NOT the same key as `itemCon`, which SetItemConsume carries.
func (b *Builder) SetConsumeItemId(v int32) *Builder { b.consumeItemId = v; return b }

func (b *Builder) SetMddR(v int32) *Builder    { b.mddR = v; return b }
func (b *Builder) SetSubTime(v int32) *Builder { b.subTime = v; return b }
func (b *Builder) SetPadX(v int32) *Builder    { b.padX = v; return b }
func (b *Builder) SetMesoR(v int32) *Builder   { b.mesoR = v; return b }
