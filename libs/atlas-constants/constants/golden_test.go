package constants_test

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/constants"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

// TestGoldenAnchors pins the PRD's (docs/tasks/task-187-version-aware-id-semantics/prd.md
// §10) acceptance anchors end-to-end through constants.For -- this is the
// task-187 feature's acceptance test, not a unit test of any one layer.
// Every case below is grounded in the committed audit
// (docs/tasks/task-187-version-aware-id-semantics/audit/*.csv, *.md).
func TestGoldenAnchors(t *testing.T) {
	// v0.48 (pre-Pirate): skill 5101004 and job 500 are GM/SuperGM, not
	// Pirate/Brawler -- audit/v048-v062.md, audit/v048-gm-supergm-skill-ranges.md.
	t.Run("v48_skill5101004_is_SuperGmHide", func(t *testing.T) {
		v48 := constants.For("GMS", 48, 1)
		got, ok := v48.Skill.Resolve(skill.Id(5101004))
		if !ok || got != skill.SuperGmHide {
			t.Fatalf("v48 Skill.Resolve(5101004) = (%v, %v), want (SuperGmHide, true)", got, ok)
		}
	})
	t.Run("v48_job500_is_Gm", func(t *testing.T) {
		v48 := constants.For("GMS", 48, 1)
		got, ok := v48.Job.Resolve(job.Id(500))
		if !ok || got != job.Gm {
			t.Fatalf("v48 Job.Resolve(500) = (%v, %v), want (Gm, true)", got, ok)
		}
	})

	// v0.72 (post-Pirate): the same wire ids now mean the canonical
	// post-v83 identities -- audit/v048-v062.md.
	t.Run("v72_skill5101004_is_BrawlerCorkscrewBlow", func(t *testing.T) {
		v72 := constants.For("GMS", 72, 1)
		got, ok := v72.Skill.Resolve(skill.Id(5101004))
		if !ok || got != skill.BrawlerCorkscrewBlow {
			t.Fatalf("v72 Skill.Resolve(5101004) = (%v, %v), want (BrawlerCorkscrewBlow, true)", got, ok)
		}
	})
	t.Run("v72_job500_is_Pirate", func(t *testing.T) {
		v72 := constants.For("GMS", 72, 1)
		got, ok := v72.Job.Resolve(job.Id(500))
		if !ok || got != job.Pirate {
			t.Fatalf("v72 Job.Resolve(500) = (%v, %v), want (Pirate, true)", got, ok)
		}
	})

	// v0.61: last pre-Pirate release, but the WZ already forward-declares a
	// Pirate stub (Pirate itself ships v0.62). Semantic presence (Wire)
	// must succeed while Available must be false -- present-but-
	// unreleased is the whole point of the two-axis model (PRD §1, §4.3).
	t.Run("v61_Pirate_present_but_unavailable", func(t *testing.T) {
		v61 := constants.For("GMS", 61, 1)
		if _, ok := v61.Job.Wire(job.Pirate); !ok {
			t.Fatal("v61 Job.Wire(Pirate) must resolve -- the Pirate WZ stub is semantically present pre-release")
		}
		if v61.Job.Available(job.Pirate) {
			t.Fatal("v61 Job.Available(Pirate) must be false -- Pirate did not release until v0.62")
		}
	})

	// Big Bang (v0.92 -> v0.95): audit/divergences.csv rows 22-25 and
	// audit/bigbang-v092-v095.md document a many-to-one MERGE
	// (WarriorImprovedHpRecovery(1000000) + WarriorImprovedMaxHpIncrease(1000001)
	// + WarriorEndure(1000002) @v92 -> WarriorHpBoost(1000006) @v95), not a
	// bijective wireId<->Identity remap: unlike v48<->v72's single "wire X
	// now means identity Y" swap, three source identities collapse into
	// one target identity with no clean inverse, so there is no single
	// "wire X remaps to Y across the boundary" assertion to make. The
	// generator-verifiable effect IS real, though, and it's a
	// PRESENCE/AVAILABILITY difference: the merge-target identity
	// WarriorHpBoost is entirely absent from v92's Set (its constituent
	// skills hadn't merged yet) and present+available at v95.
	t.Run("bigbang_WarriorHpBoost_absent_v92_present_v95", func(t *testing.T) {
		v92 := constants.For("GMS", 92, 1)
		if _, ok := v92.Skill.Wire(skill.WarriorHpBoost); ok {
			t.Fatal("v92 Skill.Wire(WarriorHpBoost) must NOT resolve -- the merge-target skill does not exist until v95")
		}

		v95 := constants.For("GMS", 95, 1)
		wireId, ok := v95.Skill.Wire(skill.WarriorHpBoost)
		if !ok || wireId != skill.Id(1000006) {
			t.Fatalf("v95 Skill.Wire(WarriorHpBoost) = (%v, %v), want (1000006, true)", wireId, ok)
		}
		if !v95.Skill.Available(skill.WarriorHpBoost) {
			t.Fatal("v95 Skill.Available(WarriorHpBoost) must be true -- released as part of the Big Bang reorg")
		}
	})

	// v0.48-correctness dispatch intent: BrawlerCorkscrewBlow genuinely IS
	// a keydown skill identity under the post-v62 convention -- but a v0.48
	// Super GM pressing wire 5101004 must never be dispatched as that
	// keydown attack, because v48's Set resolves 5101004 to the
	// non-keydown SuperGmHide, not to BrawlerCorkscrewBlow. Together these
	// two facts are the PRD's motivating backend bug (§1) and its fix.
	t.Run("v48_dispatch_intent_hide_not_keydown", func(t *testing.T) {
		if !skill.IsKeyDownSkillIdentity(skill.BrawlerCorkscrewBlow) {
			t.Fatal("BrawlerCorkscrewBlow must be a keydown skill identity")
		}
		if skill.IsKeyDownSkillIdentity(skill.SuperGmHide) {
			t.Fatal("SuperGmHide must NOT be a keydown skill identity")
		}

		v48 := constants.For("GMS", 48, 1)
		got, ok := v48.Skill.Resolve(skill.Id(5101004))
		if !ok || got != skill.SuperGmHide {
			t.Fatalf("v48 wire 5101004 must resolve to SuperGmHide, got (%v, %v)", got, ok)
		}
		if skill.IsKeyDownSkillIdentity(got) {
			t.Fatal("v48 wire 5101004's resolved identity must not be dispatched as a keydown skill")
		}
	})
}
