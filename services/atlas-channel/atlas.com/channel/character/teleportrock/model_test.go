package teleportrock

import (
	"testing"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

func TestModelContains(t *testing.T) {
	m := NewModel([]_map.Id{100000000}, []_map.Id{104040000})
	if !m.Contains(false, 100000000) || m.Contains(false, 104040000) {
		t.Fatalf("regular membership wrong")
	}
	if !m.Contains(true, 104040000) || m.Contains(true, 100000000) {
		t.Fatalf("vip membership wrong")
	}
}
