package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldTransport version=gms_v79 ida=0x537526
// packet-audit:verify packet=field/clientbound/FieldTransport version=gms_v83 ida=0x54dd08
// packet-audit:verify packet=field/clientbound/FieldTransport version=gms_v87 ida=0x577c21
// packet-audit:verify packet=field/clientbound/FieldTransport version=gms_v95 ida=0x54d5a0
// packet-audit:verify packet=field/clientbound/FieldTransport version=jms_v185 ida=0x58e280
// packet-audit:verify packet=field/clientbound/FieldTransport version=gms_v84 ida=0x55a547
func TestFieldTransport(t *testing.T) {
	input := NewFieldTransport(TransportStateMove1, true)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// TestFieldTransportByteOutputV79 pins the gms_v79 FIELD_TRANSPORT
// clientbound read. IDA: CField_ContiMove::OnContiState @0x537526
// (GMS_v79_1_DEVM.exe) reads Decode1(state) + DecodeBool(overrideAppear).
// Encode is WriteByte(state) + WriteBool(overrideAppear): state=
// TransportStateMove1 (0x02), overrideAppear=true (WriteBool -> 0x01).
func TestFieldTransportByteOutputV79(t *testing.T) {
	input := NewFieldTransport(TransportStateMove1, true)
	ctx := test.CreateContext("GMS", 79, 1)
	expected := []byte{0x02, 0x01}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v79 golden mismatch: got %v want %v", actual, expected)
	}
}
