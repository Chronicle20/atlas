package socket

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// countingWG is a test-only WaitGrouper that records the number of Add/Done
// calls it observes, so a test can assert accounting at the accept site
// without depending on sync.WaitGroup internals.
type countingWG struct {
	mu    sync.Mutex
	adds  int
	dones int
}

func (c *countingWG) Add(delta int) { c.mu.Lock(); c.adds += delta; c.mu.Unlock() }
func (c *countingWG) Done()         { c.mu.Lock(); c.dones++; c.mu.Unlock() }
func (c *countingWG) Adds() int     { c.mu.Lock(); defer c.mu.Unlock(); return c.adds }
func (c *countingWG) Dones() int    { c.mu.Lock(); defer c.mu.Unlock(); return c.dones }

func testLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetOutput(nopWriter{})
	return l
}

func TestBindSucceedsOnEphemeralPort(t *testing.T) {
	lis, err := Bind(testLogger(), "127.0.0.1", 0)
	if err != nil {
		t.Fatalf("Bind returned error: %v", err)
	}
	if lis == nil {
		t.Fatal("Bind returned a nil listener")
	}
	defer func() { _ = lis.Close() }()

	addr, ok := lis.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener address is not *net.TCPAddr: %T", lis.Addr())
	}
	if addr.Port == 0 {
		t.Fatal("listener port is 0, want an assigned ephemeral port")
	}
}

func TestBindFailsWhenPortAlreadyBound(t *testing.T) {
	held, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to bind a port to occupy: %v", err)
	}
	defer func() { _ = held.Close() }()

	port := held.Addr().(*net.TCPAddr).Port

	lis, err := Bind(testLogger(), "127.0.0.1", port)
	if err == nil {
		t.Fatal("Bind on an already-bound port returned nil error")
	}
	if lis != nil {
		t.Fatal("Bind on an already-bound port returned a non-nil listener")
	}
}

func TestServeReturnsWhenListenerClosedWhileContextLive(t *testing.T) {
	lis, err := Bind(testLogger(), "127.0.0.1", 0)
	if err != nil {
		t.Fatalf("Bind returned error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg := &sync.WaitGroup{}
	sessionWg := &sync.WaitGroup{}

	errCh := make(chan error, 1)
	go func() {
		errCh <- Serve(testLogger(), ctx, wg, sessionWg, lis)
	}()

	time.Sleep(50 * time.Millisecond)
	_ = lis.Close()

	select {
	case err := <-errCh:
		if !errors.Is(err, net.ErrClosed) {
			t.Fatalf("Serve returned error %v, want errors.Is(err, net.ErrClosed)", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve did not return on a closed listener — accept loop is spinning")
	}
}

func TestServeIncrementsSessionWaitGroupAtAcceptSite(t *testing.T) {
	lis, err := Bind(testLogger(), "127.0.0.1", 0)
	if err != nil {
		t.Fatalf("Bind returned error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg := &countingWG{}
	sessionWg := &countingWG{}

	go func() {
		_ = Serve(testLogger(), ctx, wg, sessionWg, lis)
	}()

	conn, err := net.Dial("tcp", lis.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for sessionWg.Adds() < 1 {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for sessionWg.Add to be observed after dial")
		}
		time.Sleep(10 * time.Millisecond)
	}

	if err := conn.Close(); err != nil {
		t.Fatalf("close client conn: %v", err)
	}

	deadline = time.Now().Add(2 * time.Second)
	for sessionWg.Dones() < 1 {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for sessionWg.Done to be observed after conn close")
		}
		time.Sleep(10 * time.Millisecond)
	}

	if adds := sessionWg.Adds(); adds != 1 {
		t.Fatalf("sessionWg.Adds() = %d, want 1", adds)
	}
	if dones := sessionWg.Dones(); dones != 1 {
		t.Fatalf("sessionWg.Dones() = %d, want 1", dones)
	}
	if adds := wg.Adds(); adds != 1 {
		t.Fatalf("wg.Adds() = %d, want 1 (Serve brackets its own lifetime once)", adds)
	}
}
