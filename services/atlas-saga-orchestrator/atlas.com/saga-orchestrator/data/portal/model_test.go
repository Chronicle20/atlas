package portal

import (
	"testing"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func TestSpawnPoint(t *testing.T) {
	tests := []struct {
		name       string
		portalType uint8
		want       bool
	}{
		{name: "type 0 is a spawn point", portalType: 0, want: true},
		{name: "type 1 is a spawn point", portalType: 1, want: true},
		{name: "type 2 is not a spawn point", portalType: 2, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{portalType: tt.portalType}
			if got := SpawnPoint(m); got != tt.want {
				t.Errorf("SpawnPoint(type=%d) = %v, want %v", tt.portalType, got, tt.want)
			}
		})
	}
}

// TestSpawnPointNoTargetComposition pins the SpawnPoint && NoTarget combination
// the way RandomSpawnPointIdProvider composes it (processor.go:46-58), so the
// widening of SpawnPoint to types 0 and 1 is verified at the level the warp
// actions actually use.
func TestSpawnPointNoTargetComposition(t *testing.T) {
	tests := []struct {
		name        string
		portalType  uint8
		targetMapId uint32
		want        bool
	}{
		{name: "type 0, no target passes", portalType: 0, targetMapId: 999999999, want: true},
		{name: "type 1, no target passes", portalType: 1, targetMapId: 999999999, want: true},
		{name: "type 2, no target fails on type", portalType: 2, targetMapId: 999999999, want: false},
		{name: "type 0, has target fails on target", portalType: 0, targetMapId: 100000000, want: false},
		{name: "type 1, has target fails on target", portalType: 1, targetMapId: 100000000, want: false},
	}

	filters := model.Filters(SpawnPoint, NoTarget)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{portalType: tt.portalType, targetMapId: _map.Id(tt.targetMapId)}
			got := true
			for _, f := range filters {
				if !f(m) {
					got = false
					break
				}
			}
			if got != tt.want {
				t.Errorf("SpawnPoint && NoTarget (type=%d, targetMapId=%d) = %v, want %v", tt.portalType, tt.targetMapId, got, tt.want)
			}
		})
	}
}
