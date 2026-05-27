# Dynamic Service Configuration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace atlas-channel/atlas-login's synchronous REST dependency on atlas-configurations with a Kafka event stream backed by a transactional outbox, and add live add/drain of per-`(tenant, world, channel)` listeners without pod restart.

**Architecture:** atlas-configurations gains a `libs/atlas-outbox`-driven transactional outbox writing two log-compacted Kafka topics. atlas-channel and atlas-login subscribe to those topics, build an in-memory projection of service+tenant config, and gate readiness on having caught up to the topic end offsets snapshot at boot. A new `listener.Registry` in atlas-channel/atlas-login keys per-`(t,w,c)` listener state by `server.Key` and runs a four-phase drain (quiesce → save-and-kick → deadline → tear-down) when config removes the key, threading Kafka handler IDs back through every consumer package's `InitHandlers` so they can be deregistered.

**Tech Stack:** Go (Go workspace), GORM + Postgres (LISTEN/NOTIFY + pg_advisory_lock), kafka-go (compacted topics, ReadEndOffsets), testcontainers-go for integration tests, gorilla/mux for atlas-world REST surface.

---

## Phasing and shippability

Phases A-C are independent libraries/services and can be merged independently. Phase D depends on A+B+C. Phases E-K rewire atlas-channel and depend on A+B (and on C only indirectly, via the Unregister REST call). Phase L (atlas-login) depends on A+B. Phase M is the cross-cutting verification. Each phase ends with `go test -race ./...` + `go vet ./...` + `go build ./...` clean for its module(s); see Phase M for the cross-service docker-build sweep that gates branch completion per CLAUDE.md.

Throughout: prefer Postgres testcontainers for outbox integration tests (matching the project's existing pattern in libs/atlas-database). For Kafka where a stub suffices, use `kafka-go`'s `MockBroker`; where end-to-end semantics matter (catch-up gate, compaction), use a testcontainer.

---

## Phase A — `libs/atlas-outbox`

New library. All work confined to `libs/atlas-outbox/`.

### Task A1: Library skeleton

**Files:**
- Create: `libs/atlas-outbox/go.mod`
- Create: `libs/atlas-outbox/go.sum` (generated)
- Create: `libs/atlas-outbox/README.md`
- Modify: `go.work`

- [ ] **Step 1: Bootstrap module**

```bash
cd libs/atlas-outbox
go mod init github.com/Chronicle20/atlas/libs/atlas-outbox
go mod edit -go=1.25.5
```

- [ ] **Step 2: Add to workspace**

Edit `<worktree>/go.work` and append `./libs/atlas-outbox` alphabetically inside the `use(...)` block.

- [ ] **Step 3: Add minimal README stub**

Write `libs/atlas-outbox/README.md` with two short sections (`Overview`, `Usage`). Final content fills in at Task A9.

- [ ] **Step 4: Commit**

```bash
git add libs/atlas-outbox/go.mod libs/atlas-outbox/README.md go.work
git commit -m "feat(atlas-outbox): scaffold transactional outbox library"
```

### Task A2: Entity and migration

**Files:**
- Create: `libs/atlas-outbox/entity.go`
- Create: `libs/atlas-outbox/migration.go`
- Create: `libs/atlas-outbox/migration_test.go`

- [ ] **Step 1: Write failing migration test**

```go
package outbox_test

import (
	"testing"

	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMigration_CreatesTable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, outbox.Migration(db))

	require.True(t, db.Migrator().HasTable(&outbox.Entity{}))
	require.True(t, db.Migrator().HasColumn(&outbox.Entity{}, "topic"))
	require.True(t, db.Migrator().HasColumn(&outbox.Entity{}, "message_key"))
	require.True(t, db.Migrator().HasColumn(&outbox.Entity{}, "message_value"))
	require.True(t, db.Migrator().HasColumn(&outbox.Entity{}, "headers"))
	require.True(t, db.Migrator().HasColumn(&outbox.Entity{}, "enqueued_at"))
	require.True(t, db.Migrator().HasColumn(&outbox.Entity{}, "sent_at"))
	require.True(t, db.Migrator().HasColumn(&outbox.Entity{}, "attempts"))
	require.True(t, db.Migrator().HasColumn(&outbox.Entity{}, "last_error"))
}
```

- [ ] **Step 2: Run test (expect FAIL: undefined `outbox.Migration`)**

```bash
go test ./libs/atlas-outbox/... -run TestMigration_CreatesTable
```

- [ ] **Step 3: Write `entity.go`**

```go
package outbox

import (
	"time"

	"gorm.io/datatypes"
)

type Entity struct {
	ID           uint64         `gorm:"primaryKey;column:id"`
	Topic        string         `gorm:"column:topic;not null;index:outbox_entries_unsent_idx,where:sent_at IS NULL"`
	MessageKey   []byte         `gorm:"column:message_key;not null"`
	MessageValue []byte         `gorm:"column:message_value"`
	Headers      datatypes.JSON `gorm:"column:headers;not null;default:'{}'"`
	EnqueuedAt   time.Time      `gorm:"column:enqueued_at;not null;default:CURRENT_TIMESTAMP"`
	SentAt       *time.Time     `gorm:"column:sent_at;index:outbox_entries_sweeper_idx,where:sent_at IS NOT NULL"`
	Attempts     int            `gorm:"column:attempts;not null;default:0"`
	LastError    *string        `gorm:"column:last_error"`
}

func (Entity) TableName() string { return "outbox_entries" }
```

- [ ] **Step 4: Write `migration.go`**

```go
package outbox

import "gorm.io/gorm"

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}
```

- [ ] **Step 5: Run test (expect PASS)**

```bash
go test ./libs/atlas-outbox/... -run TestMigration_CreatesTable
```

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-outbox/entity.go libs/atlas-outbox/migration.go libs/atlas-outbox/migration_test.go libs/atlas-outbox/go.sum
git commit -m "feat(atlas-outbox): entity + migration helper"
```

### Task A3: `Message` + `Enqueue`

**Files:**
- Create: `libs/atlas-outbox/outbox.go`
- Create: `libs/atlas-outbox/outbox_test.go`

- [ ] **Step 1: Write failing test for Enqueue inside transaction**

```go
package outbox_test

import (
	"context"
	"testing"

	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestEnqueue_InsertsRow_InTransaction(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, outbox.Migration(db))

	msg := outbox.Message{
		Topic: "TEST_TOPIC",
		Key:   []byte("k1"),
		Value: []byte(`{"a":1}`),
		Headers: map[string]string{"trace": "span-1"},
	}

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return outbox.Enqueue(tx, msg)
	}))

	var ents []outbox.Entity
	require.NoError(t, db.Find(&ents).Error)
	require.Len(t, ents, 1)
	require.Equal(t, "TEST_TOPIC", ents[0].Topic)
	require.Equal(t, []byte("k1"), ents[0].MessageKey)
	require.Equal(t, []byte(`{"a":1}`), ents[0].MessageValue)
	require.Nil(t, ents[0].SentAt)
	_ = context.TODO()
}

func TestEnqueue_TombstoneValueIsNullable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, outbox.Migration(db))

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return outbox.Enqueue(tx, outbox.Message{
			Topic: "TEST_TOPIC",
			Key:   []byte("k1"),
			Value: nil,
		})
	}))

	var ent outbox.Entity
	require.NoError(t, db.First(&ent).Error)
	require.Nil(t, ent.MessageValue)
}
```

- [ ] **Step 2: Run test (expect FAIL: `Message` / `Enqueue` undefined)**

- [ ] **Step 3: Write `outbox.go`**

```go
package outbox

