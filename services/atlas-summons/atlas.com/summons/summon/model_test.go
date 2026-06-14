package summon

import (
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

func testField() field.Model {
	return field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
}

func TestBuilderRoundTrip(t *testing.T) {
	exp := time.Unix(1700000000, 0).UTC()
	m := NewBuilder().
		SetId(1000001).
		SetOwnerCharacterId(42).
		SetSkillId(3111002).
		SetSkillLevel(20).
		SetSummonType(SummonTypePuppet).
		SetMovementType(MovementStationary).
		SetField(testField()).
		SetX(100).SetY(-50).
		SetHp(800).SetMaxHp(800).
		SetExpiresAt(exp).
		Build()

	if m.Id() != 1000001 || m.OwnerCharacterId() != 42 || m.SkillId() != 3111002 {
		t.Fatalf("identity getters wrong: %+v", m)
	}
	if m.SummonType() != SummonTypePuppet || m.MovementType() != MovementStationary {
		t.Fatalf("classification getters wrong")
	}
	if m.Hp() != 800 || m.MaxHp() != 800 || m.X() != 100 || m.Y() != -50 {
		t.Fatalf("numeric getters wrong")
	}
	if !m.ExpiresAt().Equal(exp) {
		t.Fatalf("expiresAt wrong")
	}
}

func TestAddHPClampsAtZero(t *testing.T) {
	m := NewBuilder().SetHp(100).SetMaxHp(100).Build()
	m2 := m.AddHP(-250)
	if m2.Hp() != 0 {
		t.Fatalf("expected hp clamped to 0, got %d", m2.Hp())
	}
	// original unchanged (immutability)
	if m.Hp() != 100 {
		t.Fatalf("original mutated")
	}
}
