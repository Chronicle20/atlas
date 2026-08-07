package model

import (
	"encoding/binary"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

func TestIsMobAffectingBuff_PriestDoom(t *testing.T) {
	if !isMobAffectingBuff(skill.PriestDoomId) {
		t.Fatalf("isMobAffectingBuff(PriestDoomId) = false, want true")
	}
}

// TestDecodeDispelGms61NoCastXY pins the task-163 DIV-1 finding: gms_48 and
// gms_61 have NO is_antirepeat_buff_skill gate in CUserLocal::SendSkillUseRequest
// at all (gms_61 IDA-verified @0x7BA213 — the function goes straight from
// Encode1(nSLV) to the bitmap/delay/mob-count blocks; gms_48 @0x6AFA91 is
// identical in shape). Wire layout for a 2311001 (Dispel) cast on gms_61:
//
//	updateTime(4) skillId(4) slv(1) bitmap(1) delay(2) mobCount(1) mobIds(4xN) delay(2)
//
// Before the fix, the version-blind decoder unconditionally read 4 phantom
// castX/castY bytes for 2311001 (a member of isAntiRepeatBuffSkill), consuming
// the bitmap byte and the first delay byte as castX/castY and reading the real
// bitmap byte from the low byte of the client's actual delay field — decoding
// to 0 or garbage instead of the true non-zero party bitmap. See
// docs/tasks/task-163-priest-dispel-party/version-findings.md DIV-1.
func TestDecodeDispelGms61NoCastXY(t *testing.T) {
	const wantBitmap = byte(0b100001) // caster (bit5) + one party slot (bit0)
	const mobId = uint32(100100)      // Green Snail (arbitrary — not exercised structurally)

	buf := make([]byte, 0, 19)
	buf = binary.LittleEndian.AppendUint32(buf, 12345)                        // updateTime
	buf = binary.LittleEndian.AppendUint32(buf, uint32(skill.PriestDispelId)) // skillId (2311001)
	buf = append(buf, 20)                                                     // skill level
	buf = append(buf, wantBitmap)                                             // affectedPartyMemberBitmap
	buf = binary.LittleEndian.AppendUint16(buf, 500)                          // delay (party arm — overwritten below)
	buf = append(buf, 1)                                                      // nMobCount
	buf = binary.LittleEndian.AppendUint32(buf, mobId)                        // affectedMobIds[0]
	buf = binary.LittleEndian.AppendUint16(buf, 500)                          // delay (mob arm — the real trailing delay)

	req := request.Request(buf)
	reader := request.NewRequestReader(&req, 0)
	ctx := pt.CreateContext("GMS", 61, 1)
	m := &SkillUsageInfo{}
	m.Decode(nil, ctx)(&reader, nil)

	if m.SkillId() != uint32(skill.PriestDispelId) {
		t.Fatalf("skillId = %d, want %d", m.SkillId(), skill.PriestDispelId)
	}
	if m.AffectedPartyMemberBitmap() != wantBitmap {
		t.Fatalf("AffectedPartyMemberBitmap = %#b, want %#b — gms_61 has no castX/castY block; reading it unconditionally misaligns the bitmap", m.AffectedPartyMemberBitmap(), wantBitmap)
	}
	if len(m.AffectedMobIds()) != 1 || m.AffectedMobIds()[0] != mobId {
		t.Fatalf("AffectedMobIds = %v, want [%d]", m.AffectedMobIds(), mobId)
	}
	if m.Delay() != 500 {
		t.Fatalf("Delay = %d, want 500", m.Delay())
	}
	if reader.Available() > 0 {
		t.Fatalf("reader has %d unconsumed bytes after decode", reader.Available())
	}
}

// TestDecodeDispelGms83HasCastXY is the regression guard for the gms_61 fix
// above: gms_83 (and every version >= gms_72, IDA-verified via
// is_antirepeat_buff_skill @0x96d6ca on v83) DOES gate castX/castY, so the
// wire layout carries the extra 4 bytes and must still decode correctly.
func TestDecodeDispelGms83HasCastXY(t *testing.T) {
	const wantBitmap = byte(0b100001)
	const mobId = uint32(100100)

	buf := make([]byte, 0, 23)
	buf = binary.LittleEndian.AppendUint32(buf, 12345)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(skill.PriestDispelId))
	buf = append(buf, 20)
	buf = binary.LittleEndian.AppendUint16(buf, 300) // castX
	buf = binary.LittleEndian.AppendUint16(buf, 400) // castY
	buf = append(buf, wantBitmap)
	buf = binary.LittleEndian.AppendUint16(buf, 500)
	buf = append(buf, 1)
	buf = binary.LittleEndian.AppendUint32(buf, mobId)
	buf = binary.LittleEndian.AppendUint16(buf, 500)

	req := request.Request(buf)
	reader := request.NewRequestReader(&req, 0)
	ctx := pt.CreateContext("GMS", 83, 1)
	m := &SkillUsageInfo{}
	m.Decode(nil, ctx)(&reader, nil)

	if m.AffectedPartyMemberBitmap() != wantBitmap {
		t.Fatalf("AffectedPartyMemberBitmap = %#b, want %#b — gms_83 castX/castY gate should still be honored", m.AffectedPartyMemberBitmap(), wantBitmap)
	}
	if len(m.AffectedMobIds()) != 1 || m.AffectedMobIds()[0] != mobId {
		t.Fatalf("AffectedMobIds = %v, want [%d]", m.AffectedMobIds(), mobId)
	}
	if reader.Available() > 0 {
		t.Fatalf("reader has %d unconsumed bytes after decode", reader.Available())
	}
}

