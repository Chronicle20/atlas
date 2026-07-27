package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldSetObjectState version=gms_v48 ida=0x4cbe02
// packet-audit:verify packet=field/clientbound/FieldSetObjectState version=gms_v83 ida=0x537a1e
// packet-audit:verify packet=field/clientbound/FieldSetObjectState version=gms_v84 ida=0x543d1c
// packet-audit:verify packet=field/clientbound/FieldSetObjectState version=gms_v87 ida=0x55f399
// packet-audit:verify packet=field/clientbound/FieldSetObjectState version=gms_v95 ida=0x539890
// packet-audit:verify packet=field/clientbound/FieldSetObjectState version=jms_v185 ida=0x574c57
func TestSetObjectStateGolden(t *testing.T) {
	input := NewSetObjectState("abc", 0x01020304)
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{0x03, 0x00, 0x61, 0x62, 0x63, 0x04, 0x03, 0x02, 0x01}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

// TestSetObjectStateByteOutputV48 pins the gms_v48 SET_OBJECT_STATE (op 0x61=97)
// clientbound wire. IDA: CField::OnSetObjectState @0x4cbe02 (GMS_v48_1_DEVM.exe)
// reads DecodeStr(name) @0x4cbe17 + Decode4(state) @0x4cbe23 — byte-identical read
// order to the version-invariant golden.
func TestSetObjectStateByteOutputV48(t *testing.T) {
	input := NewSetObjectState("abc", 0x01020304)
	ctx := test.CreateContext("GMS", 48, 1)
	expected := []byte{0x03, 0x00, 0x61, 0x62, 0x63, 0x04, 0x03, 0x02, 0x01}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v48 golden mismatch: got %v want %v", actual, expected)
	}
}

func TestSetObjectStateRoundTrip(t *testing.T) {
	input := NewSetObjectState("abc", 0x01020304)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
