package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldTournamentSetPrize version=gms_v83 ida=0x57b815
// packet-audit:verify packet=field/clientbound/FieldTournamentSetPrize version=gms_v84 ida=0x58b326
// packet-audit:verify packet=field/clientbound/FieldTournamentSetPrize version=gms_v87 ida=0x5a9f62
// packet-audit:verify packet=field/clientbound/FieldTournamentSetPrize version=gms_v95 ida=0x5633a0
// packet-audit:verify packet=field/clientbound/FieldTournamentSetPrize version=jms_v185 ida=0x5cffa7
func TestTournamentSetPrizeGolden(t *testing.T) {
	input := NewTournamentSetPrize(0x01, 0x02, 0x00000457, 0x00000005)
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{0x01, 0x02, 0x57, 0x04, 0x00, 0x00, 0x05, 0x00, 0x00, 0x00}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestTournamentSetPrizeRoundTrip(t *testing.T) {
	input := NewTournamentSetPrize(0x01, 0x02, 0x00000457, 0x00000005)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
