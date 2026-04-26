package recipe

import (
	"atlas-npc-conversations/test"
	"context"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)

func TestGetByItemIdProvider_OrdersByNpcThenStateId(t *testing.T) {
	db := test.SetupTestDB(t, MigrateTable)
	defer test.CleanupTestDB(t, db)

	tenantId := uuid.New()
	te, _ := tenant.Create(tenantId, "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), te)

	convX := uuid.New()
	for _, m := range []Model{
		newRecipe(t, tenantId, convX, 2040020, "craftB", 9999),
		newRecipe(t, tenantId, convX, 2040020, "craftA", 9999),
		newRecipe(t, tenantId, uuid.New(), 1010000, "craftZ", 9999),
		newRecipe(t, tenantId, uuid.New(), 5050000, "craft", 1234),
	} {
		if _, err := createRecipe(db.WithContext(ctx))(tenantId)(m); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	got, err := getByItemIdProvider(9999)(db.WithContext(ctx))()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(got))
	}
	if got[0].NpcID != 1010000 {
		t.Errorf("[0] npcId = %d, want 1010000 (lowest npcId first)", got[0].NpcID)
	}
	if got[1].NpcID != 2040020 || got[1].StateID != "craftA" {
		t.Errorf("[1] = (npc=%d state=%q), want (2040020, craftA)", got[1].NpcID, got[1].StateID)
	}
	if got[2].NpcID != 2040020 || got[2].StateID != "craftB" {
		t.Errorf("[2] = (npc=%d state=%q), want (2040020, craftB)", got[2].NpcID, got[2].StateID)
	}
}

func TestGetByNpcIdProvider_OrdersByStateIdAndScopesByTenant(t *testing.T) {
	db := test.SetupTestDB(t, MigrateTable)
	defer test.CleanupTestDB(t, db)

	tenantA := uuid.New()
	tenantB := uuid.New()
	teA, _ := tenant.Create(tenantA, "GMS", 83, 1)
	teB, _ := tenant.Create(tenantB, "GMS", 83, 1)
	ctxA := tenant.WithContext(context.Background(), teA)
	ctxB := tenant.WithContext(context.Background(), teB)

	for _, m := range []Model{
		newRecipe(t, tenantA, uuid.New(), 2040020, "craftB", 1),
		newRecipe(t, tenantA, uuid.New(), 2040020, "craftA", 2),
	} {
		if _, err := createRecipe(db.WithContext(ctxA))(tenantA)(m); err != nil {
			t.Fatalf("seed A: %v", err)
		}
	}
	if _, err := createRecipe(db.WithContext(ctxB))(tenantB)(newRecipe(t, tenantB, uuid.New(), 2040020, "craftZ", 999)); err != nil {
		t.Fatalf("seed B: %v", err)
	}

	got, err := getByNpcIdProvider(2040020)(db.WithContext(ctxA))()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("tenant A expected 2 rows, got %d", len(got))
	}
	if got[0].StateID != "craftA" || got[1].StateID != "craftB" {
		t.Errorf("ordering: %q, %q", got[0].StateID, got[1].StateID)
	}
}
