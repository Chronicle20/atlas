package summon

import (
	"testing"

	monstermsg "atlas-summons/monster"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

// countStatusEvents returns the number of captured SUMMON_STATUS events of the
// given type.
func countStatusEvents(captured *[]capturedMessage, eventType string) int {
	n := 0
	for i := range *captured {
		c := &(*captured)[i]
		if c.topic == EnvEventTopicSummonStatus && c.payload["type"] == eventType {
			n++
		}
	}
	return n
}

func TestPuppetDamageDecrementsAndDestroysAtZero(t *testing.T) {
	p, captured := newPuppetProcessor(t, effectWithX(100, 60000))
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).SetInstance(uuid.Nil).Build()

	m, err := p.Spawn(f, 42, 3111002, 20, 100, -50, 0, 0)
	if err != nil {
		t.Fatalf("Spawn returned error: %v", err)
	}
	if m.Hp() != 100 {
		t.Fatalf("expected initial hp 100, got %d", m.Hp())
	}

	// First hit: owner 42, 30 damage -> hp 70, DAMAGED emitted, still alive.
	if err := p.Damage(m.Id(), 42, 30, 9300018); err != nil {
		t.Fatalf("Damage returned error: %v", err)
	}
	after, err := GetRegistry().Get(p.ctx, p.t, m.Id())
	if err != nil {
		t.Fatalf("expected summon alive after non-lethal damage: %v", err)
	}
	if after.Hp() != 70 {
		t.Fatalf("expected hp 70 after 30 damage, got %d", after.Hp())
	}
	if got := countStatusEvents(captured, EventSummonStatusDamaged); got != 1 {
		t.Fatalf("expected 1 DAMAGED event, got %d", got)
	}
	if got := countStatusEvents(captured, EventSummonStatusDestroyed); got != 0 {
		t.Fatalf("expected no DESTROYED event yet, got %d", got)
	}

	// Second hit: 100 damage -> hp 0, DESTROYED + REMOVE_PUPPET, gone.
	if err := p.Damage(m.Id(), 42, 100, 9300018); err != nil {
		t.Fatalf("Damage returned error: %v", err)
	}
	if _, err := GetRegistry().Get(p.ctx, p.t, m.Id()); err == nil {
		t.Fatalf("expected summon gone after lethal damage")
	}
	if got := countStatusEvents(captured, EventSummonStatusDamaged); got != 2 {
		t.Fatalf("expected 2 DAMAGED events, got %d", got)
	}
	if got := countStatusEvents(captured, EventSummonStatusDestroyed); got != 1 {
		t.Fatalf("expected 1 DESTROYED event, got %d", got)
	}
	if remove := findCommand(captured, monstermsg.CommandTypeRemovePuppet); remove == nil {
		t.Fatalf("expected REMOVE_PUPPET emitted on lethal damage")
	}
}

func TestDamageByNonOwnerDropped(t *testing.T) {
	p, captured := newPuppetProcessor(t, effectWithX(100, 60000))
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).SetInstance(uuid.Nil).Build()

	m, err := p.Spawn(f, 42, 3111002, 20, 100, -50, 0, 0)
	if err != nil {
		t.Fatalf("Spawn returned error: %v", err)
	}

	// Non-owner 99 attempts to damage -> dropped, hp unchanged, no DAMAGED.
	if err := p.Damage(m.Id(), 99, 50, 9300018); err != nil {
		t.Fatalf("Damage returned error: %v", err)
	}
	after, err := GetRegistry().Get(p.ctx, p.t, m.Id())
	if err != nil {
		t.Fatalf("expected summon still present: %v", err)
	}
	if after.Hp() != 100 {
		t.Fatalf("expected hp unchanged (100) after non-owner damage, got %d", after.Hp())
	}
	if got := countStatusEvents(captured, EventSummonStatusDamaged); got != 0 {
		t.Fatalf("expected no DAMAGED event from non-owner, got %d", got)
	}
}
