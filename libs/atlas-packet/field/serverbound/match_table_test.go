package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/serverbound/FieldMatchTable version=gms_v79 ida=0x51a2b8
// packet-audit:verify packet=field/serverbound/FieldMatchTable version=gms_v83 ida=0x52ec6c
// packet-audit:verify packet=field/serverbound/FieldMatchTable version=gms_v84 ida=0x53ad6d
// packet-audit:verify packet=field/serverbound/FieldMatchTable version=gms_v87 ida=0x555dff
// packet-audit:verify packet=field/serverbound/FieldMatchTable version=gms_v95 ida=0x5445eb
// packet-audit:verify packet=field/serverbound/FieldMatchTable version=jms_v185 ida=0x56b971
// TestMatchTableByteOutputV79 pins the gms_v79 MATCH_TABLE (op 0xCE) serverbound
// wire. IDA: CField::SendChatMsgSlash send-site @0x51a2b8 (GMS_v79_1_DEVM.exe) —
// COutPacket(0xCE) @0x51a2c0 then Encode1(bool flag) @0x51a2da (single byte body).
func TestMatchTableByteOutputV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	input := NewMatchTable(0x01)
	expected := []byte{0x01}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v79 match_table golden mismatch: got %v want %v", actual, expected)
	}
}

func TestMatchTableGolden(t *testing.T) {
	input := NewMatchTable(0x01)
	ctx := pt.CreateContext("GMS", 83, 1)
	expected := []byte{0x01}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestMatchTableRoundTrip(t *testing.T) {
	input := NewMatchTable(0x01)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MatchTable{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Flag() != input.Flag() {
				t.Errorf("round-trip mismatch: got %+v want %+v", output, input)
			}
		})
	}
}
