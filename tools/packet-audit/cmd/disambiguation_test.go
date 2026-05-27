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
