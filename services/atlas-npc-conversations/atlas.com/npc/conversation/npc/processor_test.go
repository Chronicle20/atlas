package npc

import (
	"atlas-npc-conversations/conversation"
	"atlas-npc-conversations/conversation/recipe"
	"atlas-npc-conversations/test"
	"context"
	"testing"
	"time"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

func countTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	te, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}
	return te
}

func insertCountRow(t *testing.T, p Processor, npcId uint32) {
	t.Helper()
	m := createTestModel(t, npcId)
	if _, err := p.Create(m); err != nil {
		t.Fatalf("Create npc conversation %d: %v", npcId, err)
	}
}

func TestProcessorImpl_Count_Empty(t *testing.T) {
	l, _ := logtest.NewNullLogger()
	te := countTestTenant(t)
	ctx := tenant.WithContext(context.Background(), te)
	db := test.SetupTestDB(t, MigrateTable)
	defer test.CleanupTestDB(t, db)

	p := NewProcessor(l, ctx, db)
	count, updated, err := p.Count()
	if err != nil {
		t.Fatalf("Count() returned error: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count 0, got %d", count)
	}
	if updated != nil {
		t.Errorf("Expected nil updatedAt, got %v", updated)
	}
}

func TestProcessorImpl_Count_Populated(t *testing.T) {
	l, _ := logtest.NewNullLogger()
	te := countTestTenant(t)
	ctx := tenant.WithContext(context.Background(), te)
	db := test.SetupTestDB(t, MigrateTable, recipe.MigrateTable)
	defer test.CleanupTestDB(t, db)

	p := NewProcessor(l, ctx, db)
	insertCountRow(t, p, 1000)
	insertCountRow(t, p, 1001)

	count, updated, err := p.Count()
	if err != nil {
		t.Fatalf("Count() returned error: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected count 2, got %d", count)
	}
	if updated == nil {
		t.Fatalf("updatedAt is nil; expected non-nil")
	}
	if time.Since(*updated) > 5*time.Second {
		t.Errorf("updatedAt too old: %v", *updated)
	}
}

func TestProcessorImpl_Count_TenantIsolation(t *testing.T) {
	l, _ := logtest.NewNullLogger()
	te1 := countTestTenant(t)
	te2 := countTestTenant(t)
	ctx1 := tenant.WithContext(context.Background(), te1)
	ctx2 := tenant.WithContext(context.Background(), te2)
	db := test.SetupTestDB(t, MigrateTable, recipe.MigrateTable)
	defer test.CleanupTestDB(t, db)

	p1 := NewProcessor(l, ctx1, db)
	p2 := NewProcessor(l, ctx2, db)

	insertCountRow(t, p1, 2000)
	insertCountRow(t, p1, 2001)
	insertCountRow(t, p2, 3000)
	insertCountRow(t, p2, 3001)
	insertCountRow(t, p2, 3002)

	count, _, err := p1.Count()
	if err != nil {
		t.Fatalf("Count() returned error: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected count 2 for tenant 1, got %d", count)
	}
}

func craftStateForNpc(t *testing.T, stateId, itemId string) conversation.StateModel {
	t.Helper()
	ca, err := conversation.NewCraftActionBuilder().
		SetItemId(itemId).
		SetMaterials([]uint32{4011000}).
		SetQuantities([]uint32{3}).
		SetMesoCost(18000).
		SetSuccessState("end").
		SetFailureState("end").
		SetMissingMaterialsState("end").
		Build()
	if err != nil {
		t.Fatalf("ca build: %v", err)
	}
	state, err := conversation.NewStateBuilder().SetId(stateId).SetCraftAction(ca).Build()
	if err != nil {
		t.Fatalf("state build: %v", err)
	}
	return state
}

func craftConversationModel(t *testing.T, npcId uint32, craftStates ...conversation.StateModel) Model {
	t.Helper()
	base := createTestModel(t, npcId)

	builder := NewBuilder().SetNpcId(npcId).SetStartState(base.StartState())
	for _, s := range base.States() {
		builder.AddState(s)
	}
	for _, s := range craftStates {
		builder.AddState(s)
	}
	m, err := builder.Build()
	if err != nil {
		t.Fatalf("conversation build: %v", err)
	}
	return m
}

func TestProcessor_Create_PopulatesRecipeIndex(t *testing.T) {
	l, _ := logtest.NewNullLogger()
	te := countTestTenant(t)
	ctx := tenant.WithContext(context.Background(), te)
	db := test.SetupTestDB(t, MigrateTable, recipe.MigrateTable)
	defer test.CleanupTestDB(t, db)

	p := NewProcessor(l, ctx, db)
	m := craftConversationModel(t, 2040020,
		craftStateForNpc(t, "craft0", "1082007"),
		craftStateForNpc(t, "craft1", "1082008"),
	)

	if _, err := p.Create(m); err != nil {
		t.Fatalf("create: %v", err)
	}

	rp := recipe.NewProcessor(l, ctx, db)
	rows, err := rp.ByNpcIdProvider(2040020)()
	if err != nil {
		t.Fatalf("recipe lookup: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("expected 2 recipes, got %d", len(rows))
	}
}

func TestProcessor_Update_RewritesRecipeIndex(t *testing.T) {
	l, _ := logtest.NewNullLogger()
	te := countTestTenant(t)
	ctx := tenant.WithContext(context.Background(), te)
	db := test.SetupTestDB(t, MigrateTable, recipe.MigrateTable)
	defer test.CleanupTestDB(t, db)

	p := NewProcessor(l, ctx, db)
	created, err := p.Create(craftConversationModel(t, 2040020, craftStateForNpc(t, "old", "1")))
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	updated := craftConversationModel(t, 2040020, craftStateForNpc(t, "new", "2"))
	if _, err := p.Update(created.Id(), updated); err != nil {
		t.Fatalf("update: %v", err)
	}

	rp := recipe.NewProcessor(l, ctx, db)
	rows, _ := rp.ByNpcIdProvider(2040020)()
	if len(rows) != 1 || rows[0].StateId() != "new" || rows[0].ItemId() != 2 {
		t.Errorf("after update, expected one row stateId=new itemId=2, got %+v", rows)
	}
}
