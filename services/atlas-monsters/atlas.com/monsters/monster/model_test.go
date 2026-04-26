package monster

import (
	"sort"
	"testing"
)

func makeModelWithEntries(entries []entry) Model {
	return Model{damageEntries: entries}
}

func TestDamageSummaryPassthrough(t *testing.T) {
	src := []entry{
		{CharacterId: 1, Damage: 100, LastHitMs: 10},
		{CharacterId: 2, Damage: 200, LastHitMs: 20},
	}
	m := makeModelWithEntries(src)
	got := m.DamageSummary()
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got))
	}
	sort.Slice(got, func(i, j int) bool { return got[i].CharacterId < got[j].CharacterId })
	if got[0].CharacterId != 1 || got[0].Damage != 100 {
		t.Errorf("got[0]: %+v", got[0])
	}
	if got[1].CharacterId != 2 || got[1].Damage != 200 {
		t.Errorf("got[1]: %+v", got[1])
	}
}

func TestDamageLeaderOverAggregatedEntries(t *testing.T) {
	m := makeModelWithEntries([]entry{
		{CharacterId: 1, Damage: 50, LastHitMs: 1},
		{CharacterId: 2, Damage: 200, LastHitMs: 2},
		{CharacterId: 3, Damage: 150, LastHitMs: 3},
	})
	leader := m.DamageLeader()
	if leader != 2 {
		t.Fatalf("expected leader=2, got %d", leader)
	}
}
