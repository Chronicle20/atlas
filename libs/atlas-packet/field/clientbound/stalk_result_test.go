package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldStalkResult version=gms_v79 ida=0x522dc3
// packet-audit:verify packet=field/clientbound/FieldStalkResult version=gms_v83 ida=0x537a6a
// packet-audit:verify packet=field/clientbound/FieldStalkResult version=gms_v87 ida=0x55f3e5
// packet-audit:verify packet=field/clientbound/FieldStalkResult version=gms_v95 ida=0x539910
// packet-audit:verify packet=field/clientbound/FieldStalkResult version=jms_v185 ida=0x574ca3
// packet-audit:verify packet=field/clientbound/FieldStalkResult version=gms_v72 ida=0x51bce9
//
// v84 is VERSION-ABSENT: CField::OnStalkResult does not exist in the v84 IDB/export
// (the foothold/stalk cluster is version-divergent) — no marker, recorded ⬜.
func TestStalkResultGolden(t *testing.T) {
	// One stalkee, insert branch (flag=0): count=1, charId=0x11223344, flag=0,
	// name="GM", x=100, y=200.
	input := NewStalkResult(1, 0x11223344, 0, "GM", 100, 200)
	ctx := test.CreateContext("GMS", 83, 1)
	actual := test.Encode(t, ctx, input.Encode, nil)
	want := []byte{
		0x01, 0x00, 0x00, 0x00, // count=1            (Decode4 @0x537a6a)
		0x44, 0x33, 0x22, 0x11, // charId=0x11223344  (Decode4)
		0x00,                   // flag=0 (insert)    (Decode1)
		0x02, 0x00, 'G', 'M',   // name="GM"          (DecodeStr)
		0x64, 0x00, 0x00, 0x00, // x=100              (Decode4)
		0xC8, 0x00, 0x00, 0x00, // y=200              (Decode4)
	}
	if !bytes.Equal(actual, want) {
		t.Fatalf("golden mismatch:\n got %v\nwant %v", actual, want)
	}
}

// TestStalkResultByteOutputV79 pins the gms_v79 IDA_0X09C (op 0x094) clientbound
// wire. IDA: CField::OnStalkResult (was sub_522DC3) @0x522dc3 (GMS_v79_1_DEVM.exe).
// Count-prefixed loop read order: Decode4 count @0x522dd4, then per stalkee
// Decode4 charId @0x522deb, Decode1 flag @0x522ded; on the insert arm (flag==0)
// DecodeStr name @0x522dff, Decode4 x @0x522e11, Decode4 y @0x522e13. The export
// flattens one insert iteration (count + charId + flag + name + x + y), matching
// the v83/v87/v95/jms layout; v79 is byte-identical to the v83 golden.
func TestStalkResultByteOutputV79(t *testing.T) {
	input := NewStalkResult(1, 0x11223344, 0, "GM", 100, 200)
	ctx := test.CreateContext("GMS", 79, 1)
	actual := test.Encode(t, ctx, input.Encode, nil)
	want := []byte{
		0x01, 0x00, 0x00, 0x00, // count=1            (Decode4 @0x522dd4)
		0x44, 0x33, 0x22, 0x11, // charId=0x11223344  (Decode4 @0x522deb)
		0x00,                 // flag=0 (insert)    (Decode1 @0x522ded)
		0x02, 0x00, 'G', 'M', // name="GM"          (DecodeStr @0x522dff)
		0x64, 0x00, 0x00, 0x00, // x=100              (Decode4 @0x522e11)
		0xC8, 0x00, 0x00, 0x00, // y=200              (Decode4 @0x522e13)
	}
	if !bytes.Equal(actual, want) {
		t.Fatalf("v79 golden mismatch:\n got %v\nwant %v", actual, want)
	}
}

// TestStalkResultByteOutputV72 pins the gms_v72 IDA_0X09C (op 0x090) clientbound
// wire. IDA: CField::OnStalkResult = sub_51BCE9 @0x51bce9 (GMS_v72.1_U_DEVM.exe,
// dispatched via CField::OnPacket @0x515879 case 144). Count-prefixed loop read
// order: Decode4 count @0x51bcfa, then per stalkee Decode4 charId @0x51bd11,
// Decode1 flag @0x51bd13; on the insert arm (flag==0) DecodeStr name @0x51bd25,
// Decode4 x @0x51bd37, Decode4 y @0x51bd39 (remove arm calls sub_781FB8). The export
// flattens one insert iteration, matching the v79/v83/v87/v95/jms layout;
// byte-identical to the v79 golden.
func TestStalkResultByteOutputV72(t *testing.T) {
	input := NewStalkResult(1, 0x11223344, 0, "GM", 100, 200)
	ctx := test.CreateContext("GMS", 72, 1)
	actual := test.Encode(t, ctx, input.Encode, nil)
	want := []byte{
		0x01, 0x00, 0x00, 0x00, // count=1            (Decode4 @0x51bcfa)
		0x44, 0x33, 0x22, 0x11, // charId=0x11223344  (Decode4 @0x51bd11)
		0x00,                 // flag=0 (insert)    (Decode1 @0x51bd13)
		0x02, 0x00, 'G', 'M', // name="GM"          (DecodeStr @0x51bd25)
		0x64, 0x00, 0x00, 0x00, // x=100              (Decode4 @0x51bd37)
		0xC8, 0x00, 0x00, 0x00, // y=200              (Decode4 @0x51bd39)
	}
	if !bytes.Equal(actual, want) {
		t.Fatalf("v72 golden mismatch:\n got %v\nwant %v", actual, want)
	}
}

func TestStalkResultRoundTrip(t *testing.T) {
	input := NewStalkResult(1, 0x11223344, 0, "GM", 100, 200)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
