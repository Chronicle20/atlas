package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldWitchTowerScoreUpdate version=gms_v83 ida=0x585279
// packet-audit:verify packet=field/clientbound/FieldWitchTowerScoreUpdate version=gms_v84 ida=0x594fe2
// packet-audit:verify packet=field/clientbound/FieldWitchTowerScoreUpdate version=gms_v87 ida=0x5b40ed
// packet-audit:verify packet=field/clientbound/FieldWitchTowerScoreUpdate version=gms_v95 ida=0x531020
// packet-audit:verify packet=field/clientbound/FieldWitchTowerScoreUpdate version=jms_v185 ida=0x5da14c
func TestWitchTowerScoreUpdateGolden(t *testing.T) {
	// GMS<95: score byte only.
	input := NewWitchTowerScoreUpdate(0x05, 0x11223344)
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{0x05}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch (gms_v83): got %v want %v", actual, expected)
	}
}

func TestWitchTowerScoreUpdateGoldenV95(t *testing.T) {
	// GMS v95: score byte + seconds uint32 (little-endian).
	input := NewWitchTowerScoreUpdate(0x05, 0x11223344)
	ctx := test.CreateContext("GMS", 95, 1)
	expected := []byte{0x05, 0x44, 0x33, 0x22, 0x11}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch (gms_v95): got %v want %v", actual, expected)
	}
}

func TestWitchTowerScoreUpdateRoundTrip(t *testing.T) {
	input := NewWitchTowerScoreUpdate(0x05, 0x11223344)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
