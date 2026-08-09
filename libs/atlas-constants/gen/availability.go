// availability.go implements the atlas-constants per-version availability
// generator (task-187 Task 5): the second, orthogonal axis layered on top of
// Task 4's semantics. Semantics answers "which wire id means this Identity
// in version V" (presence); availability answers "was this Identity
// actually released/playable in version V" -- a SUBSET of presence (an
// identity can be present in the WZ data as an unreleased stub, e.g. the
// v61 Pirate job, well before its class actually shipped).
//
// Pipeline:
//
//	availability.csv (Task 1 audit ledger: release CLASS label -> released,
//	  per version -- identityName in this CSV is a class label like "Pirate"
//	  or "Aran", not an Identity constant)
//	  --[loadReleaseMatrix]--> map[versionKey]map[classLabel]released
//	  --[classOf]--> identity's canonical token -> classLabel
//	    (version-independent: the SAME token always maps to the SAME class)
//	  --[computeAvailable]--> present (Task 4) ∩ released (this file)
//	    == this version's available identity set
//	  --[EmitSemantics]--> available_<key> + names_<key> maps baked into
//	    each generated newSet_<key>()
package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
)

// availabilityCSVPath is the Task 1 audit ledger this generator reads.
const availabilityCSVPath = "../../../docs/tasks/task-187-version-aware-id-semantics/audit/availability.csv"

// classOf returns the release-class label for domain's identity canonical
// token, or "" if the identity has no gating class ("Explorer/stable" in
// the task-187 Task 5 brief -- always available at every provisioned
// version). It is VERSION-INDEPENDENT: the same (domain, canonicalToken)
// pair always maps to the same class regardless of which version is asking.
//
// Ranges verified against every domain:job and domain:skill entry in
// gen/identities.yaml (2026-07-30, task-187 Task 5): the 82 job identities'
// canonicalToken values fall exactly into these boundaries with no
// stragglers, and every one of the 567 skill identities' canonicalToken,
// floor-divided by 10000, reproduces its OWNING job's canonicalToken
// exactly (e.g. PirateBulletTime=5000000 -> 500 == job Pirate's token;
// NoblesseThreeSnails=10001000 -> 1000 == job Noblesse's token;
// EvanStage10SoulStone=22181003 -> 2218 == job EvanStage10's token) --
// see task-5-report.md for the full per-class dump. This floor-by-10000
// relationship is what lets one range table serve both domains: a skill's
// class is always its owning job's class.
//
// Mechanic and Resistance (two of availability.csv's nine release-tracked
// classes) have NO identity in the namespace at all (no job/skill
// canonicalToken maps to them) -- classOf never returns those labels; their
// availability.csv rows are inert for this generator (no identity's
// Available() is ever gated by them). DualBlade was a third such class until
// task-204 added the 430-434 job identities and their 43xxxxx skills, at
// which point its rows stopped being inert; see the DualBlade arm below.
func classOf(domain string, canonicalToken uint64) string {
	var t uint64
	switch domain {
	case "job":
		t = canonicalToken
	case "skill":
		t = canonicalToken / 10000
	default:
		return ""
	}
	switch {
	case t == 900:
		return "GM"
	case t == 910:
		return "SuperGM"
	case t >= 500 && t <= 599:
		return "Pirate"
	// DualBlade (task-204): the third Rogue branch, GMS v0.88. Jobs 430-434
	// sit INSIDE the Explorer thief range, so this arm MUST stay ahead of
	// any broader 4xx handling -- there is none today (Explorer jobs are
	// classless/"stable"), and adding one later without keeping this arm
	// first would silently un-gate Dual Blade at gms 12-87.
	//
	// Present-but-unreleased at gms 87: the WZ carries jobs 430-434 there
	// but only 17 of the 26 skill images the released versions ship, and
	// 4300000 has no WZ name (divergences.csv gms,87,1,job,430). Presence
	// is not release -- availability.csv holds that line, exactly as it does
	// for CygnusStage4 below.
	case t >= 430 && t <= 434:
		return "DualBlade"
	// CygnusStage4 (task-202 FR-2.1): the five Cygnus 4th-job branches are
	// PRESENT in every supported version's Skill.wz but their `skill` node is
	// empty -- the tier was never released in the version range we support
	// (docs/tasks/task-202-version-correct-job-hierarchy/investigation.md
	// Finding 3, corroborated by a live GET /api/data/jobs/{id}/skills sweep
	// across gms 79/83/84/87/92/95 and jms 185). Presence != release, so the
	// identities stay in identities.yaml and Set.Resolve/Set.Wire keep
	// answering for them; only Set.Available flips.
	//
	// Deliberately an explicit token list rather than arithmetic: the list is
	// greppable and a sixth Cygnus branch must be added on purpose. This arm
	// MUST stay above the 1000..1599 range arm below.
	case t == 1112, t == 1212, t == 1312, t == 1412, t == 1512:
		return "CygnusStage4"
	case t >= 1000 && t <= 1599:
		return "Cygnus"
	case t == 2000 || (t >= 2100 && t <= 2199):
		return "Aran"
	case t == 2001 || (t >= 2200 && t <= 2299):
		return "Evan"
	default:
		return ""
	}
}

