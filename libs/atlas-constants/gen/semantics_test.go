package main

import (
	"strings"
	"testing"
)

// TestEmitSemantics_v48_HideNotCorkscrew is the PRD-motivating regression:
// at GMS v48 (pre-Pirate), wire id 500/510 are the Gm/SuperGm jobs and skill
// wire id 5101004 is SuperGmHide, NOT BrawlerCorkscrewBlow (which is what a
// naive auto-bind against the canonical v83 token table would produce, since
// 5101004 is BrawlerCorkscrewBlow's canonical token). See
// docs/tasks/task-187-version-aware-id-semantics/audit/divergences.csv.
func TestEmitSemantics_v48_HideNotCorkscrew(t *testing.T) {
	m, err := BuildSemantics("gms", 48, 1) // map[wireId]identityName, both domains
	if err != nil {
		t.Fatal(err)
	}
	if m.Skill[5101004] != "SuperGmHide" {
		t.Fatalf("v48 wire 5101004 should be SuperGmHide, got %q", m.Skill[5101004])
	}
	if m.Job[500] != "Gm" {
		t.Fatalf("v48 job 500 should be Gm, got %q", m.Job[500])
	}
}

// TestEmitSemantics_v62_CorkscrewAndPirate pins the earliest provisioned
// post-Pirate column (gms 72; gms 61 is a pre-Pirate WZ stub): here the
// same wire ids resolve to the canonical post-Pirate identities.
func TestEmitSemantics_v62_CorkscrewAndPirate(t *testing.T) {
	m, err := BuildSemantics("gms", 72, 1) // 72 is the earliest provisioned post-Pirate col
	if err != nil {
		t.Fatal(err)
	}
	if m.Skill[5101004] != "BrawlerCorkscrewBlow" {
		t.Fatal("v72 5101004 should be BrawlerCorkscrewBlow")
	}
	if m.Job[500] != "Pirate" {
		t.Fatal("v72 job 500 should be Pirate")
	}
}

// TestBuildSemantics_AllProvisionedVersions smoke-tests every provisioned
// (region,major,minor) builds cleanly -- the generator must never silently
// drop a version.
func TestBuildSemantics_AllProvisionedVersions(t *testing.T) {
	for _, v := range provisionedVersions {
		m, err := BuildSemantics(v.Region, v.Major, v.Minor)
		if err != nil {
			t.Fatalf("%s %d.%d: %v", v.Region, v.Major, v.Minor, err)
		}
		if len(m.Skill) == 0 || len(m.Job) == 0 {
			t.Fatalf("%s %d.%d: empty semantics map (skill=%d job=%d)", v.Region, v.Major, v.Minor, len(m.Skill), len(m.Job))
		}
	}
}

// TestBuildSemantics_UnknownVersionErrors -- an unprovisioned version has no
// semantics.yaml / snapshot pinned, so BuildSemantics must error rather than
// silently returning an empty/zero map (the caller -- Task 6's `for.go` --
// is the one responsible for falling back to the canonical baseline).
func TestBuildSemantics_UnknownVersionErrors(t *testing.T) {
	if _, err := BuildSemantics("gms", 200, 7); err == nil {
		t.Fatal("expected error for unprovisioned version, got nil")
	}
}

// TestEmitSemantics_GoldenSnippets checks EmitSemantics renders the
// divergent v48 skill/job bindings into the generated Go source (not just
// into the in-memory join map), and that the v72 baseline binding is also
// present.
func TestEmitSemantics_GoldenSnippets(t *testing.T) {
	skillGo, jobGo, err := EmitSemantics("gms", 48, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(skillGo, "package skill") {
		t.Fatalf("skillGo missing package clause:\n%s", skillGo)
	}
	if !strings.Contains(skillGo, "5101004: SuperGmHide") {
		t.Fatalf("skillGo missing v48 5101004->SuperGmHide binding:\n%s", skillGo)
	}
	if !strings.Contains(jobGo, "package job") {
		t.Fatalf("jobGo missing package clause:\n%s", jobGo)
	}
	if !strings.Contains(jobGo, "500: Gm") {
		t.Fatalf("jobGo missing v48 500->Gm binding:\n%s", jobGo)
	}
	if !strings.Contains(jobGo, "func newSet_gms_48_1() Set") {
		t.Fatalf("jobGo missing newSet_gms_48_1 constructor:\n%s", jobGo)
	}

	skillGo72, _, err := EmitSemantics("gms", 72, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(skillGo72, "5101004: BrawlerCorkscrewBlow") {
		t.Fatalf("skillGo72 missing v72 5101004->BrawlerCorkscrewBlow binding:\n%s", skillGo72)
	}
}

// TestLoadDivergences_ExcludesDocumentationRows verifies the resolve-or-
// exclude contract: a semantic-override row (bare identifier that resolves
// against identities.yaml) is included in the version's divergent overlay;
// a documentation row (Big Bang bracketed classification, the DualBlade gap)
// is excluded but logged, never silently dropped.
func TestLoadDivergences_ExcludesDocumentationRows(t *testing.T) {
	sf, err := loadSemanticsFile("gms", 92, 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range sf.Divergent {
		if strings.ContainsAny(e.IdentityName, "[( ") {
			t.Fatalf("v92 divergent list must only contain resolving bare identifiers, found %q", e.IdentityName)
		}
	}
	if len(sf.Excluded) == 0 {
		t.Fatal("v92 excluded list should record the Big Bang documentation rows, found none")
	}

	// gms 48 must have real semantic overrides, not just documentation.
	sf48, err := loadSemanticsFile("gms", 48, 1)
	if err != nil {
		t.Fatal(err)
	}
	foundHide := false
	for _, e := range sf48.Divergent {
		if e.Domain == "skill" && e.WireId == 5101004 {
			if e.IdentityName != "SuperGmHide" {
				t.Fatalf("v48 divergent 5101004 = %q, want SuperGmHide", e.IdentityName)
			}
			foundHide = true
		}
	}
	if !foundHide {
		t.Fatal("v48 divergent list missing skill wireId 5101004 -> SuperGmHide override")
	}
}
