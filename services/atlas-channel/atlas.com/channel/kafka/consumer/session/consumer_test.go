package session

import (
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	testlog "github.com/sirupsen/logrus/hooks/test"

	reportcb "github.com/Chronicle20/atlas/libs/atlas-packet/report/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	socketwriter "github.com/Chronicle20/atlas/libs/atlas-socket/writer"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// --- claimEnableAnnouncer tests: task-17 (atlas-channel claim-enable
// emission on session bootstrap). processStateReturn itself depends on
// several live-service processors (character, buddylist, location, guild,
// key, buff, world, macro, note) that aren't practical to stand up in a
// unit test, so - mirroring the reportAnnouncer seam in
// kafka/consumer/report/consumer.go - the claim-enable send is exercised
// directly via claimEnableAnnouncer against a real session.Model, rather
// than through the full bootstrap goroutine chain. ---

func newTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

// TestClaimEnableAnnouncer_WriterNotFound_SkipsWithoutError proves the
// writer-not-found skip: a tenant whose config lacks the ClaimSvrStatusChanged
// writer (v61 sue-only, jms, gms-92) must complete cleanly - debug log only,
// no error/warn level entry, no second write attempted.
func TestClaimEnableAnnouncer_WriterNotFound_SkipsWithoutError(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)

	serverConn, clientConn := net.Pipe()
	defer func() { _ = serverConn.Close() }()
	defer func() { _ = clientConn.Close() }()

	s := session.NewSession(uuid.New(), tm, 0, serverConn)
	characterId := uint32(6001)

	logger, hook := testlog.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	notFound := errors.New("writer not found")
	wp := writer.Producer(func(_ string) (socketwriter.BodyFunc, error) {
		return nil, notFound
	})

	done := make(chan struct{})
	go func() {
		claimEnableAnnouncer(logger, ctx, wp, s, characterId)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for claimEnableAnnouncer to return")
	}

	for _, e := range hook.AllEntries() {
		if e.Level <= logrus.WarnLevel {
			t.Fatalf("writer-not-found must not error/warn (config presence is the feature gate); got %s-level entry: %q", e.Level, e.Message)
		}
	}

	foundDebugSkip := false
	for _, e := range hook.AllEntries() {
		if e.Level == logrus.DebugLevel {
			foundDebugSkip = true
		}
	}
	if !foundDebugSkip {
		t.Fatal("want a debug-level skip log entry when the tenant has no mapped claim writer, got none")
	}
}

// TestClaimEnableAnnouncer_WriterFound_DeliversBothPackets is the positive
// control for the skip-path test above: with a writer.Producer that resolves
// both claim writers and a session backed by a real net.Conn (net.Pipe), the
// real claimEnableAnnouncer must write TWO packets to the connection -
// ClaimSvrStatusChanged then ClaimAvailableTime - proving the skip-path test
// above exercises a genuine short-circuit, not a fixture that never delivers
// regardless of writer availability.
func TestClaimEnableAnnouncer_WriterFound_DeliversBothPackets(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)

	serverConn, clientConn := net.Pipe()
	defer func() { _ = serverConn.Close() }()
	defer func() { _ = clientConn.Close() }()

	s := session.NewSession(uuid.New(), tm, 0, serverConn)
	characterId := uint32(6002)

	logger, hook := testlog.NewNullLogger()

	var gotWriters []string
	wp := writer.Producer(func(writerName string) (socketwriter.BodyFunc, error) {
		gotWriters = append(gotWriters, writerName)
		return socketwriter.BodyFunc(func(_ logrus.FieldLogger, _ context.Context) func(packet.Encode) []byte {
			return func(_ packet.Encode) []byte { return []byte{0x01, 0x02, 0x03} }
		}), nil
	})

	reads := make(chan int, 2)
	readErrs := make(chan error, 2)
	go func() {
		for i := 0; i < 2; i++ {
			buf := make([]byte, 64)
			n, err := clientConn.Read(buf)
			reads <- n
			readErrs <- err
		}
	}()

	done := make(chan struct{})
	go func() {
		claimEnableAnnouncer(logger, ctx, wp, s, characterId)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for claimEnableAnnouncer to return")
	}

	for i := 0; i < 2; i++ {
		select {
		case n := <-reads:
			if err := <-readErrs; err != nil {
				t.Fatalf("read %d from session connection: %v", i, err)
			}
			if n == 0 {
				t.Fatalf("read %d: want bytes written to the real session connection, got none", i)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for write %d to the session connection", i)
		}
	}

	// claimEnableAnnouncer pre-checks the status writer's availability before
	// sending anything (the writer-not-found skip below relies on this), so
	// ClaimSvrStatusChangedWriter is looked up twice - once for the
	// pre-check, once inside session.Announce's own writerProducer call -
	// before ClaimAvailableTimeWriter is looked up once. This mirrors
	// reportAnnouncer's identical double-lookup shape
	// (kafka/consumer/report/consumer.go).
	wantWriters := []string{reportcb.ClaimSvrStatusChangedWriter, reportcb.ClaimSvrStatusChangedWriter, reportcb.ClaimAvailableTimeWriter}
	if len(gotWriters) != len(wantWriters) {
		t.Fatalf("want %d writer lookups %v, got %d: %v", len(wantWriters), wantWriters, len(gotWriters), gotWriters)
	}
	for i, want := range wantWriters {
		if gotWriters[i] != want {
			t.Fatalf("writer lookup %d: want %s, got %s", i, want, gotWriters[i])
		}
	}

	for _, e := range hook.AllEntries() {
		if e.Level <= logrus.WarnLevel {
			t.Fatalf("writer-found delivery must not error/warn; got %s-level entry: %q", e.Level, e.Message)
		}
	}
}
