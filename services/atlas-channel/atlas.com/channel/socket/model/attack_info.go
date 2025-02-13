package model

import (
	"github.com/Chronicle20/atlas-constants/skill"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type AttackType byte

const (
	AttackTypeMelee  = AttackType(0)
	AttackTypeRanged = AttackType(1)
	AttackTypeMagic  = AttackType(2)
)

func NewAttackInfo(attackType AttackType) *AttackInfo {
	return &AttackInfo{attackType: attackType}
}

type AttackInfo struct {
	attackType           AttackType
	fieldKey             byte
	dr0                  uint32
	dr1                  uint32
	dr2                  uint32
	dr3                  uint32
	skillId              uint32
	skillLevel           byte
	randomDr             uint32
	crc32                uint32
	skillDataCrc         uint32
	skillDataCrc2        uint32
	keyDown              uint32
	finalAfterSlashBlast int
	shadowPartner        int
	unknown1             int
	serialAttackSkillId  int
	unknown2             int
	attackAction         int
	left                 int
	anotherCrc           uint32
	attackActionType     byte
	attackSpeed          byte
	attackTime           uint32
	damageInfo           []DamageInfo
	targetX              uint16
	targetY              uint16
	grenadeX             uint16
	grenadeY             uint16
	reserveSpark         uint32
	javlin               bool
	properBulletPosition uint16
	pnCashItemPos        uint16
	nShootRange          byte
	bulletItemId         uint32
	dragon               bool
	dragonX              uint16
	dragonY              uint16
}

func (m *AttackInfo) Decode(l logrus.FieldLogger, t tenant.Model, options map[string]interface{}) func(r *request.Reader) {
	return func(r *request.Reader) {
		m.fieldKey = r.ReadByte()
		if t.Region() == "GMS" && t.MajorVersion() >= 95 {
			m.dr0 = r.ReadUint32()
			m.dr1 = r.ReadUint32()
		}
		numAttackedAndDamageMask := r.ReadByte()
		hits := numAttackedAndDamageMask & 0xF
		damage := uint32((numAttackedAndDamageMask >> 4) & 0xF)

		if t.Region() == "GMS" && t.MajorVersion() >= 95 {
			m.dr2 = r.ReadUint32()
			m.dr3 = r.ReadUint32()
		}

		m.skillId = r.ReadUint32()
		if t.Region() == "GMS" && t.MajorVersion() >= 95 {
			m.skillLevel = r.ReadByte() // nCombatOrders
		}

		if t.Region() == "GMS" && t.MajorVersion() >= 95 {
			m.randomDr = r.ReadUint32()
			m.crc32 = r.ReadUint32()

			if m.attackType == AttackTypeMagic {
				// TODO
				_ = r.ReadUint32() //2dr0
				_ = r.ReadUint32() //2dr1
				_ = r.ReadUint32() //2dr2
				_ = r.ReadUint32() //2dr3
				_ = r.ReadUint32() //2rnd
				_ = r.ReadUint32() //2crc
			}
		}

		m.skillDataCrc = r.ReadUint32()
		m.skillDataCrc2 = r.ReadUint32()

		if skill.IsKeyDownSkill(skill.Id(m.skillId)) {
			m.keyDown = r.ReadUint32()
		} else if skill.NeedsCharging(skill.Id(m.skillId)) {
			m.keyDown = r.ReadUint32()
		}
		mask1 := r.ReadByte()
		m.finalAfterSlashBlast = int(mask1 & 0x07)       // Extract lowest 3 bits (0b00000111)
		m.shadowPartner = int((mask1 >> 3) & 0x01)       // Extract bit 3
		m.unknown1 = int((mask1 >> 4) & 0x01)            // Extract bit 4
		m.serialAttackSkillId = int((mask1 >> 5) & 0x01) // Extract bit 5 (boolean flag)
		m.unknown2 = int((mask1 >> 7) & 0x7F)            // Extract bits 7-13 (7-bit value)

		if t.Region() == "GMS" && t.MajorVersion() >= 95 {
			if m.attackType == AttackTypeRanged {
				m.javlin = r.ReadBool()
			}
		}

		mask2 := r.ReadUint16()
		m.attackAction = int(mask2 & 0x7FFF) // Extract lower 15 bits
		m.left = int((mask2 >> 15) & 0x01)   // Extract bit 15
		if t.Region() == "GMS" && t.MajorVersion() >= 95 {
			m.anotherCrc = r.ReadUint32()
		}
		m.attackActionType = r.ReadByte()
		m.attackSpeed = r.ReadByte()
		m.attackTime = r.ReadUint32()

		if m.attackType == AttackTypeMelee {
			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				// TODO battle mage related
				_ = r.ReadUint32()
			}
		} else if m.attackType == AttackTypeRanged {
			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				_ = r.ReadUint32()
			}
			m.properBulletPosition = r.ReadUint16()
			m.pnCashItemPos = r.ReadUint16()
			m.nShootRange = r.ReadByte()

			if m.javlin && !skill.IsShootSkillNotConsumingBullet(skill.Id(m.skillId)) {
				m.bulletItemId = r.ReadUint32()
			}
		} else if m.attackType == AttackTypeMagic {
			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				_ = r.ReadUint32()
			}
		}

		for range damage {
			di := NewDamageInfo(hits)
			di.Decode(l, t, options)(r)
			m.damageInfo = append(m.damageInfo, *di)
		}

		m.targetX = r.ReadUint16()
		m.targetY = r.ReadUint16()
		if skill.Id(m.skillId) == skill.NightWalkerStage3PoisonBombId {
			m.grenadeX = r.ReadUint16()
			m.grenadeY = r.ReadUint16()
		} else if skill.Id(m.skillId) == skill.ThunderBreakerStage3SparkId {
			m.reserveSpark = r.ReadUint32()
		}
		if m.attackType == AttackTypeMagic {
			m.dragon = r.ReadBool()
			if m.dragon {
				m.dragonX = r.ReadUint16()
				m.dragonY = r.ReadUint16()
			}
		}
	}
}

func (m *AttackInfo) DamageInfo() []DamageInfo {
	return m.damageInfo
}
