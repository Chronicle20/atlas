package monster

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAutoAggroGateAdmit(t *testing.T) {
	base := time.Unix(1_700_000_000, 0)
	tm := newTestTenant(t)

	type call struct {
		characterId uint32
		mobId       uint32
		aggroed     bool
		at          time.Time
	}

	tests := []struct {
		name     string
		calls    []call
		expected []bool
	}{
		{
			name:     "first claim admits",
			calls:    []call{{7, 42, false, base}},
			expected: []bool{true},
		},
		{
			name: "unaggroed repeat inside 1s is blocked",
			calls: []call{
				{7, 42, false, base},
				{7, 42, false, base.Add(900 * time.Millisecond)},
			},
			expected: []bool{true, false},
		},
		{
			name: "unaggroed repeat after 1s admits",
			calls: []call{
				{7, 42, false, base},
				{7, 42, false, base.Add(1100 * time.Millisecond)},
			},
			expected: []bool{true, true},
		},
		{
			name: "aggroed refresh inside 5s is blocked",
			calls: []call{
				{7, 42, false, base},
				{7, 42, true, base.Add(2 * time.Second)},
			},
			expected: []bool{true, false},
		},
		{
			name: "aggroed refresh after 5s admits",
			calls: []call{
				{7, 42, false, base},
				{7, 42, true, base.Add(6 * time.Second)},
			},
			expected: []bool{true, true},
		},
		{
			name: "different mob is independent",
			calls: []call{
				{7, 42, false, base},
				{7, 43, false, base},
			},
			expected: []bool{true, true},
		},
		{
			name: "different character is independent",
			calls: []call{
				{7, 42, false, base},
				{8, 42, false, base},
			},
			expected: []bool{true, true},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := &AutoAggroGate{perTenant: map[uuid.UUID]map[autoAggroKey]time.Time{}}
			for i, c := range tc.calls {
				got := g.Admit(tm, c.characterId, c.mobId, c.aggroed, c.at)
				if got != tc.expected[i] {
					t.Errorf("call %d: Admit() = %v, want %v", i, got, tc.expected[i])
				}
			}
		})
	}
}

func TestAutoAggroGateTenantIsolation(t *testing.T) {
	base := time.Unix(1_700_000_000, 0)
	g := &AutoAggroGate{perTenant: map[uuid.UUID]map[autoAggroKey]time.Time{}}

	tm1 := newTestTenant(t)
	tm2 := newTestTenant(t)

	if got := g.Admit(tm1, 7, 42, false, base); !got {
		t.Errorf("tenant 1 Admit() = %v, want true", got)
	}
	if got := g.Admit(tm2, 7, 42, false, base); !got {
		t.Errorf("tenant 2 Admit() = %v, want true", got)
	}
}

func TestAutoAggroGateSweepStale(t *testing.T) {
	base := time.Unix(1_700_000_000, 0)
	g := &AutoAggroGate{perTenant: map[uuid.UUID]map[autoAggroKey]time.Time{}}
	tm := newTestTenant(t)

	if got := g.Admit(tm, 7, 42, false, base); !got {
		t.Fatalf("Admit() = %v, want true", got)
	}

	if got := g.SweepStale(base.Add(31*time.Minute), 30*time.Minute); got != 1 {
		t.Errorf("SweepStale() = %d, want 1", got)
	}

	if got := g.Admit(tm, 7, 42, false, base.Add(31*time.Minute)); !got {
		t.Errorf("Admit() after sweep = %v, want true", got)
	}
}

func TestAutoAggroGateEvictTenant(t *testing.T) {
	base := time.Unix(1_700_000_000, 0)
	g := &AutoAggroGate{perTenant: map[uuid.UUID]map[autoAggroKey]time.Time{}}
	tm := newTestTenant(t)

	if got := g.Admit(tm, 7, 42, false, base); !got {
		t.Fatalf("Admit() = %v, want true", got)
	}

	g.EvictTenant(tm.Id())

	if got := g.Admit(tm, 7, 42, false, base.Add(100*time.Millisecond)); !got {
		t.Errorf("Admit() after evict = %v, want true", got)
	}
}

func TestAutoAggroConstants(t *testing.T) {
	if AutoAggroProximityThreshold != 40 {
		t.Errorf("AutoAggroProximityThreshold = %d, want 40", AutoAggroProximityThreshold)
	}
	if AutoAggroRefreshInterval != 5*time.Second {
		t.Errorf("AutoAggroRefreshInterval = %v, want 5s", AutoAggroRefreshInterval)
	}
}
