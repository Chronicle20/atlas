package listener_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"atlas-channel/listener"
	"atlas-channel/server"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// nopDeps is a Dependencies where every callback succeeds and returns
// nothing. Tests that need to assert side effects override fields.
func nopDeps() listener.Dependencies {
	return listener.Dependencies{
		UnregisterChannel:  func(channel.Model) error { return nil },
		SessionsForKey:     func(server.Key) []listener.Session { return nil },
		SendShutdownNotice: func(listener.Session) {},
		DestroySession:     func(listener.Session) error { return nil },
		RemoveHandler:      func(string, string) error { return nil },
	}
}

func nullLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)
	return l
}

func makeTenant(t *testing.T) tenant.Model {
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tm
}

func makeServerModel(t *testing.T, tm tenant.Model, w world.Id, c channel.Id) server.Model {
	return server.Register(tm, channel.NewModel(w, c), "127.0.0.1", 8585+int(c))
}

func TestRegistry_AddStoresAndSnapshotsHandle(t *testing.T) {
	tm := makeTenant(t)
	sc := makeServerModel(t, tm, 1, 0)
	key := server.KeyOf(sc)
	defer server.GetRegistry().Deregister(key)

	r := listener.NewRegistry(nullLogger(), nopDeps(), listener.Config{})
	called := false
	_, err := r.Add(context.Background(), key, sc, func(h *listener.Handle) ([]listener.HandlerHandle, error) {
		called = true
		require.Equal(t, key, h.Key)
		require.Equal(t, listener.Active, h.State)
		return []listener.HandlerHandle{{Topic: "T", Id: "h1"}}, nil
	})
	require.NoError(t, err)
	require.True(t, called)

	snap := r.Snapshot()
	require.Len(t, snap, 1)
	require.Equal(t, key, snap[0].Key)
	require.Len(t, snap[0].KafkaHandlers, 1)
}

func TestRegistry_DrainRunsAllFourPhases(t *testing.T) {
	tm := makeTenant(t)
	sc := makeServerModel(t, tm, 1, 1)
	key := server.KeyOf(sc)
	defer server.GetRegistry().Deregister(key)

	var unregCalls atomic.Int32
	var destroyCalls atomic.Int32
	var removeHandlerCalls atomic.Int32

	deps := nopDeps()
	deps.UnregisterChannel = func(channel.Model) error { unregCalls.Add(1); return nil }
	deps.SessionsForKey = func(server.Key) []listener.Session {
		return []listener.Session{"s1", "s2", "s3"}
	}
	deps.DestroySession = func(listener.Session) error { destroyCalls.Add(1); return nil }
	deps.RemoveHandler = func(string, string) error { removeHandlerCalls.Add(1); return nil }

	r := listener.NewRegistry(nullLogger(), deps, listener.Config{DrainDeadline: 200 * time.Millisecond})

	h, err := r.Add(context.Background(), key, sc, func(*listener.Handle) ([]listener.HandlerHandle, error) {
		return []listener.HandlerHandle{
			{Topic: "T1", Id: "h1"},
			{Topic: "T2", Id: "h2"},
		}, nil
	})
	require.NoError(t, err)
	require.NoError(t, r.Drain(key))

	require.EqualValues(t, 1, unregCalls.Load(), "atlas-world DELETE called once")
	require.EqualValues(t, 3, destroyCalls.Load(), "all 3 sessions destroyed")
	require.EqualValues(t, 2, removeHandlerCalls.Load(), "both kafka handlers removed")
	require.Equal(t, context.Canceled, h.Ctx.Err(), "ctx canceled in phase 4")

	_, ok := server.GetRegistry().Get(key)
	require.False(t, ok, "server.Registry no longer has key after drain")

	_, ok = r.Get(key)
	require.False(t, ok, "listener.Registry removes entry after Removed")
}

