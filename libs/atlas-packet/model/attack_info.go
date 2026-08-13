package model

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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

// ExplodedMesoDrop is one entry of the meso-explosion trailing drop list: the
// detonated meso drop's object id (CDrop field +32) plus the client's bitmask
// of which attacked-mob indices that drop's explosion damaged. The hit mask is
// retained for wire fidelity only; server logic consumes just the ids.
type ExplodedMesoDrop struct {
	dropId  uint32
	hitMask byte
}

func NewExplodedMesoDrop(dropId uint32, hitMask byte) ExplodedMesoDrop {
	return ExplodedMesoDrop{dropId: dropId, hitMask: hitMask}
}

func (e ExplodedMesoDrop) DropId() uint32 { return e.dropId }
func (e ExplodedMesoDrop) HitMask() byte  { return e.hitMask }

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

// gmsAttackDrBlocks reports whether the serverbound attack head carries the
// damage-randomizer / anti-hack blocks at all. GMS v84+.
//
// v83's magic sender (@0x956da2) writes fieldKey, the numAttacked/damage mask,
// skillId, the two skill-data CRCs and then goes straight to mask1 -- no dr
// words anywhere. v84's (@0x9942f3) inserts dr0/dr1 after fieldKey, dr2/dr3
// after the mask, and randomDr/crc32 after skillId.
func gmsAttackDrBlocks(t tenant.Model) bool {
	return t.Region() == "GMS" && t.MajorVersion() >= 84
}

// gmsMagicSecondaryDrBlock reports whether a MAGIC attack carries a SECOND
// damage-randomizer block (2dr0..2dr3, 2rnd, 2crc -- six uint32s, 24 bytes)
// immediately after the primary randomDr/crc32 pair.
//
// It is present on every GMS client that carries the primary block, i.e. the
// two travel together and this predicate is deliberately the same boundary as
// gmsAttackDrBlocks rather than an independent one. IDA-verified per version,
// reading the magic sender's Encode sequence from its COutPacket ctor:
//
//	v83 @0x956da2 — absent (no dr block of any kind)
//	v84 @0x9942f3 — PRESENT: Encode4 ×4 @0x99442c/0x994440/0x994454/0x994468,
//	                Random ×2 → Encode4 @0x9944c4, GetCrc32 → Encode4 @0x9944eb
//	v87 @0x9d8656 — PRESENT: Encode4 ×4 @0x9d878f/0x9d87a3/0x9d87b7/0x9d87cb,
//	                Random ×2 → Encode4 @0x9d8827, GetCrc32 → Encode4 @0x9d884e
//	v92 @0x9086e4 — PRESENT: Encode4 ×4 @0x9087db/0x9087ef/0x908803/0x908817,
//	                Random ×2 → Encode4 @0x90883a, GetCrc32 → Encode4 @0x908861
//	v95           — PRESENT (the version this read was originally written for)
//
// This REPLACES a `>= 95` gate whose comment asserted "the v84 magic sender
// (30 Encode tokens) is shorter than v84 melee and carries no second dr-block,
// so this must NOT read for v84..94". The v84 sender above disproves the
// premise. The cost of the wrong gate was 24 unread bytes on every v84/v87/v92
// magic attack: skillDataCrc onward decoded from the middle of the secondary
// block, so the per-mob damage loop read garbage monster object ids and
// atlas-monsters dropped every one of them ("Unable to get monster [N]" for
// ids far outside the live range) -- magic attacks dealt no damage at all.
func gmsMagicSecondaryDrBlock(t tenant.Model) bool {
	return gmsAttackDrBlocks(t)
}

