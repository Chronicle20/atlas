package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// packet-audit:verify packet=pet/serverbound/PetChatRequest version=gms_v83 ida=0x7055e2
// packet-audit:verify packet=pet/serverbound/PetChatRequest version=gms_v87 ida=0x7492a2
// packet-audit:verify packet=pet/serverbound/PetChatRequest version=gms_v95 ida=0x6a2340
// packet-audit:verify packet=pet/serverbound/PetChatRequest version=jms_v185 ida=0x76b3a0
// packet-audit:verify packet=pet/serverbound/PetChatRequest version=gms_v84 ida=0x721d2c
func TestChatRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ChatRequest{petId: 12345, updateTime: 100, nType: 1, nAction: 2, msg: "meow"}
			output := ChatRequest{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.PetId() != input.PetId() {
				t.Errorf("petId: got %v, want %v", output.PetId(), input.PetId())
			}
			if output.NType() != input.NType() {
				t.Errorf("nType: got %v, want %v", output.NType(), input.NType())
			}
			if output.NAction() != input.NAction() {
				t.Errorf("nAction: got %v, want %v", output.NAction(), input.NAction())
			}
			if output.Msg() != input.Msg() {
				t.Errorf("msg: got %v, want %v", output.Msg(), input.Msg())
			}
		})
	}
}

// TestChatUpdateTimeGate is the byte-level oracle for the CPet::DoAction
// updateTime field. Live IDA shows v95 encodes 5 fields (updateTime present)
// vs v87's 4 (no updateTime); v84/v86 are byte-identical to v83 (task-083
// off-by-one). The packet is petId(8) [+ updateTime(4) iff GMS v95+] then
// nType. So offset 8 is the updateTime slot on v95 and nType on every other
// version. A symmetric round-trip can't catch a gate that's wrong on both
// sides — this asserts absolute bytes per version.
func TestChatUpdateTimeGate(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := ChatRequest{petId: 1, updateTime: 0x11223344, nType: 0x07, nAction: 0x09, msg: "meow"}

	// GMS v95: 4-byte little-endian updateTime sits at offset 8 (after petId).
	b95 := in.Encode(l, pt.CreateContext("GMS", 95, 1))(nil)
	if !bytes.Equal(b95[8:12], []byte{0x44, 0x33, 0x22, 0x11}) {
		t.Errorf("v95 updateTime @8 = % x, want 44 33 22 11", b95[8:12])
	}
	if b95[12] != 0x07 {
		t.Errorf("v95 byte @12 = 0x%02x, want nType 0x07", b95[12])
	}

	// Every pre-v95 GMS variant + JMS: NO updateTime → offset 8 is nType.
	for _, v := range []pt.TenantVariant{
		{Name: "GMS v83", Region: "GMS", MajorVersion: 83, MinorVersion: 1},
		{Name: "GMS v84", Region: "GMS", MajorVersion: 84, MinorVersion: 1},
		{Name: "GMS v86", Region: "GMS", MajorVersion: 86, MinorVersion: 1},
		{Name: "GMS v87", Region: "GMS", MajorVersion: 87, MinorVersion: 1},
		{Name: "JMS v185", Region: "JMS", MajorVersion: 185, MinorVersion: 1},
	} {
		b := in.Encode(l, pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion))(nil)
		if b[8] != 0x07 {
			t.Errorf("%s byte @8 = 0x%02x, want nType 0x07 (no updateTime)", v.Name, b[8])
		}
		if len(b95)-len(b) != 4 {
			t.Errorf("%s len = %d, want v95 len %d minus 4 (the updateTime int)", v.Name, len(b), len(b95))
		}
	}
}

// v79 PET_CHAT (sb op 164=0xA4) send order, verified GMS_v79_1_DEVM.exe (port
// 13340): CPet::DoAction@0x691d4e send block — COutPacket(164)@0x691f17,
// EncodeBuffer(petId,8)@0x691f2c, Encode1(nType/a2)@0x691f37,
// Encode1(nAction)@0x691f4b, EncodeStr(msg)@0x691f6b. NO updateTime (that field
// is GMS v95+ only, gated off here). Wire = petId(8)+nType(1)+nAction(1)+msg(2+len).
// TestChatBytesV72 pins the v72 wire = v79 (no updateTime, GMS<95). IDA
// GMS_v72.1_U_DEVM.exe @port 13339: CPet::DoAction@0x66ced9 send block builds
// COutPacket(162)@0x66d099, EncodeBuffer(petId,8)@0x66d0ae, Encode1(nType)@0x66d0b9,
// Encode1(nAction)@0x66d0cd, EncodeStr(msg)@0x66d0ed. No updateTime int.
// packet-audit:verify packet=pet/serverbound/PetChatRequest version=gms_v72 ida=0x66ced9
func TestChatBytesV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	in := ChatRequest{petId: 0x0102030405060708, updateTime: 0x11223344, nType: 0x07, nAction: 0x09, msg: "Hi"}
	got := in.Encode(nil, ctx)(nil)
	want := []byte{
		0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, // petId EncodeBuffer(8)@0x66d0ae (LE)
		0x07,       // nType Encode1@0x66d0b9 (NO updateTime, GMS<95)
		0x09,       // nAction Encode1@0x66d0cd
		0x02, 0x00, // msg length EncodeStr@0x66d0ed
		0x48, 0x69, // "Hi"
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v72 = % X, want % X", got, want)
	}
}

// packet-audit:verify packet=pet/serverbound/PetChatRequest version=gms_v79 ida=0x691d4e
func TestChatBytesV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	in := ChatRequest{petId: 0x0102030405060708, updateTime: 0x11223344, nType: 0x07, nAction: 0x09, msg: "Hi"}
	got := in.Encode(nil, ctx)(nil)
	want := []byte{
		0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, // petId EncodeBuffer(8)@0x691f2c (LE)
		0x07,       // nType Encode1@0x691f37 (NO updateTime, GMS<95)
		0x09,       // nAction Encode1@0x691f4b
		0x02, 0x00, // msg length EncodeStr@0x691f6b
		0x48, 0x69, // "Hi"
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v79 = % X, want % X", got, want)
	}
}
