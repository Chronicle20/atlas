# DB Connection Resilience Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Harden the shared libraries so transient DB pool exhaustion is retried at the acquire phase, surfaced as 503 (not 500), ridden out by the REST client, and never silently degrades a decorator — with every absorption step visible in Prometheus.

**Architecture:** Four existing lib modules gain additive-only APIs: `atlas-retry` gets a delay-hint hook, `atlas-database` gets a transient-error classifier + a retrying `driver.Connector` wrapper + DB metrics, `atlas-rest` gets a server-side 503 mapping helper, client GET retry-on-503, an auto-mounted `/metrics` endpoint, and a `degrade` observer package, and `atlas-model` gets an `ErrDecorator` combinator. atlas-inventory and atlas-login adopt these as reference implementations; a fleet-wide decorator audit fixes remaining silent-degrade sites; docs + reviewer checklist make the patterns enforceable.

**Tech Stack:** Go 1.25 workspace monorepo, gorm + pgx/v5, prometheus client_golang v1.23.2, logrus, gorilla/mux, api2go JSON:API, httptest-based tests.

## Global Constraints

- Work happens in worktree `.worktrees/task-168-db-connection-resilience` on branch `task-168-db-connection-resilience`. Verify `git branch --show-current` before every commit.
- Lib API changes are **additive only** — no existing exported signature changes.
- **No new lib module** is created; no root `Dockerfile` or `go.work` edits.
- Metric names are fixed: `atlas_db_acquire_retries_total{sqlstate}`, `atlas_db_transient_errors_total{sqlstate}`, `atlas_rest_client_retries_total{reason}`, `atlas_enrichment_degraded_total{component}`, plus the stock `go_sql_*` DBStats family with a `db_name` label. **No tenant id or per-character/entity labels ever.**
- Env knobs and defaults are fixed: `DB_ACQUIRE_RETRY_ATTEMPTS=3`, `DB_ACQUIRE_RETRY_INITIAL_DELAY=100ms`, `DB_ACQUIRE_RETRY_MAX_DELAY=400ms`. `0` or `1` attempts disables DB-side retry.
- 503 contract: `Retry-After: 1` (constant `TransientRetryAfterSeconds = 1`), JSON:API body `{"errors":[{"status":"503","title":"temporarily unavailable"}]}`.
- Client retry: **GET only**. GET default attempts 1 → 3; GET retry backoff initial 200ms, MaxDelay **2s** (down from 5s). POST/PATCH/PUT/DELETE untouched. No status other than 503 becomes retryable.
- prometheus dependency version: `github.com/prometheus/client_golang v1.23.2` (matches `libs/atlas-lock`).
- Tests use in-package test files and the project's Builder pattern; **no `*_testhelpers.go` files**.
- Never run `go work sync`. Run `go mod tidy` only after the import that needs it exists (see memory: go-workspace footguns).
- Commit after every task with message prefix `feat(task-168): ...` (docs task uses `docs(task-168): ...`).

---

### Task 1: `retry.WithDelayHint` hook in `libs/atlas-retry`

**Files:**
- Modify: `libs/atlas-retry/retry.go`
- Test: `libs/atlas-retry/retry_test.go` (append)

**Interfaces:**
- Consumes: nothing new.
- Produces: `func WithDelayHint(err error, d time.Duration) error` — wraps `err` so `Try` waits at least `d` (capped at `cfg.MaxDelay`) before the next attempt. The wrapper implements `Unwrap() error`, so `errors.Is`/`errors.As` on the original error still work. Task 6 depends on this exact name.

- [ ] **Step 1: Write the failing tests**

Append to `libs/atlas-retry/retry_test.go`:

```go
func TestWithDelayHintRespected(t *testing.T) {
	cfg := DefaultConfig().WithMaxRetries(2).WithInitialDelay(1 * time.Millisecond).WithMaxDelay(1 * time.Second)
	sentinel := errors.New("sentinel")
	start := time.Now()
	err := Try(context.Background(), cfg, func(attempt int) (bool, error) {
		if attempt == 1 {
			return true, WithDelayHint(sentinel, 300*time.Millisecond)
		}
		return false, nil
	})
	if err != nil {
		t.Fatalf("expected success on second attempt, got: %v", err)
	}
	if elapsed := time.Since(start); elapsed < 300*time.Millisecond {
		t.Fatalf("hint not respected, elapsed %v < 300ms", elapsed)
	}
}

func TestWithDelayHintCappedAtMaxDelay(t *testing.T) {
	cfg := DefaultConfig().WithMaxRetries(2).WithInitialDelay(1 * time.Millisecond).WithMaxDelay(200 * time.Millisecond)
	start := time.Now()
	_ = Try(context.Background(), cfg, func(attempt int) (bool, error) {
		if attempt == 1 {
			return true, WithDelayHint(errors.New("x"), 10*time.Second)
		}
		return false, nil
	})
	if elapsed := time.Since(start); elapsed > 1*time.Second {
		t.Fatalf("hint not capped at MaxDelay, elapsed %v", elapsed)
	}
}

func TestWithDelayHintPreservesErrorsIs(t *testing.T) {
	sentinel := errors.New("sentinel")
	if !errors.Is(WithDelayHint(sentinel, time.Second), sentinel) {
		t.Fatal("WithDelayHint broke the errors.Is chain")
	}
}
```

Add `"errors"` to the test file's imports if missing.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd libs/atlas-retry && go test ./... -run TestWithDelayHint -v`
Expected: FAIL — `undefined: WithDelayHint`

- [ ] **Step 3: Implement `WithDelayHint` and honor it in `Try`**

In `libs/atlas-retry/retry.go`, add `"errors"` to imports, then append:

```go
type delayHintError struct {
	err   error
	delay time.Duration
}

func (e *delayHintError) Error() string { return e.err.Error() }
func (e *delayHintError) Unwrap() error { return e.err }

// WithDelayHint wraps err so Try waits at least d (capped at cfg.MaxDelay)
// before the next attempt, instead of the jittered backoff when that is
// smaller. Use it to honor server-provided hints such as Retry-After.
func WithDelayHint(err error, d time.Duration) error {
	return &delayHintError{err: err, delay: d}
}
```

Then in `Try`, replace the line `delay := jitteredDelay(cfg, attempt)` with:

```go
		delay := jitteredDelay(cfg, attempt)
		var hint *delayHintError
		if errors.As(err, &hint) && hint.delay > delay {
			delay = hint.delay
			if delay > cfg.MaxDelay {
				delay = cfg.MaxDelay
			}
		}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-retry && go test -race ./... -v`
Expected: PASS (all, including pre-existing tests)

- [ ] **Step 5: Vet and commit**

```bash
cd libs/atlas-retry && go vet ./...
git add libs/atlas-retry/retry.go libs/atlas-retry/retry_test.go
git commit -m "feat(task-168): add retry.WithDelayHint for server-provided retry delays"
```

---

### Task 2: Transient classifier in `libs/atlas-database`

**Files:**
- Create: `libs/atlas-database/transient.go`
- Test: `libs/atlas-database/transient_test.go`
- Modify: `libs/atlas-database/go.mod` (pgx/v5 becomes a direct require — `go mod tidy` does it)

**Interfaces:**
- Consumes: `github.com/jackc/pgx/v5/pgconn` (already in the module graph as indirect).
- Produces: `func IsTransientConnectionError(err error) bool` and `func TransientSQLState(err error) string` — used by Tasks 3, 4, 11 and the docs.

- [ ] **Step 1: Write the failing table-driven test**

Create `libs/atlas-database/transient_test.go`:

```go
package database

import (
	"context"
	"errors"
	"fmt"
	"net"
	"syscall"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// connectRefusedError produces a real *pgconn.ConnectError by dialing a
// closed loopback port (pgconn.ConnectError's err field is unexported, so it
// cannot be constructed literally).
func connectRefusedError(t *testing.T) error {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := pgconn.Connect(ctx, "postgres://user:pass@127.0.0.1:1/db")
	if err == nil {
		t.Fatal("expected connection to closed port to fail")
	}
	return err
}

func TestIsTransientConnectionError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"53300 too_many_connections", &pgconn.PgError{Code: "53300"}, true},
		{"57P03 cannot_connect_now", &pgconn.PgError{Code: "57P03"}, true},
		{"08001 connect failure", &pgconn.PgError{Code: "08001"}, true},
		{"08006 connection failure", &pgconn.PgError{Code: "08006"}, true},
		{"wrapped 53300", fmt.Errorf("acquire: %w", &pgconn.PgError{Code: "53300"}), true},
		{"net dial op error", &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}, true},
		{"wrapped ECONNREFUSED", fmt.Errorf("x: %w", syscall.ECONNREFUSED), true},
		{"wrapped ECONNRESET", fmt.Errorf("x: %w", syscall.ECONNRESET), true},
		{"23505 unique violation", &pgconn.PgError{Code: "23505"}, false},
		{"40001 serialization failure", &pgconn.PgError{Code: "40001"}, false},
		{"57014 statement timeout", &pgconn.PgError{Code: "57014"}, false},
		{"28P01 auth failure", &pgconn.PgError{Code: "28P01"}, false},
		{"bare context deadline", context.DeadlineExceeded, false},
		{"gorm record not found", gorm.ErrRecordNotFound, false},
		{"generic error", errors.New("boom"), false},
		{"net read op error", &net.OpError{Op: "read", Net: "tcp", Err: syscall.ECONNRESET}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsTransientConnectionError(tc.err); got != tc.want {
				t.Fatalf("IsTransientConnectionError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestIsTransientConnectionErrorConnectError(t *testing.T) {
	err := connectRefusedError(t)
	var ce *pgconn.ConnectError
	if !errors.As(err, &ce) {
		t.Skipf("dial to closed port did not yield *pgconn.ConnectError: %v", err)
	}
	if !IsTransientConnectionError(err) {
		t.Fatalf("real ConnectError not classified transient: %v", err)
	}
}

func TestTransientSQLState(t *testing.T) {
	if got := TransientSQLState(&pgconn.PgError{Code: "53300"}); got != "53300" {
		t.Fatalf("want 53300, got %q", got)
	}
	if got := TransientSQLState(&net.OpError{Op: "dial", Err: syscall.ECONNREFUSED}); got != "" {
		t.Fatalf("want empty for dial-shape error, got %q", got)
	}
	if got := TransientSQLState(&pgconn.PgError{Code: "23505"}); got != "" {
		t.Fatalf("want empty for non-transient SQLSTATE, got %q", got)
	}
}
```

