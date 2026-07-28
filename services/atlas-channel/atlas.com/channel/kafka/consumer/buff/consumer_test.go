package buff

import (
	"testing"

	buff2 "atlas-channel/kafka/message/buff"
)

func TestIsBattleshipRide(t *testing.T) {
	riding := []buff2.StatChange{{Type: "MONSTER_RIDING", Amount: 1932000}}
	noRiding := []buff2.StatChange{{Type: "WEAPON_DEFENSE", Amount: 10}}
	tests := []struct {
		name     string
		sourceId int32
		changes  []buff2.StatChange
		expected bool
	}{
		{"battleship riding buff", 5221006, riding, true},
		{"battleship without riding change", 5221006, noRiding, false},
		{"other mount riding buff", 1019, riding, false},
		{"cannon is not the mount", 5221007, riding, false},
		{"empty changes", 5221006, nil, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isBattleshipRide(tc.sourceId, tc.changes); got != tc.expected {
				t.Errorf("isBattleshipRide(%d, %v) = %v, want %v", tc.sourceId, tc.changes, got, tc.expected)
			}
		})
	}
}
