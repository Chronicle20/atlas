package serverbound

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestUseSkillBookRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := UseSkillBook{updateTime: 12345, slot: 2, itemId: 2290000}
			output := UseSkillBook{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
			if output.Slot() != input.Slot() {
				t.Errorf("slot: got %v, want %v", output.Slot(), input.Slot())
			}
			if output.ItemId() != input.ItemId() {
				t.Errorf("itemId: got %v, want %v", output.ItemId(), input.ItemId())
			}
		})
	}
}

// Golden bytes, v83: updateTime(4 LE) + slot(2 LE) + itemId(4 LE).
// 12345 = 0x3039; 2 = 0x0002; 2290000 = 0x22F150.
//
// IDA evidence (task-125): CWvsContext::SendSkillLearnItemUseRequest @0xa0a1b2
// (v83 IDB MapleStory_dump.exe.i64, v83_Me build) — already named in the IDB.
// COutPacket::COutPacket(&pkt, 0x52); guard CanSendExclRequest(this,200,0);
// item-class gate a3/10000 in {228,229}; then
//
//	COutPacket::Encode4(&pkt, get_update_time())  -> updateTime
//	COutPacket::Encode2(&pkt, a2)                 -> slot
//	COutPacket::Encode4(&pkt, a3)                 -> itemId
//
// matches the codec's write order exactly. Opcode 0x52 == registry op 82.
//
// packet-audit:verify packet=character/serverbound/CharacterUseSkillBook version=gms_v83 ida=0xa0a1b2
func TestUseSkillBookGoldenBytesV83(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	l, _ := testlog.NewNullLogger()
	got := UseSkillBook{updateTime: 12345, slot: 2, itemId: 2290000}.Encode(l, ctx)(nil)
	want := []byte{0x39, 0x30, 0x00, 0x00, 0x02, 0x00, 0x50, 0xF1, 0x22, 0x00}
	if !bytes.Equal(got, want) {
		t.Errorf("golden bytes: got % X, want % X", got, want)
	}
}

// Golden bytes, v48: same 10-byte body as v83 (no version gate on this op).
// updateTime(4 LE) + slot(2 LE) + itemId(4 LE); 12345 = 0x3039; 2 = 0x0002;
// 2290000 = 0x22F150.
//
// IDA evidence (task-125): CWvsContext::SendSkillLearnItemUseRequest @0x70e3e7
// (v48 IDB GMS_v48_1_DEVM.exe.i64, session 0bb5f11a — already named in the
// IDB). item-class gate a3/10000 in {228,229} (skill-book prefix); guard
// sub_4A2518(this,200,0) (CanSendExclRequest twin); COutPacket::COutPacket(&pkt,
// 64) then:
//
//	COutPacket::Encode4(&pkt, v6)  -> updateTime
//	COutPacket::Encode2(&pkt, a2)  -> slot
//	COutPacket::Encode4(&pkt, a3)  -> itemId
//
// matches the codec's write order exactly. Opcode 0x40 == registry op 64.
//
// packet-audit:verify packet=character/serverbound/CharacterUseSkillBook version=gms_v48 ida=0x70e3e7
func TestUseSkillBookGoldenBytesV48(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	l, _ := testlog.NewNullLogger()
	got := UseSkillBook{updateTime: 12345, slot: 2, itemId: 2290000}.Encode(l, ctx)(nil)
	want := []byte{0x39, 0x30, 0x00, 0x00, 0x02, 0x00, 0x50, 0xF1, 0x22, 0x00}
	if !bytes.Equal(got, want) {
		t.Errorf("golden bytes: got % X, want % X", got, want)
	}
}

// Golden bytes, v61: same 10-byte body as v48/v83 (no version gate on this op).
// updateTime(4 LE) + slot(2 LE) + itemId(4 LE); 12345 = 0x3039; 2 = 0x0002;
// 2290000 = 0x22F150.
//
// IDA evidence (task-125): CWvsContext::SendSkillLearnItemUseRequest @0x8325d2
// (v61 IDB GMS_v61.1_U_DEVM.exe.i64, session 965202bf — already named in the
// IDB, though the registry previously mislabeled this send-site as PET_FOOD
// via a stale batch-region harvest that recorded the shared COutPacket
// ctor-site address 0x5ffc4f instead of the function's own address;
// corrected in docs/packets/registry/gms_v61.yaml, task-125). item-class gate
// a3/10000 in {228,229} (skill-book prefix); guard sub_4BDB9F(200,0)
// (CanSendExclRequest twin); COutPacket::COutPacket(&pkt, 75) then:
//
//	COutPacket::Encode4(&pkt, v6)  -> updateTime
//	COutPacket::Encode2(&pkt, a2)  -> slot
//	COutPacket::Encode4(&pkt, a3)  -> itemId
//
// matches the codec's write order exactly. Opcode 0x4B == registry op 75.
//
// packet-audit:verify packet=character/serverbound/CharacterUseSkillBook version=gms_v61 ida=0x8325d2
func TestUseSkillBookGoldenBytesV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	l, _ := testlog.NewNullLogger()
	got := UseSkillBook{updateTime: 12345, slot: 2, itemId: 2290000}.Encode(l, ctx)(nil)
	want := []byte{0x39, 0x30, 0x00, 0x00, 0x02, 0x00, 0x50, 0xF1, 0x22, 0x00}
	if !bytes.Equal(got, want) {
		t.Errorf("golden bytes: got % X, want % X", got, want)
	}
}
