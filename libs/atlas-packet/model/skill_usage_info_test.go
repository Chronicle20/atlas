package model

import (
	"context"
	"encoding/binary"
	"testing"

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
	buf = binary.LittleEndian.AppendUint32(buf, 12345)                          // updateTime
	buf = binary.LittleEndian.AppendUint32(buf, uint32(skill.BishopResurrectionId)) // skillId
	buf = append(buf, 10)                                                       // skill level
	buf = append(buf, 0b010000)                                                 // bitmap: slot-1 member (bit 5-1=4)
	buf = binary.LittleEndian.AppendUint16(buf, 0)                              // trailing delay (unread)

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