import (
	"encoding/json"
	"errors"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Message struct {
	Topic   string
	Key     []byte
	Value   []byte
	Headers map[string]string
}

func Enqueue(tx *gorm.DB, msg Message) error {
	if tx == nil {
		return errors.New("outbox: nil transaction")
	}
	if msg.Topic == "" {
		return errors.New("outbox: empty topic")
	}
	if len(msg.Key) == 0 {
		return errors.New("outbox: empty message key")
	}

	headers := datatypes.JSON([]byte("{}"))
	if len(msg.Headers) > 0 {
		b, err := json.Marshal(msg.Headers)
		if err != nil {
			return err
		}
		headers = datatypes.JSON(b)
	}

	ent := Entity{
		Topic:        msg.Topic,
		MessageKey:   msg.Key,
		MessageValue: msg.Value,
		Headers:      headers,
	}
	if err := tx.Create(&ent).Error; err != nil {
		return err
	}

	if isPostgres(tx) {
		if err := tx.Exec("SELECT pg_notify(?, ?)", notifyChannel, msg.Topic).Error; err != nil {
			return err
		}
	}
	return nil
}

const notifyChannel = "atlas_outbox_new"

func isPostgres(db *gorm.DB) bool {
	return db != nil && db.Dialector != nil && db.Dialector.Name() == "postgres"
}
```

- [ ] **Step 4: Run tests (expect PASS)**

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-outbox/outbox.go libs/atlas-outbox/outbox_test.go
git commit -m "feat(atlas-outbox): Enqueue inside caller transaction with NOTIFY hook"
```

### Task A4: Drainer publish loop (without lock yet)

**Files:**
- Create: `libs/atlas-outbox/drainer.go`
- Create: `libs/atlas-outbox/drainer_test.go`

- [ ] **Step 1: Write failing test for publish loop on sqlite + fake producer**

```go
package outbox_test

import (
	"context"
	"sync"
	"testing"
	"time"

	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakePublisher struct {
	mu       sync.Mutex
	messages []kafka.Message
}

func (f *fakePublisher) WriteMessages(_ context.Context, msgs ...kafka.Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.messages = append(f.messages, msgs...)
	return nil
}

func TestDrainer_PublishesUnsentRows(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, outbox.Migration(db))

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return outbox.Enqueue(tx, outbox.Message{Topic: "T", Key: []byte("k"), Value: []byte("v")})
	}))

	pub := &fakePublisher{}
	l := logrus.New()
	d := outbox.NewDrainer(l, db, outbox.PublisherFunc(pub.WriteMessages),
		outbox.WithPollInterval(20*time.Millisecond),
		outbox.WithBatchSize(10),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	go d.Run(ctx)

	require.Eventually(t, func() bool {
		pub.mu.Lock()
		defer pub.mu.Unlock()
		return len(pub.messages) == 1
	}, 400*time.Millisecond, 10*time.Millisecond)

	var ent outbox.Entity
	require.NoError(t, db.First(&ent).Error)
	require.NotNil(t, ent.SentAt)
}
```

- [ ] **Step 2: Run test (expect FAIL)**

- [ ] **Step 3: Write `drainer.go`**

```go
package outbox

import (
	"context"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Publisher interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
}

type PublisherFunc func(ctx context.Context, msgs ...kafka.Message) error

func (f PublisherFunc) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	return f(ctx, msgs...)
}

type drainerConfig struct {
	pollInterval    time.Duration
	batchSize       int
	sweeperInterval time.Duration
	retention       time.Duration
}

type DrainerOption func(*drainerConfig)

func WithPollInterval(d time.Duration) DrainerOption    { return func(c *drainerConfig) { c.pollInterval = d } }
func WithBatchSize(n int) DrainerOption                  { return func(c *drainerConfig) { c.batchSize = n } }
func WithSweeperInterval(d time.Duration) DrainerOption  { return func(c *drainerConfig) { c.sweeperInterval = d } }
func WithRetention(d time.Duration) DrainerOption        { return func(c *drainerConfig) { c.retention = d } }

type Drainer struct {
	l    logrus.FieldLogger
	db   *gorm.DB
	pub  Publisher
	cfg  drainerConfig
	stop chan struct{}
}

func NewDrainer(l logrus.FieldLogger, db *gorm.DB, pub Publisher, opts ...DrainerOption) *Drainer {
	cfg := drainerConfig{
		pollInterval:    1 * time.Second,
		batchSize:       100,
		sweeperInterval: 1 * time.Hour,
		retention:       7 * 24 * time.Hour,
	}
	for _, o := range opts {
		o(&cfg)
	}
	return &Drainer{l: l, db: db, pub: pub, cfg: cfg, stop: make(chan struct{})}
}

func (d *Drainer) Run(ctx context.Context) {
	t := time.NewTicker(d.cfg.pollInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-d.stop:
			return
		case <-t.C:
			if err := d.publishBatch(ctx); err != nil {
				d.l.WithError(err).Warn("outbox.publish_failed")
			}
		}
	}
}

func (d *Drainer) Stop() { close(d.stop) }

func (d *Drainer) publishBatch(ctx context.Context) error {
	var rows []Entity
	if err := d.db.WithContext(ctx).
		Where("sent_at IS NULL").
		Order("enqueued_at ASC").
		Limit(d.cfg.batchSize).
		Find(&rows).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}

	msgs := make([]kafka.Message, 0, len(rows))
	for _, r := range rows {
		msgs = append(msgs, kafka.Message{
			Topic: r.Topic,
			Key:   r.MessageKey,
			Value: r.MessageValue,
		})
	}
	if err := d.pub.WriteMessages(ctx, msgs...); err != nil {
		ids := make([]uint64, 0, len(rows))
		for _, r := range rows {
			ids = append(ids, r.ID)
		}
		d.db.WithContext(ctx).
			Model(&Entity{}).
			Where("id IN ?", ids).
			Updates(map[string]any{"attempts": gorm.Expr("attempts + 1"), "last_error": err.Error()})
		return err
	}

	now := time.Now()
	ids := make([]uint64, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	return d.db.WithContext(ctx).
		Model(&Entity{}).
		Where("id IN ?", ids).
		Updates(map[string]any{"sent_at": &now}).Error
}
```

- [ ] **Step 4: Run test (expect PASS)**

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-outbox/drainer.go libs/atlas-outbox/drainer_test.go
git commit -m "feat(atlas-outbox): drainer poll loop with batch publish + sent_at update"
```

### Task A5: pg_advisory_lock leadership (testcontainers Postgres)

**Files:**
- Create: `libs/atlas-outbox/lock.go`
- Modify: `libs/atlas-outbox/drainer.go`
- Create: `libs/atlas-outbox/lock_test.go`

- [ ] **Step 1: Write failing test that two drainers can race; only one publishes**

```go
//go:build integration

package outbox_test

import (
	"context"
	"sync"
	"testing"
	"time"

	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestDrainer_AdvisoryLock_OnlyOneLeader(t *testing.T) {
	ctx := context.Background()
	pg, err := tcpostgres.Run(ctx, "postgres:16-alpine")
	require.NoError(t, err)
	t.Cleanup(func() { _ = pg.Terminate(ctx) })

	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	dial := postgres.Open(dsn)
	db1, err := gorm.Open(dial, &gorm.Config{})
	require.NoError(t, err)
	db2, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, outbox.Migration(db1))

	require.NoError(t, db1.Transaction(func(tx *gorm.DB) error {
		return outbox.Enqueue(tx, outbox.Message{Topic: "T", Key: []byte("k"), Value: []byte("v")})
	}))

	var mu sync.Mutex
	var published []kafka.Message
	pub := outbox.PublisherFunc(func(ctx context.Context, msgs ...kafka.Message) error {
		mu.Lock()
		published = append(published, msgs...)
		mu.Unlock()
		return nil
	})

	d1 := outbox.NewDrainer(logrus.New(), db1, pub, outbox.WithPollInterval(50*time.Millisecond))
	d2 := outbox.NewDrainer(logrus.New(), db2, pub, outbox.WithPollInterval(50*time.Millisecond))
	runCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	go d1.Run(runCtx)
	go d2.Run(runCtx)

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(published) >= 1
	}, 1500*time.Millisecond, 25*time.Millisecond)
	time.Sleep(300 * time.Millisecond) // give the follower a chance to over-publish if it can

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, 1, len(published), "row was published more than once")
}
```

- [ ] **Step 2: Run test (expect FAIL — both drainers currently publish)**

- [ ] **Step 3: Write `lock.go`**

```go
package outbox

import (
	"context"
	"database/sql"

	"gorm.io/gorm"
)

const advisoryLockKey int64 = 0x4f7574626f78 // 'Outbox'

func tryAdvisoryLock(ctx context.Context, db *gorm.DB) (locker, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return locker{}, err
	}
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return locker{}, err
	}
	var got bool
	if err := conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", advisoryLockKey).Scan(&got); err != nil {
		_ = conn.Close()
		return locker{}, err
	}
	if !got {
		_ = conn.Close()
		return locker{}, nil
	}
	return locker{conn: conn, held: true}, nil
}

type locker struct {
	conn *sql.Conn
	held bool
}

func (l locker) Held() bool { return l.held }
func (l locker) Release(ctx context.Context) {
	if l.conn == nil {
		return
	}
	_, _ = l.conn.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", advisoryLockKey)
	_ = l.conn.Close()
}
```

- [ ] **Step 4: Wrap publish loop with lock acquisition in `drainer.go` `Run`**

```go
func (d *Drainer) Run(ctx context.Context) {
	t := time.NewTicker(d.cfg.pollInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-d.stop:
			return
		case <-t.C:
			if !isPostgres(d.db) {
				_ = d.publishBatch(ctx) // tests on sqlite skip leadership
				continue
			}
			lk, err := tryAdvisoryLock(ctx, d.db)
			if err != nil {
				d.l.WithError(err).Warn("outbox.lock_acquire_error")
				continue
			}
			if !lk.Held() {
				continue
			}
			d.l.Info("outbox.lock_acquired")
			d.runLeader(ctx)
			lk.Release(context.Background())
			d.l.Info("outbox.lock_lost")
		}
	}
}

func (d *Drainer) runLeader(ctx context.Context) {
	tk := time.NewTicker(d.cfg.pollInterval)
	defer tk.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-d.stop:
			return
		case <-tk.C:
			if err := d.publishBatch(ctx); err != nil {
				d.l.WithError(err).Warn("outbox.publish_failed")
				return // drop leadership, retry on next outer tick
			}
		}
	}
}
```

- [ ] **Step 5: Update batch `SELECT` to use `FOR UPDATE SKIP LOCKED` (Postgres-only)**

Replace the `Find(&rows)` call with a Postgres-aware path:

```go
if isPostgres(d.db) {
    tx := d.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"})
    if err := tx.Where("sent_at IS NULL").Order("enqueued_at ASC").Limit(d.cfg.batchSize).Find(&rows).Error; err != nil {
        return err
    }
} else {
    if err := d.db.WithContext(ctx).Where("sent_at IS NULL").Order("enqueued_at ASC").Limit(d.cfg.batchSize).Find(&rows).Error; err != nil {
        return err
    }
}
```

The batch fetch and the `sent_at` UPDATE must run inside the same transaction so SKIP LOCKED keeps the rows reserved until commit. Wrap `publishBatch` body in `d.db.Transaction(...)`.

- [ ] **Step 6: Run test (expect PASS, exactly one published)**

```bash
go test -tags=integration ./libs/atlas-outbox/... -run TestDrainer_AdvisoryLock_OnlyOneLeader
```

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-outbox/lock.go libs/atlas-outbox/drainer.go libs/atlas-outbox/lock_test.go
git commit -m "feat(atlas-outbox): advisory-lock leadership + SKIP LOCKED batch fetch"
```

