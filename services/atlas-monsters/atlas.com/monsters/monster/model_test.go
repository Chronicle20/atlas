package monster

import (
	"sort"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

func testField() field.Model {
	return field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
}

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

func TestModel_NextSkillDecision(t *testing.T) {
	zero := nextSkillDecision{}
	m := NewMonster(testField(), 1, 9000000, 0, 0, 0, 0, 0, 100, 50)
	if m.NextSkillDecision() != zero {
		t.Fatalf("default decision should be sentinel zero, got %+v", m.NextSkillDecision())
	}

	d := nextSkillDecision{
		skillId: 100, skillLevel: 1,
		decidedAtMs:            1700000000000,
		nextEligibleRepickAtMs: 1700000005000,
	}
	updated := Clone(m).SetNextSkillDecision(d).Build()
	if updated.NextSkillDecision() != d {
		t.Fatalf("decision not persisted, got %+v", updated.NextSkillDecision())
	}
}

func TestModel_LastDamageTakenMsRoundTrip(t *testing.T) {
	m := NewMonster(testField(), 1, 9000000, 0, 0, 0, 0, 0, 100, 50)
	if m.LastDamageTakenMs() != 0 {
		t.Errorf("expected zero initial lastDamageTakenMs; got %d", m.LastDamageTakenMs())
	}
	m2 := Clone(m).SetLastDamageTakenMs(123456).Build()
	if m2.LastDamageTakenMs() != 123456 {
		t.Errorf("expected 123456 after builder set; got %d", m2.LastDamageTakenMs())
	}
}
