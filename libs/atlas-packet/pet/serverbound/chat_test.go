package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

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
