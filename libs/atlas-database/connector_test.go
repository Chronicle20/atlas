package database

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sirupsen/logrus"
)

type fakeConn struct{}

func (fakeConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("not implemented") }
func (fakeConn) Close() error                        { return nil }
func (fakeConn) Begin() (driver.Tx, error)           { return nil, errors.New("not implemented") }

type queryFailConn struct {
	fakeConn
	queries *int
	err     error
}

func (c queryFailConn) QueryContext(_ context.Context, _ string, _ []driver.NamedValue) (driver.Rows, error) {
	*c.queries += 1
	return nil, c.err
}

type fakeConnector struct {
	mu       sync.Mutex
	calls    int
	failures int
	err      error
	conn     driver.Conn
}

func (f *fakeConnector) Connect(context.Context) (driver.Conn, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.calls <= f.failures {
		return nil, f.err
	}
	if f.conn != nil {
		return f.conn, nil
	}
	return fakeConn{}, nil
}

func (f *fakeConnector) Driver() driver.Driver { return nil }

func testLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)
	return l
}

func TestRetryConnectorHealsTransientFailure(t *testing.T) {
	t.Setenv("DB_ACQUIRE_RETRY_INITIAL_DELAY", "1ms")
	t.Setenv("DB_ACQUIRE_RETRY_MAX_DELAY", "5ms")
	base := &fakeConnector{failures: 2, err: &pgconn.PgError{Code: "53300"}}
	before := testutil.ToFloat64(acquireRetriesTotal.WithLabelValues("53300"))

	c := newRetryConnector(testLogger(), base)
	conn, err := c.Connect(context.Background())
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if conn == nil {
		t.Fatal("expected a connection")
	}
	if base.calls != 3 {
		t.Fatalf("expected 3 connect attempts, got %d", base.calls)
	}
	after := testutil.ToFloat64(acquireRetriesTotal.WithLabelValues("53300"))
	if after-before != 2 {
		t.Fatalf("expected 2 retry counter increments, got %v", after-before)
	}
}

func TestRetryConnectorNonTransientFailsImmediately(t *testing.T) {
	base := &fakeConnector{failures: 5, err: errors.New("boom")}
	c := newRetryConnector(testLogger(), base)
	_, err := c.Connect(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if base.calls != 1 {
		t.Fatalf("non-transient error must not be retried, got %d attempts", base.calls)
	}
}

func TestRetryConnectorExhaustsAttempts(t *testing.T) {
	t.Setenv("DB_ACQUIRE_RETRY_INITIAL_DELAY", "1ms")
	t.Setenv("DB_ACQUIRE_RETRY_MAX_DELAY", "5ms")
	base := &fakeConnector{failures: 10, err: &pgconn.PgError{Code: "53300"}}
	c := newRetryConnector(testLogger(), base)
	_, err := c.Connect(context.Background())
	if err == nil {
		t.Fatal("expected exhaustion error")
	}
	if base.calls != 3 {
		t.Fatalf("expected exactly 3 attempts (default), got %d", base.calls)
	}
}

func TestRetryConnectorDisabledPassesThrough(t *testing.T) {
	t.Setenv("DB_ACQUIRE_RETRY_ATTEMPTS", "0")
	base := &fakeConnector{failures: 1, err: &pgconn.PgError{Code: "53300"}}
	c := newRetryConnector(testLogger(), base)
	if c != driver.Connector(base) {
		t.Fatal("attempts<=1 must return the base connector unwrapped")
	}
	_, err := c.Connect(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if base.calls != 1 {
		t.Fatalf("expected single attempt with retry disabled, got %d", base.calls)
	}
}

// The mid-statement safety property: an error raised during statement
// execution flows through database/sql, never through Connector.Connect, so
// the wrapper cannot retry it even when it looks transient.
func TestMidStatementErrorIsNotRetried(t *testing.T) {
	queries := 0
	base := &fakeConnector{conn: queryFailConn{queries: &queries, err: &pgconn.PgError{Code: "53300"}}}
	db := sql.OpenDB(newRetryConnector(testLogger(), base))
	defer db.Close()

	_, err := db.QueryContext(context.Background(), "SELECT 1")
	if err == nil {
		t.Fatal("expected query error")
	}
	if queries != 1 {
		t.Fatalf("mid-statement error must execute exactly once, got %d", queries)
	}
	if base.calls != 1 {
		t.Fatalf("expected a single connection acquire, got %d", base.calls)
	}
}
