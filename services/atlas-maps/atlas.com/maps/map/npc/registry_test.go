package npc

import (
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func TestRegistryAddAndGetAll(t *testing.T) {
	getRegistry().Reset()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(108010600)).Build()
	key := FieldKey{Tenant: te, Field: f}

	m := NewModel(f, getRegistry().NextId(), 1104100, 2830, 78, 0)
	getRegistry().Add(key, m)

	all := getRegistry().GetAll(key)
	if len(all) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(all))
	}
	if all[0].NpcId() != 1104100 {
		t.Errorf("Expected NpcId 1104100, got %d", all[0].NpcId())
	}
}

func TestRegistryGetAllReturnsCopy(t *testing.T) {
	getRegistry().Reset()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(108010600)).Build()
	key := FieldKey{Tenant: te, Field: f}

	getRegistry().Add(key, NewModel(f, getRegistry().NextId(), 1104100, 0, 0, 0))
	first := getRegistry().GetAll(key)
	getRegistry().Add(key, NewModel(f, getRegistry().NextId(), 1104101, 0, 0, 0))

	if len(first) != 1 {
		t.Errorf("Expected the first snapshot to remain length 1, got %d", len(first))
	}
}

func TestRegistryReset(t *testing.T) {
	getRegistry().Reset()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(108010600)).Build()
	key := FieldKey{Tenant: te, Field: f}

	getRegistry().Add(key, NewModel(f, getRegistry().NextId(), 1104100, 0, 0, 0))
	getRegistry().Reset()

	all := getRegistry().GetAll(key)
	if len(all) != 0 {
		t.Errorf("Expected registry to be empty after Reset, got %d entries", len(all))
	}
}

func TestRegistryNextIdIsUnique(t *testing.T) {
	getRegistry().Reset()
	first := getRegistry().NextId()
	second := getRegistry().NextId()
	if first == second {
		t.Errorf("Expected NextId to return distinct values, got %d and %d", first, second)
	}
}
