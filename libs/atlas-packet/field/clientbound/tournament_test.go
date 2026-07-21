package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// Tournament read order re-derived from the live IDBs and found identical
// in every version: the leading branch condition itself consumes the FIRST
// Decode1 (mode) as part of a C `||` short-circuit against a client-local
// TSecType flag (never a wire read); whichever arm is taken then reads
// exactly one further Decode1 (value) unconditionally, and neither arm
// reads anything more. The wire is therefore a flat, unconditional two
// bytes. The prior golden modelled an unconditional THIRD byte, permanently
// desyncing the client on every packet — a false pass; corrected here
// across all versions (task-181).
// packet-audit:verify packet=field/clientbound/FieldTournament version=gms_v79 ida=0x5585af
// packet-audit:verify packet=field/clientbound/FieldTournament version=gms_v83 ida=0x57b61a
// packet-audit:verify packet=field/clientbound/FieldTournament version=gms_v84 ida=0x58b12b
// packet-audit:verify packet=field/clientbound/FieldTournament version=gms_v87 ida=0x5a9d67
// packet-audit:verify packet=field/clientbound/FieldTournament version=gms_v95 ida=0x5631a0
// packet-audit:verify packet=field/clientbound/FieldTournament version=jms_v185 ida=0x5cfdac
func TestTournamentGolden(t *testing.T) {
	input := NewTournament(0x01, 0x02)
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{0x01, 0x02}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

// TestTournamentByteOutputV79 pins the gms_v79 TOURNAMENT clientbound read.
// IDA: CField_Tournament::OnTournament @0x5585af (GMS_v79_1_DEVM.exe) —
// leading Decode1 consumed inside the branch condition itself, then exactly
// one further unconditional Decode1 in whichever arm is taken. Byte-identical
// to v83/v84/v87/v95/jms.
func TestTournamentByteOutputV79(t *testing.T) {
	input := NewTournament(0x01, 0x02)
	ctx := test.CreateContext("GMS", 79, 1)
	expected := []byte{0x01, 0x02}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v79 golden mismatch: got %v want %v", actual, expected)
	}
}

func TestTournamentRoundTrip(t *testing.T) {
	input := NewTournament(0x01, 0x02)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
