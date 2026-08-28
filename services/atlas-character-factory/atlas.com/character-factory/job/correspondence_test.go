package job

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/google/uuid"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// seedTemplate is the minimal decode shape task-7-brief.md specifies -- only the fields
// this gate needs, not a full template model.
type seedTemplate struct {
	Region       string `json:"region"`
	MajorVersion uint16 `json:"majorVersion"`
	MinorVersion uint16 `json:"minorVersion"`
	Characters   struct {
		Templates []struct {
			JobIndex    uint32 `json:"jobIndex"`
			SubJobIndex uint32 `json:"subJobIndex"`
			MapId       uint32 `json:"mapId"`
			Gender      byte   `json:"gender"`
		} `json:"templates"`
	} `json:"characters"`
}

// correspondenceCase is one (version key -> seed template file) pair task-7-brief.md's
// table names.
type correspondenceCase struct {
	key    string
	region string
	major  uint16
	minor  uint16
	file   string
}

// correspondenceCases mirrors task-7-brief.md's table verbatim, with one correction: the
// brief's table lists gms_v95's minorVersion as 0, but template_gms_95_1.json (the file
// under test) declares minorVersion 1, matching every other row's minor version. The seed
// file is the grounding source, not the brief's transcription, so minor=1 is used here;
// this also keeps the "assert file metadata equals the table" check meaningful rather than
// failing on a transcription slip unrelated to carousel/template correspondence.
var correspondenceCases = []correspondenceCase{
	{"gms_12", "GMS", 12, 1, "template_gms_12_1.json"},
	{"gms_v48", "GMS", 48, 1, "template_gms_48_1.json"},
	{"gms_v61", "GMS", 61, 1, "template_gms_61_1.json"},
	{"gms_v72", "GMS", 72, 1, "template_gms_72_1.json"},
	{"gms_v79", "GMS", 79, 1, "template_gms_79_1.json"},
	{"gms_v83", "GMS", 83, 1, "template_gms_83_1.json"},
	{"gms_v84", "GMS", 84, 1, "template_gms_84_1.json"},
	{"gms_v87", "GMS", 87, 1, "template_gms_87_1.json"},
	{"gms_v92", "GMS", 92, 1, "template_gms_92_1.json"},
	{"gms_v95", "GMS", 95, 1, "template_gms_95_1.json"},
	{"gms_jms_185", "JMS", 185, 1, "template_jms_185_1.json"},
}

// direction1Exempt names version keys whose carousel is broader than its seeded rows for
// a reason findings.md establishes is not a bug -- direction (1) ("every carousel Slot has
// a template row") is skipped for these. Direction (2) still runs for every version,
// including these. Widening this map without a matching findings.md citation is a review
// failure.
var direction1Exempt = map[string]string{
	"gms_12": "findings.md gms_12: no binary, no IDB export exists for this version. Its " +
		"carousel is the pre-Big-Bang default (Explorer only, present in every candidate " +
		"mapping); the template genuinely never seeded anything beyond (1,0), because " +
		"nothing else was ever offered on this version -- not a missing row.",
}

// direction2Exempt names (version key, Slot) pairs where a seed template row is
// deliberately NOT a carousel Slot: findings.md establishes the ordinal/sub-job's class
// identity is unverified -- neither confirmed nor positively contradicted -- so FR-21
// forbids removing the seed row, while job/carousel.go's fail-closed default (no citation,
// no entry) correctly rejects the slot at runtime. Widening this map without a matching
// findings.md citation is a review failure.
var direction2Exempt = map[string]map[Slot]string{
	"gms_v92": {
		{RaceIndex: 1, SubJobIndex: 1}: "findings.md 'Dual Blade creation job id?': on " +
			"gms_v92 the sub-job field is transmitted live, but the race-select button " +
			"handler that would confirm the (1,1) offering was not reached within budget " +
			"-- neither confirmed nor denied. job/carousel.go's race4Carousel therefore " +
			"carries no (1,1) entry (fail-closed default), which is not itself a claim " +
			"that the slot is unavailable.",
	},
	"gms_jms_185": {
		{RaceIndex: 0, SubJobIndex: 0}: jmsRaceUnverifiedReason,
		{RaceIndex: 1, SubJobIndex: 0}: jmsRaceUnverifiedReason,
		{RaceIndex: 1, SubJobIndex: 1}: jmsRaceUnverifiedReason,
		{RaceIndex: 2, SubJobIndex: 0}: jmsRaceUnverifiedReason,
		{RaceIndex: 3, SubJobIndex: 0}: jmsRaceUnverifiedReason,
	},
}