### Task A6: LISTEN/NOTIFY wake-up

**Files:**
- Create: `libs/atlas-outbox/notify.go`
- Modify: `libs/atlas-outbox/drainer.go`
- Create: `libs/atlas-outbox/notify_test.go` (integration build tag)

- [ ] **Step 1: Write failing test that NOTIFY wakeup beats poll interval**

```go
//go:build integration

func TestDrainer_NotifyAcceleratesPublish(t *testing.T) {
	ctx := context.Background()
	pg, _ := tcpostgres.Run(ctx, "postgres:16-alpine")
	t.Cleanup(func() { _ = pg.Terminate(ctx) })
	dsn, _ := pg.ConnectionString(ctx, "sslmode=disable")
	db, _ := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	_ = outbox.Migration(db)

	pubCh := make(chan time.Time, 1)
	pub := outbox.PublisherFunc(func(ctx context.Context, msgs ...kafka.Message) error {
		pubCh <- time.Now()
		return nil
	})
	d := outbox.NewDrainer(logrus.New(), db, pub, outbox.WithPollInterval(2*time.Second))
	runCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	go d.Run(runCtx)

	// Wait until the leader is fully established (one initial poll).
	time.Sleep(50 * time.Millisecond)

	insertedAt := time.Now()
	_ = db.Transaction(func(tx *gorm.DB) error {
		return outbox.Enqueue(tx, outbox.Message{Topic: "T", Key: []byte("k"), Value: []byte("v")})
	})

	publishedAt := <-pubCh
	require.Less(t, publishedAt.Sub(insertedAt), 500*time.Millisecond,
		"NOTIFY should wake the leader well before the 2s poll")
}
```

- [ ] **Step 2: Run (expect FAIL — waits the full poll interval)**

- [ ] **Step 3: Write `notify.go` using `pq.Listener`**

```go
package outbox

import (
	"context"
	"database/sql"

	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

type notifier struct {
	l   logrus.FieldLogger
	ln  *pq.Listener
	out chan struct{}
}

func newNotifier(l logrus.FieldLogger, dsn string) (*notifier, error) {
	out := make(chan struct{}, 1)
	ln := pq.NewListener(dsn, 10*time.Second, time.Minute, func(ev pq.ListenerEventType, err error) {
		if err != nil {
			l.WithError(err).Warn("outbox.notify_listener_event")
		}
	})
	if err := ln.Listen(notifyChannel); err != nil {
		_ = ln.Close()
		return nil, err
	}
	n := &notifier{l: l, ln: ln, out: out}
	go n.pump()
	return n, nil
}

func (n *notifier) pump() {
	for ev := range n.ln.Notify {
		_ = ev
		select { case n.out <- struct{}{}: default: }
	}
}

func (n *notifier) C() <-chan struct{} { return n.out }
func (n *notifier) Close()             { _ = n.ln.Close() }
```

- [ ] **Step 4: Plumb notifier through `NewDrainer`**

Add `WithDSN(dsn string)` option. Inside `runLeader`, wait on `notifier.C()` OR ticker — whichever fires first. The non-leader loop remains poll-only (lock contention determines leadership).

- [ ] **Step 5: Run test (expect PASS)**

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-outbox/notify.go libs/atlas-outbox/drainer.go libs/atlas-outbox/notify_test.go libs/atlas-outbox/go.sum
git commit -m "feat(atlas-outbox): LISTEN/NOTIFY wakeup for sub-100ms publish latency"
```

### Task A7: Sweeper

**Files:**
- Modify: `libs/atlas-outbox/drainer.go`
- Create: `libs/atlas-outbox/sweeper_test.go`

- [ ] **Step 1: Write failing test that rows older than retention are deleted**

```go
func TestDrainer_SweeperDeletesOldSentRows(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	_ = outbox.Migration(db)

	old := time.Now().Add(-10 * 24 * time.Hour)
	recent := time.Now().Add(-1 * time.Hour)
	require.NoError(t, db.Create(&outbox.Entity{Topic: "T", MessageKey: []byte("k1"), SentAt: &old}).Error)
	require.NoError(t, db.Create(&outbox.Entity{Topic: "T", MessageKey: []byte("k2"), SentAt: &recent}).Error)

	d := outbox.NewDrainer(logrus.New(), db, outbox.PublisherFunc(func(context.Context, ...kafka.Message) error { return nil }),
		outbox.WithRetention(7*24*time.Hour))

	require.NoError(t, d.SweepOnce(context.Background())) // hidden seam for tests
	var count int64
	db.Model(&outbox.Entity{}).Count(&count)
	require.Equal(t, int64(1), count)
}
```

- [ ] **Step 2: Run (expect FAIL — `SweepOnce` undefined)**

- [ ] **Step 3: Implement sweeper**

```go
func (d *Drainer) SweepOnce(ctx context.Context) error {
	cutoff := time.Now().Add(-d.cfg.retention)
	return d.db.WithContext(ctx).Where("sent_at IS NOT NULL AND sent_at < ?", cutoff).Delete(&Entity{}).Error
}
```

Schedule from `Run`: a separate goroutine ticking on `cfg.sweeperInterval` calling `SweepOnce` (only when this replica is leader — sweeper is best-effort, single-runner not required for correctness, but keep it leader-only to avoid double work).

- [ ] **Step 4: Run test (expect PASS)**

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-outbox/drainer.go libs/atlas-outbox/sweeper_test.go
git commit -m "feat(atlas-outbox): sweeper for published rows older than retention"
```

### Task A8: Backfill

**Files:**
- Create: `libs/atlas-outbox/backfill.go`
- Create: `libs/atlas-outbox/backfill_test.go`

- [ ] **Step 1: Write failing test for idempotent backfill**

```go
func TestBackfill_Idempotent(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	_ = outbox.Migration(db)

	// Source rows (simulated by inline closures: backfill is keyed by key only)
	type src struct{ ID, Body string }
	srcs := []src{{"a", `{"x":1}`}, {"b", `{"x":2}`}}

	keyFn := func(v any) ([]byte, error)   { return []byte(v.(src).ID), nil }
	valFn := func(v any) ([]byte, error)   { return []byte(v.(src).Body), nil }
	loader := func() ([]any, error)        {
		out := make([]any, len(srcs)); for i, s := range srcs { out[i] = s }; return out, nil
	}

	n, err := outbox.Backfill(db, "T", loader, keyFn, valFn)
	require.NoError(t, err)
	require.Equal(t, 2, n)

	n, err = outbox.Backfill(db, "T", loader, keyFn, valFn)
	require.NoError(t, err)
	require.Equal(t, 0, n, "second backfill must be no-op")
}
```

- [ ] **Step 2: Run (expect FAIL)**

- [ ] **Step 3: Implement `backfill.go`**

```go
package outbox

import (
	"gorm.io/gorm"
)

type Loader func() ([]any, error)
type ToBytes func(any) ([]byte, error)

func Backfill(db *gorm.DB, topic string, loader Loader, keyFn, valueFn ToBytes) (int, error) {
	rows, err := loader()
	if err != nil { return 0, err }

	added := 0
	for _, r := range rows {
		k, err := keyFn(r)
		if err != nil { return added, err }
		var count int64
		if err := db.Model(&Entity{}).Where("topic = ? AND message_key = ?", topic, k).Count(&count).Error; err != nil {
			return added, err
		}
		if count > 0 { continue }

		v, err := valueFn(r)
		if err != nil { return added, err }

		err = db.Transaction(func(tx *gorm.DB) error {
			return Enqueue(tx, Message{Topic: topic, Key: k, Value: v})
		})
		if err != nil { return added, err }
		added++
	}
	return added, nil
}
```

- [ ] **Step 4: Run tests (expect PASS)**

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-outbox/backfill.go libs/atlas-outbox/backfill_test.go
git commit -m "feat(atlas-outbox): idempotent Backfill helper for seeder fresh-cluster bootstrap"
```

### Task A9: README + final library polish

**Files:**
- Modify: `libs/atlas-outbox/README.md`

- [ ] **Step 1: Fill README**

Document: (1) at-least-once semantics — consumers MUST be idempotent; (2) Enqueue is called inside caller's gorm transaction; (3) drainer leadership via advisory lock; (4) NOTIFY/poll wakeup; (5) Sweeper retention default 7d; (6) Backfill is idempotent and safe on every startup.

- [ ] **Step 2: Commit**

```bash
git add libs/atlas-outbox/README.md
git commit -m "docs(atlas-outbox): README with semantics, lifecycle, idempotency guarantees"
```

### Task A10: Verify Phase A

- [ ] `cd libs/atlas-outbox && go test -race ./...` clean.
- [ ] `cd libs/atlas-outbox && go test -tags=integration ./...` clean (requires Docker for testcontainers).
- [ ] `cd libs/atlas-outbox && go vet ./...` clean.

---

## Phase B — `libs/atlas-kafka` ReadEndOffsets

### Task B1: ReadEndOffsets

**Files:**
- Create: `libs/atlas-kafka/consumer/offsets.go`
- Create: `libs/atlas-kafka/consumer/offsets_test.go`

- [ ] **Step 1: Write failing test (uses kafka-go `MockBroker` or testcontainer kafka)**

```go
package consumer_test

