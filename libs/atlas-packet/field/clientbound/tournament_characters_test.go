package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldTournamentCharacters version=gms_v83 ida=0x57b5b4
// packet-audit:verify packet=field/clientbound/FieldTournamentCharacters version=gms_v84 ida=0x58b0c5
// packet-audit:verify packet=field/clientbound/FieldTournamentCharacters version=gms_v87 ida=0x5a9d01
// packet-audit:verify packet=field/clientbound/FieldTournamentCharacters version=gms_v95 ida=0x563780
// packet-audit:verify packet=field/clientbound/FieldTournamentCharacters version=jms_v185 ida=0x5cfd46
func TestTournamentCharactersGolden(t *testing.T) {
	input := NewTournamentCharacters()
	ctx := test.CreateContext("GMS", 83, 1)
	actual := test.Encode(t, ctx, input.Encode, nil)
	if len(actual) != 0 {
		t.Errorf("golden mismatch: got %v want empty", actual)
	}
}

func TestTournamentCharactersRoundTrip(t *testing.T) {
	input := NewTournamentCharacters()
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
