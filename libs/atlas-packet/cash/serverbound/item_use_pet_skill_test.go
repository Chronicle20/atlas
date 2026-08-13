package serverbound

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=cash/serverbound/CashItemUsePetSkill version=jms_v185 ida=0xaef2f5
//
// FName CWvsContext::SendConsumeCashItemUseRequest, case 28.
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
// header (Encode4 at 0xaef375, before the switch), which cashsb.ItemUse already
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

// TestItemUsePetSkillRoundTrip covers the Encode direction (the byte-fixture
// test above only exercises Decode). This codec has no version-gated
// fields, so the round trip is expected to be byte-identical across every
// tenant variant -- it exists to catch a future write-width or byte-order
// slip, mirroring how item_use_pet_consumable_test.go covers its sibling.
func TestItemUsePetSkillRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ItemUsePetSkill{petId: 42}
			output := *NewItemUsePetSkill()
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.PetId() != input.PetId() {
				t.Errorf("petId: got %v, want %v", output.PetId(), input.PetId())
			}
		})
	}
}
