package key

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

// Reset is delete-all + loop-create on the keys table (class B). A failure on
// the re-create must roll the delete back, leaving the prior bindings intact.
func TestReset_RollsBackDeleteWhenCreateFails(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tid := uuid.New()
	ctx := databasetest.TenantContext(tid)
	require.NoError(t, db.Create(&entity{TenantId: tid, CharacterId: 1001, Key: 10, Type: 1, Action: 100}).Error)

	databasetest.FailWritesOn(t, db, "keys", databasetest.WriteCreate)

	l, _ := test.NewNullLogger()
	err := NewProcessor(l, ctx, db).Reset(uuid.New(), 1001)
	require.Error(t, err)

	var rows []entity
	require.NoError(t, db.Unscoped().Find(&rows).Error)
	require.Len(t, rows, 1, "pre-existing binding must survive: the delete rolled back")
	require.Equal(t, int32(10), rows[0].Key)
}
