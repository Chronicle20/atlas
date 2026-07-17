package character

import (
	"atlas-mini-games/game"
	characterKafka "atlas-mini-games/kafka/message/character"
	"context"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/miniroom"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB builds a sqlite-in-memory db with the game_records table (created
// directly via SQL, since the uuid PK can't run through AutoMigrate on sqlite)
// and the transactional-outbox table. The teardown path routes its LEFT/BALLOON
// events through the outbox (task-114), so every command — even one that writes
// no record — opens a transaction and needs a live db handle.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	database.RegisterTenantCallbacks(l, db)

	require.NoError(t, db.Exec(`
		CREATE TABLE game_records (
			tenant_id TEXT NOT NULL,
			id TEXT PRIMARY KEY,
			character_id INTEGER NOT NULL,
			game_type TEXT NOT NULL,
			wins INTEGER NOT NULL DEFAULT 0,
			ties INTEGER NOT NULL DEFAULT 0,
			losses INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME,
			updated_at DATETIME,
			UNIQUE(tenant_id, character_id, game_type)
		)
	`).Error)
	require.NoError(t, outbox.Migration(db))
	return db
}

// channelChangedHarness seeds a not-in-progress owner+visitor room for a
// fresh tenant so registry state is isolated per test, and a sqlite db the
// teardown command's outbox transaction runs against.
func channelChangedHarness(t *testing.T, owner, visitor uint32) (context.Context, tenant.Model, field.Model, *gorm.DB) {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)
	f := field.NewBuilder(1, 1, 100000).Build()
	r := game.NewBuilder(miniroom.Omok, owner, f).
		SetGameType("OMOK").
		SetVisitorId(visitor).SetLastVisitorId(visitor).
		Build()
	require.NoError(t, game.GetRegistry().Create(ten, r))
	return ctx, ten, f, setupTestDB(t)
}

func channelChangedEvent(characterId uint32) characterKafka.StatusEvent[characterKafka.ChangeChannelEventLoginBody] {
	return characterKafka.StatusEvent[characterKafka.ChangeChannelEventLoginBody]{
		TransactionId: uuid.New(),
		WorldId:       1,
		CharacterId:   characterId,
		Type:          characterKafka.EventCharacterStatusTypeChannelChanged,
		Body: characterKafka.ChangeChannelEventLoginBody{
			ChannelId:    2,
			OldChannelId: 1,
			MapId:        100000,
		},
	}
}

func testLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	return l
}

func TestHandleChannelChanged_VisitorTornDown(t *testing.T) {
	owner := uint32(21001)
	visitor := uint32(21002)
	ctx, ten, _, db := channelChangedHarness(t, owner, visitor)

	handleStatusEventChannelChanged(db)(testLogger(), ctx, channelChangedEvent(visitor))

	r, ok := game.GetRegistry().Get(ten, owner)
	require.True(t, ok, "room stays open when the visitor is torn down")
	assert.Equal(t, uint32(0), r.VisitorId(), "visitor slot cleared on channel change")
}

func TestHandleChannelChanged_OwnerClosesRoom(t *testing.T) {
	owner := uint32(21101)
	visitor := uint32(21102)
	ctx, ten, _, db := channelChangedHarness(t, owner, visitor)

	handleStatusEventChannelChanged(db)(testLogger(), ctx, channelChangedEvent(owner))

	_, ok := game.GetRegistry().Get(ten, owner)
	assert.False(t, ok, "owner's channel change closes the room")
}

func TestHandleChannelChanged_WrongTypeIsNoOp(t *testing.T) {
	owner := uint32(21201)
	visitor := uint32(21202)
	ctx, ten, _, db := channelChangedHarness(t, owner, visitor)

	e := channelChangedEvent(visitor)
	e.Type = characterKafka.EventCharacterStatusTypeLogin
	handleStatusEventChannelChanged(db)(testLogger(), ctx, e)

	r, ok := game.GetRegistry().Get(ten, owner)
	require.True(t, ok)
	assert.Equal(t, visitor, r.VisitorId(), "non-CHANNEL_CHANGED event leaves the room untouched")
}
