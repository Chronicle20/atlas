package family

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
)

// failNthWriteTo fails the nth (and every later) create/update statement
// against the named table. AddJunior saves senior then junior to the same
// table, so verb-scoped databasetest.FailWritesOn cannot isolate the second
// write — this counting callback can.
func failNthWriteTo(t *testing.T, db *gorm.DB, table string, n int) {
	t.Helper()
	count := 0
	fail := func(d *gorm.DB) {
		if d.Statement != nil && d.Statement.Table == table {
			count++
			if count >= n {
				_ = d.AddError(fmt.Errorf("test: injected failure on write %d to %q", count, table))
			}
		}
	}
	require.NoError(t, db.Callback().Create().Before("gorm:create").Register("test:fail_nth_create", fail))
	require.NoError(t, db.Callback().Update().Before("gorm:update").Register("test:fail_nth_update", fail))
}

// AddJunior updates the senior's junior list and the junior's senior link as
// two writes (class B). Failing the second must roll back the first.
func TestAddJunior_RollsBackSeniorSaveWhenJuniorSaveFails(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tid := uuid.New()
	ctx := databasetest.TenantContext(tid)
	require.NoError(t, db.Create(&Entity{CharacterId: 1001, TenantId: tid, Level: 100, World: 0}).Error)
	require.NoError(t, db.Create(&Entity{CharacterId: 1002, TenantId: tid, Level: 100, World: 0}).Error)

	failNthWriteTo(t, db, "family_members", 2)

	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db).(*ProcessorImpl)
	_, err := p.AddJunior(nil)(world.Id(0), 1001, 100, 1002, 100)()
	require.Error(t, err)

	var senior, junior Entity
	require.NoError(t, db.Where("character_id = ?", 1001).First(&senior).Error)
	require.NoError(t, db.Where("character_id = ?", 1002).First(&junior).Error)
	require.Empty(t, senior.JuniorIds, "senior's junior-list update must roll back")
	require.Nil(t, junior.SeniorId, "junior must remain unlinked")
}
