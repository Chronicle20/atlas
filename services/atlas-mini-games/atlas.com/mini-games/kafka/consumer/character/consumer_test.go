package character

import (
	"atlas-mini-games/game"
	characterKafka "atlas-mini-games/kafka/message/character"
	"context"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// channelChangedHarness seeds a not-in-progress owner+visitor room for a
// fresh tenant so registry state is isolated per test. The rooms are idle
// (not InProgress), so the teardown path never touches the db — nil is safe.
func channelChangedHarness(t *testing.T, owner, visitor uint32) (context.Context, tenant.Model, field.Model) {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)
	f := field.NewBuilder(1, 1, 100000).Build()
	r := game.NewBuilder(game.RoomTypeOmok, owner, f).
		SetGameType("OMOK").
		SetVisitorId(visitor).SetLastVisitorId(visitor).
		Build()
	require.NoError(t, game.GetRegistry().Create(ten, r))
	return ctx, ten, f
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
	ctx, ten, _ := channelChangedHarness(t, owner, visitor)

	handleStatusEventChannelChanged(nil)(testLogger(), ctx, channelChangedEvent(visitor))

	r, ok := game.GetRegistry().Get(ten, owner)
	require.True(t, ok, "room stays open when the visitor is torn down")
	assert.Equal(t, uint32(0), r.VisitorId(), "visitor slot cleared on channel change")
}

func TestHandleChannelChanged_OwnerClosesRoom(t *testing.T) {
	owner := uint32(21101)
	visitor := uint32(21102)
	ctx, ten, _ := channelChangedHarness(t, owner, visitor)

	handleStatusEventChannelChanged(nil)(testLogger(), ctx, channelChangedEvent(owner))

	_, ok := game.GetRegistry().Get(ten, owner)
	assert.False(t, ok, "owner's channel change closes the room")
}

func TestHandleChannelChanged_WrongTypeIsNoOp(t *testing.T) {
	owner := uint32(21201)
	visitor := uint32(21202)
	ctx, ten, _ := channelChangedHarness(t, owner, visitor)

	e := channelChangedEvent(visitor)
	e.Type = characterKafka.EventCharacterStatusTypeLogin
	handleStatusEventChannelChanged(nil)(testLogger(), ctx, e)

	r, ok := game.GetRegistry().Get(ten, owner)
	require.True(t, ok)
	assert.Equal(t, visitor, r.VisitorId(), "non-CHANNEL_CHANGED event leaves the room untouched")
}
