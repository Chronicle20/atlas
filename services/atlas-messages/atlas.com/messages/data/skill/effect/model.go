package effect

import (
	"atlas-messages/data/skill/effect/statup"
)

type Model struct {
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
	overtime      bool
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

func (m Model) StatUps() []statup.Model {
	return m.statups
}

func (m Model) HPConsume() uint16 {
	return m.hpCon
}

func (m Model) MPConsume() uint16 {
	return m.mpCon
}

func (m Model) Duration() int32 {
	return m.duration
}

func (m Model) Cooldown() uint32 {
	return m.cooldown
}
