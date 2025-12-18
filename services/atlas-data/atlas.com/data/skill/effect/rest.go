package effect

import (
	"atlas-data/skill/effect/statup"
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
	MonsterStatus        map[string]uint32  `json:"monsterStatus"`
	CardStats            cardItemUp         `json:"cardStats"`
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