// TestDecodeBishopResurrectionReadsPartyBitmap pins the v83 wire layout of a
// Bishop Resurrection (2321006) skill-use request. IDA-verified against
// CUserLocal::SendSkillUseRequest @0x96d399 (v83): updateTime(4) skillId(4)
// slv(1) bitmap(1) delay(2). 2321006 is NOT in the client's
// is_antirepeat_buff_skill (@0x96d6ca — no castX/castY) and is not
// mob-affecting; the bitmap byte is always present because the client refuses
// to send the packet at all when no dead party member is in range
// (SendSkillUseRequest: `if skillId == 2321006 && bitmap == 0 return 1`).
func TestDecodeBishopResurrectionReadsPartyBitmap(t *testing.T) {
	buf := make([]byte, 0, 12)
	buf = binary.LittleEndian.AppendUint32(buf, 12345)                              // updateTime
	buf = binary.LittleEndian.AppendUint32(buf, uint32(skill.BishopResurrectionId)) // skillId
	buf = append(buf, 10)                                                           // skill level
	buf = append(buf, 0b010000)                                                     // bitmap: slot-1 member (bit 5-1=4)
	buf = binary.LittleEndian.AppendUint16(buf, 0)                                  // trailing delay (unread)

	req := request.Request(buf)
	reader := request.NewRequestReader(&req, 0)
	m := &SkillUsageInfo{}
	m.Decode(nil, pt.CreateContext("GMS", 83, 1))(&reader, nil)

	if m.SkillId() != uint32(skill.BishopResurrectionId) {
		t.Fatalf("skillId = %d, want %d", m.SkillId(), skill.BishopResurrectionId)
	}
	if m.AffectedPartyMemberBitmap() != 0b010000 {
		t.Fatalf("AffectedPartyMemberBitmap = %#b, want 0b010000 — 2321006 missing from isPartyBuff drops the bitmap byte", m.AffectedPartyMemberBitmap())
	}
}

// TestDecodeBuccaneerTimeLeapReadsPartyBitmap pins the v83 wire layout of a
// Buccaneer Time Leap (5121010) skill-use request. IDA-verified against the
// v83 client: CUserLocal::DoActiveSkill dispatches 5121010 (loc_969275) with
// dwTargetFlag=2 (party bit only, no mob bit), 5121010 is NOT in the client's
// is_antirepeat_buff_skill (@0x96d6ca — no castX/castY), and FindParty always
// sets the caster's own bit so the bitmap byte is always sent. Layout:
// updateTime(4) skillId(4) slv(1) bitmap(1) delay(2). Regression guard for
// task-155: when 5121010 was wrongly in isAntiRepeatBuffSkill the decoder
// consumed 4 phantom castX/castY bytes and read the bitmap from the wrong
// offset, so Time Leap resolved no party recipients and reset only the caster.
func TestDecodeBuccaneerTimeLeapReadsPartyBitmap(t *testing.T) {
	buf := make([]byte, 0, 12)
	buf = binary.LittleEndian.AppendUint32(buf, 12345)                             // updateTime
	buf = binary.LittleEndian.AppendUint32(buf, uint32(skill.BuccaneerTimeLeapId)) // skillId
	buf = append(buf, 30)                                                          // skill level
	buf = append(buf, 0b010000)                                                    // bitmap: slot-1 member (bit 5-1=4)
	buf = binary.LittleEndian.AppendUint16(buf, 0)                                 // trailing delay (unread)

	req := request.Request(buf)
	reader := request.NewRequestReader(&req, 0)
	m := &SkillUsageInfo{}
	m.Decode(nil, pt.CreateContext("GMS", 83, 1))(&reader, nil)

	if m.SkillId() != uint32(skill.BuccaneerTimeLeapId) {
		t.Fatalf("skillId = %d, want %d", m.SkillId(), skill.BuccaneerTimeLeapId)
	}
	if m.AffectedPartyMemberBitmap() != 0b010000 {
		t.Fatalf("AffectedPartyMemberBitmap = %#b, want 0b010000 — 5121010 wrongly in isAntiRepeatBuffSkill/isMobAffectingBuff misaligns the bitmap read", m.AffectedPartyMemberBitmap())
	}
}