func TestRegistry_DrainIdempotentUnderConcurrency(t *testing.T) {
	tm := makeTenant(t)
	sc := makeServerModel(t, tm, 2, 0)
	key := server.KeyOf(sc)
	defer server.GetRegistry().Deregister(key)

	var unregCalls atomic.Int32
	deps := nopDeps()
	deps.UnregisterChannel = func(channel.Model) error {
		unregCalls.Add(1)
		return nil
	}
	r := listener.NewRegistry(nullLogger(), deps, listener.Config{DrainDeadline: 50 * time.Millisecond})

	_, err := r.Add(context.Background(), key, sc, func(*listener.Handle) ([]listener.HandlerHandle, error) {
		return nil, nil
	})
	require.NoError(t, err)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = r.Drain(key)
		}()
	}
	wg.Wait()

	require.EqualValues(t, 1, unregCalls.Load(),
		"only one goroutine should claim the drain; UnregisterChannel must run exactly once")
}

func TestRegistry_DrainWarnsOnDeadlineButCompletes(t *testing.T) {
	tm := makeTenant(t)
	sc := makeServerModel(t, tm, 3, 0)
	key := server.KeyOf(sc)
	defer server.GetRegistry().Deregister(key)

	r := listener.NewRegistry(nullLogger(), nopDeps(), listener.Config{DrainDeadline: 30 * time.Millisecond})

	h, err := r.Add(context.Background(), key, sc, func(*listener.Handle) ([]listener.HandlerHandle, error) {
		return nil, nil
	})
	require.NoError(t, err)
	// Park a goroutine on h.Wg that outlasts the deadline so phase 3 times out.
	h.Wg.Add(1)
	go func() {
		time.Sleep(200 * time.Millisecond)
		h.Wg.Done()
	}()

	start := time.Now()
	require.NoError(t, r.Drain(key))
	elapsed := time.Since(start)
	require.GreaterOrEqual(t, elapsed, 30*time.Millisecond)
	require.Less(t, elapsed, 200*time.Millisecond, "phase 4 must fall through past the deadline, not wait on the goroutine")
}

func TestRegistry_EvictorFiresWhenLastListenerForTenantRemoved(t *testing.T) {
	tm1 := makeTenant(t)
	tm2 := makeTenant(t)
	sc1a := makeServerModel(t, tm1, 4, 0)
	sc1b := makeServerModel(t, tm1, 4, 1)
	sc2 := makeServerModel(t, tm2, 5, 0)
	k1a := server.KeyOf(sc1a)
	k1b := server.KeyOf(sc1b)
	k2 := server.KeyOf(sc2)
	defer server.GetRegistry().Deregister(k1a)
	defer server.GetRegistry().Deregister(k1b)
	defer server.GetRegistry().Deregister(k2)

	var evicted []uuid.UUID
	var evMu sync.Mutex
	listener.SetEvictorsForTest(t, func(tt tenant.Model) {
		evMu.Lock()
		evicted = append(evicted, tt.Id())
		evMu.Unlock()
	})

	r := listener.NewRegistry(nullLogger(), nopDeps(), listener.Config{DrainDeadline: 50 * time.Millisecond})

	noBody := func(*listener.Handle) ([]listener.HandlerHandle, error) { return nil, nil }
	_, err := r.Add(context.Background(), k1a, sc1a, noBody)
	require.NoError(t, err)
	_, err = r.Add(context.Background(), k1b, sc1b, noBody)
	require.NoError(t, err)
	_, err = r.Add(context.Background(), k2, sc2, noBody)
	require.NoError(t, err)

	require.NoError(t, r.Drain(k1a))
	evMu.Lock()
	require.Empty(t, evicted, "still one listener for tm1")
	evMu.Unlock()

	require.NoError(t, r.Drain(k1b))
	evMu.Lock()
	require.Equal(t, []uuid.UUID{tm1.Id()}, evicted, "evictor fires for tm1 once last listener drains")
	evMu.Unlock()

	require.NoError(t, r.Drain(k2))
	evMu.Lock()
	require.Equal(t, []uuid.UUID{tm1.Id(), tm2.Id()}, evicted, "evictor also fires for tm2")
	evMu.Unlock()
}
