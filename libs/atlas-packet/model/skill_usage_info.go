package model

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/constants"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type SkillUsageInfo struct {
	updateTime                uint32
	skillId                   uint32
	skillLevel                byte
	castX                     int16
	castY                     int16
	spiritJavelinItemId       uint32
	affectedPartyMemberBitmap uint8
	affectedMobIds            []uint32
	delay                     uint16
	magnetGrabs               []MagnetGrab
	direction                 bool
}

// MagnetGrab is one entry of the Monster Magnet grab table: the CMob object
// id the client picked up and whether it reports the grab as successful.
//
// Immutable value type (FR-1.6). objectId 0 is a LEGITIMATE wire value, not a
// sentinel: the client's encode loop walks its whole candidate array, and
// slots whose CanGoThrough/CanWalkThrough probe failed were released earlier in
// the same function, leaving a null ZRef whose id reads 0 (gms_83
// CUserLocal::TryDoingMonsterMagnet @0x96C215). Dropping such entries is the
// server-side validator's job, not the decoder's.
type MagnetGrab struct {
	objectId uint32
	grabbed  bool
}

func NewMagnetGrab(objectId uint32, grabbed bool) MagnetGrab {
	return MagnetGrab{objectId: objectId, grabbed: grabbed}
}

func (m MagnetGrab) ObjectId() uint32 { return m.objectId }

func (m MagnetGrab) Grabbed() bool { return m.grabbed }

func (m *SkillUsageInfo) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		t := tenant.MustFromContext(ctx)
		m.updateTime = r.ReadUint32()
		m.skillId = r.ReadUint32()
		m.skillLevel = r.ReadByte()
		// Monster Magnet is delivered on this same opcode but its body is
		// written by a DIFFERENT client function (CUserLocal::TryDoingMonsterMagnet)
		// than every other skill here (CUserLocal::SendSkillUseRequest), and it
		// diverges immediately after skillLevel. Consume it and return: the
		// magnet shares no suffix with the arms below, and an early return makes
		// the mutual exclusion structural rather than list-maintained (FR-1.3).
		if isMonsterMagnet(t, m.skillId) {
			m.decodeMagnet(r, legacyMagnetLayout(t))
			return
		}
		// gms_48 (CUserLocal::SendSkillUseRequest @0x6AFA91) and gms_61 (@0x7BA213)
		// have NO is_antirepeat_buff_skill gate at all in SendSkillUseRequest — the
		// function goes straight from Encode1(nSLV) to the 4121006/bitmap/mob-count
		// blocks. The gate first appears at gms_72 (@0x8774D9, is_antirepeat_buff_skill
		// @0x877789) and is present on every version from there through gms_95, and on
		// jms_185 (@0xA3DE65, is_antirepeat_buff_skill @0xA3E223 — verified, 2311001 is
		// a member). Unconditionally reading castX/castY for isAntiRepeatBuffSkill
		// members on gms_48/gms_61 over-reads 4 bytes and misaligns every field after
		// it (task-163 DIV-1, docs/tasks/task-163-priest-dispel-party/version-findings.md).
		if isAntiRepeatBuffSkill(skill.Id(m.skillId)) &&
			((t.IsRegion("GMS") && t.MajorAtLeast(72)) || t.IsRegion("JMS")) {
			m.castX = r.ReadInt16()
			m.castY = r.ReadInt16()
		}
		if skill.Id(m.skillId) == skill.NightLordShadowStarsId {
			m.spiritJavelinItemId = r.ReadUint32()
		}

		if isPartyBuff(skill.Id(m.skillId)) {
			m.affectedPartyMemberBitmap = r.ReadByte()
			if skill.Id(m.skillId) == skill.PriestDispelId {
				m.delay = r.ReadUint16()
			}
		}
		if isMobAffectingBuff(skill.Id(m.skillId)) {
			nMobCount := r.ReadByte()
			m.affectedMobIds = make([]uint32, 0, nMobCount)
			for range nMobCount {
				m.affectedMobIds = append(m.affectedMobIds, r.ReadUint32())
			}
			m.delay = r.ReadUint16()
		}
	}
}

func (m *SkillUsageInfo) SkillId() uint32 {
	return m.skillId
}

func (m *SkillUsageInfo) SkillLevel() byte {
	return m.skillLevel
}

