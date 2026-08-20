package socket

import (
	"atlas-channel/server"
	"context"
	"net"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	env "github.com/Chronicle20/atlas/libs/atlas-env"
	socket "github.com/Chronicle20/atlas/libs/atlas-socket"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func testTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Register(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("register test tenant: %v", err)
	}
	return tm
}

// countingWG is a test-only WaitGrouper that records the number of Add/Done
// calls it observes, so a test can assert accounting without depending on
// sync.WaitGroup internals -- same shape as atlas-socket's own countingWG.
type countingWG struct {
	mu    sync.Mutex
	adds  int
	dones int
}

func (c *countingWG) Add(delta int) { c.mu.Lock(); c.adds += delta; c.mu.Unlock() }
func (c *countingWG) Done()         { c.mu.Lock(); c.dones++; c.mu.Unlock() }
func (c *countingWG) Adds() int     { c.mu.Lock(); defer c.mu.Unlock(); return c.adds }
func (c *countingWG) Dones() int    { c.mu.Lock(); defer c.mu.Unlock(); return c.dones }

// TestCreateSocketServiceReturnsErrorWhenPortIsAlreadyBound pins the
// bind-before-side-effects property that makes listener.Registry.Add's
// rollback sufficient: a bind failure must not have started the accept
// loop or registered a single session (task-244 design.md §4.2/§6).
func TestCreateSocketServiceReturnsErrorWhenPortIsAlreadyBound(t *testing.T) {
	pre, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("bind a port to occupy: %v", err)
	}
	defer func() { _ = pre.Close() }()
	port := pre.Addr().(*net.TCPAddr).Port

	tm := testTenant(t)
	ctx := NewListenerContext(context.Background(), tm)
	sc := server.NewProcessor(logrus.New(), ctx).Register(tm, channel.NewModel(1, 0), "127.0.0.1", port)
	defer server.GetRegistry().Deregister(server.KeyOf(sc))

	wg := &countingWG{}
	sessionWg := &countingWG{}

	hp := func() map[uint16]request.Handler { return nil }
	rw := socket.ShortReadWriter{}

	lis, err := CreateSocketService(logrus.New(), ctx, wg, sessionWg)(hp, rw, nil, sc, "127.0.0.1", port)
	if lis != nil {
		t.Fatal("CreateSocketService returned a non-nil listener on a bind failure")
	}
	if err == nil {
		t.Fatal("CreateSocketService returned a nil error on a bind failure")
	}
	if adds := wg.Adds(); adds != 0 {
		t.Fatalf("wg.Adds() = %d, want 0 -- a bind failure must not start the accept loop", adds)
	}
	if adds := sessionWg.Adds(); adds != 0 {
		t.Fatalf("sessionWg.Adds() = %d, want 0 -- a bind failure must not register a session", adds)
	}
}

// TestCreateSocketServiceReturnsTheBoundListener pins the happy path:
// CreateSocketService binds synchronously and returns the bound listener
// so buildListener can install Handle.CloseListener.
func TestCreateSocketServiceReturnsTheBoundListener(t *testing.T) {
	tm := testTenant(t)
	ctx, cancel := context.WithCancel(NewListenerContext(context.Background(), tm))
	defer cancel()

	sc := server.NewProcessor(logrus.New(), ctx).Register(tm, channel.NewModel(1, 0), "127.0.0.1", 0)
	t.Cleanup(func() { server.GetRegistry().Deregister(server.KeyOf(sc)) })

	wg := &countingWG{}
	sessionWg := &countingWG{}

	hp := func() map[uint16]request.Handler { return nil }
	rw := socket.ShortReadWriter{}

	lis, err := CreateSocketService(logrus.New(), ctx, wg, sessionWg)(hp, rw, nil, sc, "127.0.0.1", 0)
	if err != nil {
		t.Fatalf("CreateSocketService returned an error: %v", err)
	}
	if lis == nil {
		t.Fatal("CreateSocketService returned a nil listener")
	}
	addr, ok := lis.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener address is not *net.TCPAddr: %T", lis.Addr())
	}
	if addr.Port == 0 {
		t.Fatal("listener port is 0, want an assigned port")
	}
	if err := lis.Close(); err != nil {
		t.Fatalf("lis.Close() = %v, want nil on first close", err)
	}
}

func TestDualWaitGroupFansOutAddAndDone(t *testing.T) {
	a, b := &countingWG{}, &countingWG{}
	d := dualWaitGroup{a: a, b: b}
	d.Add(1)
	d.Add(2)
	d.Done()

	if a.Adds() != 3 || b.Adds() != 3 {
		t.Fatalf("a.Adds() = %d, b.Adds() = %d, want 3 and 3", a.Adds(), b.Adds())
	}
	if a.Dones() != 1 || b.Dones() != 1 {
		t.Fatalf("a.Dones() = %d, b.Dones() = %d, want 1 and 1", a.Dones(), b.Dones())
	}
}

// TestNewListenerContextCarriesThisPodsEnvironment pins the buildListener
// socket-registration path (main.go calls socket.NewListenerContext):
// CreateSocketService wires Create/Destroy/SendPing directly from this
// context, bypassing socket/handler.AdaptHandler entirely, so this path
// must also originate the environment from env.Self() -- a second,
// parallel context path alongside the per-request handler path.
func TestNewListenerContextCarriesThisPodsEnvironment(t *testing.T) {
	t.Setenv(env.SelfVar, "pr-123")
	ctx := NewListenerContext(context.Background(), testTenant(t))
	if got := env.MustFromContext(ctx); got != env.Id("pr-123") {
		t.Fatalf("environment = %q, want \"pr-123\" from ATLAS_ENVIRONMENT", got)
	}
}

func TestNewListenerContextOnMainIsTheLegacyValue(t *testing.T) {
	t.Setenv(env.SelfVar, "")
	ctx := NewListenerContext(context.Background(), testTenant(t))
	if got := env.MustFromContext(ctx); got != env.Id("") {
		t.Fatalf("environment = %q, want the empty id", got)
	}
}

// TestWithSelfEnvironmentCarriesThisPodsEnvironment pins the helper that
// lets a package outside env-domain-guard's permitted list (like
// character/combo's DecayTick) originate env.Self() on a per-event
// context without importing atlas-env directly.
func TestWithSelfEnvironmentCarriesThisPodsEnvironment(t *testing.T) {
	t.Setenv(env.SelfVar, "pr-123")
	ctx := WithSelfEnvironment(context.Background())
	if got := env.MustFromContext(ctx); got != env.Id("pr-123") {
		t.Fatalf("environment = %q, want \"pr-123\" from ATLAS_ENVIRONMENT", got)
	}
}
