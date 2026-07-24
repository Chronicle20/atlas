package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// sampleTournamentMatchTableBuffer builds a deterministic, non-zero 0x300
// byte fixture: each byte is its own low-order index, so a golden mismatch
// pinpoints the offending offset immediately.
func sampleTournamentMatchTableBuffer() [TournamentMatchTableBufferSize]byte {
	var buf [TournamentMatchTableBufferSize]byte
	for i := range buf {
		buf[i] = byte(i)
	}
	return buf
}

// TournamentMatchTable read order re-derived from the live IDBs: the
// OnTournamentMatchTable handler itself only builds + DoModal's a
// CMatchTableDlg; every wire read happens in that dialog's constructor,
// identical in every version: CInPacket::DecodeBuffer(m_aaMatch, 0x300)
// then CInPacket::Decode1() into m_nState. The prior golden asserted the
// stub encoder's empty output — a false pass; corrected here across all
// versions (task-181).
// packet-audit:verify packet=field/clientbound/FieldTournamentMatchTable version=gms_v79 ida=0x55871f
// packet-audit:verify packet=field/clientbound/FieldTournamentMatchTable version=gms_v83 ida=0x57b78a
// packet-audit:verify packet=field/clientbound/FieldTournamentMatchTable version=gms_v84 ida=0x58b29b
// packet-audit:verify packet=field/clientbound/FieldTournamentMatchTable version=gms_v87 ida=0x5a9ed7
// packet-audit:verify packet=field/clientbound/FieldTournamentMatchTable version=gms_v95 ida=0x5630d0
// packet-audit:verify packet=field/clientbound/FieldTournamentMatchTable version=jms_v185 ida=0x5cff1c
func TestTournamentMatchTableGolden(t *testing.T) {
	match := sampleTournamentMatchTableBuffer()
	input := NewTournamentMatchTable(match, 0x02)
	ctx := test.CreateContext("GMS", 83, 1)
	expected := append(append([]byte{}, match[:]...), 0x02)
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %d bytes want %d bytes", len(actual), len(expected))
	}
}

// TestTournamentMatchTableByteOutputV79 pins the gms_v79 TOURNAMENT_MATCH_TABLE
// clientbound read. IDA: CField_Tournament::OnTournamentMatchTable @0x55871f
// (GMS_v79_1_DEVM.exe) delegates the actual decode to its ctor helper
// sub_750E40 @0x750e40 -- CInPacket::DecodeBuffer(0x300) then
// CInPacket::Decode1(). Byte-identical to v83/v84/v87/v95/jms.
func TestTournamentMatchTableByteOutputV79(t *testing.T) {
	match := sampleTournamentMatchTableBuffer()
	input := NewTournamentMatchTable(match, 0x02)
	ctx := test.CreateContext("GMS", 79, 1)
	expected := append(append([]byte{}, match[:]...), 0x02)
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v79 golden mismatch: got %d bytes want %d bytes", len(actual), len(expected))
	}
}

func TestTournamentMatchTableRoundTrip(t *testing.T) {
	match := sampleTournamentMatchTableBuffer()
	input := NewTournamentMatchTable(match, 0x02)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