func (m *SkillUsageInfo) SpiritJavelinItemId() uint32 {
	return m.spiritJavelinItemId
}

func (m *SkillUsageInfo) AffectedPartyMemberBitmap() byte {
	return m.affectedPartyMemberBitmap
}

func (m *SkillUsageInfo) AffectedMobIds() []uint32 {
	return m.affectedMobIds
}

func (m *SkillUsageInfo) Delay() uint16 {
	return m.delay
}

// MagnetGrabs returns the Monster Magnet grab table. Empty for every other
// skill. On gms_48 the client's leading caster entry has already been
// discarded and every remaining entry is marked grabbed (that version sends no
// per-entry result).
func (m *SkillUsageInfo) MagnetGrabs() []MagnetGrab {
	return m.magnetGrabs
}

// Direction is the caster's facing bit (CUserLocal.stance & 1; true = facing
// left) as sent on the Monster Magnet body. Always false on gms_48, which
// sends no direction byte, and for every non-magnet skill.
func (m *SkillUsageInfo) Direction() bool {
	return m.direction
}

// SkillUsageInfoBuilder fluently constructs SkillUsageInfo values for
// callers that don't go through Decode (today: tests). The wire decoder
// remains the canonical production path.
type SkillUsageInfoBuilder struct {
	info SkillUsageInfo
}

func NewSkillUsageInfoBuilder() *SkillUsageInfoBuilder {
	return &SkillUsageInfoBuilder{}
}

func (b *SkillUsageInfoBuilder) SetUpdateTime(v uint32) *SkillUsageInfoBuilder {
	b.info.updateTime = v
	return b
}

func (b *SkillUsageInfoBuilder) SetSkillId(v uint32) *SkillUsageInfoBuilder {
	b.info.skillId = v
	return b
}

func (b *SkillUsageInfoBuilder) SetSkillLevel(v byte) *SkillUsageInfoBuilder {
	b.info.skillLevel = v
	return b
}

func (b *SkillUsageInfoBuilder) SetCastX(v int16) *SkillUsageInfoBuilder {
	b.info.castX = v
	return b
}

func (b *SkillUsageInfoBuilder) SetCastY(v int16) *SkillUsageInfoBuilder {
	b.info.castY = v
	return b
}

func (b *SkillUsageInfoBuilder) SetSpiritJavelinItemId(v uint32) *SkillUsageInfoBuilder {
	b.info.spiritJavelinItemId = v
	return b
}

func (b *SkillUsageInfoBuilder) SetAffectedPartyMemberBitmap(v uint8) *SkillUsageInfoBuilder {
	b.info.affectedPartyMemberBitmap = v
	return b
}

func (b *SkillUsageInfoBuilder) SetAffectedMobIds(v []uint32) *SkillUsageInfoBuilder {
	b.info.affectedMobIds = v
	return b
}

func (b *SkillUsageInfoBuilder) SetDelay(v uint16) *SkillUsageInfoBuilder {
	b.info.delay = v
	return b
}

func (b *SkillUsageInfoBuilder) SetMagnetGrabs(v []MagnetGrab) *SkillUsageInfoBuilder {
	b.info.magnetGrabs = v
	return b
}

func (b *SkillUsageInfoBuilder) SetDirection(v bool) *SkillUsageInfoBuilder {
	b.info.direction = v
	return b
}

func (b *SkillUsageInfoBuilder) Build() SkillUsageInfo {
	return b.info
}