Note the `net read op error → true` row: a `*net.OpError` with `Op == "read"` is NOT matched by the dial check, but its inner `syscall.ECONNRESET` matches the `errors.Is` check. That is intended — a reset is connection-level, and the connector wrapper (Task 4) only ever sees acquire-phase errors anyway.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-database && go test ./... -run 'TestIsTransient|TestTransientSQLState' -v`
Expected: FAIL — `undefined: IsTransientConnectionError`

- [ ] **Step 3: Implement the classifier**

Create `libs/atlas-database/transient.go`:

```go
package database

import (
	"errors"
	"net"
	"syscall"

	"github.com/jackc/pgx/v5/pgconn"
)

// transientSQLStates are acquire-phase SQLSTATEs that are safe to retry: the
// server rejected the connection before any statement was sent. Any SQLSTATE
// produced after a statement began executing is deliberately absent.
var transientSQLStates = map[string]bool{
	"53300": true, // too_many_connections / reserved connection slots
	"57P03": true, // cannot_connect_now (server starting up / shutting down)
	"08001": true, // sqlclient_unable_to_establish_sqlconnection
	"08006": true, // connection_failure during establishment
}

// IsTransientConnectionError reports whether err is a connection-acquire-phase
// failure that is safe to retry (no statement was ever sent). Coded server
// errors are classified strictly by SQLSTATE — checked before the connect-error
// shape so an auth failure raised during connect is NOT transient. A bare
// context.DeadlineExceeded is ambiguous (could be mid-query) and is NOT
// transient; a deadline inside a *pgconn.ConnectError IS, because the connect
// provably never completed.
func IsTransientConnectionError(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return transientSQLStates[pgErr.Code]
	}
	var connectErr *pgconn.ConnectError
	if errors.As(err, &connectErr) {
		return true
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) && opErr.Op == "dial" {
		return true
	}
	if errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.ECONNRESET) {
		return true
	}
	return false
}

// TransientSQLState returns the SQLSTATE that classified err transient, or ""
// when classification came from a dial/connect error shape. Used for metric
// labels.
func TransientSQLState(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && transientSQLStates[pgErr.Code] {
		return pgErr.Code
	}
	return ""
}
```

- [ ] **Step 4: Tidy and run tests**

```bash
cd libs/atlas-database && go mod tidy && go test -race ./... -v
```
Expected: PASS. `go.mod` now lists `github.com/jackc/pgx/v5 v5.7.4` as a direct require.

- [ ] **Step 5: Vet and commit**

```bash
cd libs/atlas-database && go vet ./...
git add libs/atlas-database/transient.go libs/atlas-database/transient_test.go libs/atlas-database/go.mod libs/atlas-database/go.sum
git commit -m "feat(task-168): transient DB connection error classifier"
```

---

### Task 3: DB metrics counters in `libs/atlas-database`

**Files:**
- Create: `libs/atlas-database/metrics.go`
- Test: `libs/atlas-database/metrics_test.go`
- Modify: `libs/atlas-database/go.mod` (gains `github.com/prometheus/client_golang v1.23.2`)

**Interfaces:**
- Consumes: `TransientSQLState` (Task 2).
- Produces: package-level `acquireRetriesTotal` counter vec (used by Task 4), exported `func CountTransient(err error)` (used by Task 4 and by service main.go classifier registration in Task 11), unexported `func registerDBStats(l logrus.FieldLogger, db *sql.DB, dbName string)` (used by Task 4).

- [ ] **Step 1: Write the failing test**

Create `libs/atlas-database/metrics_test.go`:

```go
package database

import (
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestCountTransientLabelsBySQLState(t *testing.T) {
	before := testutil.ToFloat64(transientErrorsTotal.WithLabelValues("53300"))
	CountTransient(&pgconn.PgError{Code: "53300"})
	after := testutil.ToFloat64(transientErrorsTotal.WithLabelValues("53300"))
	if after-before != 1 {
		t.Fatalf("expected counter delta 1, got %v", after-before)
	}
}

func TestCountTransientDialShapeUsesEmptyLabel(t *testing.T) {
	before := testutil.ToFloat64(transientErrorsTotal.WithLabelValues(""))
	CountTransient(&pgconn.PgError{Code: "53300"}) // wrong label, should not affect ""
	beforeAfterWrong := testutil.ToFloat64(transientErrorsTotal.WithLabelValues(""))
	if beforeAfterWrong != before {
		t.Fatalf("unexpected empty-label increment")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-database && go test ./... -run TestCountTransient -v`
Expected: FAIL — `undefined: transientErrorsTotal` / `undefined: CountTransient`

- [ ] **Step 3: Implement metrics.go**

Create `libs/atlas-database/metrics.go`:

```go
package database

import (
	"database/sql"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

var (
	acquireRetriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_db_acquire_retries_total",
			Help: "Number of retried transient connection-acquire failures, by SQLSTATE (empty label for dial-shape errors).",
		},
		[]string{"sqlstate"},
	)

	transientErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_db_transient_errors_total",
			Help: "Number of errors classified as transient connection failures, by SQLSTATE (empty label for dial-shape errors).",
		},
		[]string{"sqlstate"},
	)
)

// CountTransient increments the transient-error counter for err. Call only
// after IsTransientConnectionError(err) has returned true.
func CountTransient(err error) {
	transientErrorsTotal.WithLabelValues(TransientSQLState(err)).Inc()
}

// registerDBStats exposes the standard sql.DBStats gauge family (go_sql_*)
// for db on the default Prometheus registry. Registration failure (e.g. a
// duplicate registration in tests) logs a warning and continues.
func registerDBStats(l logrus.FieldLogger, db *sql.DB, dbName string) {
	if err := prometheus.DefaultRegisterer.Register(collectors.NewDBStatsCollector(db, dbName)); err != nil {
		l.WithError(err).Warnf("Unable to register DB stats collector.")
	}
}
```

- [ ] **Step 4: Tidy, test, vet**

```bash
cd libs/atlas-database && go mod tidy && go test -race ./... && go vet ./...
```
Expected: PASS; `go.mod` gains `github.com/prometheus/client_golang v1.23.2`.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-database/metrics.go libs/atlas-database/metrics_test.go libs/atlas-database/go.mod libs/atlas-database/go.sum
git commit -m "feat(task-168): DB transient/acquire-retry counters and DBStats registration helper"
```

---

### Task 4: Acquire-phase retry connector + `Connect()` swap

**Files:**
- Create: `libs/atlas-database/connector.go`
- Test: `libs/atlas-database/connector_test.go`
- Modify: `libs/atlas-database/connection.go` (the `Connect` function)
- Modify: `libs/atlas-database/go.mod` (gains `github.com/Chronicle20/atlas/libs/atlas-retry` require + replace)

**Interfaces:**
- Consumes: `retry.Try`, `retry.DefaultConfig` (existing atlas-retry API), `IsTransientConnectionError`/`TransientSQLState` (Task 2), `acquireRetriesTotal`/`CountTransient`/`registerDBStats` (Task 3), existing `getIntEnv`/`getDurationEnv` in connection.go.
- Produces: `func newRetryConnector(l logrus.FieldLogger, base driver.Connector) driver.Connector` (unexported; used only by `Connect`). `Connect`'s exported signature is unchanged.

- [ ] **Step 1: Add the atlas-retry dependency**

In `libs/atlas-database/go.mod`, add to the `require` block:

```
github.com/Chronicle20/atlas/libs/atlas-retry v0.0.0
```

and at the bottom (alongside the existing replaces):

```
replace github.com/Chronicle20/atlas/libs/atlas-retry => ../atlas-retry
```

- [ ] **Step 2: Write the failing connector tests**

Create `libs/atlas-database/connector_test.go`:

```go
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
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `cd libs/atlas-database && go test ./... -run TestRetryConnector -v`
Expected: FAIL — `undefined: newRetryConnector`

- [ ] **Step 4: Implement the connector wrapper**

Create `libs/atlas-database/connector.go`:

```go
package database

import (
	"context"
	"database/sql/driver"
	"time"

	retry "github.com/Chronicle20/atlas/libs/atlas-retry"
	"github.com/sirupsen/logrus"
)

type retryConnector struct {
	l        logrus.FieldLogger
	base     driver.Connector
	attempts int
	cfg      retry.Config
}

// newRetryConnector wraps base so transient acquire-phase failures (per
// IsTransientConnectionError) are retried with jittered backoff.
// database/sql invokes Connector.Connect only when the pool needs a new
// physical connection — before any SQL is sent on it — so nothing retried
// here can double-apply work. DB_ACQUIRE_RETRY_ATTEMPTS <= 1 disables the
// wrapper entirely.
func newRetryConnector(l logrus.FieldLogger, base driver.Connector) driver.Connector {
	attempts := getIntEnv("DB_ACQUIRE_RETRY_ATTEMPTS", 3)
	if attempts <= 1 {
		return base
	}
	cfg := retry.DefaultConfig().
		WithMaxRetries(attempts).
		WithInitialDelay(getDurationEnv("DB_ACQUIRE_RETRY_INITIAL_DELAY", 100*time.Millisecond)).
		WithMaxDelay(getDurationEnv("DB_ACQUIRE_RETRY_MAX_DELAY", 400*time.Millisecond))
	return &retryConnector{l: l, base: base, attempts: attempts, cfg: cfg}
}

