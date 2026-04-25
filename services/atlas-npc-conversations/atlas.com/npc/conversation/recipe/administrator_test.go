package recipe

import (
	"atlas-npc-conversations/test"
	"testing"

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