func isMobAffectingBuff(skillId skill.Id) bool {
	// TODO this is not all inclusive 32111004 32121007 33121007 35111013
	return skill.Is(skillId,
		skill.WarriorIronBodyId,
		skill.FighterRageId,
		skill.CrusaderArmorCrashId,
		skill.HeroMapleWarriorId,
		skill.PageThreatenId,
		skill.WhiteKnightMagicCrashId,
		skill.PaladinMapleWarriorId,
		skill.SpearmanHyperBodyId,
		skill.SpearmanIronWillId,
		skill.DragonKnightPowerCrashId,
		skill.DarkKnightMapleWarriorId,
		skill.FirePoisonWizardSlowId,
		skill.FirePoisonWizardMeditationId,
		skill.FirePoisonArchMagicianMapleWarriorId,
		skill.IceLightningWizardSlowId,
		skill.IceLightningWizardMeditationId,
		skill.IceLightningArchMagicianMapleWarriorId,
		skill.ClericBlessId,
		skill.PriestDoomId,
		skill.PriestDispelId,
		skill.PriestHolySymbolId,
		skill.BishopMapleWarriorId,
		skill.BishopHolyShieldId,
		skill.BowmasterMapleWarriorId,
		skill.BowmasterSharpEyesId,
		skill.MarksmanMapleWarriorId,
		// MarksmanSharpEyesId (3221002) is deliberately ABSENT. gms_v83
		// CUserLocal::DoActiveSkill compares esi against 3221002 at 0x967ff7
		// and dispatches to loc_969275, which pushes dwTargetFlag = 2 — the
		// party bit only, never the mob bit (4). DoActiveSkill_StatChange
		// @0x969e21 therefore leaves nMobCount at -1 and
		// CUserLocal::SendSkillUseRequest @0x96d399 emits no mob block for
		// this skill at all. Reading one here consumed the always-present
		// trailing delay short instead.
		skill.AssassinHasteId,
		skill.HermitMesoUpId,
		skill.HermitShadowWebId,
		skill.NightLordMapleWarriorId,
		skill.BanditHasteId,
		skill.ShadowerMapleWarriorId,
		skill.BuccaneerMapleWarriorId,
		skill.BuccaneerSpeedInfusionId,
		skill.CorsairMapleWarriorId,
		skill.CorsairSpeedInfusionId,
		skill.DawnWarriorStage1IronBodyId,
		skill.DawnWarriorStage2RageId,
		skill.BlazeWizardStage2SlowId,
		skill.BlazeWizardStage2MeditationId,
		skill.ThunderBreakerStage3SpeedInfusionId,
		skill.NightWalkerStage2HasteId,
		skill.NightWalkerStage3ShadowWebId,
		skill.AranStage4MapleWarriorId,
		skill.AranStage4ComboBarrierId,
		skill.EvanStage5MagicShieldId,
		skill.EvanStage6SlowId,
		skill.EvanStage7MagicResistanceId,
		skill.EvanStage8RecoveryAuraId,
		skill.EvanStage9MapleWarriorId,
		skill.EvanStage10BlessingOfTheOnyxId,
	)
}

func isPartyBuff(skillId skill.Id) bool {
	// TODO this is not all inclusive 32111004 32121007 33121007 35111013
	return skill.Is(skillId,
		skill.FighterRageId,
		skill.HeroMapleWarriorId,
		skill.PaladinMapleWarriorId,
		skill.SpearmanHyperBodyId,
		skill.SpearmanIronWillId,
		skill.DarkKnightMapleWarriorId,
		skill.FirePoisonWizardMeditationId,
		skill.FirePoisonArchMagicianMapleWarriorId,
		skill.IceLightningWizardMeditationId,
		skill.IceLightningArchMagicianMapleWarriorId,
		skill.ClericHealId,
		skill.ClericBlessId,
		skill.PriestDispelId,
		// BishopResurrectionId: the v83 client writes the affected-member
		// bitmap for 2321006 (CUserLocal::SendSkillUseRequest @0x96d399 —
		// bitmap byte present whenever non-zero, and FindParty @0x96db3f
		// special-cases 2321006 to set bits for DEAD members only; the client
		// never sends the packet with a zero bitmap). Not anti-repeat
		// (@0x96d6ca) and not mob-affecting: wire is
		// updateTime(4) skillId(4) slv(1) bitmap(1) delay(2).
		skill.BishopResurrectionId,
		skill.PriestHolySymbolId,
		skill.BishopMapleWarriorId,
		skill.BishopHolyShieldId,
		skill.BowmasterMapleWarriorId,
		skill.BowmasterSharpEyesId,
		skill.MarksmanMapleWarriorId,
		skill.MarksmanSharpEyesId,
		skill.AssassinHasteId,
		skill.HermitMesoUpId,
		skill.NightLordMapleWarriorId,
		skill.BanditHasteId,
		skill.ShadowerMapleWarriorId,
		skill.BuccaneerMapleWarriorId,
		skill.BuccaneerTimeLeapId,
		skill.CorsairMapleWarriorId,
		skill.DawnWarriorStage2RageId,
		skill.BlazeWizardStage2MeditationId,
		skill.NightWalkerStage2HasteId,
		skill.AranStage4MapleWarriorId,
		skill.AranStage4ComboBarrierId,
		skill.EvanStage5MagicShieldId,
		skill.EvanStage7MagicResistanceId,
		// skill.EvanStage8RecoveryAuraId,
		skill.EvanStage9MapleWarriorId,
	)
}

