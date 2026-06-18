package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldTournamentMatchTable version=gms_v83 ida=0x57b78a
// packet-audit:verify packet=field/clientbound/FieldTournamentMatchTable version=gms_v84 ida=0x58b29b
// packet-audit:verify packet=field/clientbound/FieldTournamentMatchTable version=gms_v87 ida=0x5a9ed7
// packet-audit:verify packet=field/clientbound/FieldTournamentMatchTable version=gms_v95 ida=0x5630d0
// packet-audit:verify packet=field/clientbound/FieldTournamentMatchTable version=jms_v185 ida=0x5cff1c
func TestTournamentMatchTableGolden(t *testing.T) {
	input := NewTournamentMatchTable()
	ctx := test.CreateContext("GMS", 83, 1)
	actual := test.Encode(t, ctx, input.Encode, nil)
	if len(actual) != 0 {
		t.Errorf("golden mismatch: got %v want empty", actual)
	}
}

func TestTournamentMatchTableRoundTrip(t *testing.T) {
	input := NewTournamentMatchTable()
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
