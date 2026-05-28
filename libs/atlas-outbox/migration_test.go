package outbox_test

import (
	"testing"

	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMigration_CreatesTable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, outbox.Migration(db))

	require.True(t, db.Migrator().HasTable(&outbox.Entity{}))
	require.True(t, db.Migrator().HasColumn(&outbox.Entity{}, "topic"))
	require.True(t, db.Migrator().HasColumn(&outbox.Entity{}, "message_key"))
	require.True(t, db.Migrator().HasColumn(&outbox.Entity{}, "message_value"))
	require.True(t, db.Migrator().HasColumn(&outbox.Entity{}, "headers"))
	require.True(t, db.Migrator().HasColumn(&outbox.Entity{}, "enqueued_at"))
	require.True(t, db.Migrator().HasColumn(&outbox.Entity{}, "sent_at"))
	require.True(t, db.Migrator().HasColumn(&outbox.Entity{}, "attempts"))
	require.True(t, db.Migrator().HasColumn(&outbox.Entity{}, "last_error"))
}
