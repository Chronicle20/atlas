package recipe

import (
	"atlas-npc-conversations/test"
	"context"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)

func TestMigrateTable_CreatesRecipesTable(t *testing.T) {
	db := test.SetupTestDB(t, MigrateTable)
	defer test.CleanupTestDB(t, db)

	if !db.Migrator().HasTable(&Entity{}) {
		t.Fatalf("recipes table was not created by MigrateTable")
	}
	for _, col := range []string{"id", "tenant_id", "conversation_id", "npc_id", "state_id", "item_id", "materials", "meso_cost", "stimulator_id", "stimulator_fail_chance", "created_at", "updated_at"} {
		if !db.Migrator().HasColumn(&Entity{}, col) {
			t.Errorf("column %q missing from recipes table", col)
		}
	}
}

func TestNewBuilder_BuildsExpectedModel(t *testing.T) {
	convId := uuid.New()
	tenantId := uuid.New()
	m, err := NewBuilder().
		SetTenantId(tenantId).
		SetConversationId(convId).
		SetNpcId(2040020).
		SetStateId("craftWarrior0").
		SetItemId(1082007).
		SetMaterials([]Material{{ItemId: 4011000, Quantity: 3}, {ItemId: 4011001, Quantity: 2}}).
		SetMesoCost(18000).
		SetStimulatorId(0).
		SetStimulatorFailChance(0).
		Build()
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	if m.NpcId() != 2040020 {
		t.Errorf("NpcId mismatch: got %d", m.NpcId())
	}
	if m.StateId() != "craftWarrior0" {
		t.Errorf("StateId mismatch: got %q", m.StateId())
	}
	if m.ItemId() != 1082007 {
		t.Errorf("ItemId mismatch: got %d", m.ItemId())
	}
	if len(m.Materials()) != 2 || m.Materials()[1].Quantity != 2 {
		t.Errorf("Materials mismatch: %+v", m.Materials())
	}
}

func TestBuild_DoesNotMutateBuilderId(t *testing.T) {
	b := NewBuilder().
		SetTenantId(uuid.New()).
		SetConversationId(uuid.New()).
		SetStateId("craftWarrior0")
	if _, err := b.Build(); err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	// Change stateId; re-Build should produce a different id, not the cached one.
	b.SetStateId("craftWarrior1")
	m2, err := b.Build()
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	expected := ComputeRecipeId(b.m.tenantId, b.m.conversationId, "craftWarrior1")
	if m2.Id() != expected {
		t.Errorf("Build did not recompute id after stateId change: got %s, want %s", m2.Id(), expected)
	}
}

func TestComputeRecipeId_Deterministic(t *testing.T) {
	tenantId := uuid.New()
	convId := uuid.New()

	a := ComputeRecipeId(tenantId, convId, "craftWarrior0")
	b := ComputeRecipeId(tenantId, convId, "craftWarrior0")
	if a != b {
		t.Errorf("ComputeRecipeId not deterministic: %s vs %s", a, b)
	}

	c := ComputeRecipeId(tenantId, convId, "craftWarrior1")
	if a == c {
		t.Errorf("ComputeRecipeId collided across stateIds")
	}
}

func newRecipe(t *testing.T, tenantId, convId uuid.UUID, npcId uint32, stateId string, itemId uint32) Model {
	t.Helper()
	m, err := NewBuilder().
		SetTenantId(tenantId).
		SetConversationId(convId).
		SetNpcId(npcId).
		SetStateId(stateId).
		SetItemId(itemId).
		SetMaterials([]Material{{ItemId: 4011000, Quantity: 3}}).
		SetMesoCost(18000).
		Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	return m
}

func TestCreateRecipe_PersistsRow(t *testing.T) {
	db := test.SetupTestDB(t, MigrateTable)
	defer test.CleanupTestDB(t, db)

	tenantId := uuid.New()
	convId := uuid.New()
	te, _ := tenant.Create(tenantId, "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), te)

	m := newRecipe(t, tenantId, convId, 2040020, "craftWarrior0", 1082007)

	saved, err := createRecipe(db.WithContext(ctx))(tenantId)(m)
	if err != nil {
		t.Fatalf("createRecipe: %v", err)
	}
	if saved.Id() != m.Id() {
		t.Errorf("id mismatch: got %s, want %s", saved.Id(), m.Id())
	}

	var entity Entity
	if err := db.WithContext(ctx).Where("id = ?", m.Id()).First(&entity).Error; err != nil {
		t.Fatalf("read back: %v", err)
	}
	if entity.ItemID != 1082007 || entity.NpcID != 2040020 {
		t.Errorf("unexpected row: %+v", entity)
	}
}

func TestDeleteRecipesByConversation_RemovesOnlyMatchingRows(t *testing.T) {
	db := test.SetupTestDB(t, MigrateTable)
	defer test.CleanupTestDB(t, db)

	tenantId := uuid.New()
	te, _ := tenant.Create(tenantId, "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), te)

	convA := uuid.New()
	convB := uuid.New()

	for _, m := range []Model{
		newRecipe(t, tenantId, convA, 1000, "s1", 1),
		newRecipe(t, tenantId, convA, 1000, "s2", 2),
		newRecipe(t, tenantId, convB, 1001, "s1", 3),
	} {
		if _, err := createRecipe(db.WithContext(ctx))(tenantId)(m); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	count, err := deleteRecipesByConversation(db.WithContext(ctx))(convA)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 deleted, got %d", count)
	}

	var remaining []Entity
	db.WithContext(ctx).Find(&remaining)
	if len(remaining) != 1 || remaining[0].ConversationID != convB {
		t.Errorf("unexpected remaining rows: %+v", remaining)
	}
}

func TestDeleteAllRecipes_TenantScoped(t *testing.T) {
	db := test.SetupTestDB(t, MigrateTable)
	defer test.CleanupTestDB(t, db)

	tenantA := uuid.New()
	tenantB := uuid.New()
	teA, _ := tenant.Create(tenantA, "GMS", 83, 1)
	teB, _ := tenant.Create(tenantB, "GMS", 83, 1)
	ctxA := tenant.WithContext(context.Background(), teA)
	ctxB := tenant.WithContext(context.Background(), teB)

	if _, err := createRecipe(db.WithContext(ctxA))(tenantA)(newRecipe(t, tenantA, uuid.New(), 1, "s", 100)); err != nil {
		t.Fatalf("seed A: %v", err)
	}
	if _, err := createRecipe(db.WithContext(ctxB))(tenantB)(newRecipe(t, tenantB, uuid.New(), 2, "s", 200)); err != nil {
		t.Fatalf("seed B: %v", err)
	}

	count, err := deleteAllRecipes(db.WithContext(ctxA))
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 deleted for tenant A, got %d", count)
	}

	var remaining []Entity
	db.WithContext(ctxB).Find(&remaining)
	if len(remaining) != 1 || remaining[0].ItemID != 200 {
		t.Errorf("tenant B rows unexpectedly affected: %+v", remaining)
	}
}