func (c *retryConnector) Connect(ctx context.Context) (driver.Conn, error) {
	var conn driver.Conn
	err := retry.Try(ctx, c.cfg, func(attempt int) (bool, error) {
		var err error
		conn, err = c.base.Connect(ctx)
		if err == nil {
			return false, nil
		}
		if !IsTransientConnectionError(err) {
			return false, err
		}
		CountTransient(err)
		if attempt < c.attempts {
			acquireRetriesTotal.WithLabelValues(TransientSQLState(err)).Inc()
			c.l.WithError(err).Warnf("Transient DB connection acquire failure (SQLSTATE [%s]); retrying.", TransientSQLState(err))
		}
		return true, err
	})
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (c *retryConnector) Driver() driver.Driver { return c.base.Driver() }
```

- [ ] **Step 5: Run connector tests**

Run: `cd libs/atlas-database && go test -race ./... -run TestRetryConnector -v` and `-run TestMidStatement`
Expected: PASS

- [ ] **Step 6: Swap `Connect()` onto the wrapped connector**

In `libs/atlas-database/connection.go`, add imports `"database/sql"`, `pgx "github.com/jackc/pgx/v5"`, `"github.com/jackc/pgx/v5/stdlib"`. Replace the body of `Connect` from `var db *gorm.DB` through the `registerTenantCallbacks(l, db)` line with:

```go
	pgxCfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		l.WithError(err).Fatalf("Failed to parse database DSN.")
	}

	var db *gorm.DB
	var sqlDB *sql.DB
	tryToConnect := func(attempt int) (bool, error) {
		sqlDB = sql.OpenDB(newRetryConnector(l, stdlib.GetConnector(*pgxCfg)))
		sqlDB.SetMaxOpenConns(getIntEnv("DB_MAX_OPEN_CONNS", 10))
		sqlDB.SetMaxIdleConns(getIntEnv("DB_MAX_IDLE_CONNS", 5))
		sqlDB.SetConnMaxLifetime(getDurationEnv("DB_CONN_MAX_LIFETIME", 5*time.Minute))
		sqlDB.SetConnMaxIdleTime(getDurationEnv("DB_CONN_MAX_IDLE_TIME", 3*time.Minute))

		var errOpen error
		db, errOpen = gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
		if errOpen != nil {
			_ = sqlDB.Close()
			return true, errOpen
		}
		return false, nil
	}

	err = try(tryToConnect, 10)
	if err != nil {
		l.WithError(err).Fatalf("Failed to connect to database.")
	}

	registerDBStats(l, sqlDB, os.Getenv("DB_NAME"))
	registerTenantCallbacks(l, db)
```

Notes: `gorm.Open` pings by default (`DisableAutomaticPing` is false), so the existing 10×1s bootstrap loop still guards startup, and that ping itself flows through the retry connector. The migrations loop and the rest of `Connect` are unchanged. The old pool-knob block inside the previous closure is gone (moved above `gorm.Open`).

- [ ] **Step 7: Full module test, tidy, vet**

```bash
cd libs/atlas-database && go mod tidy && go test -race ./... && go vet ./... && go build ./...
```
Expected: PASS/clean. `tenant_scope_test.go` (sqlite-based) must still pass — it does not go through `Connect`.

- [ ] **Step 8: Commit**

```bash
git add libs/atlas-database/connector.go libs/atlas-database/connector_test.go libs/atlas-database/connection.go libs/atlas-database/go.mod libs/atlas-database/go.sum
git commit -m "feat(task-168): acquire-phase DB retry via driver.Connector wrapper + DBStats gauges"
```

---

### Task 5: Server-side 503 contract in `libs/atlas-rest/server`

**Files:**
- Create: `libs/atlas-rest/server/error.go`
- Test: `libs/atlas-rest/server/error_test.go`

**Interfaces:**
- Consumes: nothing new (deliberately NO import of atlas-database — the classifier is injected).
- Produces: `const TransientRetryAfterSeconds = 1`, `func RegisterTransientErrorClassifier(f func(error) bool)`, `func WriteErrorResponse(l logrus.FieldLogger) func(w http.ResponseWriter) func(err error)`. Used by Task 11 and documented in Task 13.

- [ ] **Step 1: Write the failing tests**

Create `libs/atlas-rest/server/error_test.go`:

```go
package server

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func quietLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)
	return l
}

func TestWriteErrorResponseNoClassifierIs500(t *testing.T) {
	RegisterTransientErrorClassifier(nil)
	rec := httptest.NewRecorder()
	WriteErrorResponse(quietLogger())(rec)(errors.New("boom"))
	if rec.Code != 500 {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"status":"500"`) {
		t.Fatalf("expected JSON:API 500 body, got %s", rec.Body.String())
	}
}

func TestWriteErrorResponseTransientIs503(t *testing.T) {
	RegisterTransientErrorClassifier(func(error) bool { return true })
	defer RegisterTransientErrorClassifier(nil)
	rec := httptest.NewRecorder()
	WriteErrorResponse(quietLogger())(rec)(errors.New("pool exhausted"))
	if rec.Code != 503 {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
	if got := rec.Header().Get("Retry-After"); got != "1" {
		t.Fatalf("expected Retry-After: 1, got %q", got)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"status":"503"`) || !strings.Contains(body, `"title":"temporarily unavailable"`) {
		t.Fatalf("unexpected 503 body: %s", body)
	}
}

func TestWriteErrorResponseNonTransientIs500(t *testing.T) {
	RegisterTransientErrorClassifier(func(error) bool { return false })
	defer RegisterTransientErrorClassifier(nil)
	rec := httptest.NewRecorder()
	WriteErrorResponse(quietLogger())(rec)(errors.New("real bug"))
	if rec.Code != 500 {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	if got := rec.Header().Get("Retry-After"); got != "" {
		t.Fatalf("500 must not carry Retry-After, got %q", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd libs/atlas-rest && go test ./server/... -run TestWriteErrorResponse -v`
Expected: FAIL — `undefined: RegisterTransientErrorClassifier`

- [ ] **Step 3: Implement error.go**

Create `libs/atlas-rest/server/error.go`:

```go
package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync/atomic"

	"github.com/sirupsen/logrus"
)

// TransientRetryAfterSeconds is the Retry-After value (seconds) sent with 503
// responses produced by WriteErrorResponse for transient errors.
const TransientRetryAfterSeconds = 1

var transientClassifier atomic.Pointer[func(error) bool]

// RegisterTransientErrorClassifier installs the process-wide predicate used
// by WriteErrorResponse to map errors to 503 Service Unavailable. Typically
// called once from main.go:
//
//	server.RegisterTransientErrorClassifier(func(err error) bool {
//		if database.IsTransientConnectionError(err) {
//			database.CountTransient(err)
//			return true
//		}
//		return false
//	})
//
// Passing nil clears the classifier (everything maps to 500).
func RegisterTransientErrorClassifier(f func(error) bool) {
	transientClassifier.Store(&f)
}

type errorObject struct {
	Status string `json:"status"`
	Title  string `json:"title"`
}

type errorDocument struct {
	Errors []errorObject `json:"errors"`
}

// WriteErrorResponse maps err to a JSON:API error response. Errors the
// registered classifier reports as transient produce
// 503 + Retry-After: TransientRetryAfterSeconds; everything else produces
// 500. With no classifier registered, every error maps to 500.
func WriteErrorResponse(l logrus.FieldLogger) func(w http.ResponseWriter) func(err error) {
	return func(w http.ResponseWriter) func(err error) {
		return func(err error) {
			status := http.StatusInternalServerError
			title := "internal server error"
			if fp := transientClassifier.Load(); fp != nil && *fp != nil && (*fp)(err) {
				status = http.StatusServiceUnavailable
				title = "temporarily unavailable"
				w.Header().Set("Retry-After", strconv.Itoa(TransientRetryAfterSeconds))
			}
			l.WithError(err).Warnf("Writing [%d] error response.", status)
			w.WriteHeader(status)
			doc := errorDocument{Errors: []errorObject{{Status: strconv.Itoa(status), Title: title}}}
			if encodeErr := json.NewEncoder(w).Encode(doc); encodeErr != nil {
				l.WithError(encodeErr).Errorf("Encoding error response body.")
			}
		}
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-rest && go test -race ./server/... -v`
Expected: PASS (all, including pre-existing handler/pagination tests)

- [ ] **Step 5: Vet and commit**

```bash
cd libs/atlas-rest && go vet ./...
git add libs/atlas-rest/server/error.go libs/atlas-rest/server/error_test.go
git commit -m "feat(task-168): server-side transient-error 503 contract (WriteErrorResponse)"
```

---

### Task 6: Client GET retry on 503 in `libs/atlas-rest/requests`

**Files:**
- Modify: `libs/atlas-rest/requests/get.go`
- Create: `libs/atlas-rest/requests/metrics.go`
- Test: `libs/atlas-rest/requests/get_retry_test.go`
- Modify: `libs/atlas-rest/go.mod` (gains `github.com/prometheus/client_golang v1.23.2`)

**Interfaces:**
- Consumes: `retry.WithDelayHint` (Task 1).
- Produces: exported `var ErrServiceUnavailable = errors.New("service unavailable")` (sibling of `ErrBadRequest`/`ErrNotFound`; callers use `errors.Is`). GET default attempts change 1 → 3; GET backoff MaxDelay 5s → 2s. Non-GET verbs (`post.go`, `patch.go`, `put.go`, `delete.go`) are NOT modified.

- [ ] **Step 1: Write the failing tests**

Create `libs/atlas-rest/requests/get_retry_test.go`:

```go
package requests

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

type retryTestRestModel struct {
	Id   string `json:"-"`
	Name string `json:"name"`
}

func (r retryTestRestModel) GetName() string { return "tests" }
func (r retryTestRestModel) GetID() string   { return r.Id }
func (r *retryTestRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func retryQuietLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)
	return l
}

func TestGetRetries503ThenSucceeds(t *testing.T) {
	body, err := jsonapi.Marshal(retryTestRestModel{Id: "1", Name: "x"})
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	res, err := get[retryTestRestModel](retryQuietLogger(), context.Background())(srv.URL)
	if err != nil {
		t.Fatalf("expected success after 503 retry, got: %v", err)
	}
	if res.Name != "x" {
		t.Fatalf("unexpected response: %+v", res)
	}
	if attempts.Load() != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts.Load())
	}
}

func TestGetExhausted503ReturnsSentinel(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	_, err := get[retryTestRestModel](retryQuietLogger(), context.Background())(srv.URL)
	if !errors.Is(err, ErrServiceUnavailable) {
		t.Fatalf("expected ErrServiceUnavailable, got: %v", err)
	}
	if attempts.Load() != 3 {
		t.Fatalf("expected 3 attempts (new GET default), got %d", attempts.Load())
	}
}

func TestGetHonorsRetryAfter(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusNotFound) // terminal, ends the request quickly
	}))
	defer srv.Close()

	start := time.Now()
	_, err := get[retryTestRestModel](retryQuietLogger(), context.Background())(srv.URL)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound terminal, got: %v", err)
	}
	if elapsed := time.Since(start); elapsed < 1*time.Second {
		t.Fatalf("Retry-After not honored, elapsed %v", elapsed)
	}
}

