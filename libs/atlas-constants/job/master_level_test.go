package job

import (
	"fmt"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

// The GMS and JMS clients spell the 43x branch differently — GMS v95
// @0x47cb90 and GMS v92 @0x479260 compute (job-430)/2, JMS 185 @0x47d347
// computes (job%10)/2 — but they agree on every value in 430..434
// (0,0,1,1,2) and diverge only at job >= 435, which no client defines.
// This ports the GMS form; the cases below pin the agreed values.
func TestClientJobLevel(t *testing.T) {
	for _, c := range []struct {
		name  string
		jobId Id
		want  int
	}{
		{"beginner 0", Id(0), 1},
		{"warrior root 100", Id(100), 1},
		{"magician root 200", Id(200), 1},
		{"Evan beginner 2001", Id(2001), 1},
		{"Evan stage 1 (2200)", Id(2200), 1},
		{"fighter 110", Id(110), 2},
		{"crusader 111", Id(111), 3},
		{"hero 112", Id(112), 4},
		{"tier-5 non-Evan 113", Id(113), 0},
		{"Aran 4th 2112", Id(2112), 4},
		{"Aran tier-5 2113", Id(2113), 0},
		{"Blade Recruit 430", Id(430), 2},
		{"Blade Acolyte 431", Id(431), 2},
		{"Blade Specialist 432", Id(432), 3},
		{"Blade Lord 433", Id(433), 3},
		{"Blade Master 434", Id(434), 4},
		{"Evan stage 2 (2210)", Id(2210), 2},
		{"Evan stage 3 (2211)", Id(2211), 3},
		{"Evan stage 5 (2213)", Id(2213), 5},
		{"Evan stage 8 (2216)", Id(2216), 8},
		{"Evan stage 9 (2217)", Id(2217), 9},
		{"Evan stage 10 (2218)", Id(2218), 10},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := ClientJobLevel(c.jobId); got != c.want {
				t.Errorf("ClientJobLevel(%d) = %d, want %d", c.jobId, got, c.want)
			}
		})
	}
}

// masterLevelVersions is the six version columns every NeedsMasterLevel
// table below iterates: {"GMS",83}, {"GMS",84}, {"GMS",87}, {"GMS",92},
// {"GMS",95}, {"JMS",185}.
var masterLevelVersions = []struct {
	region string
	major  uint16
}{
	{"GMS", 83},
	{"GMS", 84},
	{"GMS", 87},
	{"GMS", 92},
	{"GMS", 95},
	{"JMS", 185},
}

// v83 (@0x4e8f04) and v84 (@0x4f0ad2) have no 43x arm at all, so these fall
// through to the common rule job%100 != 0 && job%10 == 2 — which is TRUE for
// job 432 and false for 430/431/433/434. That is the client's observable
// answer, not a modelling choice. v87 (@0x508fa4) returns false for all five.
// v92 (@0x479371), v95 (@0x47ccb0) and JMS 185 (@0x47d2f9) return
// ClientJobLevel(job) == 4 || skillId in {4311003,4321000,4331002,4331005}.
func TestNeedsMasterLevel_DualBladePerVersion(t *testing.T) {
	for _, c := range []struct {
		name string
		id   skill.Id
		want [6]bool
	}{
		{"job 430 generic", 4300000, [6]bool{false, false, false, false, false, false}},
		{"job 431 generic", 4310000, [6]bool{false, false, false, false, false, false}},
		{"job 432 generic", 4320000, [6]bool{true, true, false, false, false, false}},
		{"job 433 generic", 4330000, [6]bool{false, false, false, false, false, false}},
		{"job 434 generic", 4340000, [6]bool{false, false, false, true, true, true}},
		{"4311003", 4311003, [6]bool{false, false, false, true, true, true}},
		{"4321000", 4321000, [6]bool{true, true, false, true, true, true}},
		{"4331002", 4331002, [6]bool{false, false, false, true, true, true}},
		{"4331005", 4331005, [6]bool{false, false, false, true, true, true}},
	} {
		for i, v := range masterLevelVersions {
			want := c.want[i]
			t.Run(fmt.Sprintf("%s_v%d/%s", v.region, v.major, c.name), func(t *testing.T) {
				if got := NeedsMasterLevel(c.id, v.region, v.major); got != want {
					t.Errorf("NeedsMasterLevel(%d, %q, %d) = %v, want %v", c.id, v.region, v.major, got, want)
				}
			})
		}
	}
}

