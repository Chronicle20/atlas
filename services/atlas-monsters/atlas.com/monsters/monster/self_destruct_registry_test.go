package monster

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func TestRegistrySelfDestructTransitionsOnce(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := testContext(ten)
	r.Clear(ctx)

	f := field.NewBuilder(0, 0, 40000).Build()
	m := r.CreateMonster(ctx, ten, f, 5100002, 0, 0, 0, 5, 0, 4000, 0, "", "")

	s, err := r.SelfDestruct(ten, m.UniqueId())
	if err != nil {
		t.Fatalf("unexpected error on first SelfDestruct: %s", err.Error())
	}
	if !s.Killed {
		t.Fatal("expected Killed == true on the transitioning call")
	}
	if s.Monster.Hp() != 0 {
		t.Fatalf("expected Hp == 0, got %d", s.Monster.Hp())
	}
	if s.Monster.Alive() {
		t.Fatal("expected monster to be not alive after SelfDestruct")
	}

	s2, err := r.SelfDestruct(ten, m.UniqueId())
	if err != nil {
		t.Fatalf("unexpected error on second SelfDestruct: %s", err.Error())
	}
	if s2.Killed {
		t.Fatal("expected Killed == false on the second call")
	}
	if s2.Monster.Hp() != 0 {
		t.Fatalf("expected Hp == 0, got %d", s2.Monster.Hp())
	}
}

func TestRegistrySelfDestructLeavesDamageEntries(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := testContext(ten)
	r.Clear(ctx)

	f := field.NewBuilder(0, 0, 40000).Build()
	m := r.CreateMonster(ctx, ten, f, 5100002, 0, 0, 0, 5, 0, 4000, 0, "", "")

	_, err := r.ApplyDamage(ten, 777, 100, m.UniqueId(), time.Now().UnixMilli())
	if err != nil {
		t.Fatalf("unexpected error on ApplyDamage: %s", err.Error())
	}

	s, err := r.SelfDestruct(ten, m.UniqueId())
	if err != nil {
		t.Fatalf("unexpected error on SelfDestruct: %s", err.Error())
	}

	entries := s.Monster.DamageSummary()
	if len(entries) != 1 {
		t.Fatalf("expected exactly one damage entry, got %d", len(entries))
	}
	if entries[0].CharacterId != 777 {
		t.Fatalf("expected CharacterId == 777, got %d", entries[0].CharacterId)
	}
	if entries[0].Damage != 100 {
		t.Fatalf("expected Damage == 100, got %d", entries[0].Damage)
	}
	if s.Monster.DamageLeader() != 777 {
		t.Fatalf("expected DamageLeader == 777, got %d", s.Monster.DamageLeader())
	}
}

func TestRegistrySelfDestructUnknownMonster(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := testContext(ten)
	r.Clear(ctx)

	s, err := r.SelfDestruct(ten, 999999)
	if err == nil {
		t.Fatal("expected non-nil error for unknown monster")
	}
	if s.Killed {
		t.Fatal("expected Killed == false for unknown monster")
	}
}
