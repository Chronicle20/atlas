package messenger

import (
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"testing"
)

func TestSunnyDayCreate(t *testing.T) {
	r := GetRegistry()

	characterId := uint32(1)
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)

	p := r.Create(ten, characterId)
	if p.id != StartMessengerId {
		t.Fatal("Failed to generate correct initial messenger id.")
	}

	if len(p.members) != 1 {
		t.Fatal("Failed to generate correct initial members.")
	}

	if p.members[0].Id() != characterId {
		t.Fatal("Failed to generate correct initial members.")
	}
}

func TestMultiMessengerCreate(t *testing.T) {
	r := GetRegistry()

	character1Id := uint32(1)
	character2Id := uint32(2)
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)

	p := r.Create(ten, character1Id)
	if p.id != StartMessengerId {
		t.Fatal("Failed to generate correct initial messenger id.")
	}

	if len(p.members) != 1 {
		t.Fatal("Failed to generate correct initial members.")
	}

	if p.members[0].Id() != character1Id {
		t.Fatal("Failed to generate correct initial members.")
	}

	p = r.Create(ten, character2Id)
	if p.id != StartMessengerId+1 {
		t.Fatal("Failed to generate correct initial messenger id.")
	}

	if len(p.members) != 1 {
		t.Fatal("Failed to generate correct initial members.")
	}

	if p.members[0].Id() != character2Id {
		t.Fatal("Failed to generate correct initial members.")
	}
}

func TestMultiTenantCreate(t *testing.T) {
	r := GetRegistry()

	character1Id := uint32(1)
	character2Id := uint32(2)
	ten1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ten2, _ := tenant.Create(uuid.New(), "GMS", 87, 1)

	p := r.Create(ten1, character1Id)
	if p.id != StartMessengerId {
		t.Fatal("Failed to generate correct initial messenger id.")
	}

	if len(p.members) != 1 {
		t.Fatal("Failed to generate correct initial members.")
	}

	if p.members[0].Id() != character1Id {
		t.Fatal("Failed to generate correct initial members.")
	}

	p = r.Create(ten2, character2Id)
	if p.id != StartMessengerId {
		t.Fatal("Failed to generate correct initial messenger id.")
	}

	if len(p.members) != 1 {
		t.Fatal("Failed to generate correct initial members.")
	}

	if p.members[0].Id() != character2Id {
		t.Fatal("Failed to generate correct initial members.")
	}
}
