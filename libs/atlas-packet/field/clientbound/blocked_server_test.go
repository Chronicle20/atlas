package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldBlockedServer version=gms_v48 ida=0x4c6be7
// packet-audit:verify packet=field/clientbound/FieldBlockedServer version=gms_v83 ida=0x531a08
// packet-audit:verify packet=field/clientbound/FieldBlockedServer version=gms_v84 ida=0x53dc8e
// packet-audit:verify packet=field/clientbound/FieldBlockedServer version=gms_v87 ida=0x5592bd
// packet-audit:verify packet=field/clientbound/FieldBlockedServer version=gms_v95 ida=0x52f5f0
// packet-audit:verify packet=field/clientbound/FieldBlockedServer version=jms_v185 ida=0x56ee48
func TestBlockedServerGolden(t *testing.T) {
	input := NewBlockedServer(0x05)
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{0x05}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

// TestBlockedServerByteOutputV48 pins the gms_v48 BLOCKED_SERVER (op 0x4E=78)
// clientbound wire. IDA: CField::OnTransferChannelReqIgnored @0x4c6be7
// (GMS_v48_1_DEVM.exe) reads a single Decode1(reason) — byte-identical to the
// version-invariant golden.
func TestBlockedServerByteOutputV48(t *testing.T) {
	input := NewBlockedServer(0x05)
	ctx := test.CreateContext("GMS", 48, 1)
	expected := []byte{0x05}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v48 golden mismatch: got %v want %v", actual, expected)
	}
}

func TestBlockedServerRoundTrip(t *testing.T) {
	input := NewBlockedServer(0x05)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