func TestGetCapsRetryAfterAtMaxDelay(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) == 1 {
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	start := time.Now()
	_, _ = get[retryTestRestModel](retryQuietLogger(), context.Background())(srv.URL)
	if elapsed := time.Since(start); elapsed > 4*time.Second {
		t.Fatalf("Retry-After not capped at 2s MaxDelay, elapsed %v", elapsed)
	}
}

func TestDelete503IsNotRetried(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	err := delete(retryQuietLogger(), context.Background())(srv.URL)
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, ErrServiceUnavailable) {
		t.Fatalf("non-GET must not map to ErrServiceUnavailable, got: %v", err)
	}
	if attempts.Load() != 1 {
		t.Fatalf("non-GET 503 must not be retried, got %d attempts", attempts.Load())
	}
}

// Regression matrix: every non-503 status behaves exactly as before —
// single attempt, same error identity.
func TestGetNon503StatusesUnchanged(t *testing.T) {
	tests := []struct {
		status  int
		want    error // nil means "any non-nil generic error" for 500
		attempt int32
	}{
		{http.StatusBadRequest, ErrBadRequest, 1},
		{http.StatusNotFound, ErrNotFound, 1},
		{http.StatusInternalServerError, nil, 1},
		{http.StatusBadGateway, nil, 1},
	}
	for _, tc := range tests {
		var attempts atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts.Add(1)
			w.WriteHeader(tc.status)
		}))
		_, err := get[retryTestRestModel](retryQuietLogger(), context.Background())(srv.URL)
		srv.Close()
		if err == nil {
			t.Fatalf("status %d: expected error", tc.status)
		}
		if tc.want != nil && !errors.Is(err, tc.want) {
			t.Fatalf("status %d: expected %v, got %v", tc.status, tc.want, err)
		}
		if tc.want == nil && errors.Is(err, ErrServiceUnavailable) {
			t.Fatalf("status %d: must not be ErrServiceUnavailable", tc.status)
		}
		if attempts.Load() != tc.attempt {
			t.Fatalf("status %d: expected %d attempts, got %d", tc.status, tc.attempt, attempts.Load())
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd libs/atlas-rest && go test ./requests/... -run 'Test.*503|TestGetHonors|TestGetCaps|TestGetNon503' -v`
Expected: FAIL — `undefined: ErrServiceUnavailable`, and attempt-count failures.

- [ ] **Step 3: Create requests/metrics.go**

```go
package requests

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// reason ∈ {"503"}
var clientRetriesTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "atlas_rest_client_retries_total",
		Help: "Number of REST client attempts retried after a retryable response, by reason.",
	},
	[]string{"reason"},
)
```

- [ ] **Step 4: Modify get.go**

Replace `libs/atlas-rest/requests/get.go` with:

```go
package requests

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-retry"
	"github.com/sirupsen/logrus"
)

var ErrBadRequest = errors.New("bad request")
var ErrNotFound = errors.New("not found")

// ErrServiceUnavailable is returned when a GET exhausted its attempts and the
// final response was 503 Service Unavailable — the dependency is saturated
// but not broken. Callers distinguish it from ErrBadRequest/ErrNotFound via
// errors.Is.
var ErrServiceUnavailable = errors.New("service unavailable")

// errServiceUnavailableAttempt marks a single 503 attempt inside the retry
// loop; it is translated to ErrServiceUnavailable after exhaustion.
var errServiceUnavailableAttempt = errors.New("received 503 response")

type Request[A any] func(l logrus.FieldLogger, ctx context.Context) (A, error)

// parseRetryAfter parses an integer-seconds Retry-After header value.
func parseRetryAfter(v string) (time.Duration, bool) {
	if v == "" {
		return 0, false
	}
	if s, err := strconv.Atoi(v); err == nil && s >= 0 {
		return time.Duration(s) * time.Second, true
	}
	return 0, false
}

func get[A any](l logrus.FieldLogger, ctx context.Context) func(url string, configurators ...Configurator) (A, error) {
	return func(url string, configurators ...Configurator) (A, error) {
		// GETs are idempotent reads of JSON:API resources: default to 3
		// attempts (transport errors and 503 responses are retryable).
		c := &configuration{retries: 3, timeout: DefaultTimeout}
		for _, configurator := range configurators {
			configurator(c)
		}

		var statusCode int
		var status string
		var body []byte
		get := func(attempt int) (bool, error) {
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				l.WithError(err).Errorf("Error creating request.")
				return true, err
			}

			for _, hd := range c.headerDecorators {
				hd(req.Header)
			}

			reqCtx, cancel := context.WithTimeout(ctx, c.timeout)
			defer cancel()
			req = req.WithContext(reqCtx)

			l.Debugf("Issuing [%s] request to [%s].", req.Method, req.URL)
			r, err := client.Do(req)
			if err != nil {
				l.WithError(err).Warnf("Failed calling [%s] on [%s], will retry.", http.MethodGet, url)
				return true, err
			}
			defer r.Body.Close()

			statusCode = r.StatusCode
			status = r.Status
			body, err = io.ReadAll(r.Body)
			if err != nil {
				l.WithError(err).Warnf("Failed reading response from [%s] on [%s], will retry.", http.MethodGet, url)
				return true, err
			}
			if statusCode == http.StatusServiceUnavailable {
				if attempt < c.retries {
					clientRetriesTotal.WithLabelValues("503").Inc()
					l.Warnf("Received [503] from [%s] on [%s], will retry.", http.MethodGet, url)
				}
				if d, ok := parseRetryAfter(r.Header.Get("Retry-After")); ok {
					return true, retry.WithDelayHint(errServiceUnavailableAttempt, d)
				}
				return true, errServiceUnavailableAttempt
			}
			return false, nil
		}
		cfg := retry.DefaultConfig().WithMaxRetries(c.retries).WithInitialDelay(200 * time.Millisecond).WithMaxDelay(2 * time.Second)
		err := retry.Try(ctx, cfg, get)

		var resp A
		if err != nil {
			if errors.Is(err, errServiceUnavailableAttempt) {
				l.WithError(err).Errorf("Service unavailable after retries calling [%s] on [%s].", http.MethodGet, url)
				return resp, ErrServiceUnavailable
			}
			l.WithError(err).Errorf("Unable to successfully call [%s] on [%s].", http.MethodGet, url)
			return resp, err
		}
		if statusCode == http.StatusOK || statusCode == http.StatusAccepted {
			resp, err = unmarshalResponse[A](body)
			l.WithFields(logrus.Fields{"method": http.MethodGet, "status": status, "path": url, "response": resp}).Debugf("Printing request.")
			return resp, err
		}
		if statusCode == http.StatusBadRequest {
			return resp, ErrBadRequest
		}
		if statusCode == http.StatusNotFound {
			return resp, ErrNotFound
		}
		l.Debugf("Unable to successfully call [%s] on [%s], returned status code [%d].", http.MethodGet, url, statusCode)
		return resp, errors.New("unknown error")
	}
}

//goland:noinspection GoUnusedExportedFunction
func MakeGetRequest[A any](url string, configurators ...Configurator) Request[A] {
	return func(l logrus.FieldLogger, ctx context.Context) (A, error) {
		return get[A](l, ctx)(url, configurators...)
	}
}
```

`ErrBadRequest`/`ErrNotFound` stay defined in this file (their current home) — no move.

- [ ] **Step 5: Tidy and run the full requests test suite**

```bash
cd libs/atlas-rest && go mod tidy && go test -race ./requests/... -v
```
Expected: PASS, including the pre-existing `client_test.go` suite. Note `TestRetryGetsFreshTimeoutPerAttempt` uses `delete` and is unaffected by the GET default change.

- [ ] **Step 6: Vet and commit**

```bash
cd libs/atlas-rest && go vet ./...
git add libs/atlas-rest/requests/get.go libs/atlas-rest/requests/metrics.go libs/atlas-rest/requests/get_retry_test.go libs/atlas-rest/go.mod libs/atlas-rest/go.sum
git commit -m "feat(task-168): REST client GET retries 503 with Retry-After honoring"
```

---

### Task 7: Auto-mount `/metrics` in the rest-server Builder; remove the 4 explicit mounts

**Files:**
- Modify: `libs/atlas-rest/server/server.go` (the `New` function)
- Test: `libs/atlas-rest/server/server_test.go` (create)
- Modify: `services/atlas-channel/atlas.com/channel/main.go:345`
- Modify: `services/atlas-summons/atlas.com/summons/main.go:86`
- Modify: `services/atlas-doors/atlas.com/doors/main.go:91`
- Modify: `services/atlas-monsters/atlas.com/monsters/main.go:96`

**Interfaces:**
- Consumes: existing `MountHandler`, `promhttp.Handler()` (prometheus dep added in Task 6).
- Produces: every service built on `server.New(...)` now serves `GET <basePath>/metrics` without any per-service code.

- [ ] **Step 1: Write the failing test**

Create `libs/atlas-rest/server/server_test.go`:

```go
package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestMetricsRouteAutoMounted(t *testing.T) {
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)
	sb := New(l)
	h := sb.routerProducer(l)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 from auto-mounted /metrics, got %d", rec.Code)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-rest && go test ./server/... -run TestMetricsRouteAutoMounted -v`
