package session

import (
	"context"
	"testing"

	sessionKafka "atlas-maps/kafka/message/session"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

type fakeForceReturner struct {
	calledFor uint32
	called    bool
	returnVal bool
}

func (f *fakeForceReturner) ForceReturnIfTracked(characterId uint32) bool {
	f.calledFor = characterId
	f.called = true
	return f.returnVal
}

func TestHandleSessionDestroyed_ForcesReturnForTrackedCharacter(t *testing.T) {
	logger, _ := test.NewNullLogger()
	tt, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tt)

	fr := &fakeForceReturner{returnVal: true}
	h := newHandleSessionDestroyed(func(_ context.Context) ForceReturner { return fr })
	h(logger, ctx, sessionKafka.StatusEvent{
		SessionId:   uuid.New(),
		AccountId:   1,
		CharacterId: 42,
		WorldId:     world.Id(1),
		ChannelId:   channel.Id(2),
		Type:        sessionKafka.EventSessionStatusTypeDestroyed,
	})

	require.True(t, fr.called)
	require.Equal(t, uint32(42), fr.calledFor)
}

func TestHandleSessionDestroyed_IgnoresCreatedEvents(t *testing.T) {
	logger, _ := test.NewNullLogger()
	tt, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tt)

	fr := &fakeForceReturner{}
	h := newHandleSessionDestroyed(func(_ context.Context) ForceReturner { return fr })
	h(logger, ctx, sessionKafka.StatusEvent{
		Type:        sessionKafka.EventSessionStatusTypeCreated,
		CharacterId: 42,
	})
	require.False(t, fr.called)
}

func TestHandleSessionDestroyed_SkipsZeroCharacterId(t *testing.T) {
	logger, _ := test.NewNullLogger()
	tt, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tt)

	fr := &fakeForceReturner{}
	h := newHandleSessionDestroyed(func(_ context.Context) ForceReturner { return fr })
	h(logger, ctx, sessionKafka.StatusEvent{
		Type:        sessionKafka.EventSessionStatusTypeDestroyed,
		CharacterId: 0,
	})
	require.False(t, fr.called, "no-character-selected sessions must be skipped")
}
