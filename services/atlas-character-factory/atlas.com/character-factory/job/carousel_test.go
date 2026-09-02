package job

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestFromIndex_PerVersionCarousel is one case per row of every `##` version section in
// findings.md. Expected values are literals transcribed by hand from findings.md -- never
// computed by calling carouselFor/FromIndex or by reading race-carousels.json -- so this
// test cannot become tautological with the implementation it checks.
func TestFromIndex_PerVersionCarousel(t *testing.T) {
	tests := []struct {
		name        string
		region      string
		major       uint16
		minor       uint16
		raceIndex   uint32
		subJobIndex uint32
		wantJobId   job.Id
		wantOk      bool
	}{
		// gms_v95 -- findings.md CLogin::Update 0x5dee90 five-arm jump table
		{"gms_v95/0.0", "GMS", 95, 0, 0, 0, job.CitizenId, true},
		{"gms_v95/1.0", "GMS", 95, 0, 1, 0, job.BeginnerId, true},
		{"gms_v95/1.1", "GMS", 95, 0, 1, 1, job.BeginnerId, true},
		{"gms_v95/2.0", "GMS", 95, 0, 2, 0, job.NoblesseId, true},
		{"gms_v95/3.0", "GMS", 95, 0, 3, 0, job.LegendId, true},
		{"gms_v95/4.0", "GMS", 95, 0, 4, 0, job.EvanId, true},

		// gms_jms_185 -- findings.md CLogin::Update 0x66c17f, four ordinals, every class
		// unverified and no job id established for any of them.
		{"gms_jms_185/0.0", "JMS", 185, 0, 0, 0, 0, false},
		{"gms_jms_185/1.0", "JMS", 185, 0, 1, 0, 0, false},
		{"gms_jms_185/2.0", "JMS", 185, 0, 2, 0, 0, false},
		{"gms_jms_185/3.0", "JMS", 185, 0, 3, 0, 0, false},
		{"gms_jms_185/1.1", "JMS", 185, 0, 1, 1, 0, false},

		// gms_v92 -- findings.md sub_5D5680 switches at 0x5d5983/0x5d5cad, four ordinals
		{"gms_v92/0.0", "GMS", 92, 0, 0, 0, job.NoblesseId, true},
		{"gms_v92/1.0", "GMS", 92, 0, 1, 0, job.BeginnerId, true},
		{"gms_v92/2.0", "GMS", 92, 0, 2, 0, job.LegendId, true},
		{"gms_v92/3.0", "GMS", 92, 0, 3, 0, job.EvanId, true},
		{"gms_v92/1.1", "GMS", 92, 0, 1, 1, 0, false},

		// gms_v87 -- findings.md CLogin::Update 0x62c5c8, four ordinals
		{"gms_v87/0.0", "GMS", 87, 0, 0, 0, job.NoblesseId, true},
		{"gms_v87/1.0", "GMS", 87, 0, 1, 0, job.BeginnerId, true},
		{"gms_v87/2.0", "GMS", 87, 0, 2, 0, job.LegendId, true},
		{"gms_v87/3.0", "GMS", 87, 0, 3, 0, job.EvanId, true},

		// gms_v84 -- findings.md CLogin__Update 0x609e9f, four ordinals
		{"gms_v84/0.0", "GMS", 84, 0, 0, 0, job.NoblesseId, true},
		{"gms_v84/1.0", "GMS", 84, 0, 1, 0, job.BeginnerId, true},
		{"gms_v84/2.0", "GMS", 84, 0, 2, 0, job.LegendId, true},
		{"gms_v84/3.0", "GMS", 84, 0, 3, 0, job.EvanId, true},

		// gms_v83 -- findings.md anchor column, CLogin::Update 0x5f4f26, three arms only
		{"gms_v83/0.0", "GMS", 83, 0, 0, 0, job.NoblesseId, true},
		{"gms_v83/1.0", "GMS", 83, 0, 1, 0, job.BeginnerId, true},
		{"gms_v83/2.0", "GMS", 83, 0, 2, 0, job.LegendId, true},
		{"gms_v83/3.0", "GMS", 83, 0, 3, 0, 0, false},

		// gms_v79 -- findings.md CLogin__Update 0x5ca641, three arms only
		{"gms_v79/0.0", "GMS", 79, 0, 0, 0, job.NoblesseId, true},
		{"gms_v79/1.0", "GMS", 79, 0, 1, 0, job.BeginnerId, true},
		{"gms_v79/2.0", "GMS", 79, 0, 2, 0, job.LegendId, true},
		{"gms_v79/3.0", "GMS", 79, 0, 3, 0, 0, false},

		// gms_v72, gms_v61, gms_v48 -- no race field at all on the wire; the client can only
		// ever produce Explorer, shown as (1,0) for cross-version consistency
		{"gms_v72/1.0", "GMS", 72, 0, 1, 0, job.BeginnerId, true},
		{"gms_v61/1.0", "GMS", 61, 0, 1, 0, job.BeginnerId, true},
		{"gms_v48/1.0", "GMS", 48, 0, 1, 0, job.BeginnerId, true},

		// gms_12 -- no binary, no IDB; (1,0) is common to every candidate mapping
		{"gms_12/1.0", "GMS", 12, 0, 1, 0, job.BeginnerId, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ten, err := tenant.Create(uuid.New(), tc.region, tc.major, tc.minor)
			if err != nil {
				t.Fatalf("Failed to create test tenant: %v", err)
			}

			gotJobId, gotOk := FromIndex(ten, tc.raceIndex, tc.subJobIndex)
			if gotOk != tc.wantOk {
				t.Fatalf("ok mismatch: expected %v, got %v", tc.wantOk, gotOk)
			}
			if tc.wantOk && gotJobId != tc.wantJobId {
				t.Errorf("jobId mismatch: expected %d, got %d", tc.wantJobId, gotJobId)
			}
		})
	}
}

