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

// Golden bytes, v72: same 10-byte body as v48/v61/v83 (no version gate on this op).
// updateTime(4 LE) + slot(2 LE) + itemId(4 LE); 12345 = 0x3039; 2 = 0x0002;
// 2290000 = 0x22F150.
//
// IDA evidence (task-125): sub_904B55 @0x904b55 (v72 IDB
// GMS_v72.1_U_DEVM.exe.i64, session 90e36cb0 — was unnamed in this IDB;
// renamed live this pass to CWvsContext::SendSkillLearnItemUseRequest,
// cross-checked against caller CDraggableItem::OnDoubleClicked @0x4db880, a
// skill-book item double-click handler). Op was entirely ABSENT from the v72
// registry prior to this pass (matrix showed v72 ⬜ n-a) despite v48/v61/v83
// all carrying it — a stale n-a, corrected in docs/packets/registry/gms_v72.yaml.
// item-class gate a3/10000 in {228,229} (skill-book prefix, @0x904b6e/0x904b7c);
// guard sub_4DBE16(200,0) (CanSendExclRequest twin, @0x904b87);
// COutPacket::COutPacket(&pkt, 81) @0x904b95 then:
//
//	COutPacket::Encode4(&pkt, get_update_time())  -> updateTime, @0x904ba7
//	COutPacket::Encode2(&pkt, a2)                 -> slot, @0x904bb2
//	COutPacket::Encode4(&pkt, a3)                 -> itemId, @0x904bbd
//
// matches the codec's write order exactly. Opcode 0x51 == registry op 81.
//
// packet-audit:verify packet=character/serverbound/CharacterUseSkillBook version=gms_v72 ida=0x904b55
func TestUseSkillBookGoldenBytesV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	l, _ := testlog.NewNullLogger()
	got := UseSkillBook{updateTime: 12345, slot: 2, itemId: 2290000}.Encode(l, ctx)(nil)
	want := []byte{0x39, 0x30, 0x00, 0x00, 0x02, 0x00, 0x50, 0xF1, 0x22, 0x00}
	if !bytes.Equal(got, want) {
		t.Errorf("golden bytes: got % X, want % X", got, want)
	}
}

// Golden bytes, v79: same 10-byte body as v48/v61/v72/v83 (no version gate on
// this op). updateTime(4 LE) + slot(2 LE) + itemId(4 LE); 12345 = 0x3039;
// 2 = 0x0002; 2290000 = 0x22F150.
//
// IDA evidence (task-125): CWvsContext::SendSkillLearnItemUseRequest @0x955ebd
// (v79 IDB GMS_v79_1_DEVM.exe.i64, session 9a7d3642 — already named in the
// IDB; NOT the mis-ported neighbor sub_95B951). Op was entirely ABSENT from
// the v79 registry prior to this pass (matrix showed v79 ⬜ n-a) despite
// v48/v61/v72/v83 all carrying it — a stale n-a, corrected in
// docs/packets/registry/gms_v79.yaml. item-class gate a3/10000 in {228,229}
// (skill-book prefix) @0x955ed6/0x955ee4; guard CWvsContext::SetExclRequestSent
// @0x955f06; COutPacket::COutPacket(&pkt, 0x50) @0x955efd then:
//
//	COutPacket::Encode4(&pkt, v6)  -> updateTime, @0x955f0f
//	COutPacket::Encode2(&pkt, a2)  -> slot, @0x955f1a
//	COutPacket::Encode4(&pkt, a3)  -> itemId, @0x955f25
//
// matches the codec's write order exactly. Opcode 0x50 == registry op 80.
//
// packet-audit:verify packet=character/serverbound/CharacterUseSkillBook version=gms_v79 ida=0x955ebd
func TestUseSkillBookGoldenBytesV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	l, _ := testlog.NewNullLogger()
	got := UseSkillBook{updateTime: 12345, slot: 2, itemId: 2290000}.Encode(l, ctx)(nil)
	want := []byte{0x39, 0x30, 0x00, 0x00, 0x02, 0x00, 0x50, 0xF1, 0x22, 0x00}
	if !bytes.Equal(got, want) {
		t.Errorf("golden bytes: got % X, want % X", got, want)
	}
}

