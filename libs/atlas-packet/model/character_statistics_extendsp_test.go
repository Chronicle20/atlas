package model

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// newEvanStats builds a freshly-created Evan (job 2001) whose stat block takes
// the extended-SP arm of GW_CharacterStat::Decode.
func newEvanStats(jobId uint16, hasSPTable bool) CharacterStatistics {
	return NewCharacterStatistics(
		4, "Atlas", 0, 3, 20401, 30002,
		[3]uint64{0, 0, 0},
		1, jobId,
		13, 4, 4, 4,
		50, 50, 5, 5,
		0, hasSPTable, 0,
		0, 0, 0,
		0, 0,
	)
}

// TestCharacterStatisticsExtendSPBlockIsEncoded pins the extended-SP arm of the
// GMS v87 client's GW_CharacterStat::Decode (@0x501d0e):
//
//	if ( job / 100 == 22 || job == 2001 ) {
//	    nSP = 0;                                 // no read
//	    ExtendSP::Decode(iPacket);               // Decode1 count, count x (Decode1, Decode1)
//	} else {
//	    nSP = CInPacket::Decode2(iPacket);
//	}
//
// ExtendSP::Decode (@0x5019be) unconditionally consumes the count byte, so the
// Evan arm is one byte shorter than the two-byte nSP arm — never zero bytes.
// Emitting nothing leaves the character-list packet one byte short, desyncing
// every subsequent field and closing the client with error 38 (end of file).
func TestCharacterStatisticsExtendSPBlockIsEncoded(t *testing.T) {
	ctx := pt.CreateContext("GMS", 87, 1)

	plain := pt.Encode(t, ctx, newEvanStats(100, false).Encode, nil)
	evan := pt.Encode(t, ctx, newEvanStats(2001, true).Encode, nil)

	if want := len(plain) - 1; len(evan) != want {
		t.Fatalf("extended-SP stat block length: got %d bytes, want %d (two-byte nSP replaced by a one-byte ExtendSP count)", len(evan), want)
	}
}

// TestCharacterStatisticsExtendSPRoundTrip covers every Evan job the client's
// predicate (job/100 == 22 || job == 2001) accepts.
func TestCharacterStatisticsExtendSPRoundTrip(t *testing.T) {
	ctx := pt.CreateContext("GMS", 87, 1)
	for _, jobId := range []uint16{2001, 2200, 2210, 2214, 2218} {
		input := newEvanStats(jobId, true)
		output := CharacterStatistics{}
		output.hasSPTable = true
		pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)

		if output.JobId() != jobId {
			t.Errorf("job %d: jobId round-trip got %v", jobId, output.JobId())
		}
		if output.MapId() != input.MapId() {
			t.Errorf("job %d: mapId got %v, want %v", jobId, output.MapId(), input.MapId())
		}
		if output.SpawnPoint() != input.SpawnPoint() {
			t.Errorf("job %d: spawnPoint got %v, want %v", jobId, output.SpawnPoint(), input.SpawnPoint())
		}
	}
}

// TestCharacterStatisticsExtendSPIsGMSv84Plus pins that pre-Evan clients keep
// the plain two-byte nSP read: GW_CharacterStat::Decode has no extended-SP arm
// before the Evan release, so the job predicate alone must not drive the shape.
func TestCharacterStatisticsExtendSPIsGMSv84Plus(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)

	plain := pt.Encode(t, ctx, newEvanStats(100, false).Encode, nil)
	evan := pt.Encode(t, ctx, newEvanStats(2001, true).Encode, nil)

	if len(evan) != len(plain) {
		t.Fatalf("v83 stat block length: got %d bytes, want %d (no extended-SP arm before v84)", len(evan), len(plain))
	}
}
