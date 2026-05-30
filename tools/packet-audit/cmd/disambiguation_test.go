package cmd

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	csvpkg "github.com/Chronicle20/atlas/tools/packet-audit/internal/csv"
)

func TestQualifiedWriterName(t *testing.T) {
	cases := []struct {
		pkg, name, want string
	}{
		{"", "CharacterSpawn", "CharacterSpawn"},
		{"", "AuthSuccess", "AuthSuccess"},
		{"monster", "Spawn", "MonsterSpawn"},
		{"drop", "Spawn", "DropSpawn"},
		{"reactor", "Spawn", "ReactorSpawn"},
		{"pet", "Activated", "PetActivated"},
		{"monster", "MovementAck", "MonsterMovementAck"},
	}
	for _, tc := range cases {
		t.Run(tc.pkg+"/"+tc.name, func(t *testing.T) {
			if got := qualifiedWriterName(tc.pkg, tc.name); got != tc.want {
				t.Errorf("qualifiedWriterName(%q,%q) = %q, want %q", tc.pkg, tc.name, got, tc.want)
			}
		})
	}
}

func TestLocateAtlasFileDisambiguatesByPkg(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	atlasRoot := filepath.Join(repoRoot, "libs", "atlas-packet")

	// "Spawn" struct exists in monster/clientbound, drop/clientbound,
	// reactor/clientbound (and pet/serverbound). Without a pkg hint, the
	// walker picks the first match — that's an alphabetical-order accident,
	// not a correct routing decision.
	cases := []struct {
		pkg, name string
		dir       csvpkg.Direction
		wantInfix string // path substring that must appear when pkg is supplied
	}{
		{"monster", "Spawn", csvpkg.DirClientbound, "/monster/clientbound/"},
		{"drop", "Spawn", csvpkg.DirClientbound, "/drop/clientbound/"},
		{"reactor", "Spawn", csvpkg.DirClientbound, "/reactor/clientbound/"},
		{"pet", "Activated", csvpkg.DirClientbound, "/pet/clientbound/"},
		{"monster", "Damage", csvpkg.DirClientbound, "/monster/clientbound/"},
		{"reactor", "Hit", csvpkg.DirClientbound, "/reactor/clientbound/"},
	}
	for _, tc := range cases {
		t.Run(tc.pkg+"/"+tc.name, func(t *testing.T) {
			got, ok := locateAtlasFile(atlasRoot, tc.name, tc.pkg, tc.dir)
			if !ok {
				t.Fatalf("locateAtlasFile(%q,%q,%q) not found", tc.name, tc.pkg, tc.dir)
			}
			if !strings.Contains(got, tc.wantInfix) {
				t.Errorf("locateAtlasFile(%q,%q,%q) = %q, want path containing %q", tc.name, tc.pkg, tc.dir, got, tc.wantInfix)
			}
		})
	}
}

// fnameForCandidate returns the winning FName for a (pkg,name) candidate in a
// selectCandidates result, or "" if absent.
func fnameForCandidate(sel []selectedCandidate, pkg, name string) string {
	for _, sc := range sel {
		if sc.candidate.pkg == pkg && sc.candidate.name == name {
			return sc.fname
		}
	}
	return ""
}

func TestSelectCandidatesPrefersSyntheticOverDispatcher(t *testing.T) {
	// Both the bare dispatcher CWvsContext::OnGuildResult and its "#"-suffixed
	// synthetic entry map to the same guild::RequestAgreement candidate. The
	// enriched synthetic entry must win deterministically regardless of input
	// order — otherwise the verdict flips between runs (map-iteration
	// nondeterminism: the bare dispatcher has no field calls → ❌, the synthetic
	// entry has the full field list → ✅).
	for _, in := range [][]string{
		{"CWvsContext::OnGuildResult", "CWvsContext::OnGuildResult#RequestAgreement"},
		{"CWvsContext::OnGuildResult#RequestAgreement", "CWvsContext::OnGuildResult"},
	} {
		got := fnameForCandidate(selectCandidates(in), "guild", "RequestAgreement")
		if got != "CWvsContext::OnGuildResult#RequestAgreement" {
			t.Errorf("input %v: guild::RequestAgreement resolved to %q, want the #-suffixed entry", in, got)
		}
	}
}

func TestSelectCandidatesDeterministicAmongPlainFNames(t *testing.T) {
	// Two plain (non-"#") FNames map to party::Operation. The winner must be
	// stable across input orders — lexicographically smallest wins.
	for _, in := range [][]string{
		{"CField::SendCreateNewPartyMsg", "CField::SendWithdrawPartyMsg"},
		{"CField::SendWithdrawPartyMsg", "CField::SendCreateNewPartyMsg"},
	} {
		got := fnameForCandidate(selectCandidates(in), "party", "Operation")
		if got != "CField::SendCreateNewPartyMsg" {
			t.Errorf("input %v: party::Operation resolved to %q, want CField::SendCreateNewPartyMsg", in, got)
		}
	}
}

func TestOrderedExportFNamesPutsSyntheticFirstThenSorts(t *testing.T) {
	in := []string{"Bbb", "Aaa#sub", "Aaa", "Bbb#sub"}
	got := orderedExportFNames(in)
	want := []string{"Aaa#sub", "Bbb#sub", "Aaa", "Bbb"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("orderedExportFNames(%v) = %v, want %v", in, got, want)
		}
	}
}

func TestLocateAtlasFileEmptyPkgKeepsLegacyBehavior(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	atlasRoot := filepath.Join(repoRoot, "libs", "atlas-packet")

	// Without pkg, login/character names with unique short names must still resolve.
	// CharacterSpawn is unique (only one type CharacterSpawn struct in the tree).
	got, ok := locateAtlasFile(atlasRoot, "CharacterSpawn", "", csvpkg.DirClientbound)
	if !ok {
		t.Fatal("locateAtlasFile(CharacterSpawn, \"\") not found")
	}
	if !strings.Contains(got, "/character/clientbound/") {
		t.Errorf("CharacterSpawn resolved to %q, want path under /character/clientbound/", got)
	}
}
