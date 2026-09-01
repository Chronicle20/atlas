package writer

import (
	charskill "atlas-channel/character/skill"
	dataskill "atlas-channel/data/skill"
	"atlas-channel/data/skill/effect"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// masteryTestCtx builds a tenant-bearing context for one (region, major,
// minor): computeMasteryForWeapon resolves the tenant from ctx (both for
// the >=95 client-side-calc gate and, since task-187, for job identity
// resolution on the Knuckle branch).
func masteryTestCtx(t *testing.T, region string, major, minor uint16) (context.Context, tenant.Model) {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), region, major, minor)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tenant.WithContext(context.Background(), tm), tm
}

// seedSkillMastery installs a data/skill.Model for skillId directly into the
// in-process, tenant-keyed skill-data cache (data/skill.SeedForTest) so
// getMasteryFromSkill's skill3.NewProcessor(...).GetById call resolves
// in-process rather than making a live REST call to atlas-data. effectCount
// drives the level-group-size term of the mastery formula (gs=3 at
// maxLevel==30, else gs=2).
func seedSkillMastery(t *testing.T, tm tenant.Model, skillId skill2.Id, effectCount int) {
	t.Helper()
	m, err := dataskill.Extract(dataskill.RestModel{
		Id:      uint32(skillId),
		Effects: make([]effect.RestModel, effectCount),
	})
	if err != nil {
		t.Fatalf("dataskill.Extract: %v", err)
	}
	dataskill.SeedForTest(tm, uint32(skillId), m)
}

func trainedSkill(id skill2.Id, level byte) charskill.Model {
	m, err := charskill.Extract(charskill.RestModel{Id: uint32(id), Level: level})
	if err != nil {
		panic(err)
	}
	return m
}

// oneHandedSwordItemId / knuckleItemId are real weapon-classification item
// ids (item.GetWeaponType buckets by (id/10000)%100).
const (
	oneHandedSwordItemId = uint32(1302000)
	knuckleItemId        = uint32(1480000)
)

// TestComputeMasteryForWeapon_PageSwordStableAcrossVersions pins the
// task-187 brief's Step 1 mastery-preservation case: Page Sword Mastery
// (Warrior branch, a version-stable root per the audit) must compute the
// same mastery value before and after the task-187 refactor, on both the
// pre-Big-Bang wire encoding (v83) and the client-side-calc branch (v95).
// This is a no-regression pin, not a remap proof -- Page/Warrior roots do
// not remap across the provisioned GMS versions.
func TestComputeMasteryForWeapon_PageSwordStableAcrossVersions(t *testing.T) {
	skills := []charskill.Model{trainedSkill(skill2.PageSwordMasteryId, 20)}

	t.Run("v83", func(t *testing.T) {
		ctx, tm := masteryTestCtx(t, "GMS", 83, 1)
		seedSkillMastery(t, tm, skill2.PageSwordMasteryId, 30)
		got := computeMasteryForWeapon(logrus.New())(ctx)(oneHandedSwordItemId, job.PageId, 0, skills)
		if got != 7 {
			t.Fatalf("mastery = %d, want 7 (v83 wire-packed encoding)", got)
		}
	})

	t.Run("v95", func(t *testing.T) {
		ctx, tm := masteryTestCtx(t, "GMS", 95, 1)
		seedSkillMastery(t, tm, skill2.PageSwordMasteryId, 30)
		got := computeMasteryForWeapon(logrus.New())(ctx)(oneHandedSwordItemId, job.PageId, 0, skills)
		if got != 45 {
			t.Fatalf("mastery = %d, want 45 (v95 client-side-calc branch)", got)
		}
	})
}

// TestComputeMasteryForWeapon_BrawlerKnuckleWireCollision pins a divergent-id
// bug found beyond the Task 10 brief's listed sites (task-187 audit):
// job.BrawlerId/MarauderId/BuccaneerId are wire 510/511/512, and wire 510
// collides with SuperGM at v0.48 (the audit's GM/SuperGM-vs-Pirate/Brawler
// divergent job set). A raw job.IsA(jobId, job.BrawlerId, ...) compare on
// the Knuckle branch would misclassify a v0.48 SuperGM (whose JobId() wire
// value is 510) wielding a Knuckle as a Brawler. The fix resolves jobId to
// its version-blind Identity before comparing.
func TestComputeMasteryForWeapon_BrawlerKnuckleWireCollision(t *testing.T) {
	skills := []charskill.Model{trainedSkill(skill2.BrawlerKnucklerMasteryId, 11)}

	t.Run("v61_real_brawler_wire510_still_gets_mastery", func(t *testing.T) {
		ctx, tm := masteryTestCtx(t, "GMS", 61, 1)
		seedSkillMastery(t, tm, skill2.BrawlerKnucklerMasteryId, 20)
		got := computeMasteryForWeapon(logrus.New())(ctx)(knuckleItemId, job.Id(510), 0, skills)
		if got != 6 {
			t.Fatalf("mastery = %d, want 6 (v61 wire 510 = Brawler; mastery must apply)", got)
		}
	})

	t.Run("v48_supergm_wire510_must_not_get_brawler_mastery", func(t *testing.T) {
		ctx, tm := masteryTestCtx(t, "GMS", 48, 1)
		seedSkillMastery(t, tm, skill2.BrawlerKnucklerMasteryId, 20)
		got := computeMasteryForWeapon(logrus.New())(ctx)(knuckleItemId, job.Id(510), 0, skills)
		if got != 0 {
			t.Fatalf("mastery = %d, want 0 (v48 wire 510 = SuperGM, NOT Brawler; must not misapply Brawler mastery)", got)
		}
	})
}