// TestDecodeMarksmanSharpEyesReadsPartyBitmap pins the v83 wire layout of a
// Marksman Sharp Eyes (3221002) skill-use request:
// updateTime(4) skillId(4) slv(1) bitmap(1) delay(2) — 12 bytes.
//
// Client evidence: 3221002 is absent from is_antirepeat_buff_skill on every
// supported client (gms_v72 @0x877789, gms_v79 @0x8c42bd, gms_v83 @0x96d6ca,
// gms_v84 @0x9ad4e4, gms_v87 @0x9f20fc, gms_v92 @0x919150, gms_v95 @0x939dc0,
// jms_v185 @0xa3e223 — each lists 3121000/3121002/3221000 and stops), so no
// castX/castY is sent. gms_v83 CUserLocal::DoActiveSkill compares against
// 3221002 at 0x967ff7 and dispatches to loc_969275 with dwTargetFlag = 2 —
// party bit only — so DoActiveSkill_StatChange @0x969e21 passes nMobCount = -1
// and CUserLocal::SendSkillUseRequest @0x96d399 emits no mob block either.
//
// Regression guard: while 3221002 was in isAntiRepeatBuffSkill the decoder ate
// 4 phantom castX/castY bytes, the bitmap read ran off the end of the packet
// and returned 0, and SelectPartyMembersInMap resolved caster-only — Sharp
// Eyes buffed the Marksman and nobody else. Bowmaster Sharp Eyes (3121002) IS
// anti-repeat client-side, which is why only Marksmen saw the bug.
func TestDecodeMarksmanSharpEyesReadsPartyBitmap(t *testing.T) {
	buf := make([]byte, 0, 12)
	buf = binary.LittleEndian.AppendUint32(buf, 12345)                             // updateTime
	buf = binary.LittleEndian.AppendUint32(buf, uint32(skill.MarksmanSharpEyesId)) // skillId
	buf = append(buf, 30)                                                          // skill level
	buf = append(buf, 0b010001)                                                    // bitmap: caster + one member
	buf = binary.LittleEndian.AppendUint16(buf, 600)                               // trailing delay (unread)

	req := request.Request(buf)
	reader := request.NewRequestReader(&req, 0)
	m := &SkillUsageInfo{}
	m.Decode(nil, pt.CreateContext("GMS", 83, 1))(&reader, nil)

	if m.SkillId() != uint32(skill.MarksmanSharpEyesId) {
		t.Fatalf("skillId = %d, want %d", m.SkillId(), skill.MarksmanSharpEyesId)
	}
	if m.AffectedPartyMemberBitmap() != 0b010001 {
		t.Fatalf("AffectedPartyMemberBitmap = %#b, want 0b010001 — 3221002 back in isAntiRepeatBuffSkill misaligns the bitmap read", m.AffectedPartyMemberBitmap())
	}
	// The non-zero trailing delay is the mob-block canary: if 3221002 is put
	// back into isMobAffectingBuff the decoder reads 600&0xFF = 88 as a mob
	// count and manufactures 88 phantom target ids.
	if len(m.AffectedMobIds()) != 0 {
		t.Fatalf("AffectedMobIds = %d entries, want 0 — 3221002 wrongly in isMobAffectingBuff consumes the trailing delay as a mob count", len(m.AffectedMobIds()))
	}
}

// TestIsAntiRepeatExcludesMarksmanSharpEyesOnly guards the asymmetry directly:
// the client treats Bowmaster Sharp Eyes as anti-repeat and Marksman Sharp Eyes
// as not, and a well-meaning "these two should match" edit would reintroduce
// the bug.
func TestIsAntiRepeatExcludesMarksmanSharpEyesOnly(t *testing.T) {
	if !isAntiRepeatBuffSkill(skill.BowmasterSharpEyesId) {
		t.Fatalf("isAntiRepeatBuffSkill(3121002) = false, want true")
	}
	if isAntiRepeatBuffSkill(skill.MarksmanSharpEyesId) {
		t.Fatalf("isAntiRepeatBuffSkill(3221002) = true, want false — the client never lists 3221002")
	}
	if !isPartyBuff(skill.MarksmanSharpEyesId) {
		t.Fatalf("isPartyBuff(3221002) = false, want true — the bitmap byte IS on the wire")
	}
}

