package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/serverbound/AutoDistributeAp version=gms_v84 ida=0xa6f1da
// packet-audit:verify packet=character/serverbound/AutoDistributeAp version=gms_v87 ida=0xabb60b
// packet-audit:verify packet=character/serverbound/AutoDistributeAp version=gms_v95 ida=0x9f63b0
// packet-audit:verify packet=character/serverbound/AutoDistributeAp version=gms_v83 ida=0xa23b3d
func TestAutoDistributeApRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := AutoDistributeAp{
				updateTime: 12345,
				nValue:     5,
				distributes: []DistributeEntry{
					{Flag: 64, Value: 3},
					{Flag: 128, Value: 2},
				},
			}
			output := AutoDistributeAp{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
			if output.NValue() != input.NValue() {
				t.Errorf("nValue: got %v, want %v", output.NValue(), input.NValue())
			}
			if len(output.Distributes()) != len(input.Distributes()) {
				t.Fatalf("distributes count: got %v, want %v", len(output.Distributes()), len(input.Distributes()))
			}
			for i, d := range output.Distributes() {
				if d.Flag != input.distributes[i].Flag {
					t.Errorf("distributes[%d].Flag: got %v, want %v", i, d.Flag, input.distributes[i].Flag)
				}
				if d.Value != input.distributes[i].Value {
					t.Errorf("distributes[%d].Value: got %v, want %v", i, d.Value, input.distributes[i].Value)
				}
			}
		})
	}
}

// TestAutoDistributeApV79ByteOutput pins the gms_v79 AUTO_DISTRIBUTE_AP (op 0x56) wire.
//
// Sender sub_96DD07 (GMS_v79_1_DEVM.exe @0x96dd07):
//
//	COutPacket::COutPacket(v23, 86)  @0x96de27 → opcode 86 (matches registry)
//	COutPacket::Encode4(v23, v9)     @0x96de39 → update_time (get_update_time @0x96dd24)
//	COutPacket::Encode4(v23, count)  @0x96de4b → pair count (*(a2-4) array length)
//	loop i: Encode4(flag[i]) @0x96de63, Encode4(value[i]) @0x96de71
//
// Body = updateTime(4) + count(4) + count×(flag(4)+value(4)). The client writes
// the actual array length as count, so nValue == len(distributes).
//
// packet-audit:verify packet=character/serverbound/AutoDistributeAp version=gms_v79 ida=0x96dd07
func TestAutoDistributeApV79ByteOutput(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	input := AutoDistributeAp{
		updateTime: 100,
		nValue:     2, // == len(distributes): v79 writes the pair count
		distributes: []DistributeEntry{
			{Flag: 0x40, Value: 1},
			{Flag: 0x80, Value: 2},
		},
	}
	expected := []byte{
		0x64, 0x00, 0x00, 0x00, // updateTime 100 (Encode4)  /*0x96de39*/
		0x02, 0x00, 0x00, 0x00, // count 2 (Encode4)         /*0x96de4b*/
		0x40, 0x00, 0x00, 0x00, // flag 0x40 (Encode4)       /*0x96de63*/
		0x01, 0x00, 0x00, 0x00, // value 1 (Encode4)         /*0x96de71*/
		0x80, 0x00, 0x00, 0x00, // flag 0x80 (Encode4)       /*0x96de63*/
		0x02, 0x00, 0x00, 0x00, // value 2 (Encode4)         /*0x96de71*/
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v79 auto-distribute-ap golden mismatch:\n got %x\nwant %x", actual, expected)
	}
}
