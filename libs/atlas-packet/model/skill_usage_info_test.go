package model

import (
	"context"
	"encoding/binary"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

func TestIsMobAffectingBuff_PriestDoom(t *testing.T) {
	if !isMobAffectingBuff(skill.PriestDoomId) {
		t.Fatalf("isMobAffectingBuff(PriestDoomId) = false, want true")
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
	m.Decode(nil, context.Background())(&reader, nil)

	if m.SkillId() != uint32(skill.BishopResurrectionId) {
		t.Fatalf("skillId = %d, want %d", m.SkillId(), skill.BishopResurrectionId)
	}
	if m.AffectedPartyMemberBitmap() != 0b010000 {
		t.Fatalf("AffectedPartyMemberBitmap = %#b, want 0b010000 — 2321006 missing from isPartyBuff drops the bitmap byte", m.AffectedPartyMemberBitmap())
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
	info.Decode(logrus.New(), context.Background())(&reader, map[string]interface{}{})

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
			m.Decode(nil, context.Background())(&reader, nil)

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