Expected: FAIL — 404 (route not mounted)

- [ ] **Step 3: Mount /metrics in New()**

In `libs/atlas-rest/server/server.go`, add import `"github.com/prometheus/client_golang/prometheus/promhttp"` and change the line

```go
	sb.routeInitializers = make([]RouteInitializer, 0)
```

to

```go
	sb.routeInitializers = []RouteInitializer{MountHandler("/metrics", promhttp.Handler())}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd libs/atlas-rest && go test -race ./server/... -v`
Expected: PASS

- [ ] **Step 5: Remove the 4 now-duplicate per-service mounts**

In each of the four `main.go` files listed above, delete the line

```go
		AddRouteInitializer(<server-alias>.MountHandler("/metrics", promhttp.Handler())).
```

(the alias is `restserver` in atlas-channel and `server` in the other three). Then remove the `"github.com/prometheus/client_golang/prometheus/promhttp"` import from each file **only if** it has no other usage in that file (check with grep).

- [ ] **Step 6: Build the four services**

```bash
for s in channel summons doors monsters; do (cd services/atlas-$s/atlas.com/$s && go build ./... && go vet ./...); done
```
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-rest/server/server.go libs/atlas-rest/server/server_test.go \
  services/atlas-channel/atlas.com/channel/main.go services/atlas-summons/atlas.com/summons/main.go \
  services/atlas-doors/atlas.com/doors/main.go services/atlas-monsters/atlas.com/monsters/main.go
git commit -m "feat(task-168): auto-mount /metrics in rest-server Builder fleet-wide"
```

---

### Task 8: `model.ErrDecorator` combinator in `libs/atlas-model`

**Files:**
- Modify: `libs/atlas-model/model/processor.go` (append next to the `Decorator[M]` definition at line 101)
- Test: `libs/atlas-model/model/decorator_test.go` (create)

**Interfaces:**
- Consumes: existing `type Decorator[M any] func(M) M`.
- Produces: `func ErrDecorator[M any](f func(M) (M, error), onErr func(M, error)) Decorator[M]` — used by Tasks 10 and 12. The atlas-model module stays dependency-free (no logging, no prometheus here; `onErr` is the injection point).

- [ ] **Step 1: Write the failing test**

Create `libs/atlas-model/model/decorator_test.go`:

```go
package model

import (
	"errors"
	"testing"
)

func TestErrDecoratorSuccessEnriches(t *testing.T) {
	d := ErrDecorator(
		func(m int) (int, error) { return m + 1, nil },
		func(m int, err error) { t.Fatalf("onErr must not be called on success") },
	)
	if got := d(41); got != 42 {
		t.Fatalf("expected 42, got %d", got)
	}
}

