package constants_test

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/constants"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

// TestFor_ResolvesPerVersion pins the tenant-keyed selector at the top of
// the task-187 stack: constants.For must return the same per-version
// wireId<->Identity binding that skill.newSet_<key> resolves internally, and
// region matching must be case-insensitive (tenant.Region() returns
// uppercase "GMS"/"JMS", but callers should not have to worry about case).
func TestFor_ResolvesPerVersion(t *testing.T) {
	v48 := constants.For("GMS", 48, 1)
	id, ok := v48.Skill.Resolve(skill.Id(5101004))
	if !ok || id != skill.SuperGmHide {
		t.Fatalf("For(GMS,48,1).Skill.Resolve(5101004) = (%v, %v), want (SuperGmHide, true)", id, ok)
	}

	v72 := constants.For("gms", 72, 1) // case-insensitive region
	id2, ok2 := v72.Skill.Resolve(skill.Id(5101004))
	if !ok2 || id2 != skill.BrawlerCorkscrewBlow {
		t.Fatalf("For(gms,72,1).Skill.Resolve(5101004) = (%v, %v), want (BrawlerCorkscrewBlow, true)", id2, ok2)
	}
}

// TestFor_UnknownFallsBackToBaseline pins the fallback contract: an
// unprovisioned (region,major,minor) tuple must resolve identically to the
// canonical GMS 83.1 baseline rather than returning a zero-value Set.
func TestFor_UnknownFallsBackToBaseline(t *testing.T) {
	got := constants.For("GMS", 200, 7) // unprovisioned
	want := constants.For("GMS", 83, 1) // canonical baseline

	id, ok := got.Skill.Resolve(skill.Id(5101004))
	wid, wok := want.Skill.Resolve(skill.Id(5101004))
	if ok != wok || id != wid {
		t.Fatalf("For(GMS,200,7).Skill.Resolve(5101004) = (%v, %v), want (%v, %v) (GMS 83.1 baseline)", id, ok, wid, wok)
	}
}
