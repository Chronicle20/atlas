package serverbound

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// packet-audit:verify CWvsContext::SendConsumeCashItemUseRequest (case 28, jms_v185 @0xaef2f5)
//
// Verified via IDA jms_v185 (session 3c4bb8b1, MapleStory_dump_SCY.exe.i64):
// the jump-table case-28 arm (entry at 0xaf16df) contains exactly ONE encode
// call in its success path -- EncodeBuffer(pet+0x18, 8) at 0xaf1a42:
//
//	0xaf1a39  push  8            ; uSize
//	0xaf1a3b  add   eax, 18h     ; eax = pet object + 0x18
//	0xaf1a3e  push  eax          ; p
//	0xaf1a3f  lea   ecx, var_18  ; this (COutPacket)
//	0xaf1a42  call  EncodeBuffer@COutPacket
//
// No trailing (or leading) updateTime is written in this arm -- update_time
// is already consumed once in the common CWvsContext::SendConsumeCashItemUseRequest
// header (Encode4 at +0x80, before the switch), which cashsb.ItemUse already
// decodes for MajorVersion()>=87 (jms_v185 included). So this sub-body is a
// bare 8-byte pet locker SN and nothing else.
func TestItemUsePetSkillDecode(t *testing.T) {
	raw := []byte{
		0x2A, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // petId = 42 (pet locker SN, u64 LE)
	}
	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)

	p := NewItemUsePetSkill()
	p.Decode(logrus.New(), context.Background())(&reader, map[string]interface{}{})

	if p.PetId() != 42 {
		t.Errorf("petId = %d, want 42", p.PetId())
	}
}