// TestFromIndex_RejectsOffCarouselSlots proves FromIndex has no fallback (FR-1): every
// ordinal/sub-job pair that no findings.md row establishes must reject, not default to
// Beginner.
func TestFromIndex_RejectsOffCarouselSlots(t *testing.T) {
	type tc struct {
		name        string
		region      string
		major       uint16
		raceIndex   uint32
		subJobIndex uint32
	}

	var tests []tc

	// The required boundary cases: an ordinal beyond any carousel, an ordinal at the
	// uint32 max, a bogus sub-job on a valid ordinal, and a sub-job at the uint32 max.
	for _, v := range []struct {
		name  string
		major uint16
	}{
		{"gms_95", 95},
		{"gms_83", 83},
	} {
		tests = append(tests,
			tc{v.name + "/beyond-carousel", "GMS", v.major, 9, 0},
			tc{v.name + "/raceIndex-max", "GMS", v.major, 4294967295, 0},
			tc{v.name + "/bogus-subjob", "GMS", v.major, 1, 7},
			tc{v.name + "/subJobIndex-max", "GMS", v.major, 1, 4294967295},
		)
	}
	// gms_83 predates the Evan carousel: findings.md shows v83 has only three ordinals
	// (0..2), so ordinal 4 -- valid on gms_95 -- must reject here.
	tests = append(tests, tc{"gms_83/ordinal-4-pre-evan", "GMS", 83, 4, 0})

	// Structural sweep (FR-1): for every version key in findings.md, every raceIndex in
	// 0..(max raceIndex named in that version's table + 2) that has no row at all in that
	// table must reject. A fallback to job.BeginnerId would light up every one of these.
	sweep := []struct {
		version    string
		region     string
		major      uint16
		maxRaceIdx uint32
	}{
		{"gms_v95", "GMS", 95, 4},
		{"gms_jms_185", "JMS", 185, 3},
		{"gms_v92", "GMS", 92, 3},
		{"gms_v87", "GMS", 87, 3},
		{"gms_v84", "GMS", 84, 3},
		{"gms_v83", "GMS", 83, 3}, // findings.md tables ordinal 3 (marked "absent"); missing entirely above it
		{"gms_v79", "GMS", 79, 3}, // same shape as gms_v83
		{"gms_v72", "GMS", 72, 1}, // only (1,0) is ever tabled
		{"gms_v61", "GMS", 61, 1},
		{"gms_v48", "GMS", 48, 1},
		{"gms_12", "GMS", 12, 1},
	}
	for _, v := range sweep {
		known := map[uint32]bool{}
		switch v.version {
		case "gms_v95":
			known = map[uint32]bool{0: true, 1: true, 2: true, 3: true, 4: true}
		case "gms_jms_185", "gms_v92", "gms_v87", "gms_v84", "gms_v83", "gms_v79":
			known = map[uint32]bool{0: true, 1: true, 2: true, 3: true}
		case "gms_v72", "gms_v61", "gms_v48", "gms_12":
			known = map[uint32]bool{1: true}
		}
		for r := uint32(0); r <= v.maxRaceIdx+2; r++ {
			if known[r] {
				continue
			}
			tests = append(tests, tc{v.version + "/unlisted-raceIndex", v.region, v.major, r, 0})
		}
	}

	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			ten, err := tenant.Create(uuid.New(), c.region, c.major, 0)
			if err != nil {
				t.Fatalf("Failed to create test tenant: %v", err)
			}

			_, ok := FromIndex(ten, c.raceIndex, c.subJobIndex)
			if ok {
				t.Errorf("expected ok=false for raceIndex=%d subJobIndex=%d", c.raceIndex, c.subJobIndex)
			}
		})
	}
}

