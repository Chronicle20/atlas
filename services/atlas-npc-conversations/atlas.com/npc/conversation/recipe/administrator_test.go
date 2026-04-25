package recipe

import (
	"atlas-npc-conversations/test"
	"testing"
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
