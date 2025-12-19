package party

import (
	"testing"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
)

func TestSunnyDayCreate(t *testing.T) {
	r := GetRegistry()

	leaderId := uint32(1)
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)

	p := r.Create(ten, leaderId)
	if p.id != StartPartyId {
		t.Fatal("Failed to generate correct initial party id.")
	}

	if len(p.members) != 1 {
		t.Fatal("Failed to generate correct initial members.")
	}

	if p.members[0] != leaderId {
		t.Fatal("Failed to generate correct initial members.")
	}
}

func TestMultiPartyCreate(t *testing.T) {
	r := GetRegistry()

	leader1Id := uint32(1)
	leader2Id := uint32(2)
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)

	p := r.Create(ten, leader1Id)
	if p.id != StartPartyId {
		t.Fatal("Failed to generate correct initial party id.")
	}

	if len(p.members) != 1 {
		t.Fatal("Failed to generate correct initial members.")
	}

	if p.members[0] != leader1Id {
		t.Fatal("Failed to generate correct initial members.")
	}

	p = r.Create(ten, leader2Id)
	if p.id != StartPartyId+1 {
		t.Fatal("Failed to generate correct initial party id.")
	}

	if len(p.members) != 1 {
		t.Fatal("Failed to generate correct initial members.")
	}

	if p.members[0] != leader2Id {
		t.Fatal("Failed to generate correct initial members.")
	}
}

func TestMultiTenantCreate(t *testing.T) {
	r := GetRegistry()

	leader1Id := uint32(1)
	leader2Id := uint32(2)
	ten1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ten2, _ := tenant.Create(uuid.New(), "GMS", 87, 1)

	p := r.Create(ten1, leader1Id)
	if p.id != StartPartyId {
		t.Fatal("Failed to generate correct initial party id.")
	}

	if len(p.members) != 1 {
		t.Fatal("Failed to generate correct initial members.")
	}

	if p.members[0] != leader1Id {
		t.Fatal("Failed to generate correct initial members.")
	}

	p = r.Create(ten2, leader2Id)
	if p.id != StartPartyId {
		t.Fatal("Failed to generate correct initial party id.")
	}

	if len(p.members) != 1 {
		t.Fatal("Failed to generate correct initial members.")
	}

	if p.members[0] != leader2Id {
		t.Fatal("Failed to generate correct initial members.")
	}
}