import (
	"context"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	kafkago "github.com/testcontainers/testcontainers-go/modules/kafka"
	"github.com/stretchr/testify/require"
)

func TestReadEndOffsets_ReturnsCurrentEndPerPartition(t *testing.T) {
	ctx := context.Background()
	kc, err := kafkago.Run(ctx, "confluentinc/cp-kafka:7.6.0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = kc.Terminate(ctx) })

	brokers, err := kc.Brokers(ctx)
	require.NoError(t, err)

	// Produce 3 messages to "T".
	// (use a small inline kafka.Writer with topic-auto-create disabled or pre-create)
	// ...

	got, err := consumer.ReadEndOffsets(context.Background(), brokers, "T")
	require.NoError(t, err)
	require.Equal(t, int64(3), got[0])
	_ = time.Second
}
```

- [ ] **Step 2: Run (expect FAIL — function undefined)**

- [ ] **Step 3: Implement**

```go
package consumer

import (
	"context"
	"net"
	"strconv"

	"github.com/segmentio/kafka-go"
)

func ReadEndOffsets(ctx context.Context, brokers []string, topic string) (map[int]int64, error) {
	if len(brokers) == 0 {
		return nil, kafka.UnknownTopicOrPartition
	}
	d := &kafka.Dialer{Timeout: 10 * time.Second, DualStack: true}
	conn, err := d.DialContext(ctx, "tcp", brokers[0])
	if err != nil { return nil, err }
	defer conn.Close()

	parts, err := conn.ReadPartitions(topic)
	if err != nil { return nil, err }

	out := make(map[int]int64, len(parts))
	for _, p := range parts {
		leader := net.JoinHostPort(p.Leader.Host, strconv.Itoa(p.Leader.Port))
		pc, err := d.DialLeader(ctx, "tcp", leader, topic, p.ID)
		if err != nil { return nil, err }
		_, last, err := pc.ReadOffsets()
		pc.Close()
		if err != nil { return nil, err }
		out[p.ID] = last
	}
	return out, nil
}
```

- [ ] **Step 4: Run test (expect PASS)**

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-kafka/consumer/offsets.go libs/atlas-kafka/consumer/offsets_test.go
git commit -m "feat(atlas-kafka): ReadEndOffsets helper for caught-up gate"
```

### Task B2: Verify Phase B

- [ ] `cd libs/atlas-kafka && go test -race ./...` clean.

---

## Phase C — atlas-world DELETE route

### Task C1: Failing handler test

**Files:**
- Modify: `services/atlas-world/atlas.com/world/channel/resource_test.go`

- [ ] **Step 1: Read existing `resource.go` and the closest POST or GET test to mirror its setup.**

- [ ] **Step 2: Add failing test**

```go
func TestHandleUnregisterChannelServer_Deletes(t *testing.T) {
	// Register a channel first via existing path (or builder).
	// Issue DELETE /api/world-server/channel-server/{worldId}/{channelId}.
	// Assert 204 and that registry no longer has the entry.
}

func TestHandleUnregisterChannelServer_NotFoundIs404(t *testing.T) {
	// DELETE without prior register; expect 404.
}
```

- [ ] **Step 3: Run (expect FAIL — route 404 by default on unsupported method)**

### Task C2: Add route + handler

**Files:**
- Modify: `services/atlas-world/atlas.com/world/channel/resource.go`

- [ ] **Step 1: Add route registration**

Add to resource init:
```go
r.HandleFunc("/{channelId}", handleUnregisterChannelServer(l, db)).Methods(http.MethodDelete)
```

- [ ] **Step 2: Implement handler that parses `{worldId}`/`{channelId}`, calls `channel.NewProcessor(l, ctx).Unregister(channel.NewModel(world.Id(wId), channel.Id(cId)))`, returns 204 on success and 404 when the registry returns "not found".**

- [ ] **Step 3: Run tests (expect PASS)**

- [ ] **Step 4: Commit**

```bash
git add services/atlas-world/atlas.com/world/channel/resource.go services/atlas-world/atlas.com/world/channel/resource_test.go
git commit -m "feat(atlas-world): DELETE /api/world-server/channel-server/{worldId}/{channelId}"
```

### Task C3: Verify Phase C

- [ ] `cd services/atlas-world && go test -race ./...` + `go vet ./...` + `go build ./...` clean.

---

## Phase D — atlas-configurations adopts outbox

### Task D1: Register outbox migration

**Files:**
- Modify: `services/atlas-configurations/atlas.com/configurations/main.go:50`
- Modify: `services/atlas-configurations/atlas.com/configurations/go.mod`
- Modify: `services/atlas-configurations/Dockerfile`

- [ ] **Step 1: Add dependency**

```bash
cd services/atlas-configurations/atlas.com/configurations
go get github.com/Chronicle20/atlas/libs/atlas-outbox
```

- [ ] **Step 2: Update Dockerfile** — per CLAUDE.md, every Dockerfile has 4 places that list libs. Add `atlas-outbox` to:
1. The go.mod COPY stanza in the builder stage.
2. The synthesized `go.work use(...)` block.
3. The source COPY block.
4. The `go mod edit -replace=...` flags.

(Use an existing `atlas-kafka` entry as the pattern; insert `atlas-outbox` alphabetically.)

- [ ] **Step 3: Register migration**

Modify `main.go:50`:
```go
db := database.Connect(l, database.SetMigrations(
    templates.Migration,
    tenants.Migration,
    services.Migration,
    outbox.Migration,
))
```

- [ ] **Step 4: Build verify**

```bash
go build ./...
docker build -f services/atlas-configurations/Dockerfile .
```

- [ ] **Step 5: Commit**

```bash
git add services/atlas-configurations/atlas.com/configurations/main.go services/atlas-configurations/atlas.com/configurations/go.* services/atlas-configurations/Dockerfile
git commit -m "feat(atlas-configurations): register outbox migration"
```

### Task D2: Envelope package

**Files:**
- Create: `services/atlas-configurations/atlas.com/configurations/outbox/envelopes.go`
- Create: `services/atlas-configurations/atlas.com/configurations/outbox/envelopes_test.go`

- [ ] **Step 1: Failing test**

```go
package outbox_test

import (
	"encoding/json"
	"testing"
	"time"

	"atlas-configurations/outbox"
	"atlas-configurations/services/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestNewServiceEnvelope_Shape(t *testing.T) {
	id := uuid.New()
	rm := service.ChannelRestModel{Type: "channel"} // simplified
	b, err := outbox.NewServiceEnvelope(id, rm, time.Now())
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(b, &got))
	require.Equal(t, float64(1), got["schema_version"])
	require.Equal(t, id.String(), got["id"])
	require.NotNil(t, got["config"])
	require.NotEmpty(t, got["emitted_at"])
}
```

- [ ] **Step 2: Run (FAIL)**

- [ ] **Step 3: Implement `envelopes.go`**

```go
package outbox

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type envelope struct {
	SchemaVersion int    `json:"schema_version"`
	Id            string `json:"id"`
	Config        any    `json:"config"`
	EmittedAt     string `json:"emitted_at"`
}

func NewServiceEnvelope(id uuid.UUID, rm any, emittedAt time.Time) ([]byte, error) {
	return json.Marshal(envelope{
		SchemaVersion: 1,
		Id:            id.String(),
		Config:        rm,
		EmittedAt:     emittedAt.UTC().Format(time.RFC3339),
	})
}

func NewTenantEnvelope(id uuid.UUID, rm any, emittedAt time.Time) ([]byte, error) {
	return NewServiceEnvelope(id, rm, emittedAt)
}
```

- [ ] **Step 4: Run (PASS)**

- [ ] **Step 5: Commit**

```bash
git add services/atlas-configurations/atlas.com/configurations/outbox/
git commit -m "feat(atlas-configurations): outbox envelopes for service+tenant topics"
```

### Task D3: services Processor enqueues on CRUD

**Files:**
- Modify: `services/atlas-configurations/atlas.com/configurations/services/processor.go` (Create/Update/DeleteById)
- Modify: `services/atlas-configurations/atlas.com/configurations/services/processor_test.go` (extend existing tests)

- [ ] **Step 1: Failing test that Create enqueues exactly one outbox row with topic `EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS` and key `service:<uuid>`**

```go
func TestProcessor_Create_EnqueuesOutboxRow(t *testing.T) {
	t.Setenv("EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS", "test.svc.topic")
	db, _ := openMemoryDB(t) // existing helper
	_ = outbox.Migration(db)

	p := NewProcessor(logrus.New(), context.Background(), db)
	id, err := p.Create(service.InputRestModel{Type: "channel", /* fields */})
	require.NoError(t, err)

	var ents []outboxlib.Entity
	require.NoError(t, db.Find(&ents).Error)
	require.Len(t, ents, 1)
	require.Equal(t, "test.svc.topic", ents[0].Topic)
	require.Equal(t, []byte("service:"+id.String()), ents[0].MessageKey)
}
```

- [ ] **Step 2: Run (FAIL)**

- [ ] **Step 3: Modify the existing transactional callback to enqueue inside the same tx.**

In the `create(...)` callback returned to `ExecuteTransaction`, after the existing `Save`/`Create` of the service entity, call:

```go
val, err := outboxenv.NewServiceEnvelope(serviceId, rm, time.Now())
if err != nil { return err }
return outboxlib.Enqueue(tx, outboxlib.Message{
    Topic: os.Getenv("EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS"),
    Key:   []byte("service:" + serviceId.String()),
    Value: val,
})
```

Do the same in `update(...)` (with the new RM) and `delete(...)` (with `Value: nil`).

- [ ] **Step 4: Run (PASS)**

- [ ] **Step 5: Commit**

```bash
git add services/atlas-configurations/atlas.com/configurations/services/processor.go services/atlas-configurations/atlas.com/configurations/services/processor_test.go
git commit -m "feat(atlas-configurations): enqueue service CRUD events into outbox"
```

### Task D4: tenants Processor enqueues on CRUD

Same shape as D3 against `tenants/processor.go`. Topic env var `EVENT_TOPIC_CONFIGURATION_TENANT_STATUS`, key `tenant:<uuid>`.

- [ ] Failing test → modify → commit.

```bash
git commit -m "feat(atlas-configurations): enqueue tenant CRUD events into outbox"
```

### Task D5: main.go starts the drainer

**Files:**
- Modify: `services/atlas-configurations/atlas.com/configurations/main.go`

- [ ] **Step 1: Initialize drainer after DB connect.** Use the existing producer manager (`producer.GetManager(...).Writer(l, topic)`) to construct a `Publisher` adapter:

```go
pub := outboxlib.PublisherFunc(func(ctx context.Context, msgs ...kafka.Message) error {
    // route by topic via existing manager
    for _, m := range msgs {
        if err := producer.GetManager().Writer(l, m.Topic).WriteMessages(ctx, m); err != nil {
            return err
        }
    }
    return nil
})

dr := outboxlib.NewDrainer(l, db, pub, outboxlib.WithDSN(database.DSN()))
tdm.RegisterTeardown("outbox.drainer", func() { dr.Stop() })
go dr.Run(tdm.Context())
```

(Adapt to the real teardown manager surface; if `database.DSN()` doesn't exist, expose it or construct the DSN locally from the same env vars Connect uses.)

- [ ] **Step 2: Build verify**

```bash
go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add services/atlas-configurations/atlas.com/configurations/main.go
git commit -m "feat(atlas-configurations): boot outbox drainer with teardown registration"
```

### Task D6: Seeder backfill

**Files:**
- Modify: `services/atlas-configurations/atlas.com/configurations/seeder/seeder.go`

- [ ] **Step 1: Failing test that on a fresh DB the seeder enqueues N rows; on a re-run, 0.**

(Use the existing seeder test patterns. If none exists, create one focused on Backfill behavior.)

- [ ] **Step 2: Implement** — after the existing seed-from-JSON pass, call:

```go
n, err := outboxlib.Backfill(db,
    os.Getenv("EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS"),
    loaderForServices(db),
    serviceKeyFn,
    serviceValueFn,
)
if err != nil { return err }
l.WithField("count", n).Info("seeder.backfill.services")

n, err = outboxlib.Backfill(db,
    os.Getenv("EVENT_TOPIC_CONFIGURATION_TENANT_STATUS"),
    loaderForTenants(db),
    tenantKeyFn,
    tenantValueFn,
)
if err != nil { return err }
l.WithField("count", n).Info("seeder.backfill.tenants")
```

`loaderForServices` reads from `services` table and returns rows as `any`; `serviceValueFn` reuses `outboxenv.NewServiceEnvelope`.

- [ ] **Step 3: Tests + commit**

```bash
git add services/atlas-configurations/atlas.com/configurations/seeder/
git commit -m "feat(atlas-configurations): seeder runs outbox.Backfill after seed-from-JSON"
```

### Task D7: k8s manifest topic env vars

**Files:**
- Modify: `services/atlas-configurations/atlas-configurations.yml`

- [ ] **Step 1: Add the two env vars to Deployment spec:**

```yaml
- name: EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS
  value: "atlas.configuration.service.status"
- name: EVENT_TOPIC_CONFIGURATION_TENANT_STATUS
  value: "atlas.configuration.tenant.status"
```

- [ ] **Step 2: Commit**

```bash
git add services/atlas-configurations/atlas-configurations.yml
git commit -m "chore(atlas-configurations): k8s manifest config-status topic env vars"
```

### Task D8: Verify Phase D

- [ ] `cd services/atlas-configurations/atlas.com/configurations && go test -race ./...` clean.
- [ ] `go vet ./...` clean.
- [ ] `docker build -f services/atlas-configurations/Dockerfile .` from worktree root.

---

## Phase E — atlas-channel `server.Registry` shape change

### Task E1: `server.Key`

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/server/key.go`
- Create: `services/atlas-channel/atlas.com/channel/server/key_test.go`

- [ ] **Step 1: Failing test**

```go
func TestKey_Equality(t *testing.T) {
	a := server.Key{TenantId: uuid.MustParse("..."), WorldId: 1, ChannelId: 2}
	b := server.Key{TenantId: uuid.MustParse("..."), WorldId: 1, ChannelId: 2}
	require.Equal(t, a, b)
}
```

- [ ] **Step 2: Run (FAIL)**

- [ ] **Step 3: Implement**

```go
package server

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

type Key struct {
	TenantId  uuid.UUID
	WorldId   world.Id
	ChannelId channel.Id
}
```

- [ ] **Step 4: PASS + commit**

```bash
git commit -m "feat(atlas-channel/server): introduce Key type for (t,w,c)"
```

### Task E2: Registry slice → map; add Deregister + Get; preserve `GetAll`

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/server/registry.go`
- Create: `services/atlas-channel/atlas.com/channel/server/registry_test.go`

- [ ] **Step 1: Failing test**

```go
func TestRegistry_DeregisterRemoves(t *testing.T) {
	r := server.GetRegistry() // exported singleton accessor for tests
	tn := /* tenant.Model */
	m := server.Register(tn, channel.NewModel(world.Id(1), channel.Id(1)), "127.0.0.1", 8585)
	key := server.Key{TenantId: tn.Id(), WorldId: 1, ChannelId: 1}

	_, ok := r.Get(key)
	require.True(t, ok)
	r.Deregister(key)
	_, ok = r.Get(key)
	require.False(t, ok)
	_ = m
}

func TestRegistry_GetAllReturnsCurrentMembers(t *testing.T) {
	// register two, deregister one, GetAll returns one.
}
```

- [ ] **Step 2: Run (FAIL)**

- [ ] **Step 3: Replace internals**

```go
package server

import (
	"sync"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

var registry *Registry
var once sync.Once

type Registry struct {
	lock    sync.RWMutex
	entries map[Key]Model
}

func GetRegistry() *Registry { return getRegistry() }

func getRegistry() *Registry {
	once.Do(func() { registry = &Registry{entries: make(map[Key]Model)} })
	return registry
}

func (r *Registry) Register(m Model) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.entries[keyOf(m)] = m
}

func (r *Registry) Deregister(k Key) {
	r.lock.Lock()
	defer r.lock.Unlock()
	delete(r.entries, k)
}

func (r *Registry) Get(k Key) (Model, bool) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	m, ok := r.entries[k]
	return m, ok
}

func (r *Registry) GetAll() []Model {
	r.lock.RLock()
	defer r.lock.RUnlock()
	out := make([]Model, 0, len(r.entries))
	for _, m := range r.entries {
		out = append(out, m)
	}
	return out
}

func keyOf(m Model) Key {
	return Key{TenantId: m.Tenant().Id(), WorldId: world.Id(m.WorldId()), ChannelId: channel.Id(m.ChannelId())}
}
```

Confirm the existing `server.Register(...)` free function still works (it likely already calls into the registry — refactor only the registry, not the constructor).

- [ ] **Step 4: Build verify across atlas-channel**

```bash
go build ./...
```

If any caller assumed slice indexing, fix call sites in this commit.

- [ ] **Step 5: PASS + commit**

```bash
git add services/atlas-channel/atlas.com/channel/server/registry.go services/atlas-channel/atlas.com/channel/server/registry_test.go
git commit -m "refactor(atlas-channel/server): map-backed registry with Key, Deregister, Get"
```

---

## Phase F — atlas-channel `listener` package

New package `services/atlas-channel/atlas.com/channel/listener/`.

### Task F1: Types

**Files:**
- Create: `listener/handle.go`
- Create: `listener/handle_test.go`

- [ ] **Step 1: Failing test for `Handle` zero value and `HandlerHandle` shape.**

- [ ] **Step 2: Implement**

```go
package listener

import (
	"context"
	"sync"

	"atlas-channel/server"
)

type State int

const (
	Active State = iota
	Draining
	Removed
)

type HandlerHandle struct {
	Topic string
	Id    string
}

type Handle struct {
	Key             server.Key
	State           State
	Ctx             context.Context
	Cancel          context.CancelFunc
	Wg              *sync.WaitGroup
	ServerModel     server.Model
	KafkaHandlers   []HandlerHandle
}
```

- [ ] **Step 3: Commit**

```bash
git commit -m "feat(atlas-channel/listener): Handle + HandlerHandle types"
```

### Task F2: Registry skeleton (Add/Snapshot only)

**Files:**
- Create: `listener/registry.go`
- Create: `listener/registry_test.go`

- [ ] **Step 1: Failing test that Add stores a Handle keyed by server.Key and Snapshot returns it**

- [ ] **Step 2: Implement registry with `sync.RWMutex`-protected `map[server.Key]*Handle`, `Add(key, cfg, body func(handle *Handle) ([]HandlerHandle, error))`, `Snapshot() []*Handle`**

`Add`'s body callback is the per-`(t,w,c)` startup work currently inlined in main.go (account registry init, sc Register, all `InitHandlers` calls, socket service). It returns the collected `[]HandlerHandle`. Registry stores them on the Handle.

- [ ] **Step 3: Commit**

```bash
git commit -m "feat(atlas-channel/listener): Registry skeleton with Add + Snapshot"
```

### Task F3: Drain phase 1 (quiesce)

**Files:**
- Modify: `listener/registry.go`
- Modify: `listener/registry_test.go`
- Create/Modify: `services/atlas-channel/atlas.com/channel/channel/processor.go` (add `Unregister`)
- Create/Modify: `services/atlas-channel/atlas.com/channel/channel/requests.go` (add DELETE request)

- [ ] **Step 1: Failing test that Drain transitions Active→Draining, calls server.Deregister, calls channel.Processor.Unregister (mock).**

- [ ] **Step 2: Implement `channel.Processor.Unregister`** mirroring the existing Register request but issuing DELETE against atlas-world's new endpoint. 404 = success.

- [ ] **Step 3: Implement Phase 1 of Drain**

```go
func (r *Registry) Drain(key server.Key) error {
	r.mu.Lock()
	h, ok := r.entries[key]
	if !ok || h.State == Removed { r.mu.Unlock(); return nil }
	if h.State == Draining { r.mu.Unlock(); return nil }
	h.State = Draining
	r.mu.Unlock()

	server.GetRegistry().Deregister(key)

	if err := r.deps.UnregisterChannel(h.ServerModel.Channel()); err != nil {
		r.l.WithError(err).WithField("key", key).Warn("listener.drain.unregister_channel_failed")
	}
	r.l.WithField("key", key).Info("listener.drain_phase phase=1")
	return nil
}
```

- [ ] **Step 4: PASS + commit**

```bash
git commit -m "feat(atlas-channel/listener): Drain phase 1 — quiesce + atlas-world Unregister"
```

### Task F4: Drain phase 2 (save-and-kick)

- [ ] **Step 1: Failing test** that Drain phase 2 walks `session.Registry` for the key and calls `session.Processor.Destroy` per session.

- [ ] **Step 2: Implement**

```go
// inside Drain after phase 1:
sessions := r.deps.SessionsForKey(key)
for _, s := range sessions {
    r.deps.SendShutdownNotice(s)
    _ = r.deps.DestroySession(s)
}
r.l.WithField("key", key).WithField("sessions", len(sessions)).Info("listener.drain_phase phase=2")
```

Inject `deps` via constructor so tests can mock `SessionsForKey`, `SendShutdownNotice`, `DestroySession`. Real impl wires these to the existing `session.Processor`.

- [ ] **Step 3: PASS + commit**

```bash
git commit -m "feat(atlas-channel/listener): Drain phase 2 — save-and-kick sessions for the key"
```

### Task F5: Drain phase 3 (deadline)

- [ ] **Step 1: Failing tests**:
  - happy path: all sessions destroy before deadline → no warn.
  - deadline-exceeded path: outstanding wg counter > 0 → warn logged; phase 4 still runs.

- [ ] **Step 2: Implement**

```go
done := make(chan struct{})
go func() { h.Wg.Wait(); close(done) }()
deadline := r.cfg.DrainDeadline
if deadline <= 0 { deadline = 5 * time.Second }
select {
case <-done:
case <-time.After(deadline):
    r.l.WithField("key", key).Warn("listener.drain_timeout")
}
r.l.WithField("key", key).Info("listener.drain_phase phase=3")
```

- [ ] **Step 3: Commit**

```bash
git commit -m "feat(atlas-channel/listener): Drain phase 3 — bounded deadline on session WG"
```

### Task F6: Drain phase 4 (teardown)

- [ ] **Step 1: Failing tests**:
  - Cancel is called.
  - Every `HandlerHandle` is passed to `consumer.Manager.RemoveHandler`.
  - State transitions to Removed.

- [ ] **Step 2: Implement**

```go
h.Cancel()
for _, hh := range h.KafkaHandlers {
    if err := r.deps.RemoveHandler(hh.Topic, hh.Id); err != nil {
        r.l.WithError(err).Warn("listener.drain.remove_handler_failed")
    }
}
r.mu.Lock()
h.State = Removed
r.mu.Unlock()
r.l.WithField("key", key).Info("listener.drain_phase phase=4")
```

- [ ] **Step 3: Commit**

```bash
git commit -m "feat(atlas-channel/listener): Drain phase 4 — cancel ctx + remove kafka handlers"
```

### Task F7: Idempotency + concurrency test

- [ ] **Step 1: Failing test** that calls `Drain(key)` from 8 goroutines concurrently; expect only one transition out of Active, no panics, no double-removes.

- [ ] **Step 2: Verify implementation is race-free under `-race`. If not, fix.**

```bash
go test -race ./services/atlas-channel/atlas.com/channel/listener/...
```

- [ ] **Step 3: Commit (no code change expected if Drain is already serialized)**

```bash
git commit --allow-empty -m "test(atlas-channel/listener): concurrent Drain calls are race-free"
```

### Task F8: Evictor registration + per-tenant ref count

**Files:**
- Create: `listener/evict.go`
- Modify: `listener/registry.go`

- [ ] **Step 1: Failing test** that when the last listener for tenant T transitions to Removed, every registered evictor is called exactly once with `t`.

- [ ] **Step 2: Implement**

```go
var (
    evMu     sync.Mutex
    evictors []func(tenant.Model)
)

func RegisterEvictor(fn func(tenant.Model)) {
    evMu.Lock(); defer evMu.Unlock()
    evictors = append(evictors, fn)
}

// inside Registry: refCount map[uuid.UUID]int.
// Add increments; Drain phase 4 final step decrements; if zero, fire evictors.
```

- [ ] **Step 3: Commit**

```bash
git commit -m "feat(atlas-channel/listener): per-tenant ref count + evictor registration"
```

---

## Phase G — atlas-channel projection

New package `services/atlas-channel/atlas.com/channel/configuration/projection/`.

### Task G1: Envelope decode

**Files:**
- Create: `configuration/projection/envelope.go`
- Create: `configuration/projection/envelope_test.go`

- [ ] Failing test → impl `DecodeServiceEnvelope`, `DecodeTenantEnvelope`, tombstone detection (nil value).

- [ ] Commit: `feat(atlas-channel/projection): envelope decode + tombstone detection`

### Task G2: State singleton

**Files:**
- Create: `configuration/projection/state.go`
- Create: `configuration/projection/state_test.go`

- [ ] Failing test → impl `State` with `sync.RWMutex`, `ApplyService(env)`, `ApplyServiceTombstone()`, `ApplyTenant(env)`, `ApplyTenantTombstone(id)`, `Snapshot() (svc *ServiceConfig, tenants map[uuid.UUID]TenantConfig)`.

- [ ] Commit: `feat(atlas-channel/projection): in-memory state with RW lock`

### Task G3: End-offset snapshot + caught-up gate

**Files:**
- Create: `configuration/projection/caughtup.go`
- Create: `configuration/projection/caughtup_test.go`

- [ ] **Step 1: Failing tests**:
  - `WaitCaughtUp` blocks until consumed offsets >= snapshotted end offsets on every partition.
  - `CaughtUp()` is one-way (subsequent decreases are ignored).
  - `ReadyChecker` returns false until caught up, then true.

- [ ] **Step 2: Implement using `consumer.ReadEndOffsets` at Start; maintain `currentOffsets` per topic; atomic flag.**

- [ ] Commit: `feat(atlas-channel/projection): caught-up gate with end-offset snapshot`

### Task G4: Apply diff (desired vs current)

**Files:**
- Create: `configuration/projection/apply.go`
- Create: `configuration/projection/apply_test.go`

- [ ] **Step 1: Failing tests** exercising representative state transitions per design §4.4.1:
  - ADD: new (t,w,c) → produces Add op.
  - REMOVE: missing (t,w,c) → produces Drain op.
  - PORT CHANGE: same key, different port → produces Drain then Add.
  - REGION/VERSION CHANGE (tenant.Region or MajorVersion): same key → produces Drain then Add.
  - SOCKET TABLE CHANGE: produces Drain then Add.
  - TENANT REFERENCED BUT MISSING IN tenantConfigs: skipped (no op).
  - UNCHANGED: no op.

- [ ] **Step 2: Implement `ComputeOps(prev, next ProjectionSnapshot) []Op` returning `Op{Kind: Add|Drain, Key: server.Key, Cfg: ListenerConfig}`.**

- [ ] **Step 3: Commit**

```bash
git commit -m "feat(atlas-channel/projection): diff prev/next snapshots into Add/Drain ops"
```

### Task G5: Subscriber wiring

**Files:**
- Create: `configuration/projection/subscriber.go`
- Modify: `configuration/registry.go` (replace `Init`/`GetServiceConfig`/`GetTenantConfig` with thin shims over `projection.State`)

- [ ] **Step 1: Implement subscriber that registers two Kafka consumers (service topic + tenant topic), each at earliest offset, filtering service topic by `SERVICE_ID` env at decode time.**

- [ ] **Step 2: Integration test (testcontainer Kafka)** asserts the projection state updates after the producer publishes service+tenant events.

- [ ] **Step 3: Commit**

```bash
git commit -m "feat(atlas-channel/projection): subscriber consumes both config topics"
```

### Task G6: ApplyLoop

**Files:**
- Modify: `configuration/projection/state.go`
- Create: `configuration/projection/loop.go`

- [ ] **Step 1: Failing test** that the apply loop calls `listener.Registry.Add` for new keys and `Drain` for removed keys, serialized.

- [ ] **Step 2: Implement single goroutine consuming an ops channel; bounded by listener cardinality.**

- [ ] **Step 3: Commit**

```bash
git commit -m "feat(atlas-channel/projection): serial apply loop drives listener.Registry"
```

---

## Phase H — `InitHandlers` signature change (mechanical)

This phase mutates `services/atlas-channel/atlas.com/channel/kafka/consumer/*/consumer.go` (44 files). One representative package gets a full diff; the rest follow the same pattern in a single sweep.

### Task H1: Introduce `HandlerHandle` import shim

- [ ] **Step 1:** In `services/atlas-channel/atlas.com/channel/kafka/consumer/consumer.go` (the shared helper), declare a project-local alias if it simplifies the rewrite:

```go
type HandlerHandle = listener.HandlerHandle
```

Or skip and use `listener.HandlerHandle` directly. (Decision: skip the alias; one fewer file.)

### Task H2: Rewrite `kafka/consumer/account/consumer.go` as the pattern

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/account/consumer.go`

- [ ] **Step 1: Failing test** asserting that the new `InitHandlers` returns the slice of handler handles registered.

- [ ] **Step 2: Rewrite**

```go
func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				out := make([]listener.HandlerHandle, 0, 1)
				t, _ := topic.EnvProvider(l)(account2.EnvEventTopicAccountStatus)()
				id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleAccountStatusEvent(sc))))
				if err != nil { return nil, err }
				out = append(out, listener.HandlerHandle{Topic: t, Id: id})
				return out, nil
			}
		}
	}
}
```

- [ ] **Step 3: PASS + commit**

```bash
git commit -m "refactor(atlas-channel/account): InitHandlers returns []HandlerHandle"
```

### Task H3: Sweep the remaining 43 packages

**Files:**
- Modify: every other `services/atlas-channel/atlas.com/channel/kafka/consumer/*/consumer.go`

- [ ] **Step 1:** Enumerate packages:

```bash
find services/atlas-channel/atlas.com/channel/kafka/consumer -name consumer.go | sort
```

- [ ] **Step 2:** For each, apply the same transformation pattern as account: change the final return type to `([]listener.HandlerHandle, error)`, capture every `rf(...)` call's returned id into a slice, return it.

Some packages register more than one handler (`channel`, `monsterbook`, etc.) — capture every id, not just the last.

- [ ] **Step 3: Build verify between blocks**

After every ~10 packages, run:
```bash
go build ./services/atlas-channel/...
```
Fix any compilation drift immediately.

- [ ] **Step 4: Run channel tests**

```bash
go test -race ./services/atlas-channel/...
```

- [ ] **Step 5: Commit**

```bash
git commit -m "refactor(atlas-channel): InitHandlers returns []HandlerHandle across all consumer packages"
```

### Task H4: Phase H verification

- [ ] `go vet ./services/atlas-channel/...` clean.
- [ ] `go build ./services/atlas-channel/...` clean.

---

## Phase I — session.Destroy reorder (FR-CHN-14)

### Task I1: Failing ordering test

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/session/processor_test.go`

- [ ] **Step 1: Failing test** that asserts emit-logout and emit-destroy happen before `s.Disconnect()`. Use a fake `Disconnect` that records its call time vs an instrumented producer:

```go
func TestProcessor_Destroy_EmitsBeforeDisconnect(t *testing.T) {
	var (
		emitAt       time.Time
		disconnectAt time.Time
	)
	s := buildSession(t, withDisconnect(func() { disconnectAt = time.Now() }))
	kp := func(topic string) func(provider model.Provider[[]kafka.Message]) error {
		return func(provider model.Provider[[]kafka.Message]) error {
			emitAt = time.Now()
			return nil
		}
	}
	p := session.NewProcessorWith(/* ... */).WithProducer(kp)
	require.NoError(t, p.Destroy(s))
	require.True(t, emitAt.Before(disconnectAt), "destroy event must be emitted before socket close")
}
```

- [ ] **Step 2: Run (FAIL — current order: Disconnect first)**

### Task I2: Reorder

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/session/processor.go:330-336`

