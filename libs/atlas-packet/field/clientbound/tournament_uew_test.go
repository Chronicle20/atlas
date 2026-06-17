package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldTournamentUew version=gms_v83 ida=0x57ba16
// packet-audit:verify packet=field/clientbound/FieldTournamentUew version=gms_v84 ida=0x58b527
// packet-audit:verify packet=field/clientbound/FieldTournamentUew version=gms_v87 ida=0x5aa13f
// packet-audit:verify packet=field/clientbound/FieldTournamentUew version=gms_v95 ida=0x563620
// packet-audit:verify packet=field/clientbound/FieldTournamentUew version=jms_v185 ida=0x5d0186
func TestTournamentUewGolden(t *testing.T) {
	input := NewTournamentUew(0x01)
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{0x01}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestTournamentUewRoundTrip(t *testing.T) {
	input := NewTournamentUew(0x01)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
