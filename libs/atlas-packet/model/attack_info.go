package model

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
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

// legacyGmsByteAction reports whether the serverbound attack action/direction field
// is a single byte (bit7=bLeft, bits0-6=nAction) instead of a 2-byte short. Legacy
// pre-79 GMS only. IDA-verified: v72 TryDoingMeleeAttack @0x85f9c2 (Encode1) vs v79
// @0x8c2adc (Encode2). Mirrors the clientbound CUserRemote::OnAttack transition.
func legacyGmsByteAction(t tenant.Model) bool {
	return t.Region() == "GMS" && t.MajorVersion() < 79
}

// legacyGmsSingleCrc reports whether the serverbound attack head carries only a
// single skill-data CRC (v72 @0x85f96c) rather than the two CRCs GMS v79+ writes
// (v79 @0x8c2ab2 + @0x8c2abb). Legacy pre-79 GMS only.
func legacyGmsSingleCrc(t tenant.Model) bool {
	return t.Region() == "GMS" && t.MajorVersion() < 79
}

// legacyGmsNoSkillDataCrc reports whether the serverbound attack head carries NO
// skill-data CRC at all (the field appears at GMS v72; the very-legacy pre-72
// client omits it entirely). IDA-verified: v61 CLOSE_RANGE sender sub_7A45F1
// @0x7a5bc3 Encode4(skillId) is followed directly by the mask1/option Encode1
// @0x7a5d3d — there is no CRC Encode4 in between (only a conditional keydown
// Encode4 for charge skills). v72 TryDoingMeleeAttack @0x85f96c writes one CRC.
// So the head skill-data CRC is present GMS v72+ and absent below.
func legacyGmsNoSkillDataCrc(t tenant.Model) bool {
	return t.Region() == "GMS" && t.MajorVersion() < 72
}

