package guild

import (
	"errors"
	"testing"

	"atlas-guilds/guild/character"
	"atlas-guilds/guild/member"
	"atlas-guilds/kafka/message"
	guild2 "atlas-guilds/kafka/message/guild"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	"github.com/google/uuid"
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

// TestChangeEmblemAndEmit_CommitEnqueuesOutboxRow guards the Pattern A
// migration of a previously-bare (unwrapped) write: ChangeEmblemAndEmit now
// wraps its DB write and enqueue in the same database.ExecuteTransaction,
// using outbox.EmitProvider bound to that tx instead of the direct producer.
// A successful call must enqueue exactly one outbox row on the guild status
// topic (no direct-path publish is exercised or required here).
func TestChangeEmblemAndEmit_CommitEnqueuesOutboxRow(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := outboxTestDb(t)

	e := Entity{
		TenantId: ten.Id(),
		WorldId:  0,
		Name:     "EmblemGuild",
		LeaderId: 100,
		Capacity: 30,
	}
	require.NoError(t, db.Create(&e).Error)

	p := NewProcessor(l, ctx, db)
	err := p.ChangeEmblemAndEmit(e.Id, 100, 1, 2, 3, 4, uuid.New())
	require.NoError(t, err)

	require.Equal(t, int64(1), outboxRowCount(t, db))
}

// TestUpdateMemberOnlineAndEmit_CommitEnqueuesOutboxRow guards the nested-tx
// shape: UpdateMemberOnline already wraps its own database.ExecuteTransaction
// internally; UpdateMemberOnlineAndEmit now wraps an OUTER transaction around
// it (re-entrant per the recipe) so the enqueue shares the same tx as the
// member status write.
func TestUpdateMemberOnlineAndEmit_CommitEnqueuesOutboxRow(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := outboxTestDb(t)

	e := Entity{
		TenantId: ten.Id(),
		WorldId:  0,
		Name:     "OnlineGuild",
		LeaderId: 100,
		Capacity: 30,
	}
	require.NoError(t, db.Create(&e).Error)

	m := member.Entity{
		TenantId:    ten.Id(),
		GuildId:     e.Id,
		CharacterId: 100,
		Name:        "Leader",
		Level:       50,
	}
	require.NoError(t, db.Create(&m).Error)

	c := character.Entity{
		TenantId:    ten.Id(),
		CharacterId: 100,
		GuildId:     e.Id,
	}
	require.NoError(t, db.Create(&c).Error)

	p := NewProcessor(l, ctx, db)
	err := p.UpdateMemberOnlineAndEmit(100, true, uuid.New())
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
			return buf.Put(guild2.EnvStatusEventTopic, fixedGuildMessageProvider())
		})
		require.NoError(t, emitErr)
		return boom
	})
	require.ErrorIs(t, err, boom)
	require.Zero(t, outboxRowCount(t, db))
}

func fixedGuildMessageProvider() model.Provider[[]kafka.Message] {
	return model.FixedProvider([]kafka.Message{{Key: []byte("k"), Value: []byte(`{"e":1}`)}})
}
