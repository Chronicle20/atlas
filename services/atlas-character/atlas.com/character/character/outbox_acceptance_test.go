package character_test

import (
	"atlas-character/kafka/message"
	"context"
	"errors"
	"testing"

	character2 "atlas-character/kafka/message/character"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// Rollback in a migrated flow should leave zero outbox rows.
//
// This uses gorm's own db.Transaction(fn) primitive directly, NOT
// database.ExecuteTransaction. Confirmed empirically (not assumed) that
// database.ExecuteTransaction's OWN routing is inert against current library
// code: gorm.Open() populates db.Statement.ConnPool on the root *gorm.DB
// immediately, before any WithContext/query/transaction call, so
// isTransaction() (db.Statement != nil && db.Statement.ConnPool != nil,
// libs/atlas-database/transaction.go:16-18) is true for EVERY *gorm.DB —
// including a freshly-opened in-memory sqlite handle — not only
// request-scoped Postgres handles as bug_execute_transaction_noop.md
// describes. That means database.ExecuteTransaction never calls
// db.Transaction(fn) itself; fixing it is task-119's TxCommitter work (see
// docs/tasks/task-114-outbox-adoption/inventory.md), and this test file's
// scope (character acceptance tests) is not the place to fix
// libs/atlas-database.
//
// gorm's db.Transaction(fn) does genuinely BEGIN/COMMIT/ROLLBACK today, so
// using it here exercises exactly the thing Tasks 7-9 depend on: enqueuing
// an outbox row inside a real transaction, then having that row vanish when
// the transaction rolls back. This keeps the test a real regression check
// now instead of a permanently-skipped stub, and it still doubles as the
// exact case that becomes redundant (but harmless) once task-119 makes
// database.ExecuteTransaction itself route to a real transaction.
func TestOutbox_RollbackDiscardsEnqueuedEvents(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := outboxTestDb(t)

	boom := errors.New("boom")
	err := db.Transaction(func(tx *gorm.DB) error {
		err := message.Emit(outbox.EmitProvider(testLogger(), tctx, tx))(func(buf *message.Buffer) error {
			return buf.Put(character2.EnvEventTopicCharacterStatus, fixedMessageProvider())
		})
		require.NoError(t, err) // enqueue itself succeeded...
		return boom             // ...then the domain flow fails
	})
	require.ErrorIs(t, err, boom)
	require.Zero(t, outboxRowCount(t, db))
}

// Commit enqueues exactly one outbox row per emitted message; no
// drainer/publish is exercised here, only the enqueue count.
func TestOutbox_CommitYieldsExactlyEnqueuedEvents(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := outboxTestDb(t)

	require.NoError(t, database.ExecuteTransaction(db, func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(testLogger(), tctx, tx))(func(buf *message.Buffer) error {
			return buf.Put(character2.EnvEventTopicCharacterStatus, fixedMessageProvider())
		})
	}))
	require.Equal(t, int64(1), outboxRowCount(t, db))
}

func fixedMessageProvider() model.Provider[[]kafka.Message] {
	return model.FixedProvider([]kafka.Message{{Key: []byte("k"), Value: []byte(`{"e":1}`)}})
}