const jmsRaceUnverifiedReason = "findings.md 'race4-jms' (Consequences for later tasks, " +
	"Carousels required): CLogin::Update 0x66c17f branches on the same four-ordinal shape " +
	"as race4Carousel, but every per-ordinal class identity is unverified -- only the " +
	"ordinal count is established, and findings.md warns not to reuse the race4 class " +
	"labels here without checking. job/carousel.go's race4JmsCarousel therefore carries no " +
	"entries at all (every ordinal is a genuinely unestablished mapping, not a guess); " +
	"FR-21 protects the pre-existing seeded rows absent a positive contradiction."

// TestCarouselMatchesSeedTemplates is the FR-19/FR-20 gate: for every version key this
// task-283 established a carousel for, the carousel (job/carousel.go, the single mapper)
// and the seed template (services/atlas-configurations/seed-data/templates) must agree on
// exactly which (jobIndex, subJobIndex) slots exist, for both genders.
func TestCarouselMatchesSeedTemplates(t *testing.T) {
	for _, c := range correspondenceCases {
		t.Run(c.key, func(t *testing.T) {
			ten, err := tenant.Create(uuid.New(), c.region, c.major, c.minor)
			if err != nil {
				t.Fatalf("failed to create test tenant: %v", err)
			}
			carousel := carouselFor(ten)
			tpl := loadSeedTemplate(t, c.file)

			if tpl.Region != c.region || tpl.MajorVersion != c.major || tpl.MinorVersion != c.minor {
				t.Fatalf("template metadata mismatch: got region=%s major=%d minor=%d, want region=%s major=%d minor=%d",
					tpl.Region, tpl.MajorVersion, tpl.MinorVersion, c.region, c.major, c.minor)
			}

			templateGenders := map[Slot]map[byte]bool{}
			for _, row := range tpl.Characters.Templates {
				slot := Slot{RaceIndex: row.JobIndex, SubJobIndex: row.SubJobIndex}
				if templateGenders[slot] == nil {
					templateGenders[slot] = map[byte]bool{}
				}
				templateGenders[slot][row.Gender] = true
			}

			// Direction 1: every carousel Slot has a template row for gender 0 and gender 1.
			if reason, exempt := direction1Exempt[c.key]; exempt {
				t.Logf("direction 1 (carousel -> template) skipped for %s: %s", c.key, reason)
			} else {
				for slot := range carousel {
					genders := templateGenders[slot]
					if !genders[0] || !genders[1] {
						t.Errorf("carousel slot %+v has no template row for both genders in %s (got genders %v)", slot, c.file, genders)
					}
				}
			}

			// Direction 2: every template row's (jobIndex, subJobIndex) is a carousel Slot,
			// unless the pair is named in direction2Exempt with a findings.md citation.
			exemptSlots := direction2Exempt[c.key]
			for slot := range templateGenders {
				if _, ok := carousel[slot]; ok {
					continue
				}
				if reason, ok := exemptSlots[slot]; ok {
					t.Logf("direction 2 (template -> carousel) skipped for %s slot %+v: %s", c.key, slot, reason)
					continue
				}
				t.Errorf("template row %+v in %s has no matching carousel slot for %s", slot, c.file, c.key)
			}
		})
	}
}

// gms95ExpectedStartMap is one v95 Slot's expected mapId plus the findings.md citation that
// establishes it. Every value here is transcribed from
// docs/tasks/task-283-race-index-job-mapping/findings.md -- none is read out of
// template_gms_95_1.json, the file this test checks. Sourcing an expected value from the file
// under test would make the assertion untautologizable-proof only by construction, which is
// exactly the failure this test exists to close.
type gms95ExpectedStartMap struct {
	mapId  uint32
	reason string
}

