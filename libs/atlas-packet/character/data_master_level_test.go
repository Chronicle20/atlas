package character

import (
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// mk builds a minimal full CharacterData for the JMS byte-length assertions
// below, mirroring the shape at data_evan_test.go:41-54 with the extra
// Exp/MapId values TestDecodeExtendedSPNonZeroCount needs to identify the
// decoded tail.
func mk(jobId uint16, skills []SkillEntry) CharacterData {
	return CharacterData{
		Stats: CharacterStats{
			Id: 1, Name: "Blade", Level: 120, JobId: jobId,
			Hp: 100, MaxHp: 100, Mp: 100, MaxMp: 100, Sp: 3,
			Exp: 50000, MapId: 100030102,
		},
		BuddyCapacity: 20,
		Inventory: InventoryData{
			EquipCapacity: 24, UseCapacity: 24, SetupCapacity: 24,
			EtcCapacity: 24, CashCapacity: 24, EquipSlotExtExpire: 94354848000000000,
		},
		Skills: skills,
	}
}

// TestSkillsDualBladeMasterLevelJMS pins job.NeedsMasterLevel's 430-434 arm
// (job.dualBladeArm) across the four shapes a Dual Blade skill list takes:
// no arm (GMS v83), always-false (GMS v87), and ClientJobLevel==4-or-named-id
// (GMS v92/v95, JMS 185 @0x47d2f9).
func TestSkillsDualBladeMasterLevelJMS(t *testing.T) {
	l, _ := testlog.NewNullLogger()

	cd := CharacterData{Skills: []SkillEntry{
		{Id: 4300000, Level: 20}, // job 430 — no master level on any version
		{Id: 4340000, Level: 30}, // job 434 — ClientJobLevel 4: master level on v92/v95/JMS
		{Id: 4311003, Level: 20}, // job 431 — named exception id, master level on v92/v95/JMS
	}}

	// count(2) + 3 x (id 4 + level 4 + expiration 8) + cooldownCount(2)
	const base = 2 + 3*(4+4+8) + 2

	for _, c := range []struct {
		name    string
		region  string
		major   uint16
		want    int
		because string
	}{
		{"JMS 185", "JMS", 185, base + 2*4, "434 and 4311003 carry it (@0x47d2f9); 430 does not"},
		{"GMS v95", "GMS", 95, base + 2*4, "same arm (@0x47ccb0); none of the three is in the ignore list"},
		{"GMS v87", "GMS", 87, base + 0, "the v87 arm returns false for all of 430-434 (@0x508fa4)"},
		{"GMS v83", "GMS", 83, base + 0, "no arm; common rule — 430 %100==0, 434 %10==4, 431 %10==1, all false"},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := response.NewWriter(l)
			cd.encodeSkills(w, tenant.MustFromContext(pt.CreateContext(c.region, c.major, 1)))
			if got := len(w.Bytes()); got != c.want {
				t.Errorf("%s encodeSkills = %d bytes, want %d (%s)", c.name, got, c.want, c.because)
			}
		})
	}
}

// TestSkillsIgnoreCommonV95 is the live GMS v95 bug of design §2.3: before
// task-275, Atlas wrote a master-level int for all fourteen reachable
// is_ignore_master_level_for_common (@0x47cc20) ids on v95, a 4-byte-per-skill
// shift of everything after the skill block. 1120012 is a member; 1120003 is
// not and takes the ordinary 4th-job rule (jobId%10==2).
func TestSkillsIgnoreCommonV95(t *testing.T) {
	l, _ := testlog.NewNullLogger()

	cd := CharacterData{Skills: []SkillEntry{
		{Id: 1120012, Level: 30}, // job 112 — in is_ignore_master_level_for_common @0x47cc20
		{Id: 1120003, Level: 30}, // job 112 — NOT in the list; ordinary 4th-job rule
	}}

	// count(2) + 2 x (id 4 + level 4 + expiration 8) + cooldownCount(2)
	const base = 2 + 2*(4+4+8) + 2

	for _, c := range []struct {
		name    string
		region  string
		major   uint16
		want    int
		because string
	}{
		{"GMS v95", "GMS", 95, base + 1*4, "1120012 is ignored; only 1120003 carries it"},
		{"GMS v92", "GMS", 92, base + 2*4, "no ignore list; both are job-112 4th-job skills"},
		{"GMS v87", "GMS", 87, base + 2*4, "same"},
		{"JMS 185", "JMS", 185, base + 2*4, "same"},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := response.NewWriter(l)
			cd.encodeSkills(w, tenant.MustFromContext(pt.CreateContext(c.region, c.major, 1)))
			if got := len(w.Bytes()); got != c.want {
				t.Errorf("%s encodeSkills = %d bytes, want %d (%s)", c.name, got, c.want, c.because)
			}
		})
	}
}

// TestCharacterDataJMSDualBladePlainSP is the FR-10 guard: Dual Blade (43x)
// takes the plain 2-byte SP short on JMS 185 (job.UsesExtendedSP is false for
// 430/1000==0), exactly like any other non-Evan job, and carries exactly one
// master-level int (job 434 is ClientJobLevel 4). A Dual Blade Ranger twin
// with the equivalent single master-level skill must encode to the same
// length; a mismatch means Dual Blade wrongly took the extended-SP path,
// which is the conflation FR-10 forbids.
func TestCharacterDataJMSDualBladePlainSP(t *testing.T) {
	ctx := pt.CreateContext("JMS", 185, 1)

	blade := mk(434, []SkillEntry{{Id: 4340000, Level: 30, MasterLevel: 30}})
	ranger := mk(312, []SkillEntry{{Id: 3121002, Level: 30, MasterLevel: 30}})

	bladeBytes := pt.Encode(t, ctx, blade.Encode, nil)
	rangerBytes := pt.Encode(t, ctx, ranger.Encode, nil)
	if len(bladeBytes) != len(rangerBytes) {
		t.Errorf("JMS 185 Dual Blade CharacterData len %d; want ranger twin len %d (both take the plain SP short and carry one master-level int)", len(bladeBytes), len(rangerBytes))
	}
}