// Golden bytes, v84: same 10-byte body as v48/v61/v72/v79/v83 (no version gate
// on this op — v84 serverbound is byte-identical to v83).
// updateTime(4 LE) + slot(2 LE) + itemId(4 LE); 12345 = 0x3039; 2 = 0x0002;
// 2290000 = 0x22F150.
//
// IDA evidence (task-125): CWvsContext::SendSkillLearnItemUseRequest @0xa5459c
// (v84 IDB GMS_v84.1_U_DEVM.i64, session 79511a2a — already named in the IDB,
// func_query confirmed sub_A5459C == CWvsContext::SendSkillLearnItemUseRequest).
// item-class gate a3/10000 == 228 || sub_4F959A(a3) (skill/mastery-book class
// twin of the {228,229} gate); guard sub_48903A(200,0) (CanSendExclRequest
// twin); COutPacket::COutPacket(&pkt, 82) @0xa545e2 then:
//
//	COutPacket::Encode4(&pkt, sub_9C7771(v6,v5))  -> updateTime, @0xa545f4
//	COutPacket::Encode2(&pkt, a2)                 -> slot,       @0xa545ff
//	COutPacket::Encode4(&pkt, a3)                 -> itemId,     @0xa5460a
//
// matches the codec's write order exactly. Opcode 0x52 (82 decimal) ==
// registry op USE_SKILL_BOOK.
//
// packet-audit:verify packet=character/serverbound/CharacterUseSkillBook version=gms_v84 ida=0xa5459c
func TestUseSkillBookGoldenBytesV84(t *testing.T) {
	ctx := pt.CreateContext("GMS", 84, 1)
	l, _ := testlog.NewNullLogger()
	got := UseSkillBook{updateTime: 12345, slot: 2, itemId: 2290000}.Encode(l, ctx)(nil)
	want := []byte{0x39, 0x30, 0x00, 0x00, 0x02, 0x00, 0x50, 0xF1, 0x22, 0x00}
	if !bytes.Equal(got, want) {
		t.Errorf("golden bytes: got % X, want % X", got, want)
	}
}

// Golden bytes, v87: same 10-byte body as v48/v61/v72/v79/v83/v84 (no version
// gate on this op). updateTime(4 LE) + slot(2 LE) + itemId(4 LE); 12345 =
// 0x3039; 2 = 0x0002; 2290000 = 0x22F150.
//
// IDA evidence (task-125): CWvsContext::SendSkillLearnItemUseRequest @0xa9fa66
// (v87 IDB GMSv87_4GB.exe.i64, session 81f32170 — func_query name_regex
// confirms 0xa9fa66 == ?SendSkillLearnItemUseRequest@CWvsContext@@QAEXJJ@Z).
// item-class gate arg4/10000 == 228 || sub_512125(arg4) (skill/mastery-book
// class twin of the {228,229} gate); guard
// CWvsContext::CanSendExclRequest(this, 200, 0); COutPacket::COutPacket(&a3,
// 0x55) @0xa9faac then:
//
//	COutPacket::Encode4(&a3, get_update_time())  -> updateTime, @0xa9fabe
//	COutPacket::Encode2(&a3, a2)                 -> slot,       @0xa9fac9
//	COutPacket::Encode4(&a3, arg4)                -> itemId,     @0xa9fad4
//
// matches the codec's write order exactly. Opcode 0x55 (85 decimal) ==
// registry op USE_SKILL_BOOK.
//
// packet-audit:verify packet=character/serverbound/CharacterUseSkillBook version=gms_v87 ida=0xa9fa66
func TestUseSkillBookGoldenBytesV87(t *testing.T) {
	ctx := pt.CreateContext("GMS", 87, 1)
	l, _ := testlog.NewNullLogger()
	got := UseSkillBook{updateTime: 12345, slot: 2, itemId: 2290000}.Encode(l, ctx)(nil)
	want := []byte{0x39, 0x30, 0x00, 0x00, 0x02, 0x00, 0x50, 0xF1, 0x22, 0x00}
	if !bytes.Equal(got, want) {
		t.Errorf("golden bytes: got % X, want % X", got, want)
	}
}
