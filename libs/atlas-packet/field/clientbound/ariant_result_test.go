package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldAriantResult version=gms_v83 ida=0x5364c5
// packet-audit:verify packet=field/clientbound/FieldAriantResult version=gms_v84 ida=0x5427c9
// packet-audit:verify packet=field/clientbound/FieldAriantResult version=gms_v87 ida=0x55de40
// packet-audit:verify packet=field/clientbound/FieldAriantResult version=gms_v95 ida=0x538160
func TestAriantResultGolden(t *testing.T) {
	input := NewAriantResult("abc")
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{0x03, 0x00, 0x61, 0x62, 0x63}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

// TestAriantResultByteOutputV48 pins the gms_v48 ARIANT_RESULT (op 0x60 = 96)
// clientbound wire. IDA: CField::OnWarnMessage @0x4ca7d4 (GMS_v48_1_DEVM.exe) reads
// a single DecodeStr(message) @0x4ca7e6 then passes it to CUtilDlg::Notice — one
// ASCII string, byte-identical to the version-invariant golden.
// packet-audit:verify packet=field/clientbound/FieldAriantResult version=gms_v48 ida=0x4ca7d4
func TestAriantResultByteOutputV48(t *testing.T) {
	input := NewAriantResult("abc")
	ctx := test.CreateContext("GMS", 48, 1)
	expected := []byte{0x03, 0x00, 0x61, 0x62, 0x63}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v48 golden mismatch: got %v want %v", actual, expected)
	}
}

func TestAriantResultRoundTrip(t *testing.T) {
	input := NewAriantResult("abc")
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
