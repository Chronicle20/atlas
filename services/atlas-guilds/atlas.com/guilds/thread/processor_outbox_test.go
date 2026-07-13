package thread

import (
	"errors"
	"testing"

	"atlas-guilds/kafka/message"
	thread2 "atlas-guilds/kafka/message/thread"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func outboxTestDb(t *testing.T) *gorm.DB {
	db := setupTestDatabase(t)
	require.NoError(t, outbox.Migration(db))
	return db
}

func outboxRowCount(t *testing.T, db *gorm.DB) int64 {
	var n int64
	require.NoError(t, db.Model(&outbox.Entity{}).Count(&n).Error)
	return n
}

// TestCreateAndEmit_CommitEnqueuesOutboxRow guards the Pattern A migration
// of a previously-bare (unwrapped) write: CreateAndEmit now wraps its DB
// write and enqueue in the same database.ExecuteTransaction, using
// outbox.EmitProvider bound to that tx instead of the direct producer.
func TestCreateAndEmit_CommitEnqueuesOutboxRow(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := outboxTestDb(t)

	p := NewProcessor(l, ctx, db)
	_, err := p.CreateAndEmit(0, 1, 100, "Title", "Message", 0, false)
	require.NoError(t, err)

	require.Equal(t, int64(1), outboxRowCount(t, db))
}

// TestReplyAndEmit_CommitEnqueuesOutboxRow guards the nested-tx shape: Reply
// already wraps its own database.ExecuteTransaction internally;
// ReplyAndEmit now wraps an OUTER transaction around it (re-entrant per the
// recipe) so the enqueue shares the same tx as the reply write.
func TestReplyAndEmit_CommitEnqueuesOutboxRow(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := outboxTestDb(t)

	e := Entity{
		TenantId: ten.Id(),
		GuildId:  1,
		PosterId: 100,
		Title:    "Thread",
		Message:  "Message",
	}
	require.NoError(t, db.Create(&e).Error)

	p := NewProcessor(l, ctx, db)
	_, err := p.ReplyAndEmit(0, 1, e.Id, 200, "A reply")
	require.NoError(t, err)

	require.Equal(t, int64(1), outboxRowCount(t, db))
}

// TestOutbox_RollbackDiscardsEnqueuedEvents exercises the outbox seam
// directly (not through database.ExecuteTransaction, which is currently a
// verified no-op — see bug_execute_transaction_noop.md / task-119): gorm's
// db.Transaction(fn) genuinely BEGINs/ROLLBACKs today, so enqueuing via
// outbox.EmitProvider inside one and then failing the flow must leave zero
// outbox rows. Same pattern used by
// atlas-character/character/outbox_acceptance_test.go.
func TestOutbox_RollbackDiscardsEnqueuedEvents(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := outboxTestDb(t)

	boom := errors.New("boom")
	err := db.Transaction(func(tx *gorm.DB) error {
		emitErr := message.Emit(outbox.EmitProvider(l, ctx, tx))(func(buf *message.Buffer) error {
			return buf.Put(thread2.EnvStatusEventTopic, fixedThreadMessageProvider())
		})
		require.NoError(t, emitErr)
		return boom
	})
	require.ErrorIs(t, err, boom)
	require.Zero(t, outboxRowCount(t, db))
}

func fixedThreadMessageProvider() model.Provider[[]kafka.Message] {
	return model.FixedProvider([]kafka.Message{{Key: []byte("k"), Value: []byte(`{"e":1}`)}})
}
