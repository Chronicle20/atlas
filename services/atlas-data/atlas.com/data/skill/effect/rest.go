package effect

import (
	"atlas-data/skill/effect/statup"
	"github.com/Chronicle20/atlas-model/model"
)

type RestModel struct {
	WeaponAttack      int16   `json:"weaponAttack"`
	MagicAttack       int16   `json:"magicAttack"`
	WeaponDefense     int16   `json:"weaponDefense"`
	MagicDefense      int16   `json:"magicDefense"`
	Accuracy          int16   `json:"accuracy"`
	Avoidability      int16   `json:"avoidability"`
	Speed             int16   `json:"speed"`
	Jump              int16   `json:"jump"`
	HP                uint16  `json:"hp"`
	MP                uint16  `json:"mp"`
	HPR               float64 `json:"hpR"`
	MPR               float64 `json:"mpR"`
	MHPRRate          uint16  `json:"MHPRRate"`
	MMPRRate          uint16  `json:"MMPRRate"`
	MobSkill          uint16  `json:"mobSkill"`
	MobSkillLevel     uint16  `json:"mobSkillLevel"`
	MHPR              byte    `json:"mhpr"`
	MMPR              byte    `json:"mmpr"`
	HPConsume         uint16  `json:"HPConsume"`
	MPConsume         uint16  `json:"MPConsume"`
	Duration          int32   `json:"duration"`
	Target            uint32  `json:"target"`
	Barrier           int32   `json:"barrier"`
	Mob               uint32  `json:"mob"`
	OverTime          bool    `json:"overTime"`
	RepeatEffect      bool    `json:"repeatEffect"`
	MoveTo            int32   `json:"moveTo"`
	CP                uint32  `json:"cp"`
	NuffSkill         uint32  `json:"nuffSkill"`
	Skill             bool    `json:"skill"`
	X                 int16   `json:"x"`
	Y                 int16   `json:"y"`
	MobCount          uint32  `json:"mobCount"`
	MoneyConsume      uint32  `json:"moneyConsume"`
	Cooldown          uint32  `json:"cooldown"`
	MorphId           uint32  `json:"morphId"`
	Ghost             uint32  `json:"ghost"`
	Fatigue           uint32  `json:"fatigue"`
	Berserk           uint32  `json:"berserk"`
	Booster           uint32  `json:"booster"`
	Prop              float64 `json:"prop"`
	ItemConsume       uint32  `json:"itemConsume"`
	ItemConsumeAmount uint32  `json:"itemConsumeAmount"`
	Damage            uint32  `json:"damage"`
	AttackCount       uint32  `json:"attackCount"`
	FixDamage         int32   `json:"fixDamage"`
	//LT Point
	//RB Point
	BulletCount          uint16             `json:"bulletCount"`
	BulletConsume        uint16             `json:"bulletConsume"`
	MapProtection        byte               `json:"mapProtection"`
	CureAbnormalStatuses []string           `json:"cureAbnormalStatuses"`
	Statups              []statup.RestModel `json:"statups"`
	MonsterStatus        []monsterStatus    `json:"monsterStatus"`
	CardStats            cardItemUp         `json:"cardStats"`
}

func Transform(m Model) (RestModel, error) {
	su, err := model.SliceMap(statup.Transform)(model.FixedProvider(m.StatUps()))()()
	if err != nil {
		return RestModel{}, err
	}
	ms := make([]monsterStatus, 0)
	for k, v := range m.monsterStatus {
		ms = append(ms, monsterStatus{
			Status: k,
			Value:  v,
		})
	}

	return RestModel{
		WeaponAttack:         m.weaponAttack,
		MagicAttack:          m.magicAttack,
		WeaponDefense:        m.weaponDefense,
		MagicDefense:         m.magicDefense,
		Accuracy:             m.accuracy,
		Avoidability:         m.avoidability,
		Speed:                m.speed,
		Jump:                 m.jump,
		HP:                   m.hp,
		MP:                   m.mp,
		HPR:                  m.hpr,
		MPR:                  m.mpr,
		MHPRRate:             m.mhprRate,
		MMPRRate:             m.mmprRate,
		MobSkill:             m.mobSkill,
		MobSkillLevel:        m.mobSkillLevel,
		MHPR:                 m.mhpR,
		MMPR:                 m.mmpR,
		HPConsume:            m.hpCon,
		MPConsume:            m.mpCon,
		Duration:             m.duration,
		Target:               m.target,
		Barrier:              m.barrier,
		Mob:                  m.mob,
		OverTime:             m.overtime,
		RepeatEffect:         m.repeatEffect,
		MoveTo:               m.moveTo,
		CP:                   m.cp,
		NuffSkill:            m.nuffSkill,
		Skill:                m.skill,
		X:                    m.x,
		Y:                    m.y,
		MobCount:             m.mobCount,
		MoneyConsume:         m.moneyCon,
		Cooldown:             m.cooldown,
		MorphId:              m.morphId,
		Ghost:                m.ghost,
		Fatigue:              m.fatigue,
		Berserk:              m.berserk,
		Booster:              m.booster,
		Prop:                 m.prop,
		ItemConsume:          m.itemCon,
		ItemConsumeAmount:    m.itemConNo,
		Damage:               m.damage,
		AttackCount:          m.attackCount,
		FixDamage:            m.fixDamage,
		BulletCount:          m.bulletCount,
		BulletConsume:        m.bulletConsume,
		MapProtection:        m.mapProtection,
		CureAbnormalStatuses: m.cureAbnormalStatuses,
		Statups:              su,
		MonsterStatus:        ms,
		CardStats:            cardItemUp{}, // TODO?
	}, nil
}

type monsterStatus struct {
	Status string `json:"status"`
	Value  uint32 `json:"value"`
}

type cardItemUp struct {
	ItemCode    uint32 `json:"itemCode"`
	Probability uint32 `json:"probability"`
	Areas       []area `json:"areas"`
	InParty     bool   `json:"inParty"`
}

type area struct {
	Start uint32 `json:"start"`
	End   uint32 `json:"end"`
}
