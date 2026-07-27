package character

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
)

func TestPointResetPolicyFor(t *testing.T) {
	cases := []struct {
		name  string
		jobId job.Id
		want  pointResetPolicy
	}{
		{"Hero (warrior line)", job.Id(112), pointResetPolicy{takeHp: 54, takeMp: 4, gainHp: 20, gainMp: 2}},
		{"Dawn Warrior 3", job.DawnWarriorStage3Id, pointResetPolicy{takeHp: 54, takeMp: 4, gainHp: 20, gainMp: 2}},
		{"Aran 4", job.AranStage4Id, pointResetPolicy{takeHp: 54, takeMp: 4, gainHp: 20, gainMp: 2}},
		{"FP Arch Mage", job.Id(212), pointResetPolicy{takeHp: 10, takeMp: 31, gainHp: 6, gainMp: 18}},
		{"Blaze Wizard 2", job.BlazeWizardStage2Id, pointResetPolicy{takeHp: 10, takeMp: 31, gainHp: 6, gainMp: 18}},
		{"Bowmaster", job.Id(312), pointResetPolicy{takeHp: 20, takeMp: 12, gainHp: 16, gainMp: 10}},
		{"Wind Archer 1", job.WindArcherStage1Id, pointResetPolicy{takeHp: 20, takeMp: 12, gainHp: 16, gainMp: 10}},
		{"Night Lord", job.Id(412), pointResetPolicy{takeHp: 20, takeMp: 12, gainHp: 16, gainMp: 10}},
		{"Night Walker 2", job.NightWalkerStage2Id, pointResetPolicy{takeHp: 20, takeMp: 12, gainHp: 16, gainMp: 10}},
		{"Corsair", job.Id(522), pointResetPolicy{takeHp: 42, takeMp: 16, gainHp: 18, gainMp: 14}},
		{"Thunder Breaker 1", job.ThunderBreakerStage1Id, pointResetPolicy{takeHp: 42, takeMp: 16, gainHp: 18, gainMp: 14}},
		{"Beginner", job.BeginnerId, pointResetPolicy{takeHp: 12, takeMp: 8, gainHp: 8, gainMp: 6}},
		{"Noblesse", job.NoblesseId, pointResetPolicy{takeHp: 12, takeMp: 8, gainHp: 8, gainMp: 6}},
		{"Legend (Aran beginner)", job.LegendId, pointResetPolicy{takeHp: 12, takeMp: 8, gainHp: 8, gainMp: 6}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := pointResetPolicyFor(tc.jobId); got != tc.want {
				t.Errorf("pointResetPolicyFor(%d) = %+v, want %+v", tc.jobId, got, tc.want)
			}
		})
	}
}

func TestPointResetMinPools(t *testing.T) {
	const lvl = byte(30) // representative level; expectations are mult*30+off
	cases := []struct {
		name           string
		jobId          job.Id
		wantHp, wantMp int
	}{
		{"Warrior base", job.Id(100), 24*30 + 118, 4*30 + 55},
		{"Fighter line", job.Id(111), 24*30 + 418, 4*30 + 55},
		{"Page line", job.Id(121), 24*30 + 118, 4*30 + 155},
		{"Spearman line", job.Id(131), 24*30 + 118, 4*30 + 155},
		{"Dawn Warrior 1", job.DawnWarriorStage1Id, 24*30 + 118, 4*30 + 55},
		{"Dawn Warrior 2", job.DawnWarriorStage2Id, 24*30 + 418, 4*30 + 55},
		{"Aran 1", job.AranStage1Id, 24*30 + 118, 4*30 + 55},
		{"Aran 3", job.AranStage3Id, 24*30 + 418, 4*30 + 55},
		{"Magician base", job.Id(200), 10*30 + 54, 22*30 - 1},
		{"FP Wizard (2nd job)", job.Id(210), 10*30 + 54, 22*30 + 449},
		{"Blaze Wizard 1", job.BlazeWizardStage1Id, 10*30 + 54, 22*30 - 1},
		{"Blaze Wizard 2", job.BlazeWizardStage2Id, 10*30 + 54, 22*30 + 449},
		{"Bowman base", job.Id(300), 20*30 + 58, 14*30 - 15},
		{"Hunter line", job.Id(311), 20*30 + 358, 14*30 + 135},
		{"Thief base", job.Id(400), 20*30 + 58, 14*30 - 15},
		{"Bandit line", job.Id(422), 20*30 + 358, 14*30 + 135},
		{"Wind Archer 1", job.WindArcherStage1Id, 20*30 + 58, 14*30 - 15},
		{"Night Walker 2", job.NightWalkerStage2Id, 20*30 + 358, 14*30 + 135},
		{"Pirate base", job.Id(500), 22*30 + 38, 18*30 - 55},
		{"Brawler line", job.Id(512), 22*30 + 338, 18*30 + 95},
		{"Gunslinger line", job.Id(520), 22*30 + 338, 18*30 + 95},
		{"Thunder Breaker 1", job.ThunderBreakerStage1Id, 22*30 + 38, 18*30 - 55},
		{"Thunder Breaker 2", job.ThunderBreakerStage2Id, 22*30 + 338, 18*30 + 95},
		{"Beginner", job.BeginnerId, 12*30 + 38, 10*30 - 5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := pointResetMinHp(tc.jobId, lvl); got != tc.wantHp {
				t.Errorf("pointResetMinHp(%d, %d) = %d, want %d", tc.jobId, lvl, got, tc.wantHp)
			}
			if got := pointResetMinMp(tc.jobId, lvl); got != tc.wantMp {
				t.Errorf("pointResetMinMp(%d, %d) = %d, want %d", tc.jobId, lvl, got, tc.wantMp)
			}
		})
	}
}

func TestIsPointResetMagician(t *testing.T) {
	magicians := []job.Id{job.Id(200), job.Id(210), job.Id(212), job.Id(232), job.BlazeWizardStage1Id, job.BlazeWizardStage2Id}
	nonMagicians := []job.Id{job.Id(0), job.Id(100), job.Id(112), job.Id(300), job.Id(412), job.Id(500), job.Id(532), job.BeginnerId}
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