func TestErrDecoratorFailureDegradesLoudly(t *testing.T) {
	boom := errors.New("boom")
	var gotM int
	var gotErr error
	d := ErrDecorator(
		func(m int) (int, error) { return 0, boom },
		func(m int, err error) { gotM, gotErr = m, err },
	)
	if got := d(41); got != 41 {
		t.Fatalf("expected un-enriched 41, got %d", got)
	}
	if gotM != 41 || !errors.Is(gotErr, boom) {
		t.Fatalf("onErr not invoked with original model and cause: m=%d err=%v", gotM, gotErr)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-model && go test ./model/... -run TestErrDecorator -v`
Expected: FAIL — `undefined: ErrDecorator`

- [ ] **Step 3: Implement**

In `libs/atlas-model/model/processor.go`, directly below `type Decorator[M any] func(M) M`, add:

```go
// ErrDecorator adapts a fallible enrichment into a Decorator. On error it
// invokes onErr (must be non-nil) and returns m unchanged — degrade loudly,
// never fail the flow. Pair with a logging/metrics observer such as
// atlas-rest's degrade.Observe.
func ErrDecorator[M any](f func(M) (M, error), onErr func(M, error)) Decorator[M] {
	return func(m M) M {
		r, err := f(m)
		if err != nil {
			onErr(m, err)
			return m
		}
		return r
	}
}
```

- [ ] **Step 4: Test, vet, commit**

```bash
cd libs/atlas-model && go test -race ./... && go vet ./...
git add libs/atlas-model/model/processor.go libs/atlas-model/model/decorator_test.go
git commit -m "feat(task-168): model.ErrDecorator for loud enrichment degradation"
```

---

### Task 9: `degrade` observer package in `libs/atlas-rest`

**Files:**
- Create: `libs/atlas-rest/degrade/degrade.go`
- Test: `libs/atlas-rest/degrade/degrade_test.go`

**Interfaces:**
- Consumes: prometheus (dep added in Task 6), logrus.
- Produces: `func Observe(l logrus.FieldLogger, component string, entityId uint32, err error)` — used by Tasks 10 and 12. `component` is a low-cardinality static string like `"login.character.inventory"`; the entity id goes only into the log line, never a label.

- [ ] **Step 1: Write the failing test**

Create `libs/atlas-rest/degrade/degrade_test.go`:

```go
package degrade

import (
	"errors"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

func TestObserveLogsWarnAndCounts(t *testing.T) {
	logger, hook := test.NewNullLogger()
	before := testutil.ToFloat64(degradedTotal.WithLabelValues("test.component"))

	Observe(logger, "test.component", 42, errors.New("fetch failed"))

	after := testutil.ToFloat64(degradedTotal.WithLabelValues("test.component"))
	if after-before != 1 {
		t.Fatalf("expected counter delta 1, got %v", after-before)
	}
	entry := hook.LastEntry()
	if entry == nil || entry.Level != logrus.WarnLevel {
		t.Fatalf("expected a Warn entry, got %+v", entry)
	}
	if !strings.Contains(entry.Message, "test.component") || !strings.Contains(entry.Message, "42") {
		t.Fatalf("Warn must name component and entity id, got: %s", entry.Message)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-rest && go test ./degrade/... -v`
Expected: FAIL — package does not exist / `undefined: Observe`

- [ ] **Step 3: Implement**

Create `libs/atlas-rest/degrade/degrade.go`:

```go
// Package degrade is the loud-degradation observer: every enrichment or
// fallback path that drops data on failure must call Observe so the
// degradation is logged and counted — degraded results must never be
// indistinguishable from correct ones.
package degrade

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

var degradedTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "atlas_enrichment_degraded_total",
		Help: "Number of enrichment/decorator failures that degraded to a partial result, by component.",
	},
	[]string{"component"},
)

// Observe logs the degradation at Warn with the entity id and cause, and
// increments atlas_enrichment_degraded_total{component}. component must be a
// static low-cardinality string (e.g. "login.character.inventory"); entityId
// goes only into the log line, never into a metric label.
func Observe(l logrus.FieldLogger, component string, entityId uint32, err error) {
	degradedTotal.WithLabelValues(component).Inc()
	l.WithError(err).Warnf("Enrichment degraded for component [%s], entity [%d]; returning un-enriched model.", component, entityId)
}
```

- [ ] **Step 4: Tidy (logrus test hooks may need go.sum entries), test, vet, commit**

```bash
cd libs/atlas-rest && go mod tidy && go test -race ./degrade/... -v && go vet ./...
git add libs/atlas-rest/degrade/ libs/atlas-rest/go.mod libs/atlas-rest/go.sum
git commit -m "feat(task-168): degrade.Observe loud-degradation observer"
```

---

### Task 10: atlas-login `InventoryDecorator` reference fix + incident replay test

**Files:**
- Modify: `services/atlas-login/atlas.com/login/character/processor.go:108-116`
- Test: `services/atlas-login/atlas.com/login/character/processor_test.go` (create)
- Modify: `services/atlas-login/atlas.com/login/go.mod` (tidy pulls prometheus for the test helper)

**Interfaces:**
- Consumes: `model.ErrDecorator` (Task 8), `degrade.Observe` (Task 9), client 503 retry (Task 6, automatic via `requests.GetRequest`).
- Produces: reference implementation of the loud-degrade policy; component string `"login.character.inventory"`.

**Deviation from design.md, discovered during planning:** atlas-login has **no database** (`grep database.Connect services/atlas-login/atlas.com/login/main.go` → no match), so design §2.3's "atlas-login registers it too (it has a DB)" is wrong — there is nothing for a classifier to classify in login's own handlers. Do NOT register a classifier in atlas-login. atlas-inventory (Task 11) is the classifier-registration reference.

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-login/atlas.com/login/character/processor_test.go` (in-package so unexported fields are reachable):

```go
package character

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"atlas-login/inventory"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

// counterValue reads a labeled counter from the default gatherer (0 when the
// series does not exist yet).
func counterValue(t *testing.T, name, labelName, labelValue string) float64 {
	t.Helper()
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	for _, mf := range mfs {
		if mf.GetName() != name {
			continue
		}
		for _, m := range mf.GetMetric() {
			for _, lp := range m.GetLabel() {
				if lp.GetName() == labelName && lp.GetValue() == labelValue {
					return m.GetCounter().GetValue()
				}
			}
		}
	}
	return 0
}

func testContext(t *testing.T) context.Context {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tenant.WithContext(context.Background(), ten)
}

func TestInventoryDecoratorDegradesLoudly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError) // terminal, not retried
	}))
	defer srv.Close()
	t.Setenv("INVENTORY_SERVICE_URL", srv.URL+"/")

	logger, hook := test.NewNullLogger()
	before := counterValue(t, "atlas_enrichment_degraded_total", "component", "login.character.inventory")

	p := NewProcessor(logger, testContext(t)).(*ProcessorImpl)
	m := Model{id: 42}
	decorated := p.InventoryDecorator()(m)

	if !reflect.DeepEqual(decorated, m) {
		t.Fatalf("expected un-enriched model on failure")
	}
	found := false
	for _, e := range hook.AllEntries() {
		if e.Level == logrus.WarnLevel && strings.Contains(e.Message, "Enrichment degraded") && strings.Contains(e.Message, "42") {
			found = true
		}
	}
	if !found {
		t.Fatal("expected a Warn log naming the degradation and character id 42")
	}
	after := counterValue(t, "atlas_enrichment_degraded_total", "component", "login.character.inventory")
	if after-before != 1 {
		t.Fatalf("expected degradation counter delta 1, got %v", after-before)
	}
}

// Incident replay (PRD acceptance): one transient 503 from atlas-inventory
// must be absorbed by the client retry — the decorated character keeps its
// equipment and nothing degrades.
func TestInventoryDecoratorRetriesThroughTransient503(t *testing.T) {
	rm := inventory.RestModel{Id: uuid.New(), CharacterId: 42}
	body, err := jsonapi.Marshal(rm)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	defer srv.Close()
	t.Setenv("INVENTORY_SERVICE_URL", srv.URL+"/")

	logger, hook := test.NewNullLogger()
	p := NewProcessor(logger, testContext(t)).(*ProcessorImpl)
	m := Model{id: 42}
	decorated := p.InventoryDecorator()(m)

	if attempts.Load() != 2 {
		t.Fatalf("expected the 503 to be retried (2 attempts), got %d", attempts.Load())
	}
	if reflect.DeepEqual(decorated, m) {
		t.Fatal("expected enriched model after successful retry")
	}
	for _, e := range hook.AllEntries() {
		if strings.Contains(e.Message, "Enrichment degraded") {
			t.Fatalf("nothing may degrade when the retry succeeds: %s", e.Message)
		}
	}
}
```

(Do not import `errors` — it is unused in this file.)

Notes for the implementer: `Model{id: 42}` uses the unexported field backing `Model.Id()` (`services/atlas-login/atlas.com/login/character/model.go:87`) — confirm the field name there and adjust if it differs. If `tenant.Create`'s signature differs, check `libs/atlas-tenant` for the constructor used by other service tests.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-login/atlas.com/login && go test ./character/... -run TestInventoryDecorator -v`
Expected: `TestInventoryDecoratorDegradesLoudly` FAILS (no Warn, no counter — current code swallows the error). The replay test also fails on the attempts assertion only if Task 6 is missing; with Task 6 done it may pass — that is fine, it is the acceptance pin.

- [ ] **Step 3: Rewrite InventoryDecorator**

In `services/atlas-login/atlas.com/login/character/processor.go`, add import `"github.com/Chronicle20/atlas/libs/atlas-rest/degrade"` and replace lines 108-116 with:

```go
func (p *ProcessorImpl) InventoryDecorator() model.Decorator[Model] {
	return model.ErrDecorator(
		func(m Model) (Model, error) {
			i, err := p.ip.GetByCharacterId(m.Id())
			if err != nil {
				return m, err
			}
			return m.SetInventory(i), nil
		},
		func(m Model, err error) {
			degrade.Observe(p.l, "login.character.inventory", m.Id(), err)
		},
	)
}
```

- [ ] **Step 4: Tidy, test, vet**

```bash
cd services/atlas-login/atlas.com/login && go mod tidy && go test -race ./... && go vet ./... && go build ./...
```
Expected: PASS/clean. `go.mod` gains prometheus (test dep) — that is expected; do not fight it.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-login/atlas.com/login/character/processor.go services/atlas-login/atlas.com/login/character/processor_test.go services/atlas-login/atlas.com/login/go.mod services/atlas-login/atlas.com/login/go.sum
git commit -m "feat(task-168): loud-degrade InventoryDecorator + incident replay test"
```

---

### Task 11: atlas-inventory adopts the 503 contract

**Files:**
- Modify: `services/atlas-inventory/atlas.com/inventory/main.go` (register classifier, near `database.Connect` at line 62)
- Modify: `services/atlas-inventory/atlas.com/inventory/inventory/resource.go` (3 × `StatusInternalServerError` sites)
- Modify: `services/atlas-inventory/atlas.com/inventory/compartment/resource.go` (all `StatusInternalServerError` sites)
- Modify: `services/atlas-inventory/atlas.com/inventory/asset/resource.go` (all `StatusInternalServerError` sites)
- Test: `services/atlas-inventory/atlas.com/inventory/inventory/resource_test.go` (create)

**Interfaces:**
- Consumes: `server.RegisterTransientErrorClassifier`, `server.WriteErrorResponse` (Task 5), `database.IsTransientConnectionError`, `database.CountTransient` (Tasks 2-3), `server.NewHandlerDependency`/`server.NewHandlerContext` (existing, `libs/atlas-rest/server/context.go:17,32`).
- Produces: the fleet reference for per-service adoption (documented in Task 13).

- [ ] **Step 1: Write the failing handler test**

Create `services/atlas-inventory/atlas.com/inventory/inventory/resource_test.go` (in package `inventory`):

```go
package inventory

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	server "github.com/Chronicle20/atlas/libs/atlas-rest/server"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// --- fake driver whose every query fails with a fixed error ---

type failConn struct{ err error }

func (failConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("not implemented") }
func (failConn) Close() error                        { return nil }
func (failConn) Begin() (driver.Tx, error)           { return nil, errors.New("not implemented") }
func (c failConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	return nil, c.err
}
func (c failConn) ExecContext(context.Context, string, []driver.NamedValue) (driver.Result, error) {
	return nil, c.err
}

type failConnector struct{ err error }

func (f failConnector) Connect(context.Context) (driver.Conn, error) { return failConn{err: f.err}, nil }
func (f failConnector) Driver() driver.Driver                        { return nil }

func failingDB(t *testing.T, queryErr error) *gorm.DB {
	t.Helper()
	sqlDB := sql.OpenDB(failConnector{err: queryErr})
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	return db
}

type testSI struct{}

func (testSI) GetBaseURL() string { return "http://localhost" }
func (testSI) GetPrefix() string  { return "" }

func serveGetInventory(t *testing.T, db *gorm.DB) *httptest.ResponseRecorder {
	t.Helper()
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), ten)

	d := server.NewHandlerDependency(l, ctx)
	c := server.NewHandlerContext(testSI{})
	router := mux.NewRouter()
	router.HandleFunc("/characters/{characterId}/inventory", handleGetInventory(db)(&d, &c))

	req := httptest.NewRequest(http.MethodGet, "/characters/42/inventory", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestGetInventoryTransientDBErrorIs503(t *testing.T) {
	server.RegisterTransientErrorClassifier(func(err error) bool {
		if database.IsTransientConnectionError(err) {
			database.CountTransient(err)
			return true
		}
		return false
	})
	defer server.RegisterTransientErrorClassifier(nil)

	rec := serveGetInventory(t, failingDB(t, &pgconn.PgError{Code: "53300"}))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Retry-After") != "1" {
		t.Fatalf("expected Retry-After: 1, got %q", rec.Header().Get("Retry-After"))
	}
	if !strings.Contains(rec.Body.String(), "temporarily unavailable") {
		t.Fatalf("expected JSON:API 503 body, got: %s", rec.Body.String())
	}
}

func TestGetInventoryNonTransientDBErrorIs500(t *testing.T) {
	server.RegisterTransientErrorClassifier(database.IsTransientConnectionError)
	defer server.RegisterTransientErrorClassifier(nil)

	rec := serveGetInventory(t, errors.New("real bug"))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}
```

Notes for the implementer: `rest.HandlerDependency` is a type alias for `server.HandlerDependency` (`services/atlas-inventory/atlas.com/inventory/rest/rest.go:12`), so `&d` satisfies `handleGetInventory`'s `rest.GetHandler` signature directly. The tenant in ctx satisfies any `tenant.MustFromContext` calls in the processor. If `gorm.Open` itself trips on the failing driver (it pings, which succeeds here, but if the postgres dialector issues an eager query in your gorm version), make `failConn`'s error injectable after open: give `failConnector` a `*error` field the test sets post-`gorm.Open`.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-inventory/atlas.com/inventory && go test ./inventory/... -run TestGetInventory -v`
Expected: FAIL — both cases currently return 500 (`resource.go:45` writes `StatusInternalServerError` unconditionally).

- [ ] **Step 3: Register the classifier in main.go**

In `services/atlas-inventory/atlas.com/inventory/main.go`, immediately after the `db := database.Connect(...)` line (line 62), add:

```go
	server.RegisterTransientErrorClassifier(func(err error) bool {
		if database.IsTransientConnectionError(err) {
			database.CountTransient(err)
			return true
		}
		return false
	})
```

(using whatever alias main.go already imports `libs/atlas-rest/server` under — line 81 uses `server.New(l)`).

- [ ] **Step 4: Adopt WriteErrorResponse in the three resource files**

In `inventory/resource.go`, `compartment/resource.go`, `asset/resource.go`: replace **every** `w.WriteHeader(http.StatusInternalServerError)` with

```go
					server.WriteErrorResponse(d.Logger())(w)(err)
```

keeping the existing contextual `d.Logger().WithError(err).Errorf(...)` lines and every `errors.Is(err, gorm.ErrRecordNotFound)` → 404 branch exactly as they are. In blocks where the log line came *after* the WriteHeader, keep log-then-write order. `inventory/resource.go` already imports the server package; add the import to compartment/asset resource files if missing. Do not touch 400/404 paths.

- [ ] **Step 5: Tidy, test, vet, build**

```bash
cd services/atlas-inventory/atlas.com/inventory && go mod tidy && go test -race ./... && go vet ./... && go build ./...
```
Expected: PASS/clean.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-inventory/atlas.com/inventory/main.go \
  services/atlas-inventory/atlas.com/inventory/inventory/resource.go \
  services/atlas-inventory/atlas.com/inventory/inventory/resource_test.go \
  services/atlas-inventory/atlas.com/inventory/compartment/resource.go \
  services/atlas-inventory/atlas.com/inventory/asset/resource.go \
  services/atlas-inventory/atlas.com/inventory/go.mod services/atlas-inventory/atlas.com/inventory/go.sum
git commit -m "feat(task-168): atlas-inventory maps transient DB errors to 503 + Retry-After"
```

---

### Task 12: Fleet decorator audit + silent-site fixes

**Files:**
- Create: `docs/tasks/task-168-db-connection-resilience/decorator-audit.md`
- Modify: every service file the audit's fix-list identifies (pattern below)

**Interfaces:**
- Consumes: `model.ErrDecorator` (Task 8), `degrade.Observe` (Task 9).
- Produces: the FR-5.3 audit table; zero silent fallible decorators remain.

- [ ] **Step 1: Enumerate decorator implementations**

From the worktree root:

```bash
grep -rn 'model\.Decorator\[' services/ --include='*.go' \
  | grep -v '_test.go' | grep -v '/mock/' \
  | grep -vE 'decorators \.\.\.|\.\.\.model\.Decorator|\[\]model\.Decorator|Decorate\(' \
  > /tmp/decorator-sites.txt
wc -l /tmp/decorator-sites.txt
```

This yields declaration/return sites (interface methods + implementations), not the ~229 mere references. For each **implementation** (a `func ... model.Decorator[...]` with a body), read the body and classify:

- **fallible fetch** — the body calls a processor/`requests.*`/DB/Redis and branches on `err` (the incident shape: `if err != nil { return m }`),
- **pure** — in-memory transform only.

- [ ] **Step 2: Write the audit table**

Create `docs/tasks/task-168-db-connection-resilience/decorator-audit.md` with exactly these columns:

```markdown
# Decorator Audit — task-168 (FR-5.3 / FR-5.4)

Method: `grep -rn 'model.Decorator\[' services/` implementations (mocks and
mere references excluded), each body read and classified.

## model.Decorator implementations

| service | file:line | decorator | fetch kind | silent today? | disposition |
|---|---|---|---|---|---|
| atlas-login | character/processor.go:108 | InventoryDecorator | REST (inventory) | yes | fixed-in-task (task 10) |
| atlas-skills | skill/processor.go:247 | CooldownDecorator | (classify) | (yes/no) | (fixed-in-task / justified-but-now-loud / pure) |
| ... every site found in Step 1 ... |

## Non-decorator silent-degrade shapes on the character-select path (FR-5.4)

| service | file:line | shape | silent today? | disposition |
|---|---|---|---|---|
| ... |
```

Every row must be real (file:line verified). No site may end with disposition "silent".

- [ ] **Step 3: Audit the character-select path for non-decorator shapes (FR-5.4)**

Trace atlas-login's char-select flow: start from the socket handlers that build character entries —

```bash
grep -rn "GetForWorld\|ByAccountAndWorldProvider\|InventoryDecorator" services/atlas-login/atlas.com/login/socket/ services/atlas-login/atlas.com/login/session/
```

— and follow each caller: any place that fetches remote data, gets an error, and continues building the entry from partial data (dropping equipment, stats, etc. without a Warn+metric) is a finding. Add each to the second table.

- [ ] **Step 4: Fix every silent fallible site**

Apply the Task 10 transformation to each. Canonical before/after (this is the exact shape every fix follows):

```go
// BEFORE (silent)
func (p *ProcessorImpl) SomethingDecorator() model.Decorator[Model] {
	return func(m Model) Model {
		x, err := p.dep.GetByOwnerId(m.Id())
		if err != nil {
			return m
		}
		return m.SetSomething(x)
	}
}

// AFTER (loud)
func (p *ProcessorImpl) SomethingDecorator() model.Decorator[Model] {
	return model.ErrDecorator(
		func(m Model) (Model, error) {
			x, err := p.dep.GetByOwnerId(m.Id())
			if err != nil {
				return m, err
			}
			return m.SetSomething(x), nil
		},
		func(m Model, err error) {
			degrade.Observe(p.l, "<service>.<domain>.<enrichment>", m.Id(), err)
		},
	)
}
```

Rules: component string is `<service>.<package>.<what-was-fetched>` (static, lowercase, dot-separated); decorators whose degradation is *justified* (data genuinely optional) still get `degrade.Observe` — "justified" changes the audit disposition, never removes the observability; decorators taking a parameter (e.g. `CooldownDecorator(characterId uint32)`) close over it the same way. If a site's entity id is not a `uint32`, use the nearest owning uint32 id (character/account) — never change `Observe`'s signature. Run `go mod tidy && go test -race ./... && go vet ./... && go build ./...` in every touched service module.

- [ ] **Step 5: Record the touched-services list**

Append to `decorator-audit.md` a final section:

```markdown
## Services modified by this audit (drives Task 14 bake list)

- services/atlas-<name> (fix: <decorator>)
- ...
```

- [ ] **Step 6: Commit**

```bash
git add docs/tasks/task-168-db-connection-resilience/decorator-audit.md services/
git commit -m "feat(task-168): decorator audit — all fallible enrichments degrade loudly"
```

---

### Task 13: Documentation & agentic-guidelines updates

**Files:**
- Create: `.claude/skills/backend-dev-guidelines/resources/patterns-resilience.md`
- Modify: `.claude/skills/backend-dev-guidelines/SKILL.md` (reference the new resource where the other `patterns-*.md` files are referenced — find them with `grep -n "patterns-" .claude/skills/backend-dev-guidelines/SKILL.md`)
- Modify: `.claude/agents/backend-guidelines-reviewer.md` (add DOM-26, DOM-27 rows — DOM-25 is currently the highest)
- Create: `libs/atlas-database/README.md`

**Interfaces:**
- Consumes: everything built in Tasks 1-11 (documents it).
- Produces: the enforceable pattern documentation (FR-7).

- [ ] **Step 1: Write patterns-resilience.md**

Create `.claude/skills/backend-dev-guidelines/resources/patterns-resilience.md`:

```markdown
# DB & Downstream Resilience Patterns

Source task: task-168 (atlas-pr-901 naked-character incident). These patterns
are mandatory for new code and enforced by DOM-26/DOM-27.

## Transient DB error classification

`libs/atlas-database` exports:

- `database.IsTransientConnectionError(err error) bool` — true only for
  acquire-phase failures: SQLSTATE 53300, 57P03, 08001, 08006, pgx
  `*pgconn.ConnectError`, dial-shape net errors (ECONNREFUSED/ECONNRESET).
  Anything that may have started executing (constraint violations,
  serialization failures, statement timeouts, bare context deadlines) is
  NOT transient. Never retry ambiguous work.
- `database.TransientSQLState(err) string` — metric label helper.
- `database.CountTransient(err)` — increments
  `atlas_db_transient_errors_total{sqlstate}`; call only after the
  predicate returned true.

## Acquire-phase DB retry (automatic)

`database.Connect` wraps the pgx connector so transient acquire failures are
retried transparently: max `DB_ACQUIRE_RETRY_ATTEMPTS` (default 3) attempts,
full-jitter backoff `DB_ACQUIRE_RETRY_INITIAL_DELAY` (100ms) →
`DB_ACQUIRE_RETRY_MAX_DELAY` (400ms). `0`/`1` disables. Every retry logs Warn
and increments `atlas_db_acquire_retries_total{sqlstate}`. Mid-statement
errors never reach this layer (the wrapper sits on `driver.Connector.Connect`,
which the pool only calls before any SQL is sent).

## The 503 transient-error contract (server side)

Transient DB errors MUST surface as `503 Service Unavailable` +
`Retry-After: 1` with a JSON:API error body — never a generic 500. In
handlers, replace `w.WriteHeader(http.StatusInternalServerError)` with:

    server.WriteErrorResponse(d.Logger())(w)(err)

and register the classifier once in main.go (services with a DB):

    server.RegisterTransientErrorClassifier(func(err error) bool {
        if database.IsTransientConnectionError(err) {
            database.CountTransient(err)
            return true
        }
        return false
    })

Keep 404/400 branches as they are. Non-transient errors still map to 500
(now with a JSON:API body). Reference implementation: atlas-inventory
(main.go + inventory/compartment/asset resource.go).

## Client retry semantics (automatic, GET only)

The shared REST client (`libs/atlas-rest/requests`) retries GETs on 503
(and transport errors) — 3 attempts default, jittered backoff capped at 2s,
`Retry-After` honored (capped). Exhaustion returns
`requests.ErrServiceUnavailable` (check with `errors.Is`). POST/PATCH/PUT/
DELETE are never retried on 503. Do not add per-call retry loops around the
client; if a GET must not retry, pass `requests.SetRetries(1)`.

## No silent degradation (decorator policy)

A decorator or enrichment step that fails its fetch MUST NOT silently return
the un-enriched model. Use the combinator + observer pair:

    func (p *ProcessorImpl) XDecorator() model.Decorator[Model] {
        return model.ErrDecorator(
            func(m Model) (Model, error) {
                x, err := p.dep.GetById(m.Id())
                if err != nil { return m, err }
                return m.SetX(x), nil
            },
            func(m Model, err error) {
                degrade.Observe(p.l, "<svc>.<domain>.<enrichment>", m.Id(), err)
            },
        )
    }

Degrading (returning the un-enriched model) remains the correct fallback —
but it logs Warn with the entity id and increments
`atlas_enrichment_degraded_total{component}`. Component strings are static
and low-cardinality; entity ids go in the log line only. Reference:
atlas-login `character.InventoryDecorator`.

## Pool sizing guidance

Defaults: `DB_MAX_OPEN_CONNS=10`, `DB_MAX_IDLE_CONNS=5`,
`DB_CONN_MAX_LIFETIME=5m`, `DB_CONN_MAX_IDLE_TIME=3m`. Budget rule: the sum
over all DB services of `max_open × replicas × namespaces` must fit inside
postgres `max_connections` minus reserved slots — dozens of services ×
multiple ephemeral namespaces WILL exhaust slots under burst if left
unbudgeted. Watch `go_sql_wait_count_total` / `go_sql_wait_duration_seconds_total`
and `atlas_db_acquire_retries_total` for pressure before it bites.
Infrastructure-side budgets (postgres max_connections, PgBouncer) are infra
concerns — the service-side knob is `DB_MAX_OPEN_CONNS`.

## Observability summary

| Metric | Meaning |
|---|---|
| `go_sql_*{db_name}` | pool gauges (open/in-use/idle/wait) per service |
| `atlas_db_acquire_retries_total{sqlstate}` | DB-side transparent retries — rising = chronic undersizing |
| `atlas_db_transient_errors_total{sqlstate}` | transient classifications (retried or surfaced) |
| `atlas_rest_client_retries_total{reason}` | client-side 503 retries |
| `atlas_enrichment_degraded_total{component}` | loud degradations — should be ~0 |

Every REST-serving service exposes `/metrics` automatically (mounted by the
rest-server Builder); no per-service mount is needed.
```

- [ ] **Step 2: Reference it from SKILL.md**

Find where SKILL.md references the other resource files (`grep -n "patterns-" .claude/skills/backend-dev-guidelines/SKILL.md`) and add, in the same style and alphabetical/topical position:

```markdown
- `resources/patterns-resilience.md` — transient DB error classification, 503 + Retry-After contract, client retry semantics, loud-degradation decorator policy, pool sizing
```

- [ ] **Step 3: Add DOM-26 and DOM-27 to the reviewer agent**

In `.claude/agents/backend-guidelines-reviewer.md`, append after the DOM-25 row, matching the existing table format:

```markdown
| DOM-26 | Transient DB errors map to 503, never bare 500 | (a) In changed resource handlers, find every error branch that writes `http.StatusInternalServerError` directly via `w.WriteHeader`. (b) If the service has a DB (calls `database.Connect`), those branches must instead call `server.WriteErrorResponse(d.Logger())(w)(err)` and `main.go` must call `server.RegisterTransientErrorClassifier` composing `database.IsTransientConnectionError` + `database.CountTransient`. (c) 404/400 branches are exempt. | Changed handlers in DB-backed services use `WriteErrorResponse`; the classifier is registered once in main.go. A transient pool-exhaustion error surfacing as a generic 500 is a finding (task-168; see patterns-resilience.md). |
| DOM-27 | No silent degradation in decorators/enrichment | (a) In changed code, find every `model.Decorator[...]` implementation and every enrichment/fallback path whose body fetches remote data (processor/requests/DB/Redis) and branches on `err`. (b) Each failure path must either propagate the error or degrade loudly via `model.ErrDecorator` + `degrade.Observe(l, "<svc>.<domain>.<enrichment>", id, err)` (Warn log + `atlas_enrichment_degraded_total` increment). (c) A bare `if err != nil { return m }` that drops fetched data with no log and no metric is a finding regardless of justification. | Every fallible enrichment in the diff logs Warn and increments the degradation metric on failure (task-168; see patterns-resilience.md and decorator-audit.md for the fleet baseline). |
```

- [ ] **Step 4: Write libs/atlas-database/README.md**

```markdown
# atlas-database

Shared GORM/postgres connection layer with multi-tenant scoping, transient
error classification, and acquire-phase retry.

## Connection

`database.Connect(l, configurators...)` builds the DSN from env, opens the
pool through a retrying `driver.Connector` (pgx stdlib), registers tenant
callbacks, runs migrations, and registers `go_sql_*` DBStats gauges.

## Environment knobs

| Env | Default | Meaning |
|---|---|---|
| `DB_USER` / `DB_PASSWORD` / `DB_HOST` / `DB_PORT` / `DB_NAME` | — | DSN components |
| `DB_MAX_OPEN_CONNS` | `10` | pool max open connections |
| `DB_MAX_IDLE_CONNS` | `5` | pool max idle connections |
| `DB_CONN_MAX_LIFETIME` | `5m` | recycle connections after this age |
| `DB_CONN_MAX_IDLE_TIME` | `3m` | close idle connections after this |
| `DB_ACQUIRE_RETRY_ATTEMPTS` | `3` | total acquire attempts; `0`/`1` disables retry |
| `DB_ACQUIRE_RETRY_INITIAL_DELAY` | `100ms` | retry backoff initial delay (full jitter) |
| `DB_ACQUIRE_RETRY_MAX_DELAY` | `400ms` | retry backoff cap |

Pool budgeting: `max_open × replicas × namespaces` summed over services must
fit postgres `max_connections` minus reserved slots.

## Transient classification

`IsTransientConnectionError(err) bool` — true only for acquire-phase failures:

| Condition | Transient |
|---|---|
| SQLSTATE `53300` (too_many_connections) | yes |
| SQLSTATE `57P03` (cannot_connect_now) | yes |
| SQLSTATE `08001` / `08006` (connect failure) | yes |
| `*pgconn.ConnectError` (connect never completed) | yes |
| net dial errors / ECONNREFUSED / ECONNRESET | yes |
| any other SQLSTATE (23xxx, 40001, 57014, ...) | no |
| bare `context.DeadlineExceeded`, `gorm.ErrRecordNotFound`, nil | no |

Coded errors are classified strictly by SQLSTATE (checked before the
connect-error shape), so an auth failure during connect is not transient.

`TransientSQLState(err) string` returns the classifying SQLSTATE ("" for
dial-shape). `CountTransient(err)` increments
`atlas_db_transient_errors_total{sqlstate}` — call only after the predicate
returned true.

## Retry behavior

Retry wraps `driver.Connector.Connect` ONLY — the pool calls it before any
SQL is sent, so retried work can never double-apply. Mid-statement errors
are structurally unreachable from this layer. Each retry logs Warn and
increments `atlas_db_acquire_retries_total{sqlstate}`.

## Metrics

`go_sql_*{db_name}` DBStats gauges, `atlas_db_acquire_retries_total{sqlstate}`,
`atlas_db_transient_errors_total{sqlstate}`. Process-level; never
tenant-labeled. Exposed via the rest-server Builder's automatic `/metrics`
mount.
```

- [ ] **Step 5: Commit**

```bash
git add .claude/skills/backend-dev-guidelines/resources/patterns-resilience.md \
  .claude/skills/backend-dev-guidelines/SKILL.md \
  .claude/agents/backend-guidelines-reviewer.md \
  libs/atlas-database/README.md
git commit -m "docs(task-168): resilience patterns, DOM-26/27 reviewer checks, atlas-database README"
```

---

### Task 14: Fleet tidy + full verification battery

**Files:**
- Modify: `services/*/atlas.com/*/go.mod` + `go.sum` (tidy fallout from the lib dep additions — prometheus in atlas-rest/atlas-database, atlas-retry + direct pgx in atlas-database)

**Interfaces:**
- Consumes: everything.
- Produces: the PRD's verification acceptance criteria, evidenced.

- [ ] **Step 1: Tidy every service module**

The lib modules gained requires; every consuming service's `go.sum` needs the hashes before docker bake (the in-image minimal go.work resolves from the service's own go.mod/go.sum). From the worktree root:

```bash
for mod in services/*/atlas.com/*/go.mod; do
  (cd "$(dirname "$mod")" && go mod tidy) || echo "TIDY FAILED: $mod"
done
git status --short | head -50
```

Do NOT run `go work sync`. If any module fails with a missing `replace` for `atlas-retry`/`atlas-database` transitive requirements, add the replace line matching the sibling entries already in that go.mod (e.g. `replace github.com/Chronicle20/atlas/libs/atlas-retry => ../../../../libs/atlas-retry`).

- [ ] **Step 2: Test/vet/build every changed module**

```bash
for m in libs/atlas-retry libs/atlas-database libs/atlas-rest libs/atlas-model; do
  (cd $m && go test -race ./... && go vet ./... && go build ./...) || echo "FAILED: $m"
done
for svc in $(git diff --name-only main | grep '^services/' | cut -d/ -f2 | sort -u); do
  (cd services/$svc/atlas.com/${svc#atlas-} && go test -race ./... && go vet ./... && go build ./...) || echo "FAILED: $svc"
done
```

Expected: all clean; fix and re-run until clean.

- [ ] **Step 3: Redis key guard**

```bash
tools/redis-key-guard.sh
```
Expected: clean (this task adds no Redis usage — a failure means an unrelated regression; investigate).

- [ ] **Step 4: Docker bake**

Nearly every service's `go.mod`/`go.sum` changed in Step 1, so bake everything:

```bash
docker buildx bake all-go-services
```
Expected: every image builds. This is mandatory (root CLAUDE.md) — `go build` cannot catch in-image go.work/go.sum drift. Expect this to take a while; on failure, fix the offending module (usually a missing replace or stale go.sum) and re-bake that service (`docker buildx bake atlas-<svc>`) before re-baking all.

- [ ] **Step 5: Commit the tidy sweep**

```bash
git add services/ && git commit -m "chore(task-168): go mod tidy sweep for lib dependency additions"
```

- [ ] **Step 6: Acceptance checklist sweep**

Walk PRD §10 and confirm each criterion has evidence (test name or artifact):

- Classifier + table tests → Task 2. Acquire retry + not-retried-mid-statement + knobs → Task 4. Inventory 503 → Task 11. Client 503 retry matrix → Task 6. InventoryDecorator loud → Task 10. `decorator-audit.md` complete, zero silent → Task 12. Gauges + 4 counters registered → Tasks 3/4/6/9 tests + `/metrics` mount test (Task 7). Docs + DOM items → Task 13. Verification battery → this task. Incident replay → Task 10's `TestInventoryDecoratorRetriesThroughTransient503`.

Record any gap as a failure and fix it before declaring done. Then run the project's code-review step (`superpowers:requesting-code-review`) per CLAUDE.md before any PR.
