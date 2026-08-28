package _map

import (
	mapKafka "atlas-maps/kafka/message/map"
	"atlas-maps/map/backeffect"
	"atlas-maps/map/environment"
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

func TestHandleSetEnvironmentStateCommand_Applies(t *testing.T) {
	l, _ := test.NewNullLogger()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	f := field.NewBuilder(0, 1, 910010000).SetInstance(uuid.Nil).Build()

	cmd := mapKafka.Command[mapKafka.SetEnvironmentStateCommandBody]{
		TransactionId: uuid.New(),
		WorldId:       0,
		ChannelId:     1,
		MapId:         910010000,
		Instance:      uuid.Nil,
		Type:          mapKafka.CommandTypeSetEnvironmentState,
		Body: mapKafka.SetEnvironmentStateCommandBody{
			Kind:  "OBSTACLE",
			Name:  "obs3",
			State: 2,
		},
	}

	handleSetEnvironmentStateCommand()(l, ctx, cmd)

	entries := environment.NewProcessor(l, ctx).GetAll(f)
	require.Len(t, entries, 1)
	require.Equal(t, environment.ObjectEntry{Kind: field.ObjectKindObstacle, Name: "obs3", State: 2}, entries[0])
}

func TestHandleSetEnvironmentStateCommand_BlankKindDefaultsEnvironment(t *testing.T) {
	l, _ := test.NewNullLogger()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	f := field.NewBuilder(0, 1, 910010000).SetInstance(uuid.Nil).Build()

	cmd := mapKafka.Command[mapKafka.SetEnvironmentStateCommandBody]{
		TransactionId: uuid.New(),
		WorldId:       0,
		ChannelId:     1,
		MapId:         910010000,
		Instance:      uuid.Nil,
		Type:          mapKafka.CommandTypeSetEnvironmentState,
		Body: mapKafka.SetEnvironmentStateCommandBody{
			Kind:  "",
			Name:  "gate01",
			State: 1,
		},
	}

	handleSetEnvironmentStateCommand()(l, ctx, cmd)

	entries := environment.NewProcessor(l, ctx).GetAll(f)
	require.Len(t, entries, 1)
	require.Equal(t, field.ObjectKindEnvironment, entries[0].Kind)
}

func TestHandleSetEnvironmentStateCommand_WrongTypeIgnored(t *testing.T) {
	l, _ := test.NewNullLogger()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	f := field.NewBuilder(0, 1, 910010000).SetInstance(uuid.Nil).Build()

	cmd := mapKafka.Command[mapKafka.SetEnvironmentStateCommandBody]{
		TransactionId: uuid.New(),
		WorldId:       0,
		ChannelId:     1,
		MapId:         910010000,
		Instance:      uuid.Nil,
		Type:          mapKafka.CommandTypeWeatherStart,
		Body: mapKafka.SetEnvironmentStateCommandBody{
			Kind:  "OBSTACLE",
			Name:  "obs3",
			State: 2,
		},
	}

	handleSetEnvironmentStateCommand()(l, ctx, cmd)

	entries := environment.NewProcessor(l, ctx).GetAll(f)
	require.Len(t, entries, 0)
}

func TestHandleSetEnvironmentStateCommand_UnknownKindRejected(t *testing.T) {
	l, _ := test.NewNullLogger()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	f := field.NewBuilder(0, 1, 910010000).SetInstance(uuid.Nil).Build()

	cmd := mapKafka.Command[mapKafka.SetEnvironmentStateCommandBody]{
		TransactionId: uuid.New(),
		WorldId:       0,
		ChannelId:     1,
		MapId:         910010000,
		Instance:      uuid.Nil,
		Type:          mapKafka.CommandTypeSetEnvironmentState,
		Body: mapKafka.SetEnvironmentStateCommandBody{
			Kind:  "GATE",
			Name:  "obs3",
			State: 2,
		},
	}

	require.NotPanics(t, func() {
		handleSetEnvironmentStateCommand()(l, ctx, cmd)
	})

	entries := environment.NewProcessor(l, ctx).GetAll(f)
	require.Len(t, entries, 0)
}

func TestHandleSetEnvironmentStateCommand_BlankNameRejected(t *testing.T) {
	l, _ := test.NewNullLogger()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	f := field.NewBuilder(0, 1, 910010000).SetInstance(uuid.Nil).Build()

	cmd := mapKafka.Command[mapKafka.SetEnvironmentStateCommandBody]{
		TransactionId: uuid.New(),
		WorldId:       0,
		ChannelId:     1,
		MapId:         910010000,
		Instance:      uuid.Nil,
		Type:          mapKafka.CommandTypeSetEnvironmentState,
		Body: mapKafka.SetEnvironmentStateCommandBody{
			Kind:  "ENVIRONMENT",
			Name:  "",
			State: 1,
		},
	}

	handleSetEnvironmentStateCommand()(l, ctx, cmd)

	entries := environment.NewProcessor(l, ctx).GetAll(f)
	require.Len(t, entries, 0)
}

func TestHandleResetEnvironmentCommand_ClearsTracked(t *testing.T) {
	l, _ := test.NewNullLogger()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	f := field.NewBuilder(0, 1, 910010000).SetInstance(uuid.Nil).Build()

	proc := environment.NewProcessor(l, ctx)
	_, err = proc.Set(f, field.ObjectKindObstacle, "a", 1)
	require.NoError(t, err)
	_, err = proc.Set(f, field.ObjectKindEnvironment, "b", 2)
	require.NoError(t, err)

	cmd := mapKafka.Command[mapKafka.ResetEnvironmentCommandBody]{
		TransactionId: uuid.New(),
		WorldId:       0,
		ChannelId:     1,
		MapId:         910010000,
		Instance:      uuid.Nil,
		Type:          mapKafka.CommandTypeResetEnvironment,
		Body:          mapKafka.ResetEnvironmentCommandBody{},
	}

	handleResetEnvironmentCommand()(l, ctx, cmd)

	entries := environment.NewProcessor(l, ctx).GetAll(f)
	require.Len(t, entries, 0)
}

func TestHandleResetEnvironmentCommand_WrongTypeIgnored(t *testing.T) {
	l, _ := test.NewNullLogger()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	f := field.NewBuilder(0, 1, 910010000).SetInstance(uuid.Nil).Build()

	proc := environment.NewProcessor(l, ctx)
	_, err = proc.Set(f, field.ObjectKindObstacle, "a", 1)
	require.NoError(t, err)

	cmd := mapKafka.Command[mapKafka.ResetEnvironmentCommandBody]{
		TransactionId: uuid.New(),
		WorldId:       0,
		ChannelId:     1,
		MapId:         910010000,
		Instance:      uuid.Nil,
		Type:          mapKafka.CommandTypePlayJukebox,
		Body:          mapKafka.ResetEnvironmentCommandBody{},
	}

	handleResetEnvironmentCommand()(l, ctx, cmd)

	entries := environment.NewProcessor(l, ctx).GetAll(f)
	require.Len(t, entries, 1)
}

func TestHandleSetBackEffectCommand_RecordsEntry(t *testing.T) {
	l, _ := test.NewNullLogger()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	f := field.NewBuilder(0, 1, 100000000).SetInstance(uuid.Nil).Build()

	cmd := mapKafka.Command[mapKafka.SetBackEffectCommandBody]{
		TransactionId: uuid.New(),
		WorldId:       0,
		ChannelId:     1,
		MapId:         100000000,
		Instance:      uuid.Nil,
		Type:          mapKafka.CommandTypeSetBackEffect,
		Body: mapKafka.SetBackEffectCommandBody{
			Effect:   0,
			FieldId:  100000000,
			PageId:   1,
			Duration: 1000,
		},
	}

	handleSetBackEffectCommand()(l, ctx, cmd)

	entries := backeffect.NewProcessor(l, ctx).GetActive(f)
	require.Len(t, entries, 1)
	require.Equal(t, byte(0), entries[0].Effect)
	require.Equal(t, uint32(100000000), entries[0].FieldId)
	require.Equal(t, byte(1), entries[0].PageId)
	require.Equal(t, uint32(1000), entries[0].Duration)
}

func TestHandleSetBackEffectCommand_RejectsInvalidEffect(t *testing.T) {
	l, _ := test.NewNullLogger()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	f := field.NewBuilder(0, 1, 100000000).SetInstance(uuid.Nil).Build()

	cmd := mapKafka.Command[mapKafka.SetBackEffectCommandBody]{
		TransactionId: uuid.New(),
		WorldId:       0,
		ChannelId:     1,
		MapId:         100000000,
		Instance:      uuid.Nil,
		Type:          mapKafka.CommandTypeSetBackEffect,
		Body: mapKafka.SetBackEffectCommandBody{
			Effect:   2,
			FieldId:  100000000,
			PageId:   1,
			Duration: 1000,
		},
	}

	handleSetBackEffectCommand()(l, ctx, cmd)

	entries := backeffect.NewProcessor(l, ctx).GetActive(f)
	require.Len(t, entries, 0)
}

func TestHandleSetBackEffectCommand_IgnoresWrongType(t *testing.T) {
	l, _ := test.NewNullLogger()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	f := field.NewBuilder(0, 1, 100000000).SetInstance(uuid.Nil).Build()

	cmd := mapKafka.Command[mapKafka.SetBackEffectCommandBody]{
		TransactionId: uuid.New(),
		WorldId:       0,
		ChannelId:     1,
		MapId:         100000000,
		Instance:      uuid.Nil,
		Type:          mapKafka.CommandTypeWeatherStart,
		Body: mapKafka.SetBackEffectCommandBody{
			Effect:   0,
			FieldId:  100000000,
			PageId:   1,
			Duration: 1000,
		},
	}

	handleSetBackEffectCommand()(l, ctx, cmd)

	entries := backeffect.NewProcessor(l, ctx).GetActive(f)
	require.Len(t, entries, 0)
}

func TestHandleClearBackEffectCommand_RemovesEntries(t *testing.T) {
	l, _ := test.NewNullLogger()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	f := field.NewBuilder(0, 1, 100000000).SetInstance(uuid.Nil).Build()

	proc := backeffect.NewProcessor(l, ctx)
	proc.Set(f, backeffect.BackEffectEntry{Effect: 0, FieldId: 100000000, PageId: 1, Duration: 1000})
	proc.Set(f, backeffect.BackEffectEntry{Effect: 1, FieldId: 100000000, PageId: 2, Duration: 2000})

	cmd := mapKafka.Command[mapKafka.ClearBackEffectCommandBody]{
		TransactionId: uuid.New(),
		WorldId:       0,
		ChannelId:     1,
		MapId:         100000000,
		Instance:      uuid.Nil,
		Type:          mapKafka.CommandTypeClearBackEffect,
		Body:          mapKafka.ClearBackEffectCommandBody{},
	}

	handleClearBackEffectCommand()(l, ctx, cmd)

	entries := backeffect.NewProcessor(l, ctx).GetActive(f)
	require.Len(t, entries, 0)
}

func TestHandleClearBackEffectCommand_EmptyFieldIsNotAnError(t *testing.T) {
	l, _ := test.NewNullLogger()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	f := field.NewBuilder(0, 1, 100000000).SetInstance(uuid.Nil).Build()

	cmd := mapKafka.Command[mapKafka.ClearBackEffectCommandBody]{
		TransactionId: uuid.New(),
		WorldId:       0,
		ChannelId:     1,
		MapId:         100000000,
		Instance:      uuid.Nil,
		Type:          mapKafka.CommandTypeClearBackEffect,
		Body:          mapKafka.ClearBackEffectCommandBody{},
	}

	handleClearBackEffectCommand()(l, ctx, cmd)

	entries := backeffect.NewProcessor(l, ctx).GetActive(f)
	require.Len(t, entries, 0)
}