// fixtureSlot mirrors one entry of docs/packets/race-carousels.json's per-version "slots"
// array.
type fixtureSlot struct {
	RaceIndex   uint32 `json:"raceIndex"`
	SubJobIndex uint32 `json:"subJobIndex"`
	JobId       *int   `json:"jobId"`
}

type fixtureVersion struct {
	Region       string        `json:"region"`
	MajorVersion uint16        `json:"majorVersion"`
	Slots        []fixtureSlot `json:"slots"`
}

type fixture struct {
	Versions map[string]fixtureVersion `json:"versions"`
}

// TestCarouselsMatchParityFixture asserts, per version key, that carouselFor for that
// key's (region, majorVersion) returns exactly the fixture's slots -- no more, no fewer.
// A slot with jobId: null in the fixture has no established mapping (Ruling R-4) and is
// therefore excluded from the expected set.
func TestCarouselsMatchParityFixture(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "..")
	p := filepath.Join(repoRoot, "docs", "packets", "race-carousels.json")

	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var f fixture
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}

	for key, v := range f.Versions {
		t.Run(key, func(t *testing.T) {
			want := Carousel{}
			for _, s := range v.Slots {
				if s.JobId == nil {
					continue
				}
				want[Slot{RaceIndex: s.RaceIndex, SubJobIndex: s.SubJobIndex}] = job.Id(*s.JobId)
			}

			ten, err := tenant.Create(uuid.New(), v.Region, v.MajorVersion, 0)
			if err != nil {
				t.Fatalf("Failed to create test tenant: %v", err)
			}
			got := carouselFor(ten)

			for slot, jobId := range want {
				gotJobId, ok := got[slot]
				if !ok {
					t.Errorf("carousel missing slot %+v present in fixture", slot)
					continue
				}
				if gotJobId != jobId {
					t.Errorf("slot %+v: fixture jobId %d, carousel jobId %d", slot, jobId, gotJobId)
				}
			}
			for slot := range got {
				if _, ok := want[slot]; !ok {
					t.Errorf("carousel has slot %+v not present (or null) in fixture", slot)
				}
			}
		})
	}
}

// TestFromIndex_IsPerTenant is the NFR multi-tenancy proof (design §4.6): two tenants on
// different client versions get different results for the same ordinal, and the order the
// calls are made in does not matter -- there is no order-dependent package state.
func TestFromIndex_IsPerTenant(t *testing.T) {
	v95, err := tenant.Create(uuid.New(), "GMS", 95, 0)
	if err != nil {
		t.Fatalf("Failed to create v95 tenant: %v", err)
	}
	v83, err := tenant.Create(uuid.New(), "GMS", 83, 0)
	if err != nil {
		t.Fatalf("Failed to create v83 tenant: %v", err)
	}

	// raceIndex 0: Resistance/Citizen on v95, Cygnus/Noblesse on v83 -- diverging.
	v95Job, v95Ok := FromIndex(v95, 0, 0)
	v83Job, v83Ok := FromIndex(v83, 0, 0)
	if !v95Ok || !v83Ok {
		t.Fatalf("expected both tenants to resolve raceIndex 0: v95Ok=%v v83Ok=%v", v95Ok, v83Ok)
	}
	if v95Job == v83Job {
		t.Fatalf("expected diverging job ids for raceIndex 0, both resolved to %d", v95Job)
	}

	// Same calls in the opposite order must produce the same results -- no shared,
	// order-dependent state.
	v83JobAgain, v83OkAgain := FromIndex(v83, 0, 0)
	v95JobAgain, v95OkAgain := FromIndex(v95, 0, 0)
	if v83JobAgain != v83Job || v83OkAgain != v83Ok {
		t.Errorf("v83 result changed on reorder: got (%d,%v), want (%d,%v)", v83JobAgain, v83OkAgain, v83Job, v83Ok)
	}
	if v95JobAgain != v95Job || v95OkAgain != v95Ok {
		t.Errorf("v95 result changed on reorder: got (%d,%v), want (%d,%v)", v95JobAgain, v95OkAgain, v95Job, v95Ok)
	}
}
