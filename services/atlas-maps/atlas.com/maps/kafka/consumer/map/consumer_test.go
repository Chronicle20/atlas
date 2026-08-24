package _map

import (
	mapKafka "atlas-maps/kafka/message/map"
	"atlas-maps/map/jukebox"
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func TestHandlePlayJukeboxCommand_StartsWithTheCommandDuration(t *testing.T) {
	l, _ := test.NewNullLogger()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	f := field.NewBuilder(0, 1, 100000000).SetInstance(uuid.Nil).Build()

	cmd := mapKafka.Command[mapKafka.PlayJukeboxCommandBody]{
		TransactionId: uuid.New(),
		WorldId:       0,
		ChannelId:     1,
		MapId:         100000000,
		Instance:      uuid.Nil,
		Type:          mapKafka.CommandTypePlayJukebox,
		Body: mapKafka.PlayJukeboxCommandBody{
			ItemId:     5100000,
			PlayerName: "Chronicle",
			DurationMs: 45000,
		},
	}

	handlePlayJukeboxCommand()(l, ctx, cmd)

	entry, ok := jukebox.NewProcessor(l, ctx).GetActive(f)
	require.True(t, ok)
	require.Equal(t, uint32(5100000), entry.ItemId)
	require.Equal(t, "Chronicle", entry.PlayerName)
	require.True(t, entry.ExpiresAt.After(time.Now().Add(40*time.Second)))
	require.True(t, entry.ExpiresAt.Before(time.Now().Add(50*time.Second)))
}

func TestHandlePlayJukeboxCommand_CapsExcessiveDuration(t *testing.T) {
	l, _ := test.NewNullLogger()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	f := field.NewBuilder(0, 1, 100000000).SetInstance(uuid.Nil).Build()

	cmd := mapKafka.Command[mapKafka.PlayJukeboxCommandBody]{
		TransactionId: uuid.New(),
		WorldId:       0,
		ChannelId:     1,
		MapId:         100000000,
		Instance:      uuid.Nil,
		Type:          mapKafka.CommandTypePlayJukebox,
		Body: mapKafka.PlayJukeboxCommandBody{
			ItemId:     5100000,
			PlayerName: "Chronicle",
			DurationMs: 3600000,
		},
	}

	handlePlayJukeboxCommand()(l, ctx, cmd)

	entry, ok := jukebox.NewProcessor(l, ctx).GetActive(f)
	require.True(t, ok)
	require.False(t, entry.ExpiresAt.After(time.Now().Add(maxJukeboxDuration+time.Second)))
}

func TestHandlePlayJukeboxCommand_IgnoresOtherCommandTypes(t *testing.T) {
	l, _ := test.NewNullLogger()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	f := field.NewBuilder(0, 1, 100000000).SetInstance(uuid.Nil).Build()

	cmd := mapKafka.Command[mapKafka.PlayJukeboxCommandBody]{
		TransactionId: uuid.New(),
		WorldId:       0,
		ChannelId:     1,
		MapId:         100000000,
		Instance:      uuid.Nil,
		Type:          mapKafka.CommandTypeWeatherStart,
		Body: mapKafka.PlayJukeboxCommandBody{
			ItemId:     5100000,
			PlayerName: "Chronicle",
			DurationMs: 45000,
		},
	}

	handlePlayJukeboxCommand()(l, ctx, cmd)

	_, ok := jukebox.NewProcessor(l, ctx).GetActive(f)
	require.False(t, ok)
}
