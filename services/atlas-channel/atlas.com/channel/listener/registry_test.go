package listener_test

import (
	"atlas-channel/listener"
	"atlas-channel/server"
	"context"
	"errors"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	socket "github.com/Chronicle20/atlas/libs/atlas-socket"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// nopDeps is a Dependencies where every callback succeeds and returns
// nothing. Tests that need to assert side effects override fields.
func nopDeps() listener.Dependencies {
	return listener.Dependencies{
		UnregisterChannel: func(channel.Model) error { return nil },
		RemoveHandler:     func(string, string) error { return nil },
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
	return server.NewProcessor(logrus.New(), context.Background()).Register(tm, channel.NewModel(w, c), "127.0.0.1", 8585+int(c))
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
	deps.RemoveHandler = func(string, string) error { removeHandlerCalls.Add(1); return nil }

	r := listener.NewRegistry(nullLogger(), deps, listener.Config{DrainDeadline: 200 * time.Millisecond})

	h, err := r.Add(context.Background(), key, sc, func(h *listener.Handle) ([]listener.HandlerHandle, error) {
		h.Sessions = func() []listener.Session { return []listener.Session{"s1", "s2", "s3"} }
		h.Kick = func(listener.Session) error { destroyCalls.Add(1); return nil }
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

// acceptNotifyWG wraps a *sync.WaitGroup so a test can observe the
// instant the accept loop calls Add(1) via a channel receive, rather
// than by reading the WaitGroup's own state. Without this, a test that
// dials a connection and then immediately calls Drain has no
// happens-before edge between the accept site's h.Wg.Add(1) (in the
// Serve goroutine) and Drain phase 3's h.Wg.Wait() (in a
// routine.Go-spawned goroutine) -- both touch h.Wg with nothing tying
// the two goroutines together, which is a genuine sync.WaitGroup
// "Add concurrently with Wait" race, not a timing artifact.
type acceptNotifyWG struct {
	inner *sync.WaitGroup
	once  sync.Once
	ready chan struct{}
}

func (w *acceptNotifyWG) Add(delta int) {
	w.inner.Add(delta)
	if delta > 0 {
		w.once.Do(func() { close(w.ready) })
	}
}

func (w *acceptNotifyWG) Done() { w.inner.Done() }

func TestRegistry_DrainClosesTheBoundSocket(t *testing.T) {
	tm := makeTenant(t)
	sc := makeServerModel(t, tm, 6, 0)
	key := server.KeyOf(sc)
	defer server.GetRegistry().Deregister(key)

	r := listener.NewRegistry(nullLogger(), nopDeps(), listener.Config{DrainDeadline: 200 * time.Millisecond})

	accepted := make(chan struct{})
	var port int
	h, err := r.Add(context.Background(), key, sc, func(h *listener.Handle) ([]listener.HandlerHandle, error) {
		lis, err := socket.Bind(nullLogger(), "127.0.0.1", 0)
		require.NoError(t, err)
		port = lis.Addr().(*net.TCPAddr).Port
		h.CloseListener = lis.Close
		sessionWg := &acceptNotifyWG{inner: h.Wg, ready: accepted}
		go func() { _ = socket.Serve(nullLogger(), h.Ctx, &sync.WaitGroup{}, sessionWg, lis) }()
		return nil, nil
	})
	require.NoError(t, err)
	_ = h

	addr := "127.0.0.1:" + strconv.Itoa(port)
	conn, err := net.DialTimeout("tcp", addr, time.Second)
	require.NoError(t, err, "port must be live before drain")
	require.NoError(t, conn.Close())

	select {
	case <-accepted:
	case <-time.After(2 * time.Second):
		t.Fatal("server never accepted the dialed connection")
	}

	require.NoError(t, r.Drain(key))

	var dialErr error
	for i := 0; i < 20; i++ {
		conn, dialErr = net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if dialErr != nil {
			break
		}
		require.NoError(t, conn.Close())
		time.Sleep(25 * time.Millisecond)
	}
	require.Error(t, dialErr, "port %d still accepting after drain", port)
}

func TestRegistry_DrainClosesListenerBeforePhase3Wait(t *testing.T) {
	tm := makeTenant(t)
	sc := makeServerModel(t, tm, 6, 1)
	key := server.KeyOf(sc)
	defer server.GetRegistry().Deregister(key)

	r := listener.NewRegistry(nullLogger(), nopDeps(), listener.Config{DrainDeadline: 2 * time.Second})

	var port int
	h, err := r.Add(context.Background(), key, sc, func(h *listener.Handle) ([]listener.HandlerHandle, error) {
		lis, err := socket.Bind(nullLogger(), "127.0.0.1", 0)
		require.NoError(t, err)
		port = lis.Addr().(*net.TCPAddr).Port
		h.CloseListener = lis.Close
		go func() { _ = socket.Serve(nullLogger(), h.Ctx, &sync.WaitGroup{}, h.Wg, lis) }()
		return nil, nil
	})
	require.NoError(t, err)

	h.Wg.Add(1)
	go func() {
		time.Sleep(150 * time.Millisecond)
		h.Wg.Done()
	}()

	drainDone := make(chan error, 1)
	go func() { drainDone <- r.Drain(key) }()

	time.Sleep(50 * time.Millisecond)
	addr := "127.0.0.1:" + strconv.Itoa(port)
	_, dialErr := net.DialTimeout("tcp", addr, 200*time.Millisecond)
	require.Error(t, dialErr, "listener must already be closed mid-phase-3")

	select {
	case err := <-drainDone:
		require.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("Drain did not return within 3s")
	}
}

func TestRegistry_AddReturnsErrorAndLeavesNoEntryWhenBodyFails(t *testing.T) {
	tm := makeTenant(t)
	sc := makeServerModel(t, tm, 6, 2)
	key := server.KeyOf(sc)
	defer server.GetRegistry().Deregister(key)

	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer occupied.Close()
	occupiedPort := occupied.Addr().(*net.TCPAddr).Port

	var evictCalls atomic.Int32
	listener.SetEvictorsForTest(t, func(tenant.Model) { evictCalls.Add(1) })

	r := listener.NewRegistry(nullLogger(), nopDeps(), listener.Config{DrainDeadline: 200 * time.Millisecond})

	_, err = r.Add(context.Background(), key, sc, func(h *listener.Handle) ([]listener.HandlerHandle, error) {
		_, bindErr := socket.Bind(nullLogger(), "127.0.0.1", occupiedPort)
		return nil, bindErr
	})
	require.Error(t, err)

	_, ok := r.Get(key)
	require.False(t, ok)
	require.Empty(t, r.Snapshot())

	sc2 := makeServerModel(t, tm, 6, 3)
	key2 := server.KeyOf(sc2)
	defer server.GetRegistry().Deregister(key2)
	_, err = r.Add(context.Background(), key2, sc2, func(*listener.Handle) ([]listener.HandlerHandle, error) {
		return nil, nil
	})
	require.NoError(t, err)
	require.NoError(t, r.Drain(key2))

	require.EqualValues(t, 1, evictCalls.Load(), "refs must not have leaked from the failed Add")
}

func TestRegistry_AddRefusesADrainingHandle(t *testing.T) {
	tm := makeTenant(t)
	sc := makeServerModel(t, tm, 6, 4)
	key := server.KeyOf(sc)
	defer server.GetRegistry().Deregister(key)

	r := listener.NewRegistry(nullLogger(), nopDeps(), listener.Config{DrainDeadline: 2 * time.Second})

	block := make(chan struct{})
	_, err := r.Add(context.Background(), key, sc, func(h *listener.Handle) ([]listener.HandlerHandle, error) {
		h.Sessions = func() []listener.Session { return []listener.Session{"s"} }
		h.Kick = func(listener.Session) error { <-block; return nil }
		return nil, nil
	})
	require.NoError(t, err)

	drainDone := make(chan error, 1)
	go func() { drainDone <- r.Drain(key) }()

	require.Eventually(t, func() bool {
		state, ok := r.State(key)
		return ok && state == listener.Draining
	}, time.Second, 5*time.Millisecond, "handle never reported Draining")

	h2, addErr := r.Add(context.Background(), key, sc, func(*listener.Handle) ([]listener.HandlerHandle, error) {
		return nil, nil
	})
	require.Nil(t, h2)
	require.True(t, errors.Is(addErr, listener.ErrDraining))
	require.Len(t, r.Snapshot(), 1)

	close(block)
	<-drainDone
}

func TestRegistry_Phase3CompletesAsSoonAsTheWaitGroupReleases(t *testing.T) {
	tm := makeTenant(t)
	sc := makeServerModel(t, tm, 6, 5)
	key := server.KeyOf(sc)
	defer server.GetRegistry().Deregister(key)

	r := listener.NewRegistry(nullLogger(), nopDeps(), listener.Config{DrainDeadline: 3 * time.Second})

	h, err := r.Add(context.Background(), key, sc, func(*listener.Handle) ([]listener.HandlerHandle, error) {
		return nil, nil
	})
	require.NoError(t, err)

	h.Wg.Add(1)
	go func() {
		time.Sleep(100 * time.Millisecond)
		h.Wg.Done()
	}()

	start := time.Now()
	require.NoError(t, r.Drain(key))
	elapsed := time.Since(start)
	require.GreaterOrEqual(t, elapsed, 100*time.Millisecond)
	require.Less(t, elapsed, 1*time.Second)
}

func TestRegistry_Phase2ContinuesPastAKickError(t *testing.T) {
	tm := makeTenant(t)
	sc := makeServerModel(t, tm, 6, 6)
	key := server.KeyOf(sc)
	defer server.GetRegistry().Deregister(key)

	r := listener.NewRegistry(nullLogger(), nopDeps(), listener.Config{DrainDeadline: 200 * time.Millisecond})

	var kickCalls atomic.Int32
	_, err := r.Add(context.Background(), key, sc, func(h *listener.Handle) ([]listener.HandlerHandle, error) {
		h.Sessions = func() []listener.Session { return []listener.Session{"a", "b", "c"} }
		h.Kick = func(s listener.Session) error {
			kickCalls.Add(1)
			if s == "b" {
				return errors.New("boom")
			}
			return nil
		}
		return nil, nil
	})
	require.NoError(t, err)

	require.NoError(t, r.Drain(key))
	require.EqualValues(t, 3, kickCalls.Load())
}

func TestRegistry_DrainAllIsBoundedByOneDeadline(t *testing.T) {
	tm := makeTenant(t)
	r := listener.NewRegistry(nullLogger(), nopDeps(), listener.Config{DrainDeadline: 200 * time.Millisecond})

	for i := 0; i < 4; i++ {
		sc := makeServerModel(t, tm, 7, channel.Id(i))
		key := server.KeyOf(sc)
		defer server.GetRegistry().Deregister(key)
		h, err := r.Add(context.Background(), key, sc, func(*listener.Handle) ([]listener.HandlerHandle, error) {
			return nil, nil
		})
		require.NoError(t, err)
		h.Wg.Add(1)
	}

	start := time.Now()
	r.DrainAll()
	elapsed := time.Since(start)
	require.Less(t, elapsed, 700*time.Millisecond, "DrainAll must not serialize N deadlines")
	require.Empty(t, r.Snapshot())
}
