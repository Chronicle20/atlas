package character

import (
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func TestIsEvanJob(t *testing.T) {
	for _, c := range []struct {
		job  uint16
		want bool
	}{
		{2001, true},
		{2200, true},
		{2210, true},
		{2218, true},
		{2299, true},
		{0, false},
		{100, false},
		{312, false},
		{2000, false},
		{2100, false},
		{2300, false},
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
				EtcCapacity: 24, CashCapacity: 24, EquipSlotExtExpire: 94354848000000000,
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

// TestEvanSkillMasterLevelLength reproduces the task-218 field report: the
// preset Evan (job 2218, full 31-skill chain) closed the client with error 38
// on entering the world, while a level-1 Evan — same job branch, zero skills —
// logged in fine.
//
// The trailing master-level int is per-SKILL. The client asks
// is_skill_need_master_level(nSkillID), which for Evan selects only growths
// 9-10 (jobs 2217/2218) plus 22111001 / 22141002 / 22140000. The server was
// gating on job.IsFourthJob, true for the whole 2214-2218 band, so it wrote the
// int for skills the client does not read it for AND skipped it for 22111001,
// which the client does. Every mismatch shifts the remainder of
// GW_CharacterData by 4 bytes — hence "fine with no skills, error 38 with
// skills".
//
// Asserted as a byte length rather than a flag so the test fails for the same
// reason the client did.
func TestEvanSkillMasterLevelLength(t *testing.T) {
	l, _ := testlog.NewNullLogger()

	// Representative slice of the preset's chain, one per growth band.
	cd := CharacterData{Skills: []SkillEntry{
		{Id: 22001001, Level: 20}, // job 2200 — no master level
		{Id: 22111001, Level: 20}, // job 2211 — EXCEPTION, needs one
		{Id: 22131000, Level: 20}, // job 2213 — no
		{Id: 22141001, Level: 20}, // job 2214 — no (old gate wrongly wrote it)
		{Id: 22161003, Level: 15}, // job 2216 — no (Recovery Aura)
		{Id: 22171000, Level: 30}, // job 2217 — YES, growth 9
		{Id: 22181000, Level: 30}, // job 2218 — YES, growth 10
	}}

	// GMS v87: count(2) + 7 x (id 4 + level 4 + expiration 8) + cooldownCount(2)
	// = 2 + 112 + 2 = 116, plus 4 bytes for each skill that carries a master
	// level. Exactly three do: 22111001, 22171000, 22181000.
	const base = 2 + 7*(4+4+8) + 2
	w := response.NewWriter(l)
	cd.encodeSkills(w, tenant.MustFromContext(pt.CreateContext("GMS", 87, 1)))
	if got, want := len(w.Bytes()), base+3*4; got != want {
		t.Errorf("GMS v87 encodeSkills = %d bytes, want %d (exactly 3 of 7 Evan skills carry a master level)", got, want)
	}

	// JMS v185 has no exception list (@0x47d2a8), so 22111001 drops out and
	// only the two growth-9/10 skills carry it.
	wj := response.NewWriter(l)
	cd.encodeSkills(wj, tenant.MustFromContext(pt.CreateContext("JMS", 185, 1)))
	if got, want := len(wj.Bytes()), base+2*4; got != want {
		t.Errorf("JMS v185 encodeSkills = %d bytes, want %d (no Evan exception list on JMS)", got, want)
	}
}
