package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// AriantArenaUserScore read order re-derived from the live IDBs and found
// identical in every version: Decode1(count) followed by a count-length loop
// of {DecodeStr name, Decode4 score}. The prior golden modelled a single
// entry (count byte + ONE name/score pair) — a false pass; corrected here
// across all versions (task-181).
// packet-audit:verify packet=field/clientbound/FieldAriantArenaUserScore version=gms_v79 ida=0x528799
// packet-audit:verify packet=field/clientbound/FieldAriantArenaUserScore version=gms_v83 ida=0x53e5e1
// packet-audit:verify packet=field/clientbound/FieldAriantArenaUserScore version=gms_v84 ida=0x54abaa
// packet-audit:verify packet=field/clientbound/FieldAriantArenaUserScore version=gms_v87 ida=0x567b7d
// packet-audit:verify packet=field/clientbound/FieldAriantArenaUserScore version=gms_v95 ida=0x5492b0
// packet-audit:verify packet=field/clientbound/FieldAriantArenaUserScore version=jms_v185 ida=0x57dc4a
func TestAriantArenaUserScoreGolden(t *testing.T) {
	input := NewAriantArenaUserScore([]AriantArenaScoreEntry{
		{Name: "AB", Score: 0x00000064},
		{Name: "CD", Score: 0x00000032},
	})
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{
		0x02,                                           // count
		0x02, 0x00, 0x41, 0x42, 0x64, 0x00, 0x00, 0x00, // "AB", 100
		0x02, 0x00, 0x43, 0x44, 0x32, 0x00, 0x00, 0x00, // "CD", 50
	}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

// TestAriantArenaUserScoreByteOutputV79 pins the gms_v79
// ARIANT_ARENA_USER_SCORE clientbound read. IDA:
// CField_AriantArena::OnUserScore @0x528799 (GMS_v79_1_DEVM.exe) —
// Decode1(count) then count x {DecodeStr(name), Decode4(score)}.
// Byte-identical to v83/v84/v87/v95/jms.
func TestAriantArenaUserScoreByteOutputV79(t *testing.T) {
	input := NewAriantArenaUserScore([]AriantArenaScoreEntry{
		{Name: "AB", Score: 0x00000064},
		{Name: "CD", Score: 0x00000032},
	})
	ctx := test.CreateContext("GMS", 79, 1)
	expected := []byte{
		0x02,
		0x02, 0x00, 0x41, 0x42, 0x64, 0x00, 0x00, 0x00,
		0x02, 0x00, 0x43, 0x44, 0x32, 0x00, 0x00, 0x00,
	}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v79 golden mismatch: got %v want %v", actual, expected)
	}
}

// TestAriantArenaUserScoreGoldenEmpty confirms a zero-entry list encodes to
// just the count byte.
func TestAriantArenaUserScoreGoldenEmpty(t *testing.T) {
	input := NewAriantArenaUserScore(nil)
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{0x00}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("empty golden mismatch: got %v want %v", actual, expected)
	}
}

func TestAriantArenaUserScoreRoundTrip(t *testing.T) {
	input := NewAriantArenaUserScore([]AriantArenaScoreEntry{
		{Name: "AB", Score: 0x00000064},
		{Name: "CD", Score: 0x00000032},
	})
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
