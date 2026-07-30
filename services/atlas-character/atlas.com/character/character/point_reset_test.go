package character

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
)

func TestPointResetPolicyFor(t *testing.T) {
	cases := []struct {
		name string
		jid  job.Identity
		want pointResetPolicy
	}{
		{"Hero (warrior line)", job.Hero, pointResetPolicy{takeHp: 54, takeMp: 4, gainHp: 20, gainMp: 2}},
		{"Dawn Warrior 3", job.DawnWarriorStage3, pointResetPolicy{takeHp: 54, takeMp: 4, gainHp: 20, gainMp: 2}},
		{"Aran 4", job.AranStage4, pointResetPolicy{takeHp: 54, takeMp: 4, gainHp: 20, gainMp: 2}},
		{"FP Arch Mage", job.FirePoisonArchMagician, pointResetPolicy{takeHp: 10, takeMp: 31, gainHp: 6, gainMp: 18}},
		{"Blaze Wizard 2", job.BlazeWizardStage2, pointResetPolicy{takeHp: 10, takeMp: 31, gainHp: 6, gainMp: 18}},
		{"Bowmaster", job.Bowmaster, pointResetPolicy{takeHp: 20, takeMp: 12, gainHp: 16, gainMp: 10}},
		{"Wind Archer 1", job.WindArcherStage1, pointResetPolicy{takeHp: 20, takeMp: 12, gainHp: 16, gainMp: 10}},
		{"Night Lord", job.NightLord, pointResetPolicy{takeHp: 20, takeMp: 12, gainHp: 16, gainMp: 10}},
		{"Night Walker 2", job.NightWalkerStage2, pointResetPolicy{takeHp: 20, takeMp: 12, gainHp: 16, gainMp: 10}},
		{"Corsair", job.Corsair, pointResetPolicy{takeHp: 42, takeMp: 16, gainHp: 18, gainMp: 14}},
		{"Thunder Breaker 1", job.ThunderBreakerStage1, pointResetPolicy{takeHp: 42, takeMp: 16, gainHp: 18, gainMp: 14}},
		{"Beginner", job.Beginner, pointResetPolicy{takeHp: 12, takeMp: 8, gainHp: 8, gainMp: 6}},
		{"Noblesse", job.Noblesse, pointResetPolicy{takeHp: 12, takeMp: 8, gainHp: 8, gainMp: 6}},
		{"Legend (Aran beginner)", job.Legend, pointResetPolicy{takeHp: 12, takeMp: 8, gainHp: 8, gainMp: 6}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := pointResetPolicyFor(tc.jid); got != tc.want {
				t.Errorf("pointResetPolicyFor(%d) = %+v, want %+v", tc.jid, got, tc.want)
			}
		})
	}
}

// TestPointResetPolicyFor_v48GmDoesNotMatchPirate guards the task-187
// divergent-row fix directly: wire job 500 means GM at v0.48 (job 900/910
// don't exist yet), not Pirate. Since pointResetPolicyFor now takes a
// resolved job.Identity rather than a raw wire id, a v0.48 GM's resolved
// identity (job.Gm) must NOT hit the Pirate row (job.Pirate,
// job.ThunderBreakerStage1) -- it falls through to the default policy
// (GM/SuperGM have no explicit AP-reset row), exactly as a raw wire-500
// Pirate row match would have wrongly done pre-fix.
func TestPointResetPolicyFor_v48GmDoesNotMatchPirate(t *testing.T) {
	if got := pointResetPolicyFor(job.Gm); got != pointResetDefaultPolicy {
		t.Errorf("pointResetPolicyFor(job.Gm) = %+v, want default policy %+v (must not match the Pirate row)", got, pointResetDefaultPolicy)
	}
	if got := pointResetPolicyFor(job.Pirate); got != (pointResetPolicy{takeHp: 42, takeMp: 16, gainHp: 18, gainMp: 14}) {
		t.Errorf("pointResetPolicyFor(job.Pirate) = %+v, want the Pirate row", got)
	}
}

