package character

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestIsEvanJob(t *testing.T) {
	for _, c := range []struct {
		job  uint16
		want bool
	}{
		{2001, true}, {2200, true}, {2210, true}, {2218, true}, {2299, true},
		{0, false}, {100, false}, {312, false}, {2000, false}, {2100, false}, {2300, false},
	} {
		if got := isEvanJob(c.job); got != c.want {
			t.Errorf("isEvanJob(%d) = %v, want %v", c.job, got, c.want)
		}
	}
}

// TestEvanExtendedSPv84 pins the Evan extended-SP block: on GMS v84+ an Evan job
// writes a 1-byte count (0 for a freshly-created Evan) instead of the 2-byte
// single SP short. The v84 client (GW_CharacterStat::DecodeExtendSP) reads that
// byte count, not a short — a mismatch under-runs SetField and disconnects.
func TestEvanExtendedSPv84(t *testing.T) {
	mk := func(jobId uint16) CharacterData {
		return CharacterData{
			Stats: CharacterStats{
				Id: 1, Name: "Evan", Level: 1, JobId: jobId,
				Hp: 100, MaxHp: 100, Mp: 100, MaxMp: 100, Sp: 0,
				MapId: 100030102,
			},
			BuddyCapacity: 20,
			Inventory: InventoryData{
				EquipCapacity: 24, UseCapacity: 24, SetupCapacity: 24,
				EtcCapacity: 24, CashCapacity: 24, Timestamp: 94354848000000000,
			},
		}
	}
	ctx := pt.CreateContext("GMS", 84, 1)

	evan := mk(2001)
	normal := mk(312)
	evanBytes := pt.Encode(t, ctx, evan.Encode, nil)
	normalBytes := pt.Encode(t, ctx, normal.Encode, nil)
	// Evan writes a 1-byte SP count (0); a normal job writes a 2-byte SP short, so
	// the only length difference is that one byte.
	if len(evanBytes) != len(normalBytes)-1 {
		t.Errorf("Evan CharacterData len %d; want normal len %d - 1 (SP count byte vs SP short)", len(evanBytes), len(normalBytes))
	}

	// The Evan packet must round-trip (decode reads the byte count, not a short).
	out := CharacterData{}
	pt.RoundTrip(t, ctx, evan.Encode, out.Decode, nil)
	if out.Stats.JobId != 2001 {
		t.Errorf("roundtrip jobId: got %d, want 2001", out.Stats.JobId)
	}
}
