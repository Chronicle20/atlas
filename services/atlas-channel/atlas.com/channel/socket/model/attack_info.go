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
	AttackTypeEnergy = AttackType(3)
)

func NewAttackInfo(attackType AttackType) *AttackInfo {
	return &AttackInfo{attackType: attackType}
}

type AttackInfo struct {
	attackType           AttackType
	fieldKey             byte
	dr0                  uint32
	dr1                  uint32
	hits                 byte
	damage               uint32
	dr2                  uint32
	dr3                  uint32
	skillId              uint32
	skillLevel           byte
	randomDr             uint32
	crc32                uint32
	skillDataCrc         uint32
	skillDataCrc2        uint32
	mask1                byte
	mask2                uint16
	keyDown              uint32
	finalAfterSlashBlast int
	shadowPartner        int
	unknown1             int
	serialAttackSkillId  int
	unknown2             int
	attackAction         int
	left                 bool
	anotherCrc           uint32
	attackActionType     byte
	attackSpeed          byte
	attackTime           uint32
	damageInfo           []DamageInfo
	characterX           uint16
	characterY           uint16
	grenadeX             uint16
	grenadeY             uint16
	reserveSpark         uint32
	javlin               bool
	properBulletPosition uint16
	cashBulletPosition   uint16
	nShootRange          byte
	bulletItemId         uint32
	dragon               bool
	dragonX              uint16
	dragonY              uint16
	bulletX              uint16
	bulletY              uint16
}

func (m *AttackInfo) Decode(l logrus.FieldLogger, t tenant.Model, options map[string]interface{}) func(r *request.Reader) {
	return func(r *request.Reader) {
		m.fieldKey = r.ReadByte()
		if t.Region() == "GMS" && t.MajorVersion() >= 95 {
			m.dr0 = r.ReadUint32()
			m.dr1 = r.ReadUint32()
		}
		numAttackedAndDamageMask := r.ReadByte()
		m.hits = numAttackedAndDamageMask & 0xF
		m.damage = uint32((numAttackedAndDamageMask >> 4) & 0xF)

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
		m.mask1 = r.ReadByte()
		m.finalAfterSlashBlast = int(m.mask1 & 0x07)       // Extract lowest 3 bits (0b00000111)
		m.shadowPartner = int((m.mask1 >> 3) & 0x01)       // Extract bit 3
		m.unknown1 = int((m.mask1 >> 4) & 0x01)            // Extract bit 4
		m.serialAttackSkillId = int((m.mask1 >> 5) & 0x01) // Extract bit 5 (boolean flag)
		m.unknown2 = int((m.mask1 >> 7) & 0x7F)            // Extract bits 7-13 (7-bit value)

		if t.Region() == "GMS" && t.MajorVersion() >= 95 {
			if m.attackType == AttackTypeRanged {
				m.javlin = r.ReadBool()
			}
		}

		m.mask2 = r.ReadUint16()
		m.attackAction = int(m.mask2 & 0x7FFF) // Extract lower 15 bits
		m.left = int((m.mask2>>15)&0x01) == 1  // Extract bit 15
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
			m.cashBulletPosition = r.ReadUint16()
			m.nShootRange = r.ReadByte()

			if m.javlin && !skill.IsShootSkillNotConsumingBullet(skill.Id(m.skillId)) {
				m.bulletItemId = r.ReadUint32()
			}
		} else if m.attackType == AttackTypeMagic {
			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				_ = r.ReadUint32()
			}
		}

		for range m.damage {
			di := NewDamageInfo(m.hits)
			di.Decode(l, t, options)(r)
			m.damageInfo = append(m.damageInfo, *di)
		}

		m.characterX = r.ReadUint16()
		m.characterY = r.ReadUint16()
		if m.attackType == AttackTypeRanged {
			m.bulletX = r.ReadUint16()
			m.bulletY = r.ReadUint16()
		}

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

func (m *AttackInfo) SkillId() uint32 {
	return m.skillId
}

func (m *AttackInfo) SkillLevel() byte {
	return m.skillLevel
}

func (m *AttackInfo) Hits() byte {
	return m.hits
}

func (m *AttackInfo) Damage() uint32 {
	return m.damage
}

func (m *AttackInfo) Option() byte {
	return m.mask1
}

func (m *AttackInfo) Left() bool {
	return m.left
}

func (m *AttackInfo) AttackAction() int {
	return m.attackAction
}

func (m *AttackInfo) ActionSpeed() byte {
	return m.attackSpeed
}

func (m *AttackInfo) BulletItemId() uint32 {
	return m.bulletItemId
}

func (m *AttackInfo) Keydown() uint32 {
	return m.keyDown
}

func (m *AttackInfo) AttackType() AttackType {
	return m.attackType
}

func (m *AttackInfo) ProperBulletPosition() uint16 {
	return m.properBulletPosition
}

func (m *AttackInfo) CashBulletPosition() uint16 {
	return m.cashBulletPosition
}

func (m *AttackInfo) BulletX() uint16 {
	return m.bulletX
}

func (m *AttackInfo) BulletY() uint16 {
	return m.bulletY
}
