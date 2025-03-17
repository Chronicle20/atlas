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
	su, err := model.SliceMap(statup.Transform)(model.FixedProvider(m.Statups))()()
	if err != nil {
		return RestModel{}, err
	}

	ms := make([]monsterStatus, 0)
	for k, v := range m.MonsterStatus { // Fixed field name
		ms = append(ms, monsterStatus{
			Status: k,
			Value:  v,
		})
	}

	return RestModel{
		WeaponAttack:         m.WeaponAttack,
		MagicAttack:          m.MagicAttack,
		WeaponDefense:        m.WeaponDefense,
		MagicDefense:         m.MagicDefense,
		Accuracy:             m.Accuracy,
		Avoidability:         m.Avoidability,
		Speed:                m.Speed,
		Jump:                 m.Jump,
		HP:                   m.HP,
		MP:                   m.MP,
		HPR:                  m.HPR,
		MPR:                  m.MPR,
		MHPRRate:             m.MHPRRate,
		MMPRRate:             m.MMPRRate,
		MobSkill:             m.MobSkill,
		MobSkillLevel:        m.MobSkillLevel,
		MHPR:                 m.MHPR,
		MMPR:                 m.MMPR,
		HPConsume:            m.HPCon,
		MPConsume:            m.MPCon,
		Duration:             m.Duration,
		Target:               m.Target,
		Barrier:              m.Barrier,
		Mob:                  m.Mob,
		OverTime:             m.Overtime,
		RepeatEffect:         m.RepeatEffect,
		MoveTo:               m.MoveTo,
		CP:                   m.CP,
		NuffSkill:            m.NuffSkill,
		Skill:                m.Skill,
		X:                    m.X,
		Y:                    m.Y,
		MobCount:             m.MobCount,
		MoneyConsume:         m.MoneyCon,
		Cooldown:             m.Cooldown,
		MorphId:              m.MorphId,
		Ghost:                m.Ghost,
		Fatigue:              m.Fatigue,
		Berserk:              m.Berserk,
		Booster:              m.Booster,
		Prop:                 m.Prop,
		ItemConsume:          m.ItemCon,
		ItemConsumeAmount:    m.ItemConNo,
		Damage:               m.Damage,
		AttackCount:          m.AttackCount,
		FixDamage:            m.FixDamage,
		BulletCount:          m.BulletCount,
		BulletConsume:        m.BulletConsume,
		MapProtection:        m.MapProtection,
		CureAbnormalStatuses: m.CureAbnormalStatuses,
		Statups:              su,
		MonsterStatus:        ms,
		CardStats:            cardItemUp{}, // TODO?
	}, nil
}
func Extract(rm RestModel) (Model, error) {
	su, err := model.SliceMap(statup.Extract)(model.FixedProvider(rm.Statups))()()
	if err != nil {
		return Model{}, err
	}

	ms := make(map[string]uint32)
	for _, v := range rm.MonsterStatus {
		ms[v.Status] = v.Value
	}

	return Model{
		WeaponAttack:         rm.WeaponAttack,
		MagicAttack:          rm.MagicAttack,
		WeaponDefense:        rm.WeaponDefense,
		MagicDefense:         rm.MagicDefense,
		Accuracy:             rm.Accuracy,
		Avoidability:         rm.Avoidability,
		Speed:                rm.Speed,
		Jump:                 rm.Jump,
		HP:                   rm.HP,
		MP:                   rm.MP,
		HPR:                  rm.HPR,
		MPR:                  rm.MPR,
		MHPRRate:             rm.MHPRRate,
		MMPRRate:             rm.MMPRRate,
		MobSkill:             rm.MobSkill,
		MobSkillLevel:        rm.MobSkillLevel,
		MHPR:                 rm.MHPR,
		MMPR:                 rm.MMPR,
		HPCon:                rm.HPConsume,
		MPCon:                rm.MPConsume,
		Duration:             rm.Duration,
		Target:               rm.Target,
		Barrier:              rm.Barrier,
		Mob:                  rm.Mob,
		Overtime:             rm.OverTime,
		RepeatEffect:         rm.RepeatEffect,
		MoveTo:               rm.MoveTo,
		CP:                   rm.CP,
		NuffSkill:            rm.NuffSkill,
		Skill:                rm.Skill,
		X:                    rm.X,
		Y:                    rm.Y,
		MobCount:             rm.MobCount,
		MoneyCon:             rm.MoneyConsume,
		Cooldown:             rm.Cooldown,
		MorphId:              rm.MorphId,
		Ghost:                rm.Ghost,
		Fatigue:              rm.Fatigue,
		Berserk:              rm.Berserk,
		Booster:              rm.Booster,
		Prop:                 rm.Prop,
		ItemCon:              rm.ItemConsume,
		ItemConNo:            rm.ItemConsumeAmount,
		Damage:               rm.Damage,
		AttackCount:          rm.AttackCount,
		FixDamage:            rm.FixDamage,
		BulletCount:          rm.BulletCount,
		BulletConsume:        rm.BulletConsume,
		MapProtection:        rm.MapProtection,
		CureAbnormalStatuses: rm.CureAbnormalStatuses,
		Statups:              su,
		MonsterStatus:        ms,
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