// legacyGmsNoRangedBulletCoords reports whether the ranged-attack trailer OMITS the
// bulletX/bulletY world-coordinate shorts. The very-legacy pre-61 GMS shoot sender
// (v48 sub_6A228C @0x6a3965/0x6a3979: after the per-mob loop it Encode2s only
// characterX/characterY then SendPacket @0x6a3988 — no bullet coords) does not carry
// them; the head properBulletPosition/cashBulletPosition/nShootRange block is still
// present. Gate to GMS < 61 so v48 omits the 4-byte trailer while v61+/JMS are
// unchanged (their fixtures pin the existing trailer).
func legacyGmsNoRangedBulletCoords(t tenant.Model) bool {
	return t.Region() == "GMS" && t.MajorVersion() < 61
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

// Encode is the symmetric mirror of Decode: it serializes the client->server
// attack request. Every version gate here MUST match Decode field-for-field
// (the dr-block is GMS v84+, the magic 2dr block / skillLevel / anotherCrc /
// per-type ints are GMS v95+). The AttackInfo round-trip test relies on this
// symmetry — any drift surfaces as unconsumed bytes for the affected version.
func (m *AttackInfo) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		w.WriteByte(m.fieldKey)
		if t.Region() == "GMS" && t.MajorVersion() >= 84 { // primary dr-block (v84+)
			w.WriteInt(m.dr0)
			w.WriteInt(m.dr1)
		}
		w.WriteByte((m.hits & 0xF) | byte((m.damage&0xF)<<4))
		if t.Region() == "GMS" && t.MajorVersion() >= 84 { // primary dr-block (v84+)
			w.WriteInt(m.dr2)
			w.WriteInt(m.dr3)
		}
		w.WriteInt(m.skillId)
		if t.Region() == "GMS" && t.MajorVersion() >= 95 {
			w.WriteByte(m.skillLevel) // nCombatOrders
		}
		if t.Region() == "GMS" && t.MajorVersion() >= 84 { // randomDr/crc32 complete the primary dr-block (v84+)
			w.WriteInt(m.randomDr)
			w.WriteInt(m.crc32)
		}
		if t.Region() == "GMS" && t.MajorVersion() >= 95 {
			if m.attackType == AttackTypeMagic {
				// Secondary dr-block for magic attacks (v95+; absent in v84 magic).
				w.WriteInt(0) //2dr0
				w.WriteInt(0) //2dr1
				w.WriteInt(0) //2dr2
				w.WriteInt(0) //2dr3
				w.WriteInt(0) //2rnd
				w.WriteInt(0) //2crc
			}
		}
		// The head skill-data CRC block. The very-legacy pre-72 GMS client (v61)
		// writes NO CRC at all (sub_7A45F1 @0x7a5bc3→@0x7a5d3d: skillId then
		// straight to mask1). v72 writes a SINGLE CRC (TryDoingMeleeAttack
		// @0x85f96c); GMS v79+ adds a second (v79 @0x8c2ab2 + @0x8c2abb).
		if !legacyGmsNoSkillDataCrc(t) {
			w.WriteInt(m.skillDataCrc)
		}
		if !legacyGmsSingleCrc(t) {
			w.WriteInt(m.skillDataCrc2)
		}
		if skill.IsKeyDownSkill(skill.Id(m.skillId)) {
			w.WriteInt(m.keyDown)
		} else if skill.NeedsCharging(skill.Id(m.skillId)) {
			w.WriteInt(m.keyDown)
		}
		w.WriteByte(m.mask1)
		if t.Region() == "GMS" && t.MajorVersion() >= 95 {
			if m.attackType == AttackTypeRanged {
				w.WriteBool(m.javlin)
			}
		}
		// Attack-action / direction field. Legacy pre-79 GMS packs bLeft (bit7) +
		// nAction (bits0-6) into a SINGLE byte (v72 @0x85f9c2: Encode1
		// `(nAction&0x7F)|(bLeft<<7)`); GMS v79+ / JMS use a 2-byte short
		// (v79 @0x8c2adc: Encode2 `(bLeft<<15)|nAction`).
		if legacyGmsByteAction(t) {
			w.WriteByte(byte(m.mask2 & 0xFF))
		} else {
			w.WriteShort(m.mask2)
		}
		if t.Region() == "GMS" && t.MajorVersion() >= 95 {
			w.WriteInt(m.anotherCrc)
		}
		w.WriteByte(m.attackActionType)
		w.WriteByte(m.attackSpeed)
		w.WriteInt(m.attackTime)

		if m.attackType == AttackTypeMelee {
			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				w.WriteInt(0) // battle mage related
			}
		} else if m.attackType == AttackTypeRanged {
			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				w.WriteInt(0)
			}
			w.WriteShort(m.properBulletPosition)
			w.WriteShort(m.cashBulletPosition)
			w.WriteByte(m.nShootRange)
			if m.javlin && !skill.IsShootSkillNotConsumingBullet(skill.Id(m.skillId)) {
				w.WriteInt(m.bulletItemId)
			}
		} else if m.attackType == AttackTypeMagic {
			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				w.WriteInt(0)
			}
		}

		for i := range m.damageInfo {
			di := m.damageInfo[i]
			w.WriteByteArray(di.Encode(l, ctx)(options))
		}

		w.WriteShort(m.characterX)
		w.WriteShort(m.characterY)
		if m.attackType == AttackTypeRanged && !legacyGmsNoRangedBulletCoords(t) {
			w.WriteShort(m.bulletX)
			w.WriteShort(m.bulletY)
		}

		if skill.Id(m.skillId) == skill.NightWalkerStage3PoisonBombId {
			w.WriteShort(m.grenadeX)
			w.WriteShort(m.grenadeY)
		} else if skill.Id(m.skillId) == skill.ThunderBreakerStage3SparkId {
			w.WriteInt(m.reserveSpark)
		}
		// Trailing Evan-dragon block for magic attacks. ABSENT on the legacy pre-79
		// GMS client: v72 TryDoingMagicAttack @0x8625da writes characterX/Y then
		// SendPacket immediately (no dragon Encode1 after @0x863bff). Evan launched at
		// GMS v84, so the dragon field is naturally absent pre-79. Gate keeps v79+/JMS
		// unchanged.
		if m.attackType == AttackTypeMagic && !legacyGmsByteAction(t) {
			w.WriteBool(m.dragon)
			if m.dragon {
				w.WriteShort(m.dragonX)
				w.WriteShort(m.dragonY)
			}
		}
		return w.Bytes()
	}
}

