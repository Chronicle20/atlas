package diff

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/atlaspacket"
)

// TestFlattenMonsterSpawnRunsToCompletion guards Phase 2 against silent
// regressions in FlattenWithRegistry when the combat hot-path monster
// spawn encoder is analysed. MonsterSpawn calls model.MonsterModel.Encode
// through field m.monster, which the registry cannot currently resolve
// (the registry keys on unqualified struct names and there are 4 "Spawn"
// types across monster/drop/reactor/pet sub-domains — last-write-wins
// loses the monster field). This is a known FP tracked under design §3
// and is expected to surface as ❌ in Phase 2 with a manual IDA verdict
// in the report's prose. This fixture pins safe completion, not full
// expansion — until the registry handles name collisions, the
// KindRecurse marker stays.
func TestFlattenMonsterSpawnRunsToCompletion(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "libs", "atlas-packet")
	reg, err := atlaspacket.NewTypeRegistry(root)
	if err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(root, "monster", "clientbound", "spawn.go")
	calls, err := atlaspacket.AnalyzeFileWithRegistry(src, "Spawn", "Encode", reg)
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) == 0 {
		t.Fatal("AnalyzeFileWithRegistry returned no calls for MonsterSpawn")
	}
	// Must complete in bounded time without panicking. Output length may
	// equal len(calls) when the m.monster KindRecurse marker cannot be
	// resolved — that's the analyzer FP design §3 predicts.
	flat := FlattenWithRegistry(calls, atlaspacket.GuardContext{Region: "GMS", MajorVersion: 95}, reg)
	if len(flat) == 0 {
		t.Fatal("FlattenWithRegistry returned no calls for MonsterSpawn")
	}
}

// TestFlattenMonsterStatSetCycleSafe verifies that flattening monster
// StatSet (which delegates to *model.MonsterTemporaryStat.Encode) completes
// in bounded time. MonsterTemporaryStat is a complex sub-struct with mask
// dispatching; if FlattenWithRegistry's cycle guard regresses, this test
// is the canary.
func TestFlattenMonsterStatSetCycleSafe(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "libs", "atlas-packet")
	reg, err := atlaspacket.NewTypeRegistry(root)
	if err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(root, "monster", "clientbound", "stat.go")
	calls, err := atlaspacket.AnalyzeFileWithRegistry(src, "StatSet", "Encode", reg)
	if err != nil {
		t.Fatal(err)
	}
	// Should complete; cycle guard ensures bounded recursion.
	_ = FlattenWithRegistry(calls, atlaspacket.GuardContext{Region: "GMS", MajorVersion: 95}, reg)
}