// The exception list is present GMS v84+ (@0x4f0ad2, @0x508f33, @0x4792f0,
// @0x47ccb0) and absent on GMS v83 (@0x4e8f04) and JMS 185 (@0x47d2a8); and
// the 22161000/22171000 pair is the job.GetSkillBook off-by-one guard from
// skill/model_test.go:103-113 (GetSkillBook indexes 2210->1 … 2218->9, the
// client's level is 2210->2 … 2218->10, so a book-indexed "9 or 10" would
// wrongly select 2218/2219).
func TestNeedsMasterLevel_EvanArm(t *testing.T) {
	for _, c := range []struct {
		name string
		id   skill.Id
		want [6]bool
	}{
		{"22171000 (9th growth)", 22171000, [6]bool{true, true, true, true, true, true}},
		{"22181000 (10th growth)", 22181000, [6]bool{true, true, true, true, true, true}},
		{"22001001 (1st growth)", 22001001, [6]bool{false, false, false, false, false, false}},
		{"22131000 (5th growth)", 22131000, [6]bool{false, false, false, false, false, false}},
		{"22141001 (6th growth)", 22141001, [6]bool{false, false, false, false, false, false}},
		{"22151001 (7th growth)", 22151001, [6]bool{false, false, false, false, false, false}},
		{"22161003 (Recovery Aura)", 22161003, [6]bool{false, false, false, false, false, false}},
		{"22161000 (8th growth, skill-book guard)", 22161000, [6]bool{false, false, false, false, false, false}},
		{"20010000 (Evan beginner 2001)", 20010000, [6]bool{false, false, false, false, false, false}},
		{"22111001 (exception)", 22111001, [6]bool{false, true, true, true, true, false}},
		{"22141002 (exception)", 22141002, [6]bool{false, true, true, true, true, false}},
		{"22140000 (exception)", 22140000, [6]bool{false, true, true, true, true, false}},
	} {
		for i, v := range masterLevelVersions {
			want := c.want[i]
			t.Run(fmt.Sprintf("%s_v%d/%s", v.region, v.major, c.name), func(t *testing.T) {
				if got := NeedsMasterLevel(c.id, v.region, v.major); got != want {
					t.Errorf("NeedsMasterLevel(%d, %q, %d) = %v, want %v", c.id, v.region, v.major, got, want)
				}
			})
		}
	}
}

// TestNeedsMasterLevel_CommonBranch is the version-invariant fallthrough:
// the same answer on all six version columns.
func TestNeedsMasterLevel_CommonBranch(t *testing.T) {
	for _, c := range []struct {
		name string
		id   skill.Id
		want bool
	}{
		{"1120003", 1120003, true},
		{"2321000", 2321000, true},
		{"4120002", 4120002, true},
		{"21120001 (Aran 4th goes through the common branch, not the Evan one)", 21120001, true},
		{"1100000", 1100000, false},
		{"1110000", 1110000, false},
		{"21110000", 21110000, false},
		{"10000", 10000, false},
		{"1000000", 1000000, false},
		{"2000000", 2000000, false},
	} {
		for _, v := range masterLevelVersions {
			want := c.want
			t.Run(fmt.Sprintf("%s_v%d/%s", v.region, v.major, c.name), func(t *testing.T) {
				if got := NeedsMasterLevel(c.id, v.region, v.major); got != want {
					t.Errorf("NeedsMasterLevel(%d, %q, %d) = %v, want %v", c.id, v.region, v.major, got, want)
				}
			})
		}
	}
}

// Fourteen of these sixteen belong to jobs an Atlas tenant can create
// (112, 122, 132, 212, 222, 232, 312, 322, 412, 422, 512, 522), so before
// task-275 Atlas wrote a master-level int for them on GMS v95 where the
// client reads none — a live 4-byte-per-skill shift. The remaining two
// (jobs 3212, 3312) match no Atlas identity and are modelled only because
// the client's list is a flat id set and a partial port is not a port.
func TestNeedsMasterLevel_IgnoreCommonV95Only(t *testing.T) {
	ids := []skill.Id{
		1120012, 1220013, 1320011, 2120009, 2220009, 2320010, 3120010, 3120011,
		3220009, 3220010, 4120010, 4220009, 5120011, 5220012, 32120009, 33120010,
	}
	for _, id := range ids {
		for _, v := range masterLevelVersions {
			want := true
			if v.region == "GMS" && v.major == 95 {
				want = false
			}
			t.Run(fmt.Sprintf("%s_v%d/%d", v.region, v.major, id), func(t *testing.T) {
				if got := NeedsMasterLevel(id, v.region, v.major); got != want {
					t.Errorf("NeedsMasterLevel(%d, %q, %d) = %v, want %v", id, v.region, v.major, got, want)
				}
			})
		}
	}
}

// NeedsMasterLevel normalises region case-insensitively, matching
// constants.For (constants/for.go:40).
func TestNeedsMasterLevel_LowerCaseRegion(t *testing.T) {
	if got := NeedsMasterLevel(skill.Id(4340000), "jms", 185); got != true {
		t.Errorf(`NeedsMasterLevel(4340000, "jms", 185) = %v, want true`, got)
	}
}
