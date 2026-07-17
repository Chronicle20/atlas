package characterstatus

import (
	"context"
	"testing"
	"time"

	"atlas-buffs/berserk"
	characterstatus2 "atlas-buffs/kafka/message/characterstatus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/stat"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func setup(t *testing.T) context.Context {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	berserk.InitRegistry(client)
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	assert.NoError(t, err)
	return tenant.WithContext(context.Background(), ten)
}

func tracked(t *testing.T, ctx context.Context, characterId uint32) {
	t.Helper()
	assert.NoError(t, berserk.GetRegistry().Track(ctx,
		berserk.NewBuilder(world.Id(0), characterId, 10).SetChannel(channel.Id(1)).Build()))
}

func TestHandleLogoutUntracks(t *testing.T) {
	ctx := setup(t)
	l := logrus.New()
	tracked(t, ctx, 42)

	handleStatusEventLogout(l, ctx, characterstatus2.StatusEvent[characterstatus2.StatusEventLogoutBody]{
		WorldId: world.Id(0), CharacterId: 42, Type: characterstatus2.StatusEventTypeLogout,
	})

	_, err := berserk.GetRegistry().Get(ctx, 42)
	assert.ErrorIs(t, err, berserk.ErrNotFound)
}

func TestHandleLogoutWrongTypeIsNoop(t *testing.T) {
	ctx := setup(t)
	l := logrus.New()
	tracked(t, ctx, 42)

	handleStatusEventLogout(l, ctx, characterstatus2.StatusEvent[characterstatus2.StatusEventLogoutBody]{
		WorldId: world.Id(0), CharacterId: 42, Type: characterstatus2.StatusEventTypeLogin,
	})

	_, err := berserk.GetRegistry().Get(ctx, 42)
	assert.NoError(t, err, "wrong-type event must not mutate the registry")
}

func TestHandleStatChangedHpMarksDirty(t *testing.T) {
	ctx := setup(t)
	l := logrus.New()
	tracked(t, ctx, 42)

	handleStatusEventStatChanged(l, ctx, characterstatus2.StatusEvent[characterstatus2.StatusEventStatChangedBody]{
		WorldId: world.Id(0), CharacterId: 42, Type: characterstatus2.StatusEventTypeStatChanged,
		Body: characterstatus2.StatusEventStatChangedBody{ChannelId: channel.Id(2), Updates: []stat.Type{stat.TypeHp}},
	})

	m, err := berserk.GetRegistry().Get(ctx, 42)
	assert.NoError(t, err)
	assert.Equal(t, channel.Id(2), m.ChannelId(), "channel refreshed")
	assert.False(t, m.DirtyAt().IsZero(), "HP change marks dirty")
	assert.True(t, m.DirtyDue(time.Now().Add(time.Second)))
}

func TestHandleStatChangedUntrackedIsNoop(t *testing.T) {
	ctx := setup(t)
	l := logrus.New()

	handleStatusEventStatChanged(l, ctx, characterstatus2.StatusEvent[characterstatus2.StatusEventStatChangedBody]{
		WorldId: world.Id(0), CharacterId: 99, Type: characterstatus2.StatusEventTypeStatChanged,
		Body: characterstatus2.StatusEventStatChangedBody{ChannelId: channel.Id(1), Updates: []stat.Type{stat.TypeHp}},
	})

	assert.Empty(t, berserk.GetRegistry().GetAll(ctx), "untracked characters generate no entries")
}

func TestHandleMapChangedRefreshesChannelAndMarksDirty(t *testing.T) {
	ctx := setup(t)
	l := logrus.New()
	tracked(t, ctx, 42)

	handleStatusEventMapChanged(l, ctx, characterstatus2.StatusEvent[characterstatus2.StatusEventMapChangedBody]{
		WorldId: world.Id(0), CharacterId: 42, Type: characterstatus2.StatusEventTypeMapChanged,
		Body: characterstatus2.StatusEventMapChangedBody{ChannelId: channel.Id(3)},
	})

	m, err := berserk.GetRegistry().Get(ctx, 42)
	assert.NoError(t, err)
	assert.Equal(t, channel.Id(3), m.ChannelId())
	assert.False(t, m.DirtyAt().IsZero(), "Cosmic re-checks on transfer")
}

func TestHandleChannelChangedRefreshesChannel(t *testing.T) {
	ctx := setup(t)
	l := logrus.New()
	tracked(t, ctx, 42)

	handleStatusEventChannelChanged(l, ctx, characterstatus2.StatusEvent[characterstatus2.StatusEventChannelChangedBody]{
		WorldId: world.Id(0), CharacterId: 42, Type: characterstatus2.StatusEventTypeChannelChanged,
		Body: characterstatus2.StatusEventChannelChangedBody{ChannelId: channel.Id(4)},
	})

	m, err := berserk.GetRegistry().Get(ctx, 42)
	assert.NoError(t, err)
	assert.Equal(t, channel.Id(4), m.ChannelId())
}
