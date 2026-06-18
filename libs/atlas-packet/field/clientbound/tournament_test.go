package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldTournament version=gms_v83 ida=0x57b61a
// packet-audit:verify packet=field/clientbound/FieldTournament version=gms_v84 ida=0x58b12b
// packet-audit:verify packet=field/clientbound/FieldTournament version=gms_v87 ida=0x5a9d67
// packet-audit:verify packet=field/clientbound/FieldTournament version=gms_v95 ida=0x5631a0
// packet-audit:verify packet=field/clientbound/FieldTournament version=jms_v185 ida=0x5cfdac
func TestTournamentGolden(t *testing.T) {
	input := NewTournament(0x01, 0x02, 0x03)
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{0x01, 0x02, 0x03}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestTournamentRoundTrip(t *testing.T) {
	input := NewTournament(0x01, 0x02, 0x03)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
