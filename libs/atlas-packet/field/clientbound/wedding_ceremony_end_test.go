package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldWeddingCeremonyEnd version=gms_v79 ida=0x55e5c7
// packet-audit:verify packet=field/clientbound/FieldWeddingCeremonyEnd version=gms_v83 ida=0x581b49
// packet-audit:verify packet=field/clientbound/FieldWeddingCeremonyEnd version=gms_v84 ida=0x5917f2
// packet-audit:verify packet=field/clientbound/FieldWeddingCeremonyEnd version=gms_v87 ida=0x5b0756
// packet-audit:verify packet=field/clientbound/FieldWeddingCeremonyEnd version=gms_v95 ida=0x5640a0
// packet-audit:verify packet=field/clientbound/FieldWeddingCeremonyEnd version=jms_v185 ida=0x5d6637
func TestWeddingCeremonyEndGolden(t *testing.T) {
	input := NewWeddingCeremonyEnd()
	ctx := test.CreateContext("GMS", 83, 1)
	actual := test.Encode(t, ctx, input.Encode, nil)
	if len(actual) != 0 {
		t.Errorf("golden mismatch: got %v want empty", actual)
	}
}

// TestWeddingCeremonyEndByteOutputV79 pins the gms_v79
// FIELD_WEDDING_CEREMONY_END clientbound read. IDA:
// CField_Wedding::OnCeremonyEnd @0x55e5c7 (GMS_v79_1_DEVM.exe) reads no
// fields. Body is byte-identical (empty) to the v83 golden.
func TestWeddingCeremonyEndByteOutputV79(t *testing.T) {
	input := NewWeddingCeremonyEnd()
	ctx := test.CreateContext("GMS", 79, 1)
	actual := test.Encode(t, ctx, input.Encode, nil)
	if len(actual) != 0 {
		t.Errorf("v79 golden mismatch: got %v want empty", actual)
	}
}

func TestWeddingCeremonyEndRoundTrip(t *testing.T) {
	input := NewWeddingCeremonyEnd()
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