func TestSkillUsageInfoDecodeSpiritJavelinItemId(t *testing.T) {
	const (
		skillId = uint32(4121006) // NightLordShadowStars
		starId  = uint32(2070006) // Ilbi Throwing Stars
	)
	buf := make([]byte, 0, 13)
	buf = binary.LittleEndian.AppendUint32(buf, 12345) // updateTime
	buf = binary.LittleEndian.AppendUint32(buf, skillId)
	buf = append(buf, 30) // skillLevel
	buf = binary.LittleEndian.AppendUint32(buf, starId)

	req := request.Request(buf)
	reader := request.NewRequestReader(&req, 0)

	var info SkillUsageInfo
	info.Decode(logrus.New(), pt.CreateContext("GMS", 83, 1))(&reader, map[string]interface{}{})

	if got := info.SpiritJavelinItemId(); got != starId {
		t.Fatalf("SpiritJavelinItemId() = %d, want %d", got, starId)
	}
	if reader.Available() > 0 {
		t.Fatalf("reader has %d unconsumed bytes after decode", reader.Available())
	}
}

// TestDecodeCorsairBattleshipV92Prefix pins the v92 wire layout of a Corsair
// Battleship (5221006) skill-use request, and with it the invariant that
// makes the version-independent decoder correct on v92.
//
// v92 has no docs/packets/registry column, so the layout was read out of
// GMS_v92_1_DEVM.exe directly. SPECIAL_MOVE (opcode 0x66) has NO single
// per-version body: the trailer is chosen per skill category, and v92 alone
// carries at least three distinct shapes —
//
//	sub_91B8C0 (ctor @0x91B9B6): Encode4(time) Encode4(skillId) Encode1(SLV)
//	sub_91B630 (ctor @0x91B7C2): ... + Encode2(castX) Encode2(castY)
//	sub_91B130 (ctor @0x91B?..): ... + Encode2(x) Encode2(y) Encode1 Encode1
//
// What every v92 sender shares — and what v95's CUserLocal::DoActiveSkill_Heal
// (@0x93A830) also writes — is the 9-byte prefix asserted below. Battleship is
// in none of isAntiRepeatBuffSkill / isPartyBuff / isMobAffectingBuff, so
// Decode consumes exactly that prefix and stops.
//
// Trap for anyone re-treading this: v92's sub_91B630 was once labelled
// DoActiveSkill_Heal by propagation from v95. It is not — it contains no
// FindParty call and writes castX/castY where v95's real Heal writes
// Encode1(FindParty) + Encode2(0). Comparing those two functions is what
// produced the (incorrect) belief that v92's body diverged from v95's.
func TestDecodeCorsairBattleshipV92Prefix(t *testing.T) {
	prefix := func() []byte {
		buf := make([]byte, 0, 9)
		buf = binary.LittleEndian.AppendUint32(buf, 987654)                            // updateTime
		buf = binary.LittleEndian.AppendUint32(buf, uint32(skill.CorsairBattleshipId)) // skillId
		buf = append(buf, 7)                                                           // skill level
		return buf
	}

	tests := []struct {
		name string
		buf  []byte
	}{
		{"bare 9-byte body (v92 sub_91B8C0 shape)", prefix()},
		// A longer trailer must not corrupt the fields Decode does read: the
		// handler consumes nothing after SkillUsageInfo, so an unread tail is
		// inert. This is why the per-category trailer differences are harmless
		// for battleship on every version.
		{"with castX/castY trailer (v92 sub_91B630 shape)", append(prefix(), 0x10, 0x27, 0xF4, 0x01)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := request.Request(tc.buf)
			reader := request.NewRequestReader(&req, 0)
			m := &SkillUsageInfo{}
			m.Decode(nil, pt.CreateContext("GMS", 92, 1))(&reader, nil)

			if m.SkillId() != uint32(skill.CorsairBattleshipId) {
				t.Fatalf("skillId = %d, want %d", m.SkillId(), skill.CorsairBattleshipId)
			}
			if m.SkillLevel() != 7 {
				t.Fatalf("skillLevel = %d, want 7", m.SkillLevel())
			}
			// Battleship must not enter any conditional branch — if it were
			// ever added to one of the category lists, Decode would read past
			// the 9-byte prefix and misinterpret the trailer.
			if m.AffectedPartyMemberBitmap() != 0 {
				t.Fatalf("AffectedPartyMemberBitmap = %d, want 0 — battleship must not be treated as a party buff", m.AffectedPartyMemberBitmap())
			}
			if len(m.AffectedMobIds()) != 0 {
				t.Fatalf("AffectedMobIds = %v, want empty — battleship must not be treated as mob-affecting", m.AffectedMobIds())
			}
			if m.SpiritJavelinItemId() != 0 {
				t.Fatalf("SpiritJavelinItemId = %d, want 0", m.SpiritJavelinItemId())
			}
		})
	}
}
