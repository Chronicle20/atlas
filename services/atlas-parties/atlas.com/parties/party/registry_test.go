package party

import (
	"context"
	"testing"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

func setupTestRegistry(t *testing.T) {
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(rc)
}

func createTestCtx(ten tenant.Model) context.Context {
	return tenant.WithContext(context.Background(), ten)
}

func TestSunnyDayCreate(t *testing.T) {
	setupTestRegistry(t)
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := createTestCtx(ten)

	leaderId := uint32(1)
	p := GetRegistry().Create(ctx, leaderId)
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
	setupTestRegistry(t)
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := createTestCtx(ten)

	leader1Id := uint32(1)
	leader2Id := uint32(2)

	p := GetRegistry().Create(ctx, leader1Id)
	if p.id != StartPartyId {
		t.Fatal("Failed to generate correct initial party id.")
	}
	if len(p.members) != 1 || p.members[0] != leader1Id {
		t.Fatal("Failed to generate correct initial members.")
	}

	p = GetRegistry().Create(ctx, leader2Id)
	if p.id != StartPartyId+1 {
		t.Fatal("Failed to generate correct initial party id.")
	}
	if len(p.members) != 1 || p.members[0] != leader2Id {
		t.Fatal("Failed to generate correct initial members.")
	}
}

func TestMultiTenantCreate(t *testing.T) {
	setupTestRegistry(t)
	ten1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ten2, _ := tenant.Create(uuid.New(), "GMS", 87, 1)
	ctx1 := createTestCtx(ten1)
	ctx2 := createTestCtx(ten2)

	leader1Id := uint32(1)
	leader2Id := uint32(2)

	p := GetRegistry().Create(ctx1, leader1Id)
	if p.id != StartPartyId {
		t.Fatal("Failed to generate correct initial party id.")
	}
	if len(p.members) != 1 || p.members[0] != leader1Id {
		t.Fatal("Failed to generate correct initial members.")
	}

	p = GetRegistry().Create(ctx2, leader2Id)
	if p.id != StartPartyId {
		t.Fatal("Failed to generate correct initial party id.")
	}
	if len(p.members) != 1 || p.members[0] != leader2Id {
		t.Fatal("Failed to generate correct initial members.")
	}
}
