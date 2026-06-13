package skill

import "testing"

func TestIsTamedMountSkill(t *testing.T) {
	tests := []struct {
		name     string
		id       Id
		expected bool
	}{
		{"Beginner MonsterRider", 1004, true},
		{"Noblesse MonsterRider", 10001004, true},
		{"Legend MonsterRider", 20001004, true},
		{"Evan MonsterRider", 20011004, true},
		{"Broomstick (skill-only)", 1019, false},
		{"Battleship (out of scope)", 5221006, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsTamedMountSkill(tc.id); got != tc.expected {
				t.Errorf("IsTamedMountSkill(%d) = %v, want %v", tc.id, got, tc.expected)
			}
		})
	}
}

func TestSkillOnlyMountVehicleId(t *testing.T) {
	tests := []struct {
		name        string
		id          Id
		level       int
		expectedVid int32
		expectedOk  bool
	}{
		{"Broomstick beginner", 1019, 1, 1932005, true},
		{"SpaceShip beginner formula", 1013, 3, 1932003, true},
		{"Noblesse Yeti1 same vehicle", 10001019, 1, 1932003, true},
		{"Yeti1 beginner", 1017, 1, 1932003, true},
		{"Yeti2 beginner", 1018, 1, 1932004, true},
		{"Balrog beginner", 1031, 1, 1932010, true},
		{"Noblesse SpaceShip formula", 1001014, 2, 1932002, true},
		{"Legend Balrog", 20001031, 1, 1932010, true},
		{"Tamed MonsterRider not skill-only", 1004, 1, 0, false},
		{"Battleship not skill-only", 5221006, 1, 0, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			vid, ok := SkillOnlyMountVehicleId(tc.id, tc.level)
			if ok != tc.expectedOk {
				t.Errorf("SkillOnlyMountVehicleId(%d, %d) ok = %v, want %v", tc.id, tc.level, ok, tc.expectedOk)
			}
			if vid != tc.expectedVid {
				t.Errorf("SkillOnlyMountVehicleId(%d, %d) vid = %d, want %d", tc.id, tc.level, vid, tc.expectedVid)
			}
		})
	}
}
