package model

import (
	"math"

	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type MonsterAppearType int8

const (
	MonsterAppearTypeNormal    MonsterAppearType = -1
	MonsterAppearTypeRegen     MonsterAppearType = -2
	MonsterAppearTypeRevived   MonsterAppearType = -3
	MonsterAppearTypeSuspended MonsterAppearType = -4
	MonsterAppearTypeDelay     MonsterAppearType = -5
	MonsterAppearTypeEffect    MonsterAppearType = 0
)

type MonsterTemporaryStatType struct {
	bitPosition int32
	value       int32
	position    int8
}

func NewMonsterTemporaryStatType(bitPosition int32) MonsterTemporaryStatType {
	return MonsterTemporaryStatType{
		bitPosition: bitPosition,
		value:       1 << (bitPosition % 32),
		position:    int8(bitPosition >> 5),
	}
}

var MonsterTemporaryStatTypePad = NewMonsterTemporaryStatType(0)
var MonsterTemporaryStatTypePdr = NewMonsterTemporaryStatType(1)
var MonsterTemporaryStatTypeMad = NewMonsterTemporaryStatType(2)
var MonsterTemporaryStatTypeMdr = NewMonsterTemporaryStatType(3)
var MonsterTemporaryStatTypeAcc = NewMonsterTemporaryStatType(4)
var MonsterTemporaryStatTypeEva = NewMonsterTemporaryStatType(5)
var MonsterTemporaryStatTypeSpeed = NewMonsterTemporaryStatType(6)
var MonsterTemporaryStatTypeStun = NewMonsterTemporaryStatType(7)
var MonsterTemporaryStatTypeFreeze = NewMonsterTemporaryStatType(8)
var MonsterTemporaryStatTypePoison = NewMonsterTemporaryStatType(9)
var MonsterTemporaryStatTypeSeal = NewMonsterTemporaryStatType(10)
var MonsterTemporaryStatTypeDarkness = NewMonsterTemporaryStatType(11)
var MonsterTemporaryStatTypePowerUp = NewMonsterTemporaryStatType(12)
var MonsterTemporaryStatTypeMagicUp = NewMonsterTemporaryStatType(13)
var MonsterTemporaryStatTypePowerGuardUp = NewMonsterTemporaryStatType(14)
var MonsterTemporaryStatTypeMagicGuardUp = NewMonsterTemporaryStatType(15)
var MonsterTemporaryStatTypeDoom = NewMonsterTemporaryStatType(16)
var MonsterTemporaryStatTypeWeb = NewMonsterTemporaryStatType(17)
var MonsterTemporaryStatTypePImmune = NewMonsterTemporaryStatType(18)
var MonsterTemporaryStatTypeMImmune = NewMonsterTemporaryStatType(19)
var MonsterTemporaryStatTypeShowdown = NewMonsterTemporaryStatType(20)
var MonsterTemporaryStatTypeHardSkin = NewMonsterTemporaryStatType(21)
var MonsterTemporaryStatTypeAmbush = NewMonsterTemporaryStatType(22)
var MonsterTemporaryStatTypeDamagedElemAttr = NewMonsterTemporaryStatType(23)
var MonsterTemporaryStatTypeVenom = NewMonsterTemporaryStatType(24)
var MonsterTemporaryStatTypeBlind = NewMonsterTemporaryStatType(25)
var MonsterTemporaryStatTypeSealSkill = NewMonsterTemporaryStatType(26)
var MonsterTemporaryStatTypeBurned = NewMonsterTemporaryStatType(27)
var MonsterTemporaryStatTypeDazzle = NewMonsterTemporaryStatType(28)
var MonsterTemporaryStatTypePCounter = NewMonsterTemporaryStatType(29)
var MonsterTemporaryStatTypeMCounter = NewMonsterTemporaryStatType(30)
var MonsterTemporaryStatTypeDisable = NewMonsterTemporaryStatType(31)
var MonsterTemporaryStatTypeRiseByToss = NewMonsterTemporaryStatType(32)
var MonsterTemporaryStatTypeBodyPressure = NewMonsterTemporaryStatType(33)
var MonsterTemporaryStatTypeWeakness = NewMonsterTemporaryStatType(34)
var MonsterTemporaryStatTypeTimeBomb = NewMonsterTemporaryStatType(35)
var MonsterTemporaryStatTypeMagicCrash = NewMonsterTemporaryStatType(36)
var MonsterTemporaryStatTypeHealByDamage = NewMonsterTemporaryStatType(37)

type MonsterTemporaryStatValue struct {
	value    int32
	reason   int32
	duration int32
}

type MonsterBurnedInfo struct {
	characterId uint32
	skillId     uint32
	damage      uint32
	interval    uint32
	end         uint32
	dotCount    uint32
}

func (m *MonsterBurnedInfo) Encode(_ logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteInt(m.characterId)
		w.WriteInt(m.skillId)
		w.WriteInt(m.damage)
		w.WriteInt(m.interval)
		w.WriteInt(m.end)
		w.WriteInt(m.dotCount)
	}
}

