package model

import (
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
			m.skillLevel = r.ReadByte()
		}
		if m.attackType == AttackTypeMagic {
			// TODO
		}

		if t.Region() == "GMS" && t.MajorVersion() >= 95 {
			m.randomDr = r.ReadUint32()
			m.crc32 = r.ReadUint32()
		}
		m.skillDataCrc = r.ReadUint32()
		m.skillDataCrc2 = r.ReadUint32()
		if isKeyDownSkill(m.skillId) {
			m.keyDown = r.ReadUint32()
		}
		mask1 := r.ReadByte()
		m.finalAfterSlashBlast = int(mask1 & 0x07)       // Extract lowest 3 bits (0b00000111)
		m.shadowPartner = int((mask1 >> 3) & 0x01)       // Extract bit 3
		m.unknown1 = int((mask1 >> 4) & 0x01)            // Extract bit 4
		m.serialAttackSkillId = int((mask1 >> 5) & 0x01) // Extract bit 5 (boolean flag)
		m.unknown2 = int((mask1 >> 7) & 0x7F)            // Extract bits 7-13 (7-bit value)

		if m.attackType == AttackTypeRanged {
			// TODO
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

		if m.attackType == AttackTypeRanged {
			// TODO
		}

		if t.Region() == "GMS" && t.MajorVersion() >= 95 {
			// TODO battle mage related
			_ = r.ReadUint32()
		}

		for range m.damage {
			di := NewDamageInfo(m.hits)
			di.Decode(l, t, options)(r)
			m.damageInfo = append(m.damageInfo, *di)
		}

		m.targetX = r.ReadUint16()
		m.targetY = r.ReadUint16()
		if m.skillId == 14111006 { // poison bomb
			m.grenadeX = r.ReadUint16()
			m.grenadeY = r.ReadUint16()
		}
	}
}

func isKeyDownSkill(skillId uint32) bool {
	return skillId == 2321001 || skillId == 80001836 || skillId == 37121052 || skillId == 36121000 ||
		skillId == 37121003 || skillId == 36101001 || skillId == 33121114 || skillId == 33121214 ||
		skillId == 35121015 || skillId == 33121009 || skillId == 32121003 || skillId == 31211001 ||
		skillId == 31111005 || skillId == 30021238 || skillId == 31001000 || skillId == 31101000 ||
		skillId == 80001887 || skillId == 80001880 || skillId == 80001629 || skillId == 20041226 ||
		skillId == 60011216 || skillId == 65121003 || skillId == 80001587 || skillId == 131001008 ||
		skillId == 142111010 || skillId == 131001004 || skillId == 95001001 || skillId == 101110100 ||
		skillId == 101110101 || skillId == 101110102 || skillId == 27111100 || skillId == 12121054 ||
		skillId == 11121052 || skillId == 11121055 || skillId == 5311002 || skillId == 4341002 ||
		skillId == 5221004 || skillId == 5221022 || skillId == 3121020 || skillId == 3101008 ||
		skillId == 3111013 || skillId == 1311011 || skillId == 2221011 || skillId == 2221052 ||
		skillId == 25121030 || skillId == 27101202 || skillId == 25111005 || skillId == 23121000 ||
		skillId == 22171083 || skillId == 14121004 || skillId == 13111020 || skillId == 13121001 ||
		skillId == 14111006 || (skillId >= 80001389 && skillId <= 80001392) || skillId == 42121000 ||
		skillId == 42120003 || skillId == 5700010 || skillId == 5711021 || skillId == 5721001 ||
		skillId == 5721061 || skillId == 21120018 || skillId == 21120019 || skillId == 24121000 ||
		skillId == 24121005
}