func TestPointResetMinPools(t *testing.T) {
	const lvl = byte(30) // representative level; expectations are mult*30+off
	cases := []struct {
		name           string
		jid            job.Identity
		wantHp, wantMp int
	}{
		{"Warrior base", job.Warrior, 24*30 + 118, 4*30 + 55},
		{"Fighter line", job.Crusader, 24*30 + 418, 4*30 + 55},
		{"Page line", job.WhiteKnight, 24*30 + 118, 4*30 + 155},
		{"Spearman line", job.DragonKnight, 24*30 + 118, 4*30 + 155},
		{"Dawn Warrior 1", job.DawnWarriorStage1, 24*30 + 118, 4*30 + 55},
		{"Dawn Warrior 2", job.DawnWarriorStage2, 24*30 + 418, 4*30 + 55},
		{"Aran 1", job.AranStage1, 24*30 + 118, 4*30 + 55},
		{"Aran 3", job.AranStage3, 24*30 + 418, 4*30 + 55},
		{"Magician base", job.Magician, 10*30 + 54, 22*30 - 1},
		{"FP Wizard (2nd job)", job.FirePoisonWizard, 10*30 + 54, 22*30 + 449},
		{"Blaze Wizard 1", job.BlazeWizardStage1, 10*30 + 54, 22*30 - 1},
		{"Blaze Wizard 2", job.BlazeWizardStage2, 10*30 + 54, 22*30 + 449},
		{"Bowman base", job.Bowman, 20*30 + 58, 14*30 - 15},
		{"Hunter line", job.Ranger, 20*30 + 358, 14*30 + 135},
		{"Thief base", job.Rogue, 20*30 + 58, 14*30 - 15},
		{"Bandit line", job.Shadower, 20*30 + 358, 14*30 + 135},
		{"Wind Archer 1", job.WindArcherStage1, 20*30 + 58, 14*30 - 15},
		{"Night Walker 2", job.NightWalkerStage2, 20*30 + 358, 14*30 + 135},
		{"Pirate base", job.Pirate, 22*30 + 38, 18*30 - 55},
		{"Brawler line", job.Buccaneer, 22*30 + 338, 18*30 + 95},
		{"Gunslinger line", job.Gunslinger, 22*30 + 338, 18*30 + 95},
		{"Thunder Breaker 1", job.ThunderBreakerStage1, 22*30 + 38, 18*30 - 55},
		{"Thunder Breaker 2", job.ThunderBreakerStage2, 22*30 + 338, 18*30 + 95},
		{"Beginner", job.Beginner, 12*30 + 38, 10*30 - 5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := pointResetMinHp(tc.jid, lvl); got != tc.wantHp {
				t.Errorf("pointResetMinHp(%d, %d) = %d, want %d", tc.jid, lvl, got, tc.wantHp)
			}
			if got := pointResetMinMp(tc.jid, lvl); got != tc.wantMp {
				t.Errorf("pointResetMinMp(%d, %d) = %d, want %d", tc.jid, lvl, got, tc.wantMp)
			}
		})
	}
}

// TestPointResetMinPools_v48GmDoesNotMatchBrawlerOrPirate guards the
// task-187 divergent-row fix for the pool-min tables: wire job 510 means
// SuperGM at v0.48, not Brawler -- and wire 500 means GM, not Pirate. A
// resolved job.SuperGm/job.Gm identity must fall through to the default
// (Beginner/Noblesse) pool-min row, not the Brawler/Gunslinger or Pirate
// rows a raw wire-510/500 compare would have wrongly matched pre-fix.
func TestPointResetMinPools_v48GmDoesNotMatchBrawlerOrPirate(t *testing.T) {
	const lvl = byte(30)
	if gotHp := pointResetMinHp(job.SuperGm, lvl); gotHp != 12*30+38 {
		t.Errorf("pointResetMinHp(job.SuperGm, %d) = %d, want default %d (must not match Brawler/Gunslinger row)", lvl, gotHp, 12*30+38)
	}
	if gotMp := pointResetMinMp(job.SuperGm, lvl); gotMp != 10*30-5 {
		t.Errorf("pointResetMinMp(job.SuperGm, %d) = %d, want default %d (must not match Brawler/Gunslinger row)", lvl, gotMp, 10*30-5)
	}
	if gotHp := pointResetMinHp(job.Gm, lvl); gotHp != 12*30+38 {
		t.Errorf("pointResetMinHp(job.Gm, %d) = %d, want default %d (must not match Pirate row)", lvl, gotHp, 12*30+38)
	}
}

func TestIsPointResetMagician(t *testing.T) {
	magicians := []job.Identity{job.Magician, job.FirePoisonWizard, job.FirePoisonArchMagician, job.Bishop, job.BlazeWizardStage1, job.BlazeWizardStage2}
	nonMagicians := []job.Identity{job.Beginner, job.Warrior, job.Hero, job.Bowman, job.NightLord, job.Pirate, job.Identity(532)}
	for _, j := range magicians {
		if !isPointResetMagician(j) {
			t.Errorf("isPointResetMagician(%d) = false, want true", j)
		}
	}
	for _, j := range nonMagicians {
		if isPointResetMagician(j) {
			t.Errorf("isPointResetMagician(%d) = true, want false", j)
		}
	}
}

// pointResetMagicianTakeMp mirrors the client's INT-scaled magician MP-reset-out
// loss: 3*effectiveInt/40 + 30 (integer division). INT 14 reproduces §4.3's old
// flat 31, confirming it was only ever the low-INT value.
func TestPointResetMagicianTakeMp(t *testing.T) {
	cases := []struct {
		intVal uint16
		want   uint16
	}{
		{0, 30}, {14, 31}, {40, 33}, {200, 45}, {999, 104},
	}
	for _, tc := range cases {
		if got := pointResetMagicianTakeMp(tc.intVal); got != tc.want {
			t.Errorf("pointResetMagicianTakeMp(%d) = %d, want %d", tc.intVal, got, tc.want)
		}
	}
}