type MonsterTemporaryStat struct {
	burnedInfo []MonsterBurnedInfo
	stats      map[MonsterTemporaryStatType]MonsterTemporaryStatValue
}

func (m *MonsterTemporaryStat) getMask() []int32 {
	mask := make([]int32, 4)
	for k := range m.stats {
		mask[k.position] |= k.value
	}
	return mask
}

func (m *MonsterTemporaryStat) Encode(l logrus.FieldLogger, tenant tenant.Model, ops map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		mask := m.getMask()
		pCounter := int32(-1)
		mCounter := int32(-1)

		for i := len(mask) - 1; i >= 0; i-- {
			w.WriteInt32(mask[i])
		}
		for t, v := range m.stats {
			switch t {
			case MonsterTemporaryStatTypeBurned:
				w.WriteInt(uint32(len(m.burnedInfo)))
				for _, b := range m.burnedInfo {
					b.Encode(l, tenant, ops)(w)
				}
			case MonsterTemporaryStatTypeDisable:
				w.WriteBool(false)
				w.WriteBool(false)
			default:
				if t == MonsterTemporaryStatTypePCounter {
					pCounter = v.value
				} else if t == MonsterTemporaryStatTypeMCounter {
					mCounter = v.value
				}
				w.WriteInt16(int16(v.value))
				w.WriteInt32(v.reason)
				w.WriteInt16(int16(v.duration / 500))
			}

		}
		if pCounter != -1 {
			w.WriteInt32(pCounter)
		}
		if mCounter != -1 {
			w.WriteInt32(mCounter)
		}
		if pCounter != -1 || mCounter != -1 {
			w.WriteInt32(int32(math.Max(float64(pCounter), float64(mCounter))))
		}
	}
}

type Monster struct {
	monsterTemporaryStat MonsterTemporaryStat
	x                    int16
	y                    int16
	moveAction           byte
	foothold             int16
	homeFoothold         int16
	appearType           MonsterAppearType
	appearTypeOption     uint32
	team                 int8
	effectItemId         uint32
	phase                uint32
}

func NewMonster(x int16, y int16, stance byte, fh int16, appearType MonsterAppearType, team int8) Monster {
	return Monster{
		monsterTemporaryStat: MonsterTemporaryStat{},
		x:                    x,
		y:                    y,
		moveAction:           stance,
		foothold:             0,
		homeFoothold:         fh,
		appearType:           appearType,
		appearTypeOption:     0,
		team:                 team,
		effectItemId:         0,
		phase:                0,
	}
}

func (m *Monster) Encode(l logrus.FieldLogger, tenant tenant.Model, ops map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		if (tenant.Region() == "GMS" && tenant.MajorVersion() > 12) || tenant.Region() == "JMS" {
			m.monsterTemporaryStat.Encode(l, tenant, ops)(w)
		}
		w.WriteInt16(m.x)
		w.WriteInt16(m.y)
		w.WriteByte(m.moveAction)
		w.WriteInt16(m.foothold)
		w.WriteInt16(m.homeFoothold)
		w.WriteInt8(int8(m.appearType))
		if m.appearType == MonsterAppearTypeRevived || m.appearType >= 0 {
			w.WriteInt(m.appearTypeOption)
		}
		if (tenant.Region() == "GMS" && tenant.MajorVersion() > 12) || tenant.Region() == "JMS" {
			w.WriteInt8(m.team)
			w.WriteInt(m.effectItemId)
			w.WriteInt(m.phase)
		} else {
			// TODO proper temp stat encoding for GMS v12
			w.WriteInt(0)
		}
	}
}