// gms95StartMaps is the PRD §10 "verified start map" half of the v95 acceptance criterion.
var gms95StartMaps = map[Slot]gms95ExpectedStartMap{
	{RaceIndex: 0, SubJobIndex: 0}: {310010000, "findings.md 'Seed rows to add (Task 7, FR-19)': " +
		"the (0,0) Resistance/Citizen row's mapId is 310010000 -- human-supplied, not IDA-derived " +
		"(task-283 execution session, 2026-08-28); the client carries no map ids on this path."},
	{RaceIndex: 1, SubJobIndex: 0}: {10000, "findings.md 'Seed rows to correct (Task 7, FR-20)': " +
		"the shift-by-one audit names the (0,0), (2,0) and (3,0) rows as wrong and the (4,0) row " +
		"as missing, and separately confirms the (1,1) row is correct as a slot; the (1,0)/(1,1) " +
		"Explorer mapId 10000 is never named among the errors, so the audit leaves it standing."},
	{RaceIndex: 1, SubJobIndex: 1}: {10000, "findings.md 'Seed rows to correct (Task 7, FR-20)': " +
		"same citation as (1,0) -- Explorer's mapId does not vary by sub-job, and the (1,1) row " +
		"is explicitly confirmed correct as a slot."},
	{RaceIndex: 2, SubJobIndex: 0}: {130010220, "findings.md 'Seed rows to correct (Task 7, FR-20)': " +
		"130010220 is the map the same file gives to index 0 on every pre-v95 column, where index " +
		"0 is Cygnus; on v95, Cygnus is index 2 (CLogin::Update 0x5dee90), so (2,0)'s mapId is " +
		"130010220."},
	{RaceIndex: 3, SubJobIndex: 0}: {140090000, "findings.md 'Seed rows to correct (Task 7, FR-20)': " +
		"140090000 is the map the same file gives to index 2 on the pre-v95 columns, where index 2 " +
		"is Aran; on v95, Aran is index 3 (CLogin::Update 0x5dee90), so (3,0)'s mapId is 140090000."},
	{RaceIndex: 4, SubJobIndex: 0}: {100030100, "findings.md 'Seed rows to add (Task 7, FR-19)': the " +
		"missing (4,0) Evan row's mapId is unknown from IDA -- the client carries no map ids; " +
		"100030100, the value mis-filed under (3,0) before the shift correction, is the reuse " +
		"findings.md names as one sanctioned source."},
}

// TestGms95SeedStartMapsMatchFindings is the PRD §10 v95 acceptance criterion's "verified start
// map" half: every seeded v95 slot's mapId must equal the value findings.md establishes for that
// race, not merely the value the seed template happens to carry.
func TestGms95SeedStartMapsMatchFindings(t *testing.T) {
	tpl := loadSeedTemplate(t, "template_gms_95_1.json")

	seen := map[Slot]bool{}
	for _, row := range tpl.Characters.Templates {
		slot := Slot{RaceIndex: row.JobIndex, SubJobIndex: row.SubJobIndex}
		expected, ok := gms95StartMaps[slot]
		if !ok {
			t.Fatalf("template_gms_95_1.json row %+v has no findings.md-sourced expected mapId "+
				"in gms95StartMaps -- add one with a citation, or explain why this slot is exempt", slot)
		}
		if row.MapId != expected.mapId {
			t.Errorf("slot %+v mapId = %d, want %d (%s)", slot, row.MapId, expected.mapId, expected.reason)
		}
		seen[slot] = true
	}

	for slot := range gms95StartMaps {
		if !seen[slot] {
			t.Errorf("expected slot %+v not present in template_gms_95_1.json", slot)
		}
	}
}

// loadSeedTemplate locates and decodes a seed template by filename, using
// runtime.Caller(0) to find the repo root the way
// tools/packet-audit/internal/template/real_test.go does.
func loadSeedTemplate(t *testing.T, filename string) seedTemplate {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "..")
	p := filepath.Join(repoRoot, "services", "atlas-configurations", "seed-data", "templates", filename)

	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("reading %s: %v", p, err)
	}

	var tpl seedTemplate
	if err := json.Unmarshal(data, &tpl); err != nil {
		t.Fatalf("decoding %s: %v", p, err)
	}
	return tpl
}
