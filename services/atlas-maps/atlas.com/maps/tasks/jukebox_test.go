package tasks

import (
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

// jukeboxEnvMarkerKey is a test-local context key -- deliberately not
// libs/atlas-env, since tasks sits outside env-domain-guard's permitted
// import list (main.go, kafka/, rest/, socket/) and must not import
// atlas-env even from a test file.
type jukeboxEnvMarkerKey string

// TestProcessExpiredJukebox_AppliesEnvContextToEmit pins the review fix:
// this pod's own environment identity must be threaded onto each expired
// jukebox entry's per-tenant context before the jukebox-end emit. A test
// with an identity envContext would still pass if this were dropped --
// decide() would then fail open per FR-1.8 and every live deployment, not
// just this pod's, would react to the jukebox end.
func TestProcessExpiredJukebox_AppliesEnvContextToEmit(t *testing.T) {
	l, _ := test.NewNullLogger()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	f := field.NewBuilder(0, 1, 100000000).Build()
	key := jukebox.FieldKey{Tenant: ten, Field: f}
	entry := jukebox.ExpiredEntry{
		Key:   key,
		Entry: jukebox.JukeboxEntry{ItemId: 5100000, PlayerName: "Chronicle", ExpiresAt: time.Now().Add(-time.Second)},
	}

	envContext := func(ctx context.Context) context.Context {
		return context.WithValue(ctx, jukeboxEnvMarkerKey("marker"), "stamped")
	}

	var gotMarker any
	processExpiredJukebox(l, context.Background(), []jukebox.ExpiredEntry{entry}, func(ctx context.Context, e jukebox.ExpiredEntry) error {
		gotMarker = ctx.Value(jukeboxEnvMarkerKey("marker"))
		return nil
	}, envContext)

	require.Equal(t, "stamped", gotMarker, "envContext was not applied to the jukebox-end emit context")
}

// TestProcessExpiredJukebox_DeletesTheEntry confirms the sweep clears the
// registry entry after emitting the jukebox-end event, so a subsequent
// GetActive lookup no longer finds it.
func TestProcessExpiredJukebox_DeletesTheEntry(t *testing.T) {
	l, _ := test.NewNullLogger()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := context.Background()
	tctx := tenant.WithContext(ctx, ten)
	f := field.NewBuilder(0, 1, 100000001).Build()

	jukebox.NewProcessor(l, tctx).Start(f, 5100000, "Chronicle", -time.Second)

	noopEmit := func(ctx context.Context, e jukebox.ExpiredEntry) error {
		return nil
	}
	identityEnvContext := func(ctx context.Context) context.Context {
		return ctx
	}

	processExpiredJukebox(l, ctx, jukebox.GetExpired(), noopEmit, identityEnvContext)

	_, ok := jukebox.NewProcessor(l, tctx).GetActive(f)
	require.False(t, ok)
}