// gmsMagicTrailingWord reports whether a MAGIC attack writes one extra uint32
// between attackTime and the per-mob damage loop. GMS v92+.
//
// IDA-verified by reading what follows the attackTime Encode4 in each magic
// sender. v84 @0x994678 and v87 @0x9d89db are followed directly by the per-mob
// block (mob id Encode4 @0x99470d / @0x9d8a70, then the Encode1 run and the
// five Encode2s). v92 writes TWO Encode4s there -- attackTime @0x9089e9 then
// this word @0x9089f8 -- before its per-mob block at @0x908b26. v95 matches
// v92, which is the version the original `>= 95` gate was written against.
//
// So v92 was short by a further 4 bytes on top of the secondary-dr-block gap.
func gmsMagicTrailingWord(t tenant.Model) bool {
	return t.Region() == "GMS" && t.MajorVersion() >= 92
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
	exJablin             bool
	properBulletPosition uint16
	cashBulletPosition   uint16
	nShootRange          byte
	bulletItemId         uint32
	dragon               bool
	dragonX              uint16
	dragonY              uint16
	bulletX              uint16
	bulletY              uint16
	explodedMesoDrops    []ExplodedMesoDrop
	mesoDelay            uint16
}

// Encode is the symmetric mirror of Decode: it serializes the client->server
// attack request. Every version gate here MUST match Decode field-for-field:
// the primary dr-block and the magic secondary dr-block are GMS v84+
// (gmsAttackDrBlocks / gmsMagicSecondaryDrBlock), the magic trailing word is
// GMS v92+ (gmsMagicTrailingWord), and skillLevel / anotherCrc / the melee and
// ranged per-type ints are GMS v95+. The AttackInfo round-trip test relies on
// this symmetry — any drift surfaces as unconsumed bytes for the affected
// version.
func (m *AttackInfo) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		// Meso Explosion (4211006) is a CLOSE_RANGE_ATTACK variant written by a
		// dedicated client sender. Its three deltas (per-mob count byte, trailing
		// drop list, trailing delay) are byte-identical across every IDA-verified
		// version (task-150 design §2.1), so one flag and no new version gates.
		isMesoExplosion := skill.Id(m.skillId) == skill.ChiefBanditMesoExplosionId
		w := response.NewWriter(l)
		w.WriteByte(m.fieldKey)
		if gmsAttackDrBlocks(t) { // primary dr-block (v84+)
			w.WriteInt(m.dr0)
			w.WriteInt(m.dr1)
		}
		w.WriteByte((m.hits & 0xF) | byte((m.damage&0xF)<<4))
		if gmsAttackDrBlocks(t) { // primary dr-block (v84+)
			w.WriteInt(m.dr2)
			w.WriteInt(m.dr3)
		}
		w.WriteInt(m.skillId)
		if t.Region() == "GMS" && t.MajorVersion() >= 95 {
			w.WriteByte(m.skillLevel) // nCombatOrders
		}
		if gmsAttackDrBlocks(t) { // randomDr/crc32 complete the primary dr-block (v84+)
			w.WriteInt(m.randomDr)
			w.WriteInt(m.crc32)
		}
		if m.attackType == AttackTypeMagic && gmsMagicSecondaryDrBlock(t) {
			// Secondary dr-block for magic attacks — GMS v84+, see the
			// predicate for the per-version IDA evidence.
			w.WriteInt(0) // 2dr0
			w.WriteInt(0) // 2dr1
			w.WriteInt(0) // 2dr2
			w.WriteInt(0) // 2dr3
			w.WriteInt(0) // 2rnd
			w.WriteInt(0) // 2crc
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
				w.WriteBool(m.exJablin)
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
			if m.spiritJavelin() && !skill.IsShootSkillNotConsumingBullet(skill.Id(m.skillId)) {
				w.WriteInt(m.bulletItemId)
			}
		} else if m.attackType == AttackTypeMagic {
			// GMS v92+ — see gmsMagicTrailingWord. v87 omits it.
			if gmsMagicTrailingWord(t) {
				w.WriteInt(0)
			}
		}

		for i := range m.damageInfo {
			di := m.damageInfo[i]
			w.WriteByteArray(di.Encode(l, ctx)(options))
		}

		w.WriteShort(m.characterX)
		w.WriteShort(m.characterY)
		if isMesoExplosion {
			w.WriteByte(byte(len(m.explodedMesoDrops)))
			for _, e := range m.explodedMesoDrops {
				w.WriteInt(e.dropId)
				w.WriteByte(e.hitMask)
			}
			w.WriteShort(m.mesoDelay)
		}
		if m.attackType == AttackTypeRanged && !legacyGmsNoRangedBulletCoords(t) {
			w.WriteShort(m.bulletX)
			w.WriteShort(m.bulletY)
		}

		if skill.IsGrenadeSkill(skill.Id(m.skillId)) {
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
		// Primary damage-randomizer (dr/crc anti-hack) block — GMS v84+, see
		// gmsAttackDrBlocks. v83 writes no dr words at all; v84/v87/v92/v95
		// insert dr0/dr1 here (after fieldKey), dr2/dr3 after the numAttacked
		// mask, and randomDr/crc32 after skillId.
		//
		// The magic secondary dr-block below is the SAME boundary, not v95+:
		// the claim that "the v84 magic sender is +6 only" was wrong and cost
		// v84/v87/v92 every magic attack (gmsMagicSecondaryDrBlock).
		if gmsAttackDrBlocks(t) {
			m.dr0 = r.ReadUint32()
			m.dr1 = r.ReadUint32()
		}
		numAttackedAndDamageMask := r.ReadByte()
		m.hits = numAttackedAndDamageMask & 0xF
		m.damage = uint32((numAttackedAndDamageMask >> 4) & 0xF)

		if gmsAttackDrBlocks(t) { // primary dr-block (v84+, see above)
			m.dr2 = r.ReadUint32()
			m.dr3 = r.ReadUint32()
		}

		m.skillId = r.ReadUint32()
		// Meso Explosion (4211006) is a CLOSE_RANGE_ATTACK variant written by a
		// dedicated client sender. Its three deltas (per-mob count byte, trailing
		// drop list, trailing delay) are byte-identical across every IDA-verified
		// version (task-150 design §2.1), so one flag and no new version gates.
		isMesoExplosion := skill.Id(m.skillId) == skill.ChiefBanditMesoExplosionId
		if t.Region() == "GMS" && t.MajorVersion() >= 95 {
			m.skillLevel = r.ReadByte() // nCombatOrders
		}

		if gmsAttackDrBlocks(t) { // randomDr/crc32 complete the primary dr-block (v84+, see above)
			m.randomDr = r.ReadUint32()
			m.crc32 = r.ReadUint32()
		}

		if m.attackType == AttackTypeMagic && gmsMagicSecondaryDrBlock(t) {
			// Secondary dr-block for magic attacks — see the predicate for the
			// per-version IDA evidence. Skipping it on v84/v87/v92 left the
			// decode 24 bytes short and turned every magic attack's mob ids
			// into garbage.
			_ = r.ReadUint32() // 2dr0
			_ = r.ReadUint32() // 2dr1
			_ = r.ReadUint32() // 2dr2
			_ = r.ReadUint32() // 2dr3
			_ = r.ReadUint32() // 2rnd
			_ = r.ReadUint32() // 2crc
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
		// bit 6 is the Spirit Javelin (Shadow Stars active) flag — see spiritJavelin().
		m.unknown2 = int((m.mask1 >> 7) & 0x01) // Extract bit 7

		// GMS v95+ writes an explicit "ExJablin applied" bool right after mask1
		// (ranged only). It is a SEPARATE field from the mask1 bit-6 gate below —
		// consume it for alignment but do NOT use it to gate the star id. Verified
		// across every client version: the bulletItemId gate is mask1 bit 6.
		if t.Region() == "GMS" && t.MajorVersion() >= 95 {
			if m.attackType == AttackTypeRanged {
				m.exJablin = r.ReadBool()
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

			// Spirit Javelin / Shadow Stars star id. When the caster has Shadow Stars
			// active the client draws imbued throwing stars from the buff and appends
			// the chosen star id here, gated on mask1 bit 6 (verified in every GMS
			// client v48–v95; v87's IDB names it nSpiritJavelin; jms v185 assumed to
			// follow v87). Missing this read decodes the per-mob damage-info loop 4
			// bytes misaligned, so monster damage silently drops.
			if m.spiritJavelin() && !skill.IsShootSkillNotConsumingBullet(skill.Id(m.skillId)) {
				m.bulletItemId = r.ReadUint32()
			}
		} else if m.attackType == AttackTypeMagic {
			// One trailing word between attackTime and the per-mob loop, GMS
			// v92+ — see gmsMagicTrailingWord for the per-version evidence.
			// v87 does NOT write it, so the previous `>= 95` gate was right for
			// v87 and wrong for v92.
			if gmsMagicTrailingWord(t) {
				_ = r.ReadUint32()
			}
		}

		for range m.damage {
			var di *DamageInfo
			if isMesoExplosion {
				di = NewMesoExplosionDamageInfo()
			} else {
				di = NewDamageInfo(m.hits)
			}
			di.Decode(l, ctx)(r, options)
			m.damageInfo = append(m.damageInfo, *di)
		}

		m.characterX = r.ReadUint16()
		m.characterY = r.ReadUint16()
		if isMesoExplosion {
			dropCount := r.ReadByte()
			for range dropCount {
				m.explodedMesoDrops = append(m.explodedMesoDrops, ExplodedMesoDrop{
					dropId:  r.ReadUint32(),
					hitMask: r.ReadByte(),
				})
			}
			m.mesoDelay = r.ReadUint16()
		}
		if m.attackType == AttackTypeRanged && !legacyGmsNoRangedBulletCoords(t) {
			m.bulletX = r.ReadUint16()
			m.bulletY = r.ReadUint16()
		}

		if skill.IsGrenadeSkill(skill.Id(m.skillId)) {
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

// SpiritJavelin reports whether mask1 bit 6 — the client's Spirit Javelin
// (Shadow Stars active) flag — is set. When set, the caster throws stars imbued
// by the Shadow Stars buff, so the ranged attack carries an explicit star id and
// per-attack projectile consumption is skipped (the stars were charged in bulk
// at cast time).
func (m *AttackInfo) SpiritJavelin() bool {
	return m.spiritJavelin()
}

func (m *AttackInfo) spiritJavelin() bool {
	return (m.mask1>>6)&0x01 == 1
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

// GrenadeX / GrenadeY report the landing point of a thrown-grenade attack --
// the trailing coordinate pair the client appends for the grenade skills (see
// Decode's skill.IsGrenadeSkill arm). It is the point the BOMB lands
// on, which is a function of how long the key was held, and is NOT the
// caster's own position: a server-side effect anchored at the caster instead
// of here lands in the wrong place by the whole throw distance (task-218
// field report #4).
//
// Zero for every attack that carries no grenade block, so callers must gate on
// the skill rather than on the value.
func (m *AttackInfo) GrenadeX() uint16 {
	return m.grenadeX
}

func (m *AttackInfo) GrenadeY() uint16 {
	return m.grenadeY
}

// ExplodedMesoDrops returns the drop object ids listed by a meso-explosion
// attack. Empty for every other attack (FR-3).
func (m *AttackInfo) ExplodedMesoDrops() []uint32 {
	ids := make([]uint32, 0, len(m.explodedMesoDrops))
	for _, e := range m.explodedMesoDrops {
		ids = append(ids, e.dropId)
	}
	return ids
}

// ExplodedMesoDropEntries returns the full wire entries (id + hit mask).
func (m *AttackInfo) ExplodedMesoDropEntries() []ExplodedMesoDrop {
	return m.explodedMesoDrops
}

func (m *AttackInfo) MesoDelay() uint16 {
	return m.mesoDelay
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

// SetGrenadePosition sets the thrown-grenade landing point. Only encoded for
// the grenade skills (see Encode/Decode).
func (m *AttackInfo) SetGrenadePosition(grenadeX uint16, grenadeY uint16) *AttackInfo {
	m.grenadeX = grenadeX
	m.grenadeY = grenadeY
	return m
}

// SetBulletItemId sets the Spirit Javelin / Shadow Stars star id carried on the
// wire when mask1 bit 6 is set. Only encoded for ranged attacks with the bit set.
func (m *AttackInfo) SetBulletItemId(bulletItemId uint32) *AttackInfo {
	m.bulletItemId = bulletItemId
	return m
}

func (m *AttackInfo) AddDamageInfo(di DamageInfo) *AttackInfo {
	m.damageInfo = append(m.damageInfo, di)
	return m
}

func (m *AttackInfo) SetExplodedMesoDrops(entries []ExplodedMesoDrop) *AttackInfo {
	m.explodedMesoDrops = entries
	return m
}

func (m *AttackInfo) SetMesoDelay(delay uint16) *AttackInfo {
	m.mesoDelay = delay
	return m
}