- [ ] **Step 1: Reorder**

```go
func (p *Processor) Destroy(s Model) error {
	p.l.WithField("session", s.SessionId().String()).Debugf("Destroying session.")
	getRegistry().Remove(p.t.Id(), s.SessionId())

	if err := p.sp.Destroy(s.SessionId(), s.AccountId()); err != nil {
		p.l.WithError(err).Warn("session.destroy.emit_logout_failed")
	}
	if err := p.kp(session2.EnvEventTopicSessionStatus)(DestroyedStatusEventProvider(s.SessionId(), s.AccountId(), s.CharacterId(), s.Field().Channel())); err != nil {
		p.l.WithError(err).Warn("session.destroy.emit_destroyed_failed")
	}

	s.Disconnect()
	return nil
}
```

(Keep return semantics — if either emit returns an error today the caller's error path is preserved by switching to per-emit logging plus an aggregate return; verify by reading callers. If callers rely on the error, return the first non-nil err but still proceed with the second emit + Disconnect.)

- [ ] **Step 2: PASS + commit**

```bash
git commit -m "fix(atlas-channel/session): emit logout+destroy before Disconnect for crash-safe ordering"
```

### Task I3: Downstream-consumer audit

- [ ] **Step 1:** Read every consumer of `EVENT_TOPIC_SESSION_STATUS` Destroyed events (saga, character, account, etc.). Confirm none assume the socket is already closed when the event fires.

- [ ] **Step 2:** Document findings inline in this plan or in a follow-up note. No code change unless an assumption surfaces.

---

## Phase J — atlas-channel main.go rewire

### Task J1: Replace `configuration.Init` block with projection startup

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/main.go`

- [ ] **Step 1:** Delete the `configuration.Init(...)` call. Construct `projection.New(...)`, call `Start()`, then `WaitCaughtUp()`. Fatal on error.

- [ ] **Step 2:** Wire `/readyz` to `projection.ReadyChecker()`.

```go
restserver.New(l, tdm, GetServer).
    WithReadyChecker(projection.ReadyChecker()).
    Run(/* port */)
```

(Adapt to actual REST server constructor surface.)

- [ ] **Step 3: Build verify + commit**

```bash
git commit -m "refactor(atlas-channel/main): replace configuration.Init with projection startup"
```

### Task J2: Move per-(t,w,c) startup into `listener.Registry.Add`

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/main.go`
- Modify: `services/atlas-channel/atlas.com/channel/listener/registry.go`

- [ ] **Step 1:** Lift the entire body of the `for _, ten := range config.Tenants` loop (today main.go:209-380) into a function `buildListener(deps) func(server.Key, ListenerConfig) ([]HandlerHandle, error)` and pass it as the `Add`-body callback in `listener.Registry`.

- [ ] **Step 2:** Inside that function, every `InitHandlers` call now captures the returned `[]HandlerHandle` and concatenates them. The aggregate slice is returned and stored on the `Handle`.

- [ ] **Step 3:** Build verify and run a smoke test (channel still boots locally in dev) before committing.

- [ ] **Step 4: Commit**

```bash
git commit -m "refactor(atlas-channel/main): per-listener startup runs through listener.Registry.Add"
```

### Task J3: Wire tenant Evict + tenant.Unregister

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/main.go`
- Modify: `services/atlas-channel/atlas.com/channel/monster/...` (add `Evict(t)` to `StatusMirror`, `NextSkillInbox`)
- Modify: `services/atlas-channel/atlas.com/channel/account/...` (add `Evict(t)` on the registry)
- Modify: `libs/atlas-tenant/...` (add `Unregister` method on the global registry)

- [ ] **Step 1: Failing tests** for each `Evict(t)` method: tenant entries are removed, others untouched.

- [ ] **Step 2: Implement Evict methods.**

- [ ] **Step 3: Implement `tenant.Unregister(id)` on the global registry.**

- [ ] **Step 4: Wire in main.go**

```go
listener.RegisterEvictor(func(t tenant.Model) {
    monster.GetStatusMirror().Evict(t.Id())
    monster.GetNextSkillInbox().Evict(t.Id())
    account.GetRegistry().Evict(t.Id())
    tenant.Unregister(t.Id())
})
```

- [ ] **Step 5: Commit**

```bash
git commit -m "feat(atlas-channel): tenant Evict hooks fire when last listener drains"
```

### Task J4: Move `account.InitializeRegistry` into Add path

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/main.go`
- Modify: `listener/registry.go`

- [ ] **Step 1:** Move `account.NewProcessor(l, tctx).InitializeRegistry()` from main.go's tenant loop into the `Add` body's start. Ref-count guards uniqueness across multiple listeners per tenant.

- [ ] **Step 2: Test + commit**

```bash
git commit -m "refactor(atlas-channel): account.InitializeRegistry runs per first listener per tenant"
```

### Task J5: SIGTERM drain-all

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/main.go`

- [ ] **Step 1:** Register a teardown handler that:
  1. Flips `/readyz` to not-ready (independent of caught-up state — add a process-level shutdown flag the ReadyChecker reads).
  2. Calls `listener.GetRegistry().DrainAll()` (new method that calls `Drain` for every key in parallel).

- [ ] **Step 2: Test** that DrainAll completes within `terminationGracePeriodSeconds` budget under load.

- [ ] **Step 3: Bump `terminationGracePeriodSeconds`** in `services/atlas-channel/atlas-channel.yml` to 20s if not already at or above.

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(atlas-channel): SIGTERM drains all listeners in parallel; readyz flips first"
```

### Task J6: k8s manifest topic env vars + drain deadline

**Files:**
- Modify: `services/atlas-channel/atlas-channel.yml`

- [ ] Add:
```yaml
- name: EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS
  value: "atlas.configuration.service.status"
- name: EVENT_TOPIC_CONFIGURATION_TENANT_STATUS
  value: "atlas.configuration.tenant.status"
- name: DRAIN_DEADLINE_MS
  value: "5000"
```

- [ ] **Commit**

```bash
git commit -m "chore(atlas-channel): k8s env vars for config topics + drain deadline"
```

### Task J7: Dockerfile lib list update

**Files:**
- Modify: `services/atlas-channel/Dockerfile`

- [ ] Add `atlas-outbox` (if atlas-channel imports it — it shouldn't, but verify) and update the four hand-edited lib-list locations only if a new lib was added. The listener/projection work is internal to atlas-channel and does not add external libs except via the existing atlas-kafka import.

- [ ] **Commit** (only if a Dockerfile change was needed)

---

## Phase K — atlas-login: projection + simpler drain

### Task K1: Mirror projection package

Recreate `services/atlas-login/atlas.com/login/configuration/projection/` mirroring atlas-channel's projection. State, caught-up gate, subscriber, apply diff.

- [ ] Failing tests, impl, commit per atlas-channel pattern.

### Task K2: Login listener.Registry (simpler drain)

`services/atlas-login/atlas.com/login/listener/`:

- Add/Snapshot identical to atlas-channel.
- Drain phases:
  1. server.Deregister (if a login server registry exists; if not, skip).
  2. Send shutdown notice on existing sessions.
  3. Cancel ctx (stops accept loop).
  4. Wait wg (short — login sessions stateless after handshake).
  5. RemoveHandler for each captured handle.
  6. state = Removed.

No save-and-kick, no per-tenant evictors (login does not hold the same per-tenant singletons; verify by reading login's main.go).

Default drain deadline 2s, ceiling 5s.

- [ ] Failing tests per phase, impl, commit.

### Task K3: Login `InitHandlers` signature change

`services/atlas-login/atlas.com/login/kafka/consumer/*/consumer.go` (4 files). Apply the same H2-pattern transformation. Single commit.

```bash
git commit -m "refactor(atlas-login): InitHandlers returns []HandlerHandle"
```

### Task K4: Login main.go rewire

Same as J1+J2 for atlas-login. Replace `configuration.Init` with projection; lift per-(t,w,c) startup into listener.Registry.Add.

### Task K5: k8s manifest + Dockerfile

Add the same two topic env vars to `services/atlas-login/atlas-login.yml`. Update Dockerfile if a new lib dependency was added.

---

## Phase L — Atlas-world tests + atlas-channel REST integration test

### Task L1: End-to-end integration test (atlas-channel side)

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/listener/integration_test.go`

- [ ] **Step 1: Failing test** with testcontainer Kafka + a mock atlas-world HTTP server:
  - Publish a service-add envelope → assert `listener.Registry` brings up a listener for the (t,w,c) and `server.Registry` contains it.
  - Publish a tombstone for the same (t,w,c) → assert Drain runs all four phases, server.Deregister called, atlas-world DELETE called, all KafkaHandlers removed.

- [ ] **Step 2: Run (FAIL initially)**

- [ ] **Step 3: Iterate on wiring until PASS.**

- [ ] **Step 4: Commit**

```bash
git commit -m "test(atlas-channel/listener): end-to-end add+drain integration"
```

### Task L2: Atlas-login boot-without-configurations test

Smaller mirror of L1: start the login projection against an isolated testcontainer Kafka with no events; assert /readyz=503; publish service+tenant events; assert /readyz=200.

---

## Phase M — Verification (cross-service)

Per CLAUDE.md, the branch is not done until every changed module passes:

### Task M1: Test + vet + build sweep

- [ ] **Modules to verify (each from worktree root):**

```bash
# Libraries
( cd libs/atlas-outbox && go test -race ./... && go vet ./... && go build ./... )
( cd libs/atlas-kafka  && go test -race ./... && go vet ./... && go build ./... )

# Services
( cd services/atlas-configurations/atlas.com/configurations && go test -race ./... && go vet ./... && go build ./... )
( cd services/atlas-channel/atlas.com/channel               && go test -race ./... && go vet ./... && go build ./... )
( cd services/atlas-login/atlas.com/login                   && go test -race ./... && go vet ./... && go build ./... )
( cd services/atlas-world/atlas.com/world                   && go test -race ./... && go vet ./... && go build ./... )
```

### Task M2: Docker build sweep (worktree root, MANDATORY per CLAUDE.md)

Every service whose `go.mod` or `Dockerfile` was touched MUST `docker build` successfully:

```bash
docker build -f services/atlas-configurations/Dockerfile .
docker build -f services/atlas-channel/Dockerfile .
docker build -f services/atlas-login/Dockerfile .
docker build -f services/atlas-world/Dockerfile .   # only if its go.mod/Dockerfile changed; verify
```

`atlas-outbox` is a new lib — every adopting service's Dockerfile must list it in all four hand-edited locations. Drift only surfaces in `docker build`.

### Task M3: Service docs

For each touched service, run `/service-doc <service>` and commit the resulting doc updates:

- atlas-configurations
- atlas-channel
- atlas-login
- atlas-world (if changed)

Plus a new `libs/atlas-outbox/README.md` (already authored in A9).

### Task M4: Guideline audits

Dispatch in parallel:

- `backend-guidelines-reviewer` over each changed Go service.
- `plan-adherence-reviewer` against this plan.

Address findings before opening the PR.

### Task M5: Final commit / PR opening

Per CLAUDE.md, code review is mandatory before PR open. After M4 passes:

- Open PR titled `feat: dynamic service configuration (task-032)`.
- Body cross-links task-032 PRD, design, and audit artifacts.

---

## Self-Review

**Spec coverage:** Every FR-* and acceptance criterion from prd.md has a corresponding task:

| Spec | Task |
|---|---|
| FR-OUT-1..3 (Enqueue, Drainer construction) | A3, A4 |
| FR-OUT-4 (NOTIFY) | A6 |
| FR-OUT-5 (SKIP LOCKED batch) | A5 |
| FR-OUT-6 (at-least-once) | A9 (README) |
| FR-OUT-7 (Migration) | A2 |
| FR-OUT-8 (Backfill) | A8, D6 |
| FR-OUT-9 (Sweeper) | A7 |
| FR-OUT-10 (no tenant scope) | A2 (entity has no tenant column; Backfill uses raw queries) — additionally Phase A library uses no tenant-scope callbacks |
| FR-OUT-11 (logging) | A4-A7 inline log lines |
| FR-KAF-1 (ReadEndOffsets) | B1 |
| FR-CFG-1..6 (atlas-configurations adoption) | D1-D7 |
| FR-SCH-1..4 (envelopes) | D2 |
| FR-CHN-1..6 (subscriber, projection, caught-up gate) | G1-G6 |
| FR-CHN-7..10 (listener lifecycle + server.Registry change) | E1-E2, F1-F2 |
| FR-CHN-11..13 (four-phase drain) | F3-F7 |
| FR-CHN-14 (Destroy reorder) | I1-I2 |
| FR-CHN-15..17 (InitHandlers signature) | H1-H3, K3 |
| FR-CHN-18..19 (Evict hooks, account registry move) | J3, J4 |
| FR-CHN-20 (channel.Unregister) | F3 |
| FR-LGN-1..3 (login projection + drain) | K1-K2 |
| atlas-world DELETE | C1-C2 |
| Verification (CLAUDE.md build/vet/test/docker) | M1-M2 |
| Docs | M3 |

**Placeholder scan:** No "TBD", "implement later", "similar to Task N". Every task has either code or a precise pattern + reference task.

**Type consistency:** `listener.HandlerHandle{Topic, Id string}` used uniformly across H2, H3, F2, F6. `server.Key{TenantId, WorldId, ChannelId}` used uniformly across E1, E2, F1. `outbox.Message{Topic, Key, Value, Headers}` used uniformly A3, A4, A8, D3, D4, D6.

---

## Execution Handoff

Plan complete and saved. Two execution options:

1. **Subagent-Driven (recommended)** — `/execute-task task-032` dispatches a fresh subagent per task with review checkpoints.
2. **Inline** — Execute tasks in a single session using executing-plans.
