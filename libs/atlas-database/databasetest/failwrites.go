package databasetest

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// WriteVerb identifies one GORM write pipeline for FailWritesOn.
type WriteVerb string

const (
	WriteCreate WriteVerb = "create"
	WriteUpdate WriteVerb = "update"
	WriteDelete WriteVerb = "delete"
)

// FailWritesOn registers create/update/delete callbacks that fail any write to
// the named table, for rollback testing (fail a flow's later statement, then
// assert its earlier writes rolled back). With no verbs, all three write verbs
// fail. Callbacks apply to every session and transaction derived from db.
// Raw .Exec(...) statements bypass GORM callbacks and are not intercepted.
func FailWritesOn(t *testing.T, db *gorm.DB, table string, verbs ...WriteVerb) {
	t.Helper()
	if len(verbs) == 0 {
		verbs = []WriteVerb{WriteCreate, WriteUpdate, WriteDelete}
	}
	fail := func(d *gorm.DB) {
		if d.Statement != nil && d.Statement.Table == table {
			_ = d.AddError(fmt.Errorf("databasetest: injected failure writing to %q", table))
		}
	}
	for _, v := range verbs {
		name := fmt.Sprintf("databasetest:fail_%s_%s", v, table)
		switch v {
		case WriteCreate:
			require.NoError(t, db.Callback().Create().Before("gorm:create").Register(name, fail))
		case WriteUpdate:
			require.NoError(t, db.Callback().Update().Before("gorm:update").Register(name, fail))
		case WriteDelete:
			require.NoError(t, db.Callback().Delete().Before("gorm:delete").Register(name, fail))
		}
	}
}
