package job

import "testing"

func TestMaxLevelFor(t *testing.T) {
	tests := []struct {
		name string
		id   Id
		want byte
	}{
		{"beginner", BeginnerId, 200},
		{"warrior", WarriorId, 200},
		{"magician", MagicianId, 200},
		{"bowman", BowmanId, 200},
		{"thief", RogueId, 200},
		{"pirate", PirateId, 200},
		{"noblesse", NoblesseId, 200},
		{"dawn warrior", Id(1100), 200},
		{"legend", LegendId, 200},
		{"evan", EvanId, 200},
		{"unknown high id", Id(9999), 200},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MaxLevelFor(tt.id); got != tt.want {
				t.Errorf("MaxLevelFor(%d) = %d; want %d", tt.id, got, tt.want)
			}
		})
	}
}

func TestMaxLevelForIsExhaustive(t *testing.T) {
	for id := range Jobs {
		if got := MaxLevelFor(id); got != 200 {
			t.Errorf("MaxLevelFor(%d) = %d; want 200", id, got)
		}
	}
}