func (m *AttackInfo) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.fieldKey = r.ReadByte()
		// Primary damage-randomizer (dr/crc anti-hack) block. Present GMS v84+,
		// NOT v95+ (off-by-one). CONFIRMED via the client attack senders: v83
		// melee (sub @0x66… in v83) writes no dr-block, while the v84, v87, and
		// v95 melee senders all insert dr0/dr1 here (after fieldKey), dr2/dr3
		// after the numAttacked mask, and randomDr/crc32 after skillId — exactly
		// +6 uint32 vs v83. The v84 magic sender is +6 only (no secondary
		// dr-block), so the magic 2dr block below stays v95+.
		if t.Region() == "GMS" && t.MajorVersion() >= 84 {
			m.dr0 = r.ReadUint32()
			m.dr1 = r.ReadUint32()
		}
		numAttackedAndDamageMask := r.ReadByte()
		m.hits = numAttackedAndDamageMask & 0xF
		m.damage = uint32((numAttackedAndDamageMask >> 4) & 0xF)

		if t.Region() == "GMS" && t.MajorVersion() >= 84 { // primary dr-block (v84+, see above)
			m.dr2 = r.ReadUint32()
			m.dr3 = r.ReadUint32()
		}

		m.skillId = r.ReadUint32()
		if t.Region() == "GMS" && t.MajorVersion() >= 95 {
			m.skillLevel = r.ReadByte() // nCombatOrders
		}

		if t.Region() == "GMS" && t.MajorVersion() >= 84 { // randomDr/crc32 complete the primary dr-block (v84+, see above)
			m.randomDr = r.ReadUint32()
			m.crc32 = r.ReadUint32()
		}

		if t.Region() == "GMS" && t.MajorVersion() >= 95 {
			if m.attackType == AttackTypeMagic {
				// Secondary dr-block for magic attacks. v95+ only: the v84 magic
				// sender (30 Encode tokens) is shorter than v84 melee and carries
				// no second dr-block, so this must NOT read for v84..94.
				_ = r.ReadUint32() //2dr0
				_ = r.ReadUint32() //2dr1
				_ = r.ReadUint32() //2dr2
				_ = r.ReadUint32() //2dr3
				_ = r.ReadUint32() //2rnd
				_ = r.ReadUint32() //2crc
			}
		}

		if !legacyGmsNoSkillDataCrc(t) {
			m.skillDataCrc = r.ReadUint32()
		}
		if !legacyGmsSingleCrc(t) {
			m.skillDataCrc2 = r.ReadUint32()
		}

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

		if legacyGmsByteAction(t) {
			b := r.ReadByte()
			m.mask2 = uint16(b)
			m.attackAction = int(b & 0x7F) // legacy: lower 7 bits
			m.left = int((b>>7)&0x01) == 1 // legacy: bit 7
		} else {
			m.mask2 = r.ReadUint16()
			m.attackAction = int(m.mask2 & 0x7FFF) // Extract lower 15 bits
			m.left = int((m.mask2>>15)&0x01) == 1  // Extract bit 15
		}
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

			// TODO(task-007): the `javlin` flag is tied to a specific skill mechanic
			// whose gameplay semantics are not yet fully understood (the original name
			// is a poor translation). Projectile consumption in atlas-channel's
			// character_attack_projectile.go intentionally bails out when javlin=true
			// to avoid mis-consuming. Revisit the gate at both sites when the mechanic
			// is characterized.
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
			di.Decode(l, ctx)(r, options)
			m.damageInfo = append(m.damageInfo, *di)
		}

		m.characterX = r.ReadUint16()
		m.characterY = r.ReadUint16()
		if m.attackType == AttackTypeRanged && !legacyGmsNoRangedBulletCoords(t) {
			m.bulletX = r.ReadUint16()
			m.bulletY = r.ReadUint16()
		}

		if skill.Id(m.skillId) == skill.NightWalkerStage3PoisonBombId {
			m.grenadeX = r.ReadUint16()
			m.grenadeY = r.ReadUint16()
		} else if skill.Id(m.skillId) == skill.ThunderBreakerStage3SparkId {
			m.reserveSpark = r.ReadUint32()
		}
		// Evan-dragon block absent on legacy pre-79 GMS (see Encode note).
		if m.attackType == AttackTypeMagic && !legacyGmsByteAction(t) {
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

func (m *AttackInfo) Javlin() bool {
	return m.javlin
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

// Builder methods for constructing AttackInfo in the server-send path.

func (m *AttackInfo) SetDamage(damage uint32) *AttackInfo {
	m.damage = damage
	return m
}

func (m *AttackInfo) SetHits(hits byte) *AttackInfo {
	m.hits = hits
	return m
}

func (m *AttackInfo) SetSkillId(skillId uint32) *AttackInfo {
	m.skillId = skillId
	return m
}

func (m *AttackInfo) SetOption(option byte) *AttackInfo {
	m.mask1 = option
	return m
}

func (m *AttackInfo) SetLeft(left bool) *AttackInfo {
	m.left = left
	return m
}

func (m *AttackInfo) SetAttackAction(attackAction int) *AttackInfo {
	m.attackAction = attackAction
	return m
}

func (m *AttackInfo) SetActionSpeed(actionSpeed byte) *AttackInfo {
	m.attackSpeed = actionSpeed
	return m
}

func (m *AttackInfo) SetKeydown(keydown uint32) *AttackInfo {
	m.keyDown = keydown
	return m
}

func (m *AttackInfo) SetBulletPosition(bulletX uint16, bulletY uint16) *AttackInfo {
	m.bulletX = bulletX
	m.bulletY = bulletY
	return m
}

func (m *AttackInfo) AddDamageInfo(di DamageInfo) *AttackInfo {
	m.damageInfo = append(m.damageInfo, di)
	return m
}
