package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldStopClock version=gms_v83 ida=0x53184a
// packet-audit:verify packet=field/clientbound/FieldStopClock version=gms_v84 ida=0x53dad0
// packet-audit:verify packet=field/clientbound/FieldStopClock version=gms_v87 ida=0x5590cf
// packet-audit:verify packet=field/clientbound/FieldStopClock version=gms_v95 ida=0x52a7c0
// packet-audit:verify packet=field/clientbound/FieldStopClock version=jms_v185 ida=0x56ec69
func TestStopClockGolden(t *testing.T) {
	input := NewStopClock()
	ctx := test.CreateContext("GMS", 83, 1)
	actual := test.Encode(t, ctx, input.Encode, nil)
	if len(actual) != 0 {
		t.Errorf("golden mismatch: got %v want empty", actual)
	}
}

// TestStopClockByteOutputV48 pins the gms_v48 STOP_CLOCK (op 0x62 = 98) clientbound
// wire. IDA: sub_4C6AEF @0x4c6aef (GMS_v48_1_DEVM.exe) reads NOTHING — it destroys
// the clock CWnd (this[92]) and decodes no packet fields. Empty wire matches the
// version-invariant golden.
// packet-audit:verify packet=field/clientbound/FieldStopClock version=gms_v48 ida=0x4c6aef
func TestStopClockByteOutputV48(t *testing.T) {
	input := NewStopClock()
	ctx := test.CreateContext("GMS", 48, 1)
	actual := test.Encode(t, ctx, input.Encode, nil)
	if len(actual) != 0 {
		t.Errorf("v48 golden mismatch: got %v want empty", actual)
	}
}

func TestStopClockRoundTrip(t *testing.T) {
	input := NewStopClock()
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
