package effect

import "atlas-data/skill/effect/statup"

type Model struct {
	WeaponAttack         int16             `json:"weapon_attack"`
	MagicAttack          int16             `json:"magic_attack"`
	WeaponDefense        int16             `json:"weapon_defense"`
	MagicDefense         int16             `json:"magic_defense"`
	Accuracy             int16             `json:"accuracy"`
	Avoidability         int16             `json:"avoidability"`
	Speed                int16             `json:"speed"`
	Jump                 int16             `json:"jump"`
	HP                   uint16            `json:"hp"`
	MP                   uint16            `json:"mp"`
	HPR                  float64           `json:"hpr"`
	MPR                  float64           `json:"mpr"`
	MHPRRate             uint16            `json:"mhpr_rate"`
	MMPRRate             uint16            `json:"mmpr_rate"`
	MobSkill             uint16            `json:"mob_skill"`
	MobSkillLevel        uint16            `json:"mob_skill_level"`
	MHPR                 byte              `json:"mhp_r"`
	MMPR                 byte              `json:"mmp_r"`
	HPCon                uint16            `json:"hp_con"`
	MPCon                uint16            `json:"mp_con"`
	Duration             int32             `json:"duration"`
	Target               uint32            `json:"target"`
	Barrier              int32             `json:"barrier"`
	Mob                  uint32            `json:"mob"`
	Overtime             bool              `json:"overtime"`
	RepeatEffect         bool              `json:"repeat_effect"`
	MoveTo               int32             `json:"move_to"`
	CP                   uint32            `json:"cp"`
	NuffSkill            uint32            `json:"nuff_skill"`
	Skill                bool              `json:"skill"`
	X                    int16             `json:"x"`
	Y                    int16             `json:"y"`
	MobCount             uint32            `json:"mob_count"`
	MoneyCon             uint32            `json:"money_con"`
	Cooldown             uint32            `json:"cooldown"`
	MorphId              uint32            `json:"morph_id"`
	Ghost                uint32            `json:"ghost"`
	Fatigue              uint32            `json:"fatigue"`
	Berserk              uint32            `json:"berserk"`
	Booster              uint32            `json:"booster"`
	Prop                 float64           `json:"prop"`
	ItemCon              uint32            `json:"item_con"`
	ItemConNo            uint32            `json:"item_con_no"`
	Damage               uint32            `json:"damage"`
	AttackCount          uint32            `json:"attack_count"`
	FixDamage            int32             `json:"fix_damage"`
	BulletCount          uint16            `json:"bullet_count"`
	BulletConsume        uint16            `json:"bullet_consume"`
	MapProtection        byte              `json:"map_protection"`
	CureAbnormalStatuses []string          `json:"cure_abnormal_statuses"`
	Statups              []statup.Model    `json:"statups"`
	MonsterStatus        map[string]uint32 `json:"monster_status"`
}

func NewModelBuilder() *ModelBuilder {
	return &ModelBuilder{}
}

type ModelBuilder struct {
	weaponAttack  int16
	magicAttack   int16
	weaponDefense int16
	magicDefense  int16
	accuracy      int16
	avoidability  int16
	speed         int16
	jump          int16
	hp            uint16
	mp            uint16
	hpr           float64
	mpr           float64
	mhprRate      uint16
	mmprRate      uint16
	mobSkill      uint16
	mobSkillLevel uint16
	mhpR          byte
	mmpR          byte
	hpCon         uint16
	mpCon         uint16
	duration      int32
	target        uint32
	barrier       int32
	mob           uint32
	overTime      bool
	repeatEffect  bool
	moveTo        int32
	cp            uint32
	nuffSkill     uint32
	skill         bool
	x             int16
	y             int16
	mobCount      uint32
	moneyCon      uint32
	cooldown      uint32
	morphId       uint32
	ghost         uint32
	fatigue       uint32
	berserk       uint32
	booster       uint32
	prop          float64
	itemCon       uint32
	itemConNo     uint32
	damage        uint32
	attackCount   uint32
	fixDamage     int32
	//LT Point
	//RB Point
	bulletCount          uint16
	bulletConsume        uint16
	mapProtection        byte
	cureAbnormalStatuses []string
	statups              []statup.Model
	monsterStatus        map[string]uint32
}

func (b *ModelBuilder) SetDuration(duration int32) *ModelBuilder {
	b.duration = duration
	return b
}

func (b *ModelBuilder) SetHP(hp uint16) *ModelBuilder {
	b.hp = hp
	return b
}

func (b *ModelBuilder) SetHPRecovery(hpr float64) *ModelBuilder {
	b.hpr = hpr
	return b
}

func (b *ModelBuilder) SetMP(mp uint16) *ModelBuilder {
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

func (b *ModelBuilder) SetStatups(statups []statup.Model) *ModelBuilder {
	b.statups = statups
	return b
}
func (b *ModelBuilder) Build() Model {
	return Model{
		WeaponAttack:         b.weaponAttack,
		MagicAttack:          b.magicAttack,
		WeaponDefense:        b.weaponDefense,
		MagicDefense:         b.magicDefense,
		Accuracy:             b.accuracy,
		Avoidability:         b.avoidability,
		Speed:                b.speed,
		Jump:                 b.jump,
		HP:                   b.hp,
		MP:                   b.mp,
		HPR:                  b.hpr,
		MPR:                  b.mpr,
		MHPRRate:             b.mhprRate,
		MMPRRate:             b.mmprRate,
		MobSkill:             b.mobSkill,
		MobSkillLevel:        b.mobSkillLevel,
		MHPR:                 b.mhpR,
		MMPR:                 b.mmpR,
		HPCon:                b.hpCon,
		MPCon:                b.mpCon,
		Duration:             b.duration,
		Target:               b.target,
		Barrier:              b.barrier,
		Mob:                  b.mob,
		Overtime:             b.overTime, // Kept lowercase `b.overTime` as per request
		RepeatEffect:         b.repeatEffect,
		MoveTo:               b.moveTo,
		CP:                   b.cp,
		NuffSkill:            b.nuffSkill,
		Skill:                b.skill,
		X:                    b.x,
		Y:                    b.y,
		MobCount:             b.mobCount,
		MoneyCon:             b.moneyCon,
		Cooldown:             b.cooldown,
		MorphId:              b.morphId,
		Ghost:                b.ghost,
		Fatigue:              b.fatigue,
		Berserk:              b.berserk,
		Booster:              b.booster,
		Prop:                 b.prop,
		ItemCon:              b.itemCon,
		ItemConNo:            b.itemConNo,
		Damage:               b.damage,
		AttackCount:          b.attackCount,
		FixDamage:            b.fixDamage,
		BulletCount:          b.bulletCount,
		BulletConsume:        b.bulletConsume,
		MapProtection:        b.mapProtection,
		CureAbnormalStatuses: b.cureAbnormalStatuses,
		Statups:              b.statups,
		MonsterStatus:        b.monsterStatus,
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
