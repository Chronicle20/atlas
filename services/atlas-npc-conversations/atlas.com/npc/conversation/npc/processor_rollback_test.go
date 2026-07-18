package npc

import (
	"atlas-npc-conversations/conversation/recipe"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
)

// DeleteAllForTenant clears recipe rows then conversation rows (class A).
// Failing the conversation delete must restore the cleared recipes.
func TestDeleteAllForTenant_RollsBackRecipeClearWhenConversationDeleteFails(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, MigrateTable, recipe.MigrateTable)
	tid := uuid.New()
	ctx := databasetest.TenantContext(tid)

	convId := uuid.New()
	require.NoError(t, db.Create(&Entity{ID: convId, TenantID: tid, NpcID: 9000000, Data: `{}`}).Error)
	require.NoError(t, db.Create(&recipe.Entity{ID: uuid.New(), TenantID: tid, ConversationID: convId, NpcID: 9000000, StateID: "craft", ItemID: 4000000, Materials: `[]`}).Error)

	databasetest.FailWritesOn(t, db, "conversations", databasetest.WriteDelete)

	l, _ := test.NewNullLogger()
	_, err := NewProcessor(l, ctx, db).DeleteAllForTenant()
	require.Error(t, err)

	var recipes int64
	require.NoError(t, db.Model(&recipe.Entity{}).Count(&recipes).Error)
	require.EqualValues(t, 1, recipes, "recipe clear must roll back with the failed conversation delete")
}
