package job_test

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
)

func TestAdvancement(t *testing.T) {
	cases := []struct {
		name  string
		jobId job.Id
		want  int
	}{
		{"Beginner", job.BeginnerId, 0},
		{"Noblesse", job.NoblesseId, 0},
		{"Legend (Aran beginner)", job.LegendId, 0},
		{"Evan beginner (2001)", job.EvanId, 0},
		{"Warrior", job.Id(100), 1},
		{"Fighter", job.Id(110), 2},
		{"Crusader", job.Id(111), 3},
		{"Hero", job.Id(112), 4},
		{"Page", job.Id(120), 2},
		{"Paladin", job.Id(122), 4},
		{"Spearman", job.Id(130), 2},
		{"Dark Knight", job.Id(132), 4},
		{"Magician", job.Id(200), 1},
		{"FP Wizard", job.Id(210), 2},
		{"IL Arch Mage", job.Id(222), 4},
		{"Bowman", job.Id(300), 1},
		{"Bowmaster", job.Id(312), 4},
		{"Thief", job.Id(400), 1},
		{"Night Lord", job.Id(412), 4},
		{"Pirate", job.Id(500), 1},
		{"Corsair", job.Id(522), 4},
		{"Dawn Warrior 1", job.DawnWarriorStage1Id, 1},
		{"Dawn Warrior 2", job.DawnWarriorStage2Id, 2},
		{"Dawn Warrior 3", job.DawnWarriorStage3Id, 3},
		{"Dawn Warrior 4", job.DawnWarriorStage4Id, 4},
		{"Aran 1", job.AranStage1Id, 1},
		{"Aran 2", job.AranStage2Id, 2},
		{"Aran 3", job.AranStage3Id, 3},
		{"Aran 4", job.AranStage4Id, 4},
		{"Evan stage 1 (excluded)", job.EvanStage1Id, -1},
		{"Evan stage 5 (excluded)", job.EvanStage5Id, -1},
		{"Evan stage 10 (excluded)", job.EvanStage10Id, -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := job.Advancement(tc.jobId); got != tc.want {
				t.Errorf("Advancement(%d) = %d, want %d", tc.jobId, got, tc.want)
			}
		})
	}
}