// releaseRow is one row of availability.csv
// (region,major,minor,domain,identityName,released,meymink). identityName
// here is a release CLASS LABEL (GM, SuperGM, Pirate, Cygnus, Aran, Evan,
// DualBlade, Resistance, Mechanic), not an Identity constant name -- see
// docs/tasks/task-187-version-aware-id-semantics/audit/README.md.
type releaseRow struct {
	Region     string
	Major      uint16
	Minor      uint16
	Domain     string
	ClassLabel string
	Released   bool
	Meymink    string
}

// loadReleaseRows parses availability.csv.
func loadReleaseRows(path string) ([]releaseRow, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	r := csv.NewReader(f)
	r.FieldsPerRecord = 7
	all, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if len(all) == 0 {
		return nil, fmt.Errorf("%s: empty file", path)
	}
	header := all[0]
	col := func(name string) int {
		for i, h := range header {
			if h == name {
				return i
			}
		}
		return -1
	}
	regionI, majorI, minorI, domainI := col("region"), col("major"), col("minor"), col("domain")
	identityI, releasedI, meyminkI := col("identityName"), col("released"), col("meymink")
	if regionI < 0 || majorI < 0 || minorI < 0 || domainI < 0 || identityI < 0 || releasedI < 0 || meyminkI < 0 {
		return nil, fmt.Errorf("%s: header missing required column(s): %v", path, header)
	}

	var out []releaseRow
	for i, rec := range all[1:] {
		major, err := strconv.ParseUint(rec[majorI], 10, 16)
		if err != nil {
			return nil, fmt.Errorf("%s row %d: parsing major %q: %w", path, i+2, rec[majorI], err)
		}
		minor, err := strconv.ParseUint(rec[minorI], 10, 16)
		if err != nil {
			return nil, fmt.Errorf("%s row %d: parsing minor %q: %w", path, i+2, rec[minorI], err)
		}
		var released bool
		switch rec[releasedI] {
		case "true":
			released = true
		case "false":
			released = false
		default:
			return nil, fmt.Errorf("%s row %d: released must be \"true\" or \"false\", got %q", path, i+2, rec[releasedI])
		}
		out = append(out, releaseRow{
			Region:     rec[regionI],
			Major:      uint16(major),
			Minor:      uint16(minor),
			Domain:     rec[domainI],
			ClassLabel: rec[identityI],
			Released:   released,
			Meymink:    rec[meyminkI],
		})
	}
	return out, nil
}

// buildReleaseMatrices groups releaseRows by version into a
// classLabel->released lookup.
func buildReleaseMatrices(rows []releaseRow) map[versionKey]map[string]bool {
	out := make(map[versionKey]map[string]bool)
	for _, row := range rows {
		v := versionKey{Region: row.Region, Major: row.Major, Minor: row.Minor}
		if out[v] == nil {
			out[v] = make(map[string]bool)
		}
		out[v][row.ClassLabel] = row.Released
	}
	return out
}

// loadReleaseMatrix returns one version's classLabel->released matrix; error
// if availability.csv has no rows at all for that version (every
// provisioned version must have a full 9-class row set -- see the
// task-187 Task 5 gms_12 reconciliation in
// docs/tasks/task-187-version-aware-id-semantics/audit/README.md).
func loadReleaseMatrix(region string, major, minor uint16) (map[string]bool, error) {
	rows, err := loadReleaseRows(availabilityCSVPath)
	if err != nil {
		return nil, err
	}
	matrices := buildReleaseMatrices(rows)
	v := versionKey{Region: region, Major: major, Minor: minor}
	m, ok := matrices[v]
	if !ok {
		return nil, fmt.Errorf("availability: no release rows for %s %d.%d in %s -- run the task-5 reconciliation to add them", region, major, minor, availabilityCSVPath)
	}
	return m, nil
}