func isAntiRepeatBuffSkill(skillId skill.Id) bool {
	// TODO this is not all inclusive 32111004 32121007 33121007 35111013
	return skill.Is(skillId,
		skill.WarriorIronBodyId,
		skill.FighterRageId,
		skill.CrusaderArmorCrashId,
		skill.HeroMapleWarriorId,
		skill.PageThreatenId,
		skill.WhiteKnightMagicCrashId,
		skill.PaladinMapleWarriorId,
		skill.SpearmanHyperBodyId,
		skill.SpearmanIronWillId,
		skill.DragonKnightPowerCrashId,
		skill.DarkKnightMapleWarriorId,
		skill.FirePoisonWizardSlowId,
		skill.FirePoisonWizardMeditationId,
		skill.FirePoisonArchMagicianMapleWarriorId,
		skill.IceLightningWizardSlowId,
		skill.IceLightningWizardMeditationId,
		skill.IceLightningArchMagicianMapleWarriorId,
		skill.ClericBlessId,
		skill.PriestDispelId,
		skill.PriestHolySymbolId,
		skill.BishopMapleWarriorId,
		skill.BishopHolyShieldId,
		skill.BowmasterMapleWarriorId,
		skill.BowmasterSharpEyesId,
		skill.MarksmanMapleWarriorId,
		// MarksmanSharpEyesId (3221002) is deliberately ABSENT — the client's
		// is_antirepeat_buff_skill lists 3121000, 3121002 (Bowmaster Sharp
		// Eyes) and 3221000, and then stops; 3221002 appears nowhere in it on
		// ANY supported client. Verified by decompiling that function per
		// version: gms_v72 @0x877789, gms_v79 @0x8c42bd, gms_v83 @0x96d6ca,
		// gms_v84 @0x9ad4e4, gms_v87 @0x9f20fc, gms_v92 @0x919150, gms_v95
		// @0x939dc0, jms_v185 @0xa3e223. Listing it here made the decoder eat
		// 4 castX/castY bytes the client never sends, pushing the
		// affected-party bitmap past the end of the 12-byte packet so it read
		// 0 and Sharp Eyes only ever buffed the Marksman themselves.
		skill.AssassinHasteId,
		skill.HermitMesoUpId,
		skill.HermitShadowWebId,
		skill.NightLordMapleWarriorId,
		skill.BanditHasteId,
		skill.ShadowerMapleWarriorId,
		skill.BuccaneerMapleWarriorId,
		skill.BuccaneerSpeedInfusionId,
		skill.CorsairMapleWarriorId,
		skill.CorsairSpeedInfusionId,
		skill.DawnWarriorStage1IronBodyId,
		skill.DawnWarriorStage2RageId,
		skill.BlazeWizardStage2SlowId,
		skill.BlazeWizardStage2MeditationId,
		skill.ThunderBreakerStage3SpeedInfusionId,
		skill.NightWalkerStage2HasteId,
		skill.NightWalkerStage3ShadowWebId,
		skill.AranStage4MapleWarriorId,
		skill.AranStage4ComboBarrierId,
		skill.EvanStage5MagicShieldId,
		skill.EvanStage6SlowId,
		skill.EvanStage7MagicResistanceId,
		skill.EvanStage8RecoveryAuraId,
		skill.EvanStage9MapleWarriorId,
		skill.EvanStage10BlessingOfTheOnyxId,
	)
}

// magnetEntrySizeModern is the on-wire size of one gms_61+/jms grab entry:
// objectId uint32 + grabbed byte. magnetEntrySizeLegacy is the gms_48 entry:
// objectId uint32, no result byte.
const (
	magnetEntrySizeModern = 5
	magnetEntrySizeLegacy = 4
)