// TestCharacterDataJMSEvanExtendedSP is the JMS half of TestEvanExtendedSPv84:
// on JMS 185, an Evan (job 2218) writes the 1-byte extended-SP count
// (sub_5163A2 @0x5163a2, called @0x50eda2) instead of the 2-byte SP short, and
// a Dual Blade on the same client still takes the plain SP short.
func TestCharacterDataJMSEvanExtendedSP(t *testing.T) {
	ctx := pt.CreateContext("JMS", 185, 1)

	evan := mk(2218, nil)
	normal := mk(312, nil)
	evanBytes := pt.Encode(t, ctx, evan.Encode, nil)
	normalBytes := pt.Encode(t, ctx, normal.Encode, nil)
	if len(evanBytes) != len(normalBytes)-1 {
		t.Errorf("JMS 185 Evan CharacterData len %d; want normal len %d - 1 (SP count byte vs SP short)", len(evanBytes), len(normalBytes))
	}

	// Round-trip: decode must read the 1-byte count, not a short.
	out := CharacterData{}
	pt.RoundTrip(t, ctx, evan.Encode, out.Decode, nil)
	if out.Stats.JobId != 2218 {
		t.Errorf("roundtrip jobId: got %d, want 2218", out.Stats.JobId)
	}

	// A Dual Blade on the same client still round-trips the plain SP short.
	blade := mk(434, nil)
	outBlade := CharacterData{}
	pt.RoundTrip(t, ctx, blade.Encode, outBlade.Decode, nil)
	if outBlade.Stats.Sp != 3 {
		t.Errorf("roundtrip Dual Blade Sp: got %d, want 3", outBlade.Stats.Sp)
	}
}

// TestDecodeExtendedSPNonZeroCount is the D5 decode-only case: Atlas's
// encoder only ever writes extended-SP count 0, so a non-zero count cannot be
// produced by a round-trip. This proves the reader consumes 1 + 2*count
// bytes for a client-authored packet by splicing a non-zero count into an
// encoded JMS 185 Evan and confirming every field after the extended-SP block
// (Exp, MapId) lands at the right offset after decode.
func TestDecodeExtendedSPNonZeroCount(t *testing.T) {
	ctx := pt.CreateContext("JMS", 185, 1)

	evan := mk(2218, nil)
	normal := mk(312, nil)
	evanBytes := pt.Encode(t, ctx, evan.Encode, nil)
	normalBytes := pt.Encode(t, ctx, normal.Encode, nil)
	if len(evanBytes) != len(normalBytes)-1 {
		t.Fatalf("JMS 185 Evan CharacterData len %d; want normal len %d - 1 (SP count byte vs SP short)", len(evanBytes), len(normalBytes))
	}

	// The extended-SP count byte sits right after Ap, at the fixed offset
	// dbcharFlag(8, JMS Int64) + SN-list-size(1) + the encodeStats fields
	// up to and including Ap: Id(4) + name(13) + gender(1) + skin(1) +
	// face(4) + hair(4) + petIds(3x8, JMS) + level(1) + jobId(2) + str(2) +
	// dex(2) + int(2) + luk(2) + hp(2) + maxHp(2) + mp(2) + maxMp(2) + ap(2)
	// (see data.go encodeStats). Both mk(2218, nil) and mk(312, nil) share
	// every field up to Ap except JobId, which is the same width in both, so
	// the offset is identical between the two encodings.
	const countOffset = 8 + 1 + (4 + 13 + 1 + 1 + 4 + 4 + 3*8 + 1 + 2 + 2 + 2 + 2 + 2 + 2 + 2 + 2 + 2 + 2)
	if evanBytes[countOffset] != 0x00 {
		t.Fatalf("evanBytes[%d] = 0x%02x, want the extended-SP count byte 0x00 — offset arithmetic is stale against encodeStats", countOffset, evanBytes[countOffset])
	}
	if normalBytes[countOffset] != 0x03 || normalBytes[countOffset+1] != 0x00 {
		t.Fatalf("normalBytes[%d:%d] = %02x %02x, want the plain SP short 03 00 — offset arithmetic is stale against encodeStats", countOffset, countOffset+1, normalBytes[countOffset], normalBytes[countOffset+1])
	}

	// count(1)=0x02 then two (masterLevelIdx, sp) byte-pairs.
	spliced := make([]byte, 0, len(evanBytes)+4)
	spliced = append(spliced, evanBytes[:countOffset]...)
	spliced = append(spliced, 0x02, 0x0A, 0x03, 0x14, 0x05)
	spliced = append(spliced, evanBytes[countOffset+1:]...)

	l, _ := testlog.NewNullLogger()
	req := request.Request(spliced)
	reader := request.NewRequestReader(&req, 0)
	out := CharacterData{}
	out.Decode(l, ctx)(&reader, nil)

	if out.Stats.Exp != 50000 {
		t.Errorf("spliced decode Exp: got %d, want 50000 (reader mis-sized the extended-SP block)", out.Stats.Exp)
	}
	if out.Stats.MapId != 100030102 {
		t.Errorf("spliced decode MapId: got %d, want 100030102 (reader mis-sized the extended-SP block)", out.Stats.MapId)
	}
}