// AvailabilityMap is one version's release-class view straight from
// availability.csv: classLabel -> released. Job and Skill point at the
// SAME underlying matrix -- availability.csv's class rows are domain-inert
// (a release class like Pirate gates both the job AND skill identities of
// that class simultaneously); kept as two fields purely so callers can
// index it the same way they index SemanticsMap.
type AvailabilityMap struct {
	Job   map[string]bool
	Skill map[string]bool
}

// BuildAvailability returns one version's release-class view (task-187
// Task 5 Step 2/4 test surface): matrix[classLabel] = released.
func BuildAvailability(region string, major, minor uint16) (AvailabilityMap, error) {
	m, err := loadReleaseMatrix(region, major, minor)
	if err != nil {
		return AvailabilityMap{}, err
	}
	return AvailabilityMap{Job: m, Skill: m}, nil
}

// releaseEligible reports whether an identity with the given canonicalToken
// in domain is release-eligible at this version per matrix: true if it has
// no gating class (stable/Explorer, always available) or its class is
// released=true in matrix.
func releaseEligible(matrix map[string]bool, domain string, canonicalToken uint64) bool {
	cls := classOf(domain, canonicalToken)
	if cls == "" {
		return true
	}
	return matrix[cls]
}

// computeAvailable implements the task-187 Task 5 definition: available =
// { identities present in this version's semantics (Task 4) } MINUS
// { identities whose class is not released at this version }. ids is the
// full identities.yaml roster (used only for its name->canonicalToken
// lookup, since classOf is version-independent and operates on the
// canonical token, never on this version's wire id); sem is this version's
// Task-4 join result; matrix is this version's release-class view. Returns
// the AVAILABLE identity names per domain.
func computeAvailable(sem SemanticsMap, ids []IdentityEntry, matrix map[string]bool) (availJob map[string]bool, availSkill map[string]bool) {
	tokenByName := make(map[string]map[string]uint64, 2)
	for _, id := range ids {
		if tokenByName[id.Domain] == nil {
			tokenByName[id.Domain] = make(map[string]uint64)
		}
		tokenByName[id.Domain][id.Name] = id.CanonicalToken
	}

	availJob = make(map[string]bool, len(sem.JobByName))
	for name := range sem.JobByName {
		tok, ok := tokenByName["job"][name]
		if !ok {
			continue // BuildSemantics only binds known identities; defensive.
		}
		if releaseEligible(matrix, "job", tok) {
			availJob[name] = true
		}
	}
	availSkill = make(map[string]bool, len(sem.SkillByName))
	for name := range sem.SkillByName {
		tok, ok := tokenByName["skill"][name]
		if !ok {
			continue
		}
		if releaseEligible(matrix, "skill", tok) {
			availSkill[name] = true
		}
	}
	return availJob, availSkill
}

// ValidateAvailabilitySubset is the task-187 Task 5 Step 2 invariant check:
// every identity computed as available at (region,major,minor) must be
// present in that version's semantic map (available ⊆ semantics -- true by
// construction in computeAvailable, since it only ever considers names
// already present in sem.JobByName/SkillByName). This makes the invariant
// an explicit, machine-checked assertion instead of an implicit property of
// the implementation that a future refactor could silently break.
func ValidateAvailabilitySubset(region string, major, minor uint16) error {
	sem, err := BuildSemantics(region, major, minor)
	if err != nil {
		return err
	}
	ids, err := LoadIdentities(identitiesYAMLPath)
	if err != nil {
		return err
	}
	matrix, err := loadReleaseMatrix(region, major, minor)
	if err != nil {
		return err
	}
	availJob, availSkill := computeAvailable(sem, ids, matrix)

	for name := range availJob {
		if _, ok := sem.JobByName[name]; !ok {
			return fmt.Errorf("availability %s %d.%d: available job identity %q is not present in semantics", region, major, minor, name)
		}
	}
	for name := range availSkill {
		if _, ok := sem.SkillByName[name]; !ok {
			return fmt.Errorf("availability %s %d.%d: available skill identity %q is not present in semantics", region, major, minor, name)
		}
	}
	return nil
}