// isMonsterMagnet resolves the incoming wire id through the tenant's version
// set and reports whether it is one of the three Monster Magnet identities
// (FR-1.1). Identity-keyed rather than a raw skill.Is compare against
// 1121001/1221001/1321001 (task-187): the wire id happens to be stable across
// every provisioned version for these three, but the resolver is the contract
// this file's remaining raw-id lists exist to be migrated onto, and a raw
// compare here would be the wrong precedent for the next branch added.
func isMonsterMagnet(t tenant.Model, skillId uint32) bool {
	set := constants.For(t.Region(), t.MajorVersion(), t.MinorVersion())
	id, ok := set.Skill.Resolve(skill.Id(skillId))
	if !ok {
		return false
	}
	return skill.IsIdentity(id,
		skill.HeroMonsterMagnet,
		skill.PaladinMonsterMagnet,
		skill.DarkKnightMonsterMagnet,
	)
}

// legacyMagnetLayout reports whether this tenant's client sends the gms_48
// magnet body (byte count, no per-entry result, trailing delay short, no
// direction byte, leading caster-id entry) rather than the modern one.
//
// The split is gms_48 vs EVERYTHING else — deliberately NARROWER than the
// isAntiRepeatBuffSkill gate above, which splits at gms_72. Do not harmonise
// the two. Verified by decompiling CUserLocal::TryDoingMonsterMagnet per
// version: gms_48 @0x6AD842 (COutPacket ctor `push 46h` @0x6ADABC; entryCount
// Encode1 @0x6ADB02; per-entry Encode4 @0x6ADB1B; delay Encode2 @0x6ADB29; the
// caster's own object id is inserted at index 0 by ZArray<ulong>::InsertBefore
// @0x6AD977-0x6AD987 BEFORE the mob loop @0x6ADA89-0x6ADA99, both reading
// offset +0x654), versus the modern shape at gms_61 @0x7B9684, gms_72
// @0x876605, gms_79 @0x8C3117, gms_83 @0x96C215, gms_84 @0x9ABDB7, gms_87
// @0x9F086F, gms_92 @0x91F2A0, gms_95 @0x940570 and jms_185 @0xA3C61C — all of
// which Encode4 the grab count, Encode4/Encode1 per entry, and Encode1 the
// direction with NO trailing delay. jms takes the modern branch.
func legacyMagnetLayout(t tenant.Model) bool {
	return t.IsRegion("GMS") && !t.MajorAtLeast(61)
}

// decodeMagnet consumes the Monster Magnet body. It is a REPLACEMENT body, not
// an additive suffix of the common prefix the other arms share, which is why
// Decode returns immediately after calling it.
//
// The per-entry loops are bounded by the bytes actually available as well as by
// the client-supplied count: the shared reader returns zero WITHOUT advancing
// pos once exhausted (atlas-socket/request/reader.go), so an unbounded loop
// over a hostile 0xFFFFFFFF count would spin ~4 billion times on the channel's
// packet-handling goroutine producing nothing.
func (m *SkillUsageInfo) decodeMagnet(r *request.Reader, legacy bool) {
	if legacy {
		entryCount := int(r.ReadByte())
		if maxEntries := r.Available() / magnetEntrySizeLegacy; entryCount > maxEntries {
			entryCount = maxEntries
		}
		m.magnetGrabs = make([]MagnetGrab, 0, entryCount)
		for i := range entryCount {
			objectId := r.ReadUint32()
			// entry[0] is the CASTER's own object id, not a monster.
			if i == 0 {
				continue
			}
			m.magnetGrabs = append(m.magnetGrabs, NewMagnetGrab(objectId, true))
		}
		m.delay = r.ReadUint16()
		return
	}

	grabCount := int(r.ReadUint32())
	if maxEntries := r.Available() / magnetEntrySizeModern; grabCount > maxEntries {
		grabCount = maxEntries
	}
	m.magnetGrabs = make([]MagnetGrab, 0, grabCount)
	for range grabCount {
		objectId := r.ReadUint32()
		// gms_83 @0x96C215 encodes `COutPacket::Encode1(v65, *v40 == 3)` — a
		// BOOL, not an enum. The 3 is the client's own prop-roll sentinel.
		grabbed := r.ReadByte() != 0
		m.magnetGrabs = append(m.magnetGrabs, NewMagnetGrab(objectId, grabbed))
	}
	m.direction = r.ReadByte() != 0
}
