# Fleet-Wide Transactional Outbox Adoption — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Every transactional Atlas service persists its Kafka events as outbox rows in the same DB transaction as the domain mutation, so a rollback emits nothing and a commit publishes exactly the enqueued events.

**Architecture:** `libs/atlas-outbox` gains a publisher (`TopicWriterPool`, promoted from atlas-configurations), a buffer bridge (`EnqueueBuffer`), and an outbox-backed `producer.Provider` constructor (`EmitProvider`). Because every service-local `producer.Provider` is `func(token string) producer.MessageProducer`, existing `message.Emit`/`EmitWithResult` call sites accept `EmitProvider` without conversion — each migration is a mechanical inversion that moves the `Emit` call inside `database.ExecuteTransaction`. The lib drainer (advisory-lock leader, NOTIFY wakeup) publishes rows in `id` order with headers re-attached.

**Tech Stack:** Go, GORM (Postgres prod / sqlite tests), segmentio/kafka-go, logrus, go/analysis (CI guard).

## Global Constraints

- Design: `docs/tasks/task-114-outbox-adoption/design.md`; PRD: `prd.md` (same folder). Deviations from design are listed in `context.md` §Deviations and are binding.
- Existing lib API must not break: `Enqueue(tx, Message)`, `NewDrainer`, `Migration`, `Backfill` signatures unchanged. atlas-configurations compiles with no change other than the publisher import swap (Task 2).
- No event payload, schema, or topic changes. Consumer-visible bytes (topic, key, value, header set) identical to the direct path.
- Non-tx emits (commands, relays, socket fan-out, ticker emissions without DB writes, rejection/error status events reflecting no state change) stay on the direct producer path and are listed explicitly in `docs/tasks/task-114-outbox-adoption/inventory.md` — no silent skips.
- All committed file content uses repo-relative paths; never write absolute home paths into files.
- No `// TODO`, stubs, or 501s in landed commits (pre-existing TODO comments in untouched lines stay).
- Per changed module before its task's commit: `go test -race ./...`, `go vet ./...`, `go build ./...` — all clean, run from the module directory. Fleet-wide `docker buildx bake all-go-services`, `tools/redis-key-guard.sh`, and `tools/outbox-guard.sh` run once in Task 26 (design §6 chooses one bake over per-service bakes given breadth).
- Git: commit at the end of every task with the message given in the task. Work happens on branch `task-114-outbox-adoption` in the worktree `.worktrees/task-114-outbox-adoption`.

### Shared migration recipe (referenced by Tasks 7–23; treat as part of each task)

Every service carries local packages `kafka/message` (Buffer + `Emit`/`EmitWithResult`) and `kafka/producer` (`type Provider func(token string) producer.MessageProducer`). `outbox.EmitProvider(l, ctx, tx)` returns that exact unnamed func type, so it is assignable wherever a local `Provider` is expected. `database.ExecuteTransaction` is re-entrant (libs/atlas-database/transaction.go:9-14): a processor method that opens its own transaction runs directly on the given `tx` when it is already one, so wrapping an outer transaction around existing code is always safe.

Every migrating service file adds the import:

```go
outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
```

(in `main.go` use alias `outboxlib` to match atlas-configurations convention).

**Pattern A — `XAndEmit` inversion.** The dominant shape: `Emit` wraps a buffer-filling method that runs its own transaction internally, so today the flush happens after commit (crash window loses events).

Before (real code, services/atlas-character/atlas.com/character/character/processor.go:450-454):

```go
func (p *ProcessorImpl) ChangeJobAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model, jobId job.Id) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		return p.ChangeJob(buf)(transactionId, characterId, channel, jobId)
	})
}
```

After:

```go
func (p *ProcessorImpl) ChangeJobAndEmit(transactionId uuid.UUID, characterId uint32, channel channel.Model, jobId job.Id) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return p.WithTransaction(tx).ChangeJob(buf)(transactionId, characterId, channel, jobId)
		})
	})
}
```

The inner method is untouched: its own `ExecuteTransaction` re-enters, and its `buf.Put(...)` calls now flush to outbox rows inside the same transaction as its writes. If the processor has no `WithTransaction` method, construct a processor with `tx` as its db (`NewProcessor(l, ctx, tx)`) instead.

**Pattern B — `EmitWithResult` inversion.** Same move with the result captured outside:

```go
// before
return message.EmitWithResult[Model, Input](producer.ProviderImpl(p.l)(p.ctx))(p.DoThing)(input)

// after
var result Model
txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
	var err error
	result, err = message.EmitWithResult[Model, Input](outbox.EmitProvider(p.l, p.ctx, tx))(p.WithTransaction(tx).DoThing)(input)
	return err
})
return result, txErr
```

**Pattern C — call-site wrapping.** When the `Emit` lives outside the processor (consumers, tickers), wrap at the call site and hand the tx to the processor constructor. Real target (services/atlas-mounts/atlas.com/mounts/mount/task.go:29-35):

```go
// after
var applyTick = func(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, worldId world.Id, characterId uint32) error {
	return database.ExecuteTransaction(db.WithContext(ctx), func(tx *gorm.DB) error {
		p := NewProcessor(l, ctx, tx)
		return mountmessage.Emit(outbox.EmitProvider(l, ctx, tx))(func(mb *mountmessage.Buffer) error {
			return p.ApplyTick(mb)(worldId, characterId)
		})
	})
}
```

**Classification rule (design D7).** Migrate any emit whose events assert a DB state change: inside `ExecuteTransaction`, immediately after one, or wrapping bare `Save`/`Create`/`Update` writes (wrap those in an explicit `ExecuteTransaction` with the enqueue). Leave direct — and list in inventory.md with reason — rejection/error events reflecting no state change, pure relays, command emits to other services, socket fan-out, and emissions with no DB write.

**Per-service wiring template (design D6).** In the service `go.mod`:

```
require github.com/Chronicle20/atlas/libs/atlas-outbox v0.0.0-00010101000000-000000000000
replace github.com/Chronicle20/atlas/libs/atlas-outbox => ../../../../libs/atlas-outbox
```

then `go mod tidy` from the module dir. In `main.go` (import `outboxlib "github.com/Chronicle20/atlas/libs/atlas-outbox"`): append `outboxlib.Migration` as the last argument of the existing `database.SetMigrations(...)` call, and insert immediately after the `database.Connect(...)` line (every service main uses the `tdm := service.GetTeardownManager()` pattern; adapt the variable name if a service differs):

```go
// Boot the outbox drainer: publishes the transactional outbox to Kafka.
// Leadership is gated by a postgres advisory lock — replicas are safe.
publisher := outboxlib.NewTopicWriterPool()
drainer := outboxlib.NewDrainer(l, db, publisher, outboxlib.WithDSN(database.DSN()))
go drainer.Run(tdm.Context())
tdm.TeardownFunc(func() {
	drainer.Stop()
	publisher.Close()
})
```

Lib defaults everywhere (poll 1s, batch 100, retention 7d) — no per-service tuning.

**Per-service inventory entry** (append to `docs/tasks/task-114-outbox-adoption/inventory.md`): a `## atlas-<svc>` section with three lists: *Migrated* (file:line of each converted site), *Left direct* (file:line + one-line reason each), *Notes* (or the single line "Zero tx-coupled emit sites; no code change" where true).

---

## Phase 1 — libs/atlas-outbox

### Task 1: Header round-trip + id-ordered publishing in the drainer

The lib persists `msg.Headers` but `publishBatch` drops them (drainer.go:232-239), and it orders by `enqueued_at`, which is transaction-stable in Postgres so intra-tx order is a tie (design §1.1). Additionally — **planning-time discovery, amends design D2**: tenant version header values are raw big-endian uint16 bytes (libs/atlas-kafka/producer/header.go:38-39); they always contain a NUL byte (Postgres jsonb rejects ` `) and can be invalid UTF-8 (version 185 → 0xB9; `encoding/json` mangles it to U+FFFD). Stored header **values are therefore base64-encoded** inside the jsonb; the drainer decodes on publish. Byte-exact round trip; existing rows (atlas-configurations stores only `{}`) are unaffected.

**Files:**
- Create: `libs/atlas-outbox/headers.go`
- Modify: `libs/atlas-outbox/outbox.go` (Enqueue's marshal block), `libs/atlas-outbox/drainer.go` (publishBatch)
- Test: `libs/atlas-outbox/drainer_test.go`, `libs/atlas-outbox/outbox_test.go`

**Interfaces:**
- Consumes: existing `Enqueue`, `Entity`, `Drainer`, test helpers (`fakePublisher` in drainer_test.go).
- Produces: unexported `encodeHeaders(map[string]string) (datatypes.JSON, error)` and `decodeHeaders(datatypes.JSON) ([]kafka.Header, error)`; drainer publishes `kafka.Message` with `Headers` populated, rows ordered by `id ASC`. Public API unchanged.

- [ ] **Step 1: Write the failing tests**

Append to `libs/atlas-outbox/drainer_test.go` (reuse the existing `fakePublisher`):

```go
func TestDrainer_ReattachesHeaders(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, outbox.Migration(db))

	// Binary tenant-version value: NUL + 0xB9 (v185) — must survive byte-exact.
	binVal := string([]byte{0x00, 0xB9})
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return outbox.Enqueue(tx, outbox.Message{
			Topic: "T", Key: []byte("k"), Value: []byte("v"),
			Headers: map[string]string{"TENANT_ID": "abc", "MAJOR_VERSION": binVal},
		})
	}))

	pub := &fakePublisher{}
	d := outbox.NewDrainer(logrus.New(), db, outbox.PublisherFunc(pub.WriteMessages),
		outbox.WithPollInterval(20*time.Millisecond))
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	go d.Run(ctx)

	require.Eventually(t, func() bool {
		pub.mu.Lock()
		defer pub.mu.Unlock()
		return len(pub.messages) == 1
	}, 400*time.Millisecond, 10*time.Millisecond)

	got := map[string][]byte{}
	for _, h := range pub.messages[0].Headers {
		got[h.Key] = h.Value
	}
	require.Equal(t, []byte("abc"), got["TENANT_ID"])
	require.Equal(t, []byte{0x00, 0xB9}, got["MAJOR_VERSION"])
}

func TestDrainer_EmptyHeadersPublishNoHeaderSlice(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, outbox.Migration(db))
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return outbox.Enqueue(tx, outbox.Message{Topic: "T", Key: []byte("k"), Value: []byte("v")})
	}))

	pub := &fakePublisher{}
	d := outbox.NewDrainer(logrus.New(), db, outbox.PublisherFunc(pub.WriteMessages),
		outbox.WithPollInterval(20*time.Millisecond))
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	go d.Run(ctx)

	require.Eventually(t, func() bool {
		pub.mu.Lock()
		defer pub.mu.Unlock()
		return len(pub.messages) == 1
	}, 400*time.Millisecond, 10*time.Millisecond)
	require.Empty(t, pub.messages[0].Headers)
}

func TestDrainer_PublishesInIdOrderOnTimestampTie(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, outbox.Migration(db))

	// Same enqueued_at on every row — the Postgres intra-transaction reality.
	ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 1; i <= 3; i++ {
		require.NoError(t, db.Create(&outbox.Entity{
			Topic: "T", MessageKey: []byte("k"),
			MessageValue: []byte{byte(i)}, EnqueuedAt: ts,
		}).Error)
	}

	pub := &fakePublisher{}
	d := outbox.NewDrainer(logrus.New(), db, outbox.PublisherFunc(pub.WriteMessages),
		outbox.WithPollInterval(20*time.Millisecond))
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	go d.Run(ctx)

	require.Eventually(t, func() bool {
		pub.mu.Lock()
		defer pub.mu.Unlock()
		return len(pub.messages) == 3
	}, 400*time.Millisecond, 10*time.Millisecond)
	require.Equal(t, []byte{1}, pub.messages[0].Value)
	require.Equal(t, []byte{2}, pub.messages[1].Value)
	require.Equal(t, []byte{3}, pub.messages[2].Value)
}
```

Note: `Entity.Headers` has a gorm default of `'{}'`, so direct `db.Create` rows decode as empty maps — the id-order test needs no header handling.

- [ ] **Step 2: Run tests to verify they fail**

Run from `libs/atlas-outbox`: `go test -run 'TestDrainer_Reattaches|TestDrainer_EmptyHeaders|TestDrainer_PublishesInIdOrder' ./...`
Expected: `TestDrainer_ReattachesHeaders` FAILS (no headers published). Id-order test may pass or fail nondeterministically today (map/timestamp tie) — that's the point; it must pass deterministically after.

- [ ] **Step 3: Implement**

Create `libs/atlas-outbox/headers.go`:

```go
package outbox

import (
	"encoding/base64"
	"encoding/json"

	"github.com/segmentio/kafka-go"
	"gorm.io/datatypes"
)

// Header values are base64-encoded inside the stored jsonb. Tenant version
// headers are raw big-endian uint16 bytes (see atlas-kafka
// TenantHeaderDecorator): they always contain a NUL byte, which Postgres
// jsonb rejects, and may be invalid UTF-8 (e.g. version 185 = 0xB9), which
// encoding/json silently mangles to U+FFFD. Base64 keeps the round trip
// byte-exact. Keys are plain ASCII and stay unencoded.
func encodeHeaders(h map[string]string) (datatypes.JSON, error) {
	if len(h) == 0 {
		return datatypes.JSON([]byte("{}")), nil
	}
	enc := make(map[string]string, len(h))
	for k, v := range h {
		enc[k] = base64.StdEncoding.EncodeToString([]byte(v))
	}
	b, err := json.Marshal(enc)
	if err != nil {
		return nil, err
	}
	return datatypes.JSON(b), nil
}

func decodeHeaders(j datatypes.JSON) ([]kafka.Header, error) {
	if len(j) == 0 {
		return nil, nil
	}
	var enc map[string]string
	if err := json.Unmarshal(j, &enc); err != nil {
		return nil, err
	}
	if len(enc) == 0 {
		return nil, nil
	}
	hs := make([]kafka.Header, 0, len(enc))
	for k, v := range enc {
		b, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			return nil, err
		}
		hs = append(hs, kafka.Header{Key: k, Value: b})
	}
	return hs, nil
}
```

In `libs/atlas-outbox/outbox.go`, replace Enqueue's header-marshal block (lines 29-36):

```go
	headers, err := encodeHeaders(msg.Headers)
	if err != nil {
		return err
	}
```

(drop the now-unused `encoding/json` import from outbox.go if nothing else uses it).

In `libs/atlas-outbox/drainer.go` `publishBatch`: change both `Order("enqueued_at ASC")` occurrences (lines 220, 224) to `Order("id ASC")`, and replace the message-building loop (lines 232-239) with:

```go
		msgs := make([]kafka.Message, 0, len(rows))
		for _, r := range rows {
			hs, err := decodeHeaders(r.Headers)
			if err != nil {
				return err
			}
			msgs = append(msgs, kafka.Message{
				Topic:   r.Topic,
				Key:     r.MessageKey,
				Value:   r.MessageValue,
				Headers: hs,
			})
		}
```

Audit `libs/atlas-outbox/backfill.go` for the same ordering pattern: it iterates the caller's loader and enqueues row-by-row (no ORDER BY on outbox_entries) — no change required; record that in the commit message body.

- [ ] **Step 4: Run tests to verify they pass**

Run from `libs/atlas-outbox`: `go test -race ./...`
Expected: PASS (whole package — existing drainer/sweeper/notify tests must stay green; `{}` rows decode to nil headers so `TestDrainer_PublishesUnsentRows` is unaffected).

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-outbox
git commit -m "feat(outbox): re-attach headers on publish, order by id, base64 header storage"
```

### Task 2: Promote TopicWriterPool into the lib; swap atlas-configurations

**Files:**
- Create: `libs/atlas-outbox/publisher.go`
- Delete: `services/atlas-configurations/atlas.com/configurations/outbox/publisher.go`
- Modify: `services/atlas-configurations/atlas.com/configurations/main.go:59`

**Interfaces:**
- Produces: `outbox.TopicWriterPool` (implements `Publisher`), `outbox.NewTopicWriterPool() *TopicWriterPool`, `(*TopicWriterPool).Close()` — used by every service main from Task 6 on.

- [ ] **Step 1: Move the file**

Copy `services/atlas-configurations/atlas.com/configurations/outbox/publisher.go` verbatim to `libs/atlas-outbox/publisher.go`, then apply exactly two edits: the doc comment's first paragraph becomes:

```go
// TopicWriterPool implements Publisher by maintaining one long-lived
// kafka.Writer per topic, keyed by the message's Topic field (outbox rows
// store real topic names, not env-var tokens). Per-service topic counts
// stay small fleet-wide, so the pool stays small.
```

and the package clause stays `package outbox` (it already is). Delete the original file. Straight move — no alias, no re-export (project convention).

- [ ] **Step 2: Swap atlas-configurations**

In `services/atlas-configurations/atlas.com/configurations/main.go` change line 59 from `publisher := outbox.NewTopicWriterPool()` to `publisher := outboxlib.NewTopicWriterPool()` (the `outboxlib` import already exists at line 13). The local `"atlas-configurations/outbox"` import stays — the package still holds the envelope types (`envelopes.go`).

- [ ] **Step 3: Verify both modules**

Run: `cd libs/atlas-outbox && go test -race ./... && go vet ./...`
Then: `cd services/atlas-configurations/atlas.com/configurations && go test -race ./... && go vet ./... && go build ./...`
Expected: all clean. atlas-configurations has no other diff than main.go:59 and the deleted file.

- [ ] **Step 4: Commit**

```bash
git add libs/atlas-outbox/publisher.go services/atlas-configurations
git commit -m "refactor(outbox): promote TopicWriterPool to libs/atlas-outbox"
```

### Task 3: Lib dependencies + EnqueueBuffer bridge

**Files:**
- Modify: `libs/atlas-outbox/go.mod` (+`go.sum`)
- Create: `libs/atlas-outbox/bridge.go`
- Test: `libs/atlas-outbox/bridge_test.go`

**Interfaces:**
- Consumes: `topic.EnvProvider` (libs/atlas-kafka/topic), `producer.SpanHeaderDecorator` / `producer.TenantHeaderDecorator` / `producer.HeaderDecorator` (libs/atlas-kafka/producer), `Enqueue`.
- Produces: `EnqueueBuffer(l logrus.FieldLogger, ctx context.Context, tx *gorm.DB, contents map[string][]kafka.Message) error` and unexported `headerMap(ctx) (map[string]string, error)` — used by Task 4's EmitProvider and directly by any Buffer-less caller.

- [ ] **Step 1: Add module dependencies**

In `libs/atlas-outbox/go.mod` add to the main require block:

```
	github.com/Chronicle20/atlas/libs/atlas-kafka v0.0.0
	github.com/Chronicle20/atlas/libs/atlas-model v0.0.0
```

and at the bottom (replace directives are per-main-module, so transitive Chronicle20 deps of atlas-kafka need their own lines):

```
replace github.com/Chronicle20/atlas/libs/atlas-kafka => ../atlas-kafka

replace github.com/Chronicle20/atlas/libs/atlas-model => ../atlas-model

replace github.com/Chronicle20/atlas/libs/atlas-retry => ../atlas-retry

replace github.com/Chronicle20/atlas/libs/atlas-tenant => ../atlas-tenant
```

Run from `libs/atlas-outbox`: `go mod tidy`. No import cycle: atlas-kafka does not import atlas-outbox.

- [ ] **Step 2: Write the failing tests**

Create `libs/atlas-outbox/bridge_test.go`:

```go
package outbox_test

import (
	"context"
	"testing"

	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	kafkaproducer "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func bridgeDb(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, outbox.Migration(db))
	return db
}

func tenantCtx(t *testing.T) context.Context {
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tenant.WithContext(context.Background(), tm)
}

func TestEnqueueBuffer_ResolvesTokenAndPreservesBytes(t *testing.T) {
	db := bridgeDb(t)
	t.Setenv("EVENT_TOPIC_TEST", "real-topic-name")

	contents := map[string][]kafka.Message{
		"EVENT_TOPIC_TEST": {
			{Key: []byte("k1"), Value: []byte(`{"a":1}`)},
			{Key: []byte("k2"), Value: []byte(`{"b":2}`)},
		},
	}
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return outbox.EnqueueBuffer(logrus.New(), tenantCtx(t), tx, contents)
	}))

	var ents []outbox.Entity
	require.NoError(t, db.Order("id ASC").Find(&ents).Error)
	require.Len(t, ents, 2)
	require.Equal(t, "real-topic-name", ents[0].Topic)
	require.Equal(t, []byte("k1"), ents[0].MessageKey)
	require.Equal(t, []byte(`{"a":1}`), ents[0].MessageValue)
	require.Equal(t, []byte("k2"), ents[1].MessageKey)
}

func TestEnqueueBuffer_UnsetTokenFallsThroughToToken(t *testing.T) {
	db := bridgeDb(t)
	contents := map[string][]kafka.Message{
		"EVENT_TOPIC_UNSET": {{Key: []byte("k"), Value: []byte("v")}},
	}
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return outbox.EnqueueBuffer(logrus.New(), tenantCtx(t), tx, contents)
	}))
	var ent outbox.Entity
	require.NoError(t, db.First(&ent).Error)
	require.Equal(t, "EVENT_TOPIC_UNSET", ent.Topic)
}

// FR-2.2 acceptance: enqueued header set == the direct path's decorator fold.
func TestEnqueueBuffer_HeaderParityWithDirectPath(t *testing.T) {
	db := bridgeDb(t)
	ctx := tenantCtx(t)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return outbox.EnqueueBuffer(logrus.New(), ctx, tx,
			map[string][]kafka.Message{"T": {{Key: []byte("k"), Value: []byte("v")}}})
	}))

	// Direct-path reference: fold the same decorators the service-local
	// ProviderImpl passes to Produce.
	want := map[string][]byte{}
	for _, d := range []kafkaproducer.HeaderDecorator{
		kafkaproducer.SpanHeaderDecorator(ctx),
		kafkaproducer.TenantHeaderDecorator(ctx),
	} {
		hm, err := d()
		require.NoError(t, err)
		for k, v := range hm {
			want[k] = []byte(v)
		}
	}

	// Drain via a short Run window and compare the published header set.
	pub := &fakePublisher{}
	d := outbox.NewDrainer(logrus.New(), db, outbox.PublisherFunc(pub.WriteMessages),
		outbox.WithPollInterval(20*time.Millisecond))
	ctx2, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	go d.Run(ctx2)
	require.Eventually(t, func() bool {
		pub.mu.Lock()
		defer pub.mu.Unlock()
		return len(pub.messages) == 1
	}, 400*time.Millisecond, 10*time.Millisecond)

	got := map[string][]byte{}
	for _, h := range pub.messages[0].Headers {
		got[h.Key] = h.Value
	}
	require.Equal(t, want, got)
}

func TestEnqueueBuffer_RowFailureReturnsError(t *testing.T) {
	db := bridgeDb(t)
	contents := map[string][]kafka.Message{
		"T": {{Key: nil, Value: []byte("v")}}, // empty key -> Enqueue rejects
	}
	err := db.Transaction(func(tx *gorm.DB) error {
		return outbox.EnqueueBuffer(logrus.New(), tenantCtx(t), tx, contents)
	})
	require.Error(t, err)
	var count int64
	require.NoError(t, db.Model(&outbox.Entity{}).Count(&count).Error)
	require.Zero(t, count)
}
```

(add `"time"` to the imports for the parity test).

- [ ] **Step 3: Run tests to verify they fail**

Run from `libs/atlas-outbox`: `go test -run TestEnqueueBuffer ./...`
Expected: compile FAILURE — `outbox.EnqueueBuffer` undefined.

- [ ] **Step 4: Implement**

Create `libs/atlas-outbox/bridge.go`:

```go
package outbox

import (
	"context"

	kafkaproducer "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// EnqueueBuffer persists a message.Buffer-shaped payload (env-var token →
// messages) as outbox rows inside tx. Tokens are resolved to real topic
// names via topic.EnvProvider; span + tenant headers are derived from ctx
// exactly as the direct producer path derives them at emit time. Message
// key and value bytes pass through unchanged. Any failure returns an
// error, failing the enclosing transaction.
func EnqueueBuffer(l logrus.FieldLogger, ctx context.Context, tx *gorm.DB, contents map[string][]kafka.Message) error {
	headers, err := headerMap(ctx)
	if err != nil {
		return err
	}
	for token, msgs := range contents {
		t, err := topic.EnvProvider(l)(token)()
		if err != nil {
			return err
		}
		for _, m := range msgs {
			if err := Enqueue(tx, Message{Topic: t, Key: m.Key, Value: m.Value, Headers: headers}); err != nil {
				return err
			}
		}
	}
	return nil
}

// headerMap merges the span and tenant decorators into one map — the same
// key set the direct path's produceHeaders folds (span and tenant key sets
// are disjoint, so map-merge is equivalent to the append-fold).
func headerMap(ctx context.Context) (map[string]string, error) {
	headers := make(map[string]string)
	decorators := []kafkaproducer.HeaderDecorator{
		kafkaproducer.SpanHeaderDecorator(ctx),
		kafkaproducer.TenantHeaderDecorator(ctx),
	}
	for _, d := range decorators {
		hm, err := d()
		if err != nil {
			return nil, err
		}
		for k, v := range hm {
			headers[k] = v
		}
	}
	return headers, nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run from `libs/atlas-outbox`: `go test -race ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-outbox
git commit -m "feat(outbox): EnqueueBuffer bridge with token resolution and header parity"
```

### Task 4: EmitProvider — the outbox-backed producer.Provider

**Files:**
- Create: `libs/atlas-outbox/provider.go`
- Test: `libs/atlas-outbox/provider_test.go`

**Interfaces:**
- Consumes: `EnqueueBuffer` (Task 3), `kafkaproducer.MessageProducer` (`func(model.Provider[[]kafka.Message]) error`), `model.Provider`/`model.FixedProvider` (libs/atlas-model).
- Produces: `EmitProvider(l logrus.FieldLogger, ctx context.Context, tx *gorm.DB) func(token string) kafkaproducer.MessageProducer` — assignable to every service-local `producer.Provider`; used by Tasks 7–23.

- [ ] **Step 1: Write the failing test**

Create `libs/atlas-outbox/provider_test.go`:

```go
package outbox_test

import (
	"errors"
	"testing"

	kafkaproducer "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// localProvider mirrors every service's kafka/producer.Provider named type;
// assignment here is the compile-time proof EmitProvider satisfies it.
type localProvider func(token string) kafkaproducer.MessageProducer

func TestEmitProvider_EnqueuesThroughEmitShapedLoop(t *testing.T) {
	db := bridgeDb(t)
	ctx := tenantCtx(t)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var p localProvider = outbox.EmitProvider(logrus.New(), ctx, tx)
		// The message.Emit flush loop shape: p(token)(FixedProvider(msgs)).
		return p("EVENT_TOPIC_X")(model.FixedProvider([]kafka.Message{
			{Key: []byte("k"), Value: []byte("v")},
		}))
	}))

	var ents []outbox.Entity
	require.NoError(t, db.Find(&ents).Error)
	require.Len(t, ents, 1)
	require.Equal(t, "EVENT_TOPIC_X", ents[0].Topic)
	require.Equal(t, []byte("k"), ents[0].MessageKey)
}

func TestEmitProvider_ProviderErrorPropagates(t *testing.T) {
	db := bridgeDb(t)
	boom := errors.New("boom")
	err := db.Transaction(func(tx *gorm.DB) error {
		p := outbox.EmitProvider(logrus.New(), tenantCtx(t), tx)
		return p("T")(model.ErrorProvider[[]kafka.Message](boom))
	})
	require.ErrorIs(t, err, boom)
}

func TestEmitProvider_EnqueueErrorFailsTransaction(t *testing.T) {
	db := bridgeDb(t)
	err := db.Transaction(func(tx *gorm.DB) error {
		p := outbox.EmitProvider(logrus.New(), tenantCtx(t), tx)
		return p("T")(model.FixedProvider([]kafka.Message{{Key: nil, Value: []byte("v")}}))
	})
	require.Error(t, err)
	var count int64
	require.NoError(t, db.Model(&outbox.Entity{}).Count(&count).Error)
	require.Zero(t, count)
}
```

(if `model.ErrorProvider` does not exist in libs/atlas-model, use an inline `func() ([]kafka.Message, error) { return nil, boom }` — check `libs/atlas-model/model` first and use whichever exists.)

- [ ] **Step 2: Run test to verify it fails**

Run from `libs/atlas-outbox`: `go test -run TestEmitProvider ./...`
Expected: compile FAILURE — `outbox.EmitProvider` undefined.

- [ ] **Step 3: Implement**

Create `libs/atlas-outbox/provider.go`:

```go
package outbox

import (
	"context"

	kafkaproducer "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// EmitProvider returns a producer.Provider-shaped value whose
// MessageProducer persists messages as outbox rows inside tx instead of
// writing to Kafka. The return type is the unnamed func type underlying
// every service-local kafka/producer.Provider, so existing message.Emit /
// EmitWithResult call sites accept it without conversion. Topic tokens are
// env-resolved and span+tenant headers applied from ctx at enqueue time;
// the drainer publishes after the transaction commits.
func EmitProvider(l logrus.FieldLogger, ctx context.Context, tx *gorm.DB) func(token string) kafkaproducer.MessageProducer {
	return func(token string) kafkaproducer.MessageProducer {
		return func(provider model.Provider[[]kafka.Message]) error {
			msgs, err := provider()
			if err != nil {
				return err
			}
			return EnqueueBuffer(l, ctx, tx, map[string][]kafka.Message{token: msgs})
		}
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run from `libs/atlas-outbox`: `go test -race ./... && go vet ./...`
Expected: PASS, vet clean.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-outbox
git commit -m "feat(outbox): EmitProvider outbox-backed producer.Provider"
```

### Task 5: Lib README (FR-4) and lib-phase verification

**Files:**
- Modify: `libs/atlas-outbox/README.md`

- [ ] **Step 1: Update the README**

Add/refresh these sections (keep existing content that still holds):

1. **Delivery semantics**: at-least-once — a crash between `WriteMessages` and the `sent_at` stamp redelivers on the next tick; consumer dedup on `TransactionId` is the CD-1 follow-up task, not this lib.
2. **Ordering guarantee**: the single drainer leader publishes in `id ASC` order, preserving each service's per-transaction emission order. Caveat: across concurrent transactions, id order is allocation order, not commit order — same as the previous `enqueued_at` behavior; the guarantee is per-flow, not cross-flow. Within one flushed buffer, cross-topic order follows map iteration (as on the direct path).
3. **Headers**: persisted with base64-encoded values (binary tenant-version headers contain NUL / non-UTF-8 bytes; jsonb rejects NUL) and re-attached byte-exact at publish. The published header set equals the direct producer path's span+tenant decoration.
4. **Adoption API**: `EmitProvider(l, ctx, tx)` for `message.Emit`/`EmitWithResult` call sites; `EnqueueBuffer(l, ctx, tx, contents)` for Buffer-less callers; `NewTopicWriterPool()` as the standard drainer Publisher; the wiring template (migration + drainer + teardown, as in atlas-configurations main.go).
5. **Operations**: growth of `sent_at IS NULL` rows is the signal for a wedged drainer; publish failures log with `attempts`/`last_error`.
6. Keep the existing note that the table is intentionally not tenant-scoped; tenancy rides in the headers.

- [ ] **Step 2: Full lib verification**

Run from `libs/atlas-outbox`: `go test -race ./... && go vet ./... && go build ./...`
Expected: all clean.

- [ ] **Step 3: Commit**

```bash
git add libs/atlas-outbox/README.md
git commit -m "docs(outbox): delivery semantics, ordering, headers, adoption API"
```

---

## Phase 2 — atlas-character (reference implementation)

Module dir: `services/atlas-character/atlas.com/character`. All processor work is in `character/processor.go` (25 `ExecuteTransaction` sites, 23 `message.Emit` sites, 12 in-tx direct emits at lines 742, 750, 751, 768, 785, 812, 813, 826, 830, 892, 901, 905).

### Task 6: Wire outbox into atlas-character

**Files:**
- Modify: `services/atlas-character/atlas.com/character/go.mod`, `services/atlas-character/atlas.com/character/main.go:68`

**Interfaces:**
- Produces: outbox table migrated + drainer running in atlas-character; `outbox` importable from service code (Tasks 7–10).

- [ ] **Step 1: go.mod**

Apply the go.mod part of the wiring template (Global Constraints), then run `go mod tidy` from the module dir.

- [ ] **Step 2: main.go**

Line 68 becomes:

```go
	db := database.Connect(l, database.SetMigrations(character.Migration, history.Migration, saved_location.Migration, outboxlib.Migration))
```

Add the import `outboxlib "github.com/Chronicle20/atlas/libs/atlas-outbox"` and insert the drainer block from the wiring template immediately after the `database.Connect` line.

- [ ] **Step 3: Verify**

Run from the module dir: `go build ./... && go vet ./... && go test -race ./...`
Expected: all clean (behavioral no-op so far).

- [ ] **Step 4: Commit**

```bash
git add services/atlas-character
git commit -m "feat(character): wire outbox migration and drainer"
```

### Task 7: FR-1 — the three meso paths

`RequestChangeMeso` (:733), `AttemptMesoPickUp` (:755), `RequestDropMeso` (:776) in `character/processor.go`. Defects fixed here: unchecked `err = dynamicUpdate(...)` (two sites), nil-`err` overflow returns (two sites), in-tx direct emits (all three), fire-and-forget post-tx STAT_CHANGED (drop path).

**Files:**
- Modify: `services/atlas-character/atlas.com/character/character/processor.go:733-800`
- Test: `services/atlas-character/atlas.com/character/character/meso_outbox_test.go` (new)

**Interfaces:**
- Consumes: `outbox.EmitProvider` (Task 4), existing `message.Emit`, `dynamicUpdate`, `SetMeso`, event providers (`mesoChangedStatusEventProvider`, `statChangedProvider`, `notEnoughMesoErrorStatusEventProvider`).
- Produces: exported sentinels `character.ErrNotEnoughMeso`, `character.ErrMesoOverflow` (used by tests and future callers).

- [ ] **Step 1: Write the failing tests**

Create `character/meso_outbox_test.go` (reuse `testDatabase`/`testTenant`/`testLogger` from `processor_test.go`; note `testDatabase` must also migrate the outbox table — extend it or add a local helper):

```go
package character_test

import (
	"atlas-character/character"
	"atlas-character/kafka/message"
	"context"
	"testing"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func outboxTestDb(t *testing.T) *gorm.DB {
	db := testDatabase(t)
	require.NoError(t, outbox.Migration(db))
	return db
}

func createTestCharacter(t *testing.T, ctx context.Context, db *gorm.DB, meso uint32) character.Model {
	input := character.NewModelBuilder().SetAccountId(1000).SetWorldId(0).SetName("Atlas").SetLevel(1).SetExperience(0).Build()
	c, err := character.NewProcessor(testLogger(), ctx, db).Create(message.NewBuffer())(uuid.New(), input, _map.Id(0))
	require.NoError(t, err)
	if meso > 0 {
		require.NoError(t, character.NewProcessor(testLogger(), ctx, db).RequestChangeMeso(uuid.New(), c.Id(), int32(meso), 0, "SYSTEM", false))
	}
	return c
}

func outboxRowCount(t *testing.T, db *gorm.DB) int64 {
	var n int64
	require.NoError(t, db.Model(&outbox.Entity{}).Count(&n).Error)
	return n
}

func TestRequestChangeMeso_CommitEnqueuesExactlyTwoRows(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := outboxTestDb(t)
	c := createTestCharacter(t, tctx, db, 0)

	before := outboxRowCount(t, db)
	require.NoError(t, character.NewProcessor(testLogger(), tctx, db).RequestChangeMeso(uuid.New(), c.Id(), 500, 0, "SYSTEM", false))
	require.Equal(t, before+2, outboxRowCount(t, db)) // MESO_CHANGED + STAT_CHANGED

	got, err := character.NewProcessor(testLogger(), tctx, db).GetById()(c.Id())
	require.NoError(t, err)
	require.Equal(t, uint32(500), got.Meso())
}

func TestRequestChangeMeso_NotEnoughMesoEmitsNoOutboxRowsAndReturnsNil(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := outboxTestDb(t)
	c := createTestCharacter(t, tctx, db, 0)

	before := outboxRowCount(t, db)
	require.NoError(t, character.NewProcessor(testLogger(), tctx, db).RequestChangeMeso(uuid.New(), c.Id(), -100, 0, "SYSTEM", false))
	require.Equal(t, before, outboxRowCount(t, db))
}

func TestRequestChangeMeso_OverflowReturnsError(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := outboxTestDb(t)
	c := createTestCharacter(t, tctx, db, 10)

	before := outboxRowCount(t, db)
	err := character.NewProcessor(testLogger(), tctx, db).RequestChangeMeso(uuid.New(), c.Id(), 2147483647, 0, "SYSTEM", false)
	err2 := character.NewProcessor(testLogger(), tctx, db).RequestChangeMeso(uuid.New(), c.Id(), 2147483647, 0, "SYSTEM", false)
	// two max-int32 adds from a 10+2147483647 base guarantee crossing MaxUint32
	require.True(t, err != nil || err2 != nil)
	require.ErrorIs(t, func() error {
		if err != nil {
			return err
		}
		return err2
	}(), character.ErrMesoOverflow)
	_ = before
}
```

Notes for the implementer: `createTestCharacter` uses `message.NewBuffer()` — import the service `message` package and replace the `messageBuffer()` placeholder-name with `message.NewBuffer()` directly (shown this way here only to keep the import list obvious). Check `character.Model` exposes `Id()` and `Meso()` (it does — used throughout processor.go). If `NewModelBuilder` requires more fields for a valid insert, mirror the setup in `TestCreateSunny` (processor_test.go:52-56). The overflow test may be simplified to a single call if a builder `SetMeso` exists — inspect `NewModelBuilder` and prefer direct meso seeding over the two-call dance.

- [ ] **Step 2: Run tests to verify they fail**

Run from the module dir: `go test -run 'TestRequestChangeMeso' ./character/...`
Expected: compile FAILURE (`character.ErrMesoOverflow` undefined) — and once compiled, row-count failures because events currently go to Kafka, not the outbox.

- [ ] **Step 3: Implement**

At the top of `character/processor.go` (near existing vars/consts) add:

```go
// ErrNotEnoughMeso signals a rejected meso change: no state was written and
// the rejection status event is emitted outside the transaction.
var ErrNotEnoughMeso = errors.New("not enough meso")

// ErrMesoOverflow rejects a change that would overflow the uint32 meso field.
var ErrMesoOverflow = errors.New("meso overflow")
```

Replace the three methods:

```go
func (p *ProcessorImpl) RequestChangeMeso(transactionId uuid.UUID, characterId uint32, amount int32, actorId uint32, actorType string, showEffect bool) error {
	var rejectEmit func() error
	txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		c, err := p.WithTransaction(tx).GetById()(characterId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to retrieve character [%d] who is having their meso adjusted.", characterId)
			return err
		}
		if int64(c.Meso())+int64(amount) < 0 {
			p.l.Debugf("Request for character [%d] would leave their meso negative. Amount [%d]. Existing [%d].", characterId, amount, c.Meso())
			rejectEmit = func() error {
				return producer.ProviderImpl(p.l)(p.ctx)(character2.EnvEventTopicCharacterStatus)(notEnoughMesoErrorStatusEventProvider(transactionId, characterId, c.WorldId(), amount))
			}
			return ErrNotEnoughMeso
		}
		if amount > 0 && uint32(amount) > (math.MaxUint32-c.Meso()) {
			p.l.Errorf("Transaction for character [%d] would result in a uint32 overflow. Rejecting transaction.", characterId)
			return ErrMesoOverflow
		}

		if err = dynamicUpdate(tx)(SetMeso(uint32(int64(c.Meso()) + int64(amount))))(c); err != nil {
			return err
		}
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			if err := buf.Put(character2.EnvEventTopicCharacterStatus, mesoChangedStatusEventProvider(transactionId, characterId, c.WorldId(), amount, actorId, actorType, showEffect)); err != nil {
				return err
			}
			return buf.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel.NewModel(c.WorldId(), 0), characterId, []stat.Type{stat.TypeMeso}, nil))
		})
	})
	if errors.Is(txErr, ErrNotEnoughMeso) && rejectEmit != nil {
		_ = rejectEmit()
		return nil
	}
	return txErr
}

func (p *ProcessorImpl) AttemptMesoPickUp(transactionId uuid.UUID, field field.Model, characterId uint32, dropId uint32, meso uint32) error {
	txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		c, err := p.WithTransaction(tx).GetById()(characterId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to retrieve character [%d] who is having their meso adjusted.", characterId)
			return err
		}
		if meso > (math.MaxUint32 - c.Meso()) {
			p.l.Errorf("Transaction for character [%d] would result in a uint32 overflow. Rejecting transaction.", characterId)
			return ErrMesoOverflow
		}

		if err = dynamicUpdate(tx)(SetMeso(uint32(int64(c.Meso()) + int64(meso))))(c); err != nil {
			return err
		}
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return buf.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel.NewModel(field.WorldId(), field.ChannelId()), characterId, []stat.Type{stat.TypeMeso}, nil))
		})
	})
	if txErr != nil {
		return txErr
	}
	return drop.NewProcessor(p.l, p.ctx).RequestPickUp(field, dropId, characterId)
}

func (p *ProcessorImpl) RequestDropMeso(transactionId uuid.UUID, field field.Model, characterId uint32, amount uint32) error {
	var rejectEmit func() error
	txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		c, err := p.WithTransaction(tx).GetById()(characterId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to retrieve character [%d] who is having their meso adjusted.", characterId)
			return err
		}
		if int64(c.Meso())-int64(amount) < 0 {
			p.l.Debugf("Request for character [%d] would leave their meso negative. Amount [%d]. Existing [%d].", characterId, amount, c.Meso())
			rejectEmit = func() error {
				return producer.ProviderImpl(p.l)(p.ctx)(character2.EnvEventTopicCharacterStatus)(notEnoughMesoErrorStatusEventProvider(transactionId, characterId, c.WorldId(), int32(amount)))
			}
			return ErrNotEnoughMeso
		}

		if err = dynamicUpdate(tx)(SetMeso(c.Meso() - amount))(c); err != nil {
			return err
		}
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return buf.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel.NewModel(field.WorldId(), field.ChannelId()), characterId, []stat.Type{stat.TypeMeso}, nil))
		})
	})
	if errors.Is(txErr, ErrNotEnoughMeso) && rejectEmit != nil {
		_ = rejectEmit()
		return nil
	}
	if txErr != nil {
		return txErr
	}

	tc := GetTemporalRegistry().GetById(p.ctx, tenant.MustFromContext(p.ctx), characterId)
	// TODO determine appropriate drop type and mod
	_ = drop.NewProcessor(p.l, p.ctx).CreateForMesos(field, amount, 2, tc.X(), tc.Y(), characterId)
	return nil
}
```

Add the `outbox` import to processor.go per the recipe. Behavioral notes: the drop path's success `STAT_CHANGED` moves from post-tx fire-and-forget to in-tx enqueue (it asserts a state change — that's the migration); the rejection events keep their current "return nil to caller" behavior but now emit after the transaction closes (FR-1.3); the pre-existing `// TODO` comment line is untouched.

- [ ] **Step 4: Run tests to verify they pass**

Run from the module dir: `go test -race ./character/...`
Expected: PASS, including all pre-existing tests.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-character
git commit -m "fix(character): meso paths — check update errors, real overflow errors, outbox enqueue in-tx"
```

### Task 8: RequestChangeFame + RequestDistributeAp in-tx emits

`RequestChangeFame` (:802-815): unchecked `err = dynamicUpdate`, two in-tx success emits. `RequestDistributeAp` (:822-910): four in-tx rejection/error emits (GetById failure, not-enough-AP, invalid-ability, update-failure) and one in-tx success emit.

**Files:**
- Modify: `services/atlas-character/atlas.com/character/character/processor.go:802-910`

**Interfaces:**
- Consumes: `outbox.EmitProvider`, `message.Emit`, existing providers (`fameChangedStatusEventProvider`, `statChangedProvider`).

- [ ] **Step 1: RequestChangeFame**

```go
func (p *ProcessorImpl) RequestChangeFame(transactionId uuid.UUID, characterId uint32, amount int8, actorId uint32, actorType string) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		c, err := p.WithTransaction(tx).GetById()(characterId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to retrieve character [%d] who is having their fame adjusted.", characterId)
			return err
		}

		total := c.Fame() + int16(amount)
		if err = dynamicUpdate(tx)(SetFame(total))(c); err != nil {
			return err
		}
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			if err := buf.Put(character2.EnvEventTopicCharacterStatus, fameChangedStatusEventProvider(transactionId, characterId, c.WorldId(), amount, actorId, actorType)); err != nil {
				return err
			}
			return buf.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel.NewModel(c.WorldId(), 0), characterId, []stat.Type{stat.TypeFame}, nil))
		})
	})
}
```

- [ ] **Step 2: RequestDistributeAp**

Restructure with the reject-closure device from Task 7. The four failure-path emits (statChanged with empty/AP-only stat list — UI re-enable signals reflecting no state change) become one captured `rejectEmit` set at each failure branch and fired after the transaction returns non-nil; the success emit flushes through the outbox. The switch body (ability accumulation) is unchanged. Skeleton of the edits — failure branches change from

```go
_ = producer.ProviderImpl(p.l)(p.ctx)(character2.EnvEventTopicCharacterStatus)(statChangedProvider(transactionId, channel.NewModel(c.WorldId(), 0), characterId, []stat.Type{}, nil))
return errors.New("not enough ap")
```

to

```go
rejectEmit = func() error {
	return producer.ProviderImpl(p.l)(p.ctx)(character2.EnvEventTopicCharacterStatus)(statChangedProvider(transactionId, channel.NewModel(c.WorldId(), 0), characterId, []stat.Type{}, nil))
}
return errors.New("not enough ap")
```

(the GetById-failure branch cannot reference `c`; capture the emit with `channel.NewModel(0, 0)`? No — **it can't know the world**. Keep that one branch's emit exactly as today's arguments require: today it dereferences `c` after `err != nil`, which is itself a latent nil-model bug. Since `GetById` failing means no character, drop that branch's emit entirely and just `return err`; record this in inventory.md as a deliberate removal of an emit that read from a zero-value model). The update-failure branch captures `[]stat.Type{stat.TypeAvailableAP}` as today. The function tail becomes:

```go
	var rejectEmit func() error
	txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		// ... branches as above; success path ends with:
		if err = dynamicUpdate(tx)(eufs...)(c); err != nil {
			rejectEmit = func() error {
				return producer.ProviderImpl(p.l)(p.ctx)(character2.EnvEventTopicCharacterStatus)(statChangedProvider(transactionId, channel.NewModel(c.WorldId(), 0), characterId, []stat.Type{stat.TypeAvailableAP}, nil))
			}
			return err
		}
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return buf.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel.NewModel(c.WorldId(), 0), characterId, stats, values))
		})
	})
	if txErr != nil && rejectEmit != nil {
		_ = rejectEmit()
	}
	return txErr
```

- [ ] **Step 3: Verify**

Run from the module dir: `go test -race ./... && go vet ./...`
Expected: all clean.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-character
git commit -m "fix(character): fame and AP distribution — outbox enqueue in-tx, rejection emits post-tx"
```

### Task 9: Invert every remaining Emit site in atlas-character

**Files:**
- Modify: `services/atlas-character/atlas.com/character/character/processor.go` (the `*AndEmit` family and any remaining tx-coupled emits)

**Interfaces:**
- Consumes: recipe Patterns A/B, `outbox.EmitProvider`.

- [ ] **Step 1: Enumerate**

Run from the module dir:

```bash
grep -rn "message.Emit(\|message.EmitWithResult" --include='*.go' . | grep -v _test.go
```

Expected: 23 sites in `character/processor.go` (CreateAndEmit, DeleteAndEmit, DeleteForSagaCompensationAndEmit, DeleteByAccountIdAndEmit, LoginAndEmit, LogoutAndEmit, ChangeJobAndEmit, ChangeHair/Face/SkinAndEmit, AwardExperienceAndEmit, DeductExperienceAndEmit, AwardLevelAndEmit, ChangeHP/MP/SetHP/ClampHP/ClampMPAndEmit, ProcessLevelChangeAndEmit, ProcessJobChangeAndEmit, UpdateAndEmit, ResetStatsAndEmit, RebalanceAPAndEmit, plus RequestDistributeSp's flow). Record the exact list.

- [ ] **Step 2: Classify and transform**

For each site: if the wrapped method performs DB writes (directly or via its own `ExecuteTransaction`), apply Pattern A (or B where a result is returned). The Task 7/8 sites are already done. Sites whose wrapped method performs **no** DB write (e.g. pure registry/temporal operations, login/logout if they only touch session history — verify by reading the wrapped method, don't assume) stay direct and go on the inventory "left direct" list with the reason "no DB mutation in flow". `LoginAndEmit`/`LogoutAndEmit` write session history rows — those migrate.

- [ ] **Step 3: Sweep the rest of the module**

```bash
grep -rn "message.Emit(\|message.EmitWithResult\|producer.ProviderImpl" --include='*.go' . | grep -v _test.go | grep -v "kafka/"
```

Classify every remaining hit (session history, saved_location, drop, consumers) by the same rule. Consumers that emit **commands** to other services stay direct.

- [ ] **Step 4: Verify**

Run from the module dir: `go test -race ./... && go vet ./... && go build ./...`
Expected: all clean. The existing `kafka_integration_test.go` and `producer_test.go` may assert direct-path emission for migrated flows — if they fail, update them to assert outbox rows instead (that is the new contract), keeping non-migrated flow assertions on `producertest`.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-character
git commit -m "feat(character): migrate all tx-coupled emit sites to outbox"
```

### Task 10: atlas-character acceptance tests + inventory start

**Files:**
- Test: `services/atlas-character/atlas.com/character/character/outbox_acceptance_test.go` (new)
- Create: `docs/tasks/task-114-outbox-adoption/inventory.md`

- [ ] **Step 1: Write the acceptance tests**

PRD acceptance #3: rollback → zero events; commit → exactly the enqueued events. Create `outbox_acceptance_test.go` in package `character_test`:

```go
package character_test

import (
	"atlas-character/kafka/message"
	"context"
	"errors"
	"testing"

	character2 "atlas-character/kafka/message/character"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// Rollback in a migrated flow leaves zero outbox rows.
func TestOutbox_RollbackDiscardsEnqueuedEvents(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := outboxTestDb(t)

	boom := errors.New("boom")
	err := database.ExecuteTransaction(db, func(tx *gorm.DB) error {
		err := message.Emit(outbox.EmitProvider(testLogger(), tctx, tx))(func(buf *message.Buffer) error {
			return buf.Put(character2.EnvEventTopicCharacterStatus, fixedMessageProvider())
		})
		require.NoError(t, err) // enqueue itself succeeded...
		return boom             // ...then the domain flow fails
	})
	require.ErrorIs(t, err, boom)
	require.Zero(t, outboxRowCount(t, db))
}

// Commit publishes exactly what was enqueued, via the drainer.
func TestOutbox_CommitYieldsExactlyEnqueuedEvents(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := outboxTestDb(t)

	require.NoError(t, database.ExecuteTransaction(db, func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(testLogger(), tctx, tx))(func(buf *message.Buffer) error {
			return buf.Put(character2.EnvEventTopicCharacterStatus, fixedMessageProvider())
		})
	}))
	require.Equal(t, int64(1), outboxRowCount(t, db))
}

func fixedMessageProvider() model.Provider[[]kafka.Message] {
	return model.FixedProvider([]kafka.Message{{Key: []byte("k"), Value: []byte(`{"e":1}`)}})
}
```

(`outboxTestDb`/`outboxRowCount` come from Task 7's `meso_outbox_test.go`; the package `TestMain` already installs the noop Kafka producer.)

- [ ] **Step 2: Run tests**

Run from the module dir: `go test -race ./character/...`
Expected: PASS.

- [ ] **Step 3: Start the inventory doc**

Create `docs/tasks/task-114-outbox-adoption/inventory.md`:

```markdown
# task-114 Outbox Migration Inventory

Per-service audit record (FR-3.5). Sections are appended as each service
migrates. "Left direct" sites keep the direct producer path deliberately.

## atlas-character
```

Fill the atlas-character section from Tasks 7–9's actual work: every migrated site (method + file:line at time of change), every left-direct site with reason (rejection emits post-tx, command emits to drops service, no-DB flows, the removed zero-value-model emit from Task 8), following the inventory entry format in Global Constraints.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-character docs/tasks/task-114-outbox-adoption/inventory.md
git commit -m "test(character): outbox rollback/commit acceptance; start migration inventory"
```

---

## Phase 3 — Economy tier

Each of Tasks 11–13 follows the same five steps; the recipe (Global Constraints) is part of the task. Steps per task:

1. **Wire**: go.mod + main.go per the wiring template (exact before-lines given per task).
2. **Enumerate**: from the module dir, `grep -rn "message.Emit(\|message.EmitWithResult\|producer.ProviderImpl\|database.ExecuteTransaction" --include='*.go' . | grep -v _test.go` — reconcile against the site list given in the task; investigate any extras.
3. **Transform**: apply Patterns A/B/C to every tx-coupled site; leave listed non-tx sites direct.
4. **Verify**: `go test -race ./... && go vet ./... && go build ./...` from the module dir, all clean; fix any existing tests that asserted direct emission for migrated flows by asserting outbox rows instead.
5. **Inventory + commit** with the given message.

### Task 11: atlas-inventory

**Files:** module `services/atlas-inventory/atlas.com/inventory`; modify `go.mod`, `main.go:62` (`db := database.Connect(l, database.SetMigrations(compartment.Migration, asset.Migration))` → append `, outboxlib.Migration` + drainer block), `inventory/compartment/processor.go` (20 Emit sites, 21 tx sites), `inventory/inventory/processor.go` (3 Emit, 2 tx), `inventory/asset/processor.go` (2 Emit, 3 tx); append to `docs/tasks/task-114-outbox-adoption/inventory.md`.

**Notes:** `compartment/processor.go:108` holds a `producer.ProviderImpl` struct-field initialization — a stored `Provider` used by methods. Trace which methods flush through that field: those that couple to DB writes must take the outbox provider path (restructure the method to Pattern A; do not store a tx-bound provider on a long-lived struct — providers bind to a tx, structs outlive it). No in-tx direct emits exist in this service.

- [ ] Steps 1–5 as above. Commit: `git commit -m "feat(inventory): migrate tx-coupled emit sites to outbox"`

### Task 12: atlas-cashshop

**Files:** module `services/atlas-cashshop/atlas.com/cashshop`; modify `go.mod`, `main.go:62` (`...SetMigrations(wallet.Migration, wishlist.Migration, compartment.Migration, asset.Migration)` → append), Emit sites across `cashshop/wallet/processor.go` (3 EmitWithResult), `cashshop/wishlist/processor.go` (1 EmitWithResult), `cashshop/cashshop/processor.go`, `cashshop/cashshop/inventory/asset/processor.go`, `cashshop/cashshop/inventory/compartment/processor.go` (13 Emit total); inventory append.

**Notes:** `cashshop/processor.go:234,239` are direct error/status emits in a helper outside any tx — classify per D7 (rejection/no-state-change → left direct; state-asserting → migrate). Consumer emits at `kafka/consumer/cashshop/consumer.go:93,102,111` are command/relay emits — left direct. EmitWithResult sites use Pattern B.

- [ ] Steps 1–5. Commit: `git commit -m "feat(cashshop): migrate tx-coupled emit sites to outbox"`

### Task 13: atlas-fame

**Files:** module `services/atlas-fame/atlas.com/fame`; modify `go.mod`, `main.go:34` (`...SetMigrations(fame.Migration)` → append), `fame/fame/processor.go` (1 Emit, 2 tx sites; note `fame/processor.go:120` captures the provider into a local var used outside the tx — inversion moves the whole Emit inside), `fame/character/processor.go` (1 Emit at :70 — check whether its flow writes to this service's DB; if it only relays to atlas-character via command, it stays direct); inventory append.

- [ ] Steps 1–5. Commit: `git commit -m "feat(fame): migrate tx-coupled emit sites to outbox"`

---

## Phase 4 — Standard tier

Same five steps as Phase 3 per task.

### Task 14: atlas-buddies

**Files:** module `services/atlas-buddies/atlas.com/buddies`; `go.mod`; `main.go:57` (`...SetMigrations(list.Migration, buddy.Migration)` → append); `list/processor.go` (8 Emit, 8 tx; `list/processor.go:74` struct-init provider — same treatment as Task 11's note; `list/resource.go:69` REST-handler provider — classify); `invite/processor.go:32,37` are command emits — left direct; inventory append.

- [ ] Steps 1–5. Commit: `git commit -m "feat(buddies): migrate tx-coupled emit sites to outbox"`

### Task 15: atlas-guilds

**Files:** module `services/atlas-guilds/atlas.com/guilds`; `go.mod`; `main.go:68` (`...SetMigrations(guild.Migration, title.Migration, member.Migration, character.Migration, thread.Migration, reply.Migration)` → append); `guild/processor.go` (13 Emit, 4 tx), `thread/processor.go` (5 Emit, 4 tx), plus tx sites in `guild/title`, `guild/member`, `guild/character` processors (check whether their mutations are emitted from guild/processor.go flows); `invite/processor.go:29` command emit — left direct; inventory append.

- [ ] Steps 1–5. Commit: `git commit -m "feat(guilds): migrate tx-coupled emit sites to outbox"`

### Task 16: atlas-notes

**Files:** module `services/atlas-notes/atlas.com/notes`; `go.mod`; `main.go:55` (`...SetMigrations(note.Migration)` → append); `note/processor.go` (3 Emit + 2 EmitWithResult; the 4 tx sites live in `note/administrator.go` — Pattern A wraps at the processor, administrator re-enters); `saga/processor.go:28` command emit — left direct; inventory append.

- [ ] Steps 1–5. Commit: `git commit -m "feat(notes): migrate tx-coupled emit sites to outbox"`

### Task 17: atlas-pets

**Files:** module `services/atlas-pets/atlas.com/pets`; `go.mod`; `main.go:64` (`...SetMigrations(pet.Migration, exclude.Migration)` → append); `pet/processor.go` (11 Emit + 1 EmitWithResult, 13 tx; struct-init provider at :108 — Task 11 note applies); inventory append.

- [ ] Steps 1–5. Commit: `git commit -m "feat(pets): migrate tx-coupled emit sites to outbox"`

### Task 18: atlas-skills

**Files:** module `services/atlas-skills/atlas.com/skills`; `go.mod`; `main.go:61` (`...SetMigrations(skill.Migration, macro.Migration)` → append); `skill/processor.go` (4 Emit, 3 tx), `macro/processor.go` (1 Emit, 2 tx); left direct: `skill/processor.go:229` (`ExpireCooldowns` — registry-only background task, no DB write) and the command emits at :260/:265; inventory append.

- [ ] Steps 1–5. Commit: `git commit -m "feat(skills): migrate tx-coupled emit sites to outbox"`

### Task 19: atlas-merchant

**Files:** module `services/atlas-merchant/atlas.com/merchant`; `go.mod`; `main.go:65` (`...SetMigrations(shop.Migration, listing.Migration, message.Migration, frederick.Migration)` → append — note the local package named `message` here is a domain package, so the main.go import alias for the outbox lib must not collide; `outboxlib` is safe); `shop/processor.go` (11 Emit, 8 tx), `frederick/administrator.go` (1 tx — check for a coupled emit in its caller); `kafka/consumer/merchant/consumer.go:198` builds a producer outside any tx — classify; this service has **no EmitWithResult** and its `kafka/message` lacks it — Pattern A only; inventory append.

- [ ] Steps 1–5. Commit: `git commit -m "feat(merchant): migrate tx-coupled emit sites to outbox"`

### Task 20: atlas-npc-shops

**Files:** module `services/atlas-npc-shops/atlas.com/npc` (go.mod module name is `atlas-npc`); `go.mod`; `main.go:63` multi-line SetMigrations (`commodities.Migration,` / `shops.Migration,` / seeder func) → append `outboxlib.Migration,` before the closing paren; `shops/processor.go` (5 Emit, 2 tx; struct-init at :92), tx-only sites in `shops/administrator.go`, `commodities/administrator.go` (migrate only if a coupled emit exists); inventory append.

- [ ] Steps 1–5. Commit: `git commit -m "feat(npc-shops): migrate tx-coupled emit sites to outbox"`

### Task 21: atlas-tenants

**Files:** module `services/atlas-tenants/atlas.com/tenants`; `go.mod`; `main.go:53` (`...SetMigrations(tenant.MigrateEntities, configuration.MigrateEntities)` → append); `configuration/processor.go` (3 Emit + 6 EmitWithResult), `tenant/processor.go` (1 Emit + 2 EmitWithResult); tx sites live in the two `administrator.go` files (re-entrancy handles them); struct-init providers at `configuration/processor.go:108`, `tenant/processor.go:62`; inventory append.

**Note:** heavy EmitWithResult usage — Pattern B throughout. atlas-tenants feeds the config-status projection consumed by login/channel; byte-parity of headers matters here most (already covered by lib tests).

- [ ] Steps 1–5. Commit: `git commit -m "feat(tenants): migrate tx-coupled emit sites to outbox"`

### Task 22: atlas-mounts

**Files:** module `services/atlas-mounts/atlas.com/mounts`; `go.mod`; `main.go:61` (`...SetMigrations(mount.Migration)` → append); Emit sites are all **outside** the processor: `mount/task.go:29-35` (the `applyTick` seam — Pattern C, exact after-code in the recipe), `kafka/consumer/buff/consumer.go:35`, `kafka/consumer/food/consumer.go:26` (same Pattern C wrap around their processor calls); `mount/processor.go` holds the 3 tx sites and needs no edits beyond accepting `tx` as its db (NewProcessor(l, ctx, tx) at the wrapped call sites); no EmitWithResult in this service's message package; inventory append.

- [ ] Steps 1–5. Commit: `git commit -m "feat(mounts): migrate tx-coupled emit sites to outbox"`

### Task 23: atlas-quest (divergent — EventEmitter interface)

atlas-quest has no `message.Emit` call sites and no local `ProviderImpl`. Emission goes through the `EventEmitter` interface (`quest/event_emitter.go:17-23`) whose `KafkaEventEmitter` impl publishes directly; the 8 call sites (`quest/processor.go:216,361,461,535,580,586,866,936`) run **after** their `ExecuteTransaction` blocks. Migration: an outbox-backed `EventEmitter` built per-transaction, and the emit calls move inside the tx closures.

**Files:**
- Modify: `services/atlas-quest/atlas.com/quest/go.mod`, `main.go:57` (`...SetMigrations(quest.Migration, progress.Migration)` → append + drainer block)
- Create: `services/atlas-quest/atlas.com/quest/quest/outbox_event_emitter.go`
- Modify: `services/atlas-quest/atlas.com/quest/quest/processor.go` (emitter plumbing + 8 sites)
- Append: `docs/tasks/task-114-outbox-adoption/inventory.md`

**Interfaces:**
- Consumes: `outbox.EnqueueBuffer`, existing providers `questproducer.QuestStartedEventProvider` / `QuestCompletedEventProvider` / `QuestForfeitedEventProvider` / `QuestProgressUpdatedEventProvider` (kafka/producer/quest/producer.go), `sagaproducer.SagaCommandProvider` (kafka/producer/saga/producer.go:17-20), tokens `questmessage.EnvStatusEventTopic` (`"EVENT_TOPIC_QUEST_STATUS"`, kafka/message/quest/kafka.go:73) and `sagamessage.EnvCommandTopic` (`"COMMAND_TOPIC_SAGA"`, kafka/message/saga/kafka.go:8).
- Produces: `NewOutboxEventEmitter(l logrus.FieldLogger, ctx context.Context, tx *gorm.DB) EventEmitter`; processor field `txEmitter func(tx *gorm.DB) EventEmitter`.

- [ ] **Step 1: Create the outbox emitter**

`quest/outbox_event_emitter.go`:

```go
package quest

import (
	questmessage "atlas-quest/kafka/message/quest"
	sagamessage "atlas-quest/kafka/message/saga"
	questproducer "atlas-quest/kafka/producer/quest"
	sagaproducer "atlas-quest/kafka/producer/saga"
	"context"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// OutboxEventEmitter persists events as outbox rows inside tx instead of
// publishing directly; the drainer publishes after commit.
type OutboxEventEmitter struct {
	l   logrus.FieldLogger
	ctx context.Context
	tx  *gorm.DB
}

func NewOutboxEventEmitter(l logrus.FieldLogger, ctx context.Context, tx *gorm.DB) EventEmitter {
	return &OutboxEventEmitter{l: l, ctx: ctx, tx: tx}
}

func (e *OutboxEventEmitter) enqueue(token string, p model.Provider[[]kafka.Message]) error {
	msgs, err := p()
	if err != nil {
		return err
	}
	return outbox.EnqueueBuffer(e.l, e.ctx, e.tx, map[string][]kafka.Message{token: msgs})
}

func (e *OutboxEventEmitter) EmitQuestStarted(transactionId uuid.UUID, characterId uint32, worldId world.Id, questId uint32, progress string, items []questmessage.ItemReward) error {
	return e.enqueue(questmessage.EnvStatusEventTopic, questproducer.QuestStartedEventProvider(transactionId, characterId, worldId, questId, progress, items))
}

func (e *OutboxEventEmitter) EmitQuestCompleted(transactionId uuid.UUID, characterId uint32, worldId world.Id, questId uint32, completedAt time.Time, items []questmessage.ItemReward) error {
	return e.enqueue(questmessage.EnvStatusEventTopic, questproducer.QuestCompletedEventProvider(transactionId, characterId, worldId, questId, completedAt, items))
}

func (e *OutboxEventEmitter) EmitQuestForfeited(transactionId uuid.UUID, characterId uint32, worldId world.Id, questId uint32) error {
	return e.enqueue(questmessage.EnvStatusEventTopic, questproducer.QuestForfeitedEventProvider(transactionId, characterId, worldId, questId))
}

func (e *OutboxEventEmitter) EmitProgressUpdated(transactionId uuid.UUID, characterId uint32, worldId world.Id, questId uint32, infoNumber uint32, progress string) error {
	return e.enqueue(questmessage.EnvStatusEventTopic, questproducer.QuestProgressUpdatedEventProvider(transactionId, characterId, worldId, questId, infoNumber, progress))
}

func (e *OutboxEventEmitter) EmitSaga(s sagamessage.Saga) error {
	return e.enqueue(sagamessage.EnvCommandTopic, sagaproducer.SagaCommandProvider(s))
}
```

(Match `EmitSaga`'s parameter type to the interface — `event_emitter.go:22` uses `saga.Saga` where `saga` is `atlas-quest/kafka/message/saga`; keep the same import alias the interface file uses.)

- [ ] **Step 2: Processor plumbing**

Add a `txEmitter func(tx *gorm.DB) EventEmitter` field to `ProcessorImpl`. `NewProcessor` (processor.go:99) defaults it to `func(tx *gorm.DB) EventEmitter { return NewOutboxEventEmitter(l, ctx, tx) }`. `NewProcessorWithDependencies` (processor.go:104) sets `txEmitter: func(*gorm.DB) EventEmitter { return eventEmitter }` so injected mocks keep working, and the `WithTransaction`-style copy at :124 carries the field. Then, at each of the 8 emit call sites, move the `p.eventEmitter.EmitX(...)` call **inside** the associated `ExecuteTransaction` closure as `p.txEmitter(tx).EmitX(...)`, keeping argument lists identical. `EmitSaga` sites (:866, :936) return values from inside their flows — restructure so the enqueue happens in the tx and the awarded-items result is returned after, e.g. capture `awardedItems` outside the closure as Pattern B does.

- [ ] **Step 3: Verify**

Run from the module dir: `go test -race ./... && go vet ./... && go build ./...`
Expected: clean. Existing `processor_test.go` mocks inject via `NewProcessorWithDependencies` — they continue to see every emit through their mock (via the wrapped `txEmitter`).

- [ ] **Step 4: Inventory + commit**

```bash
git add services/atlas-quest docs/tasks/task-114-outbox-adoption/inventory.md
git commit -m "feat(quest): outbox-backed EventEmitter, emits move inside transactions"
```

### Task 24: atlas-gachapons, atlas-drop-information, atlas-data — inventory-only

The FR-3.1 sweep (recorded in context.md) found: atlas-gachapons and atlas-drop-information have **no Kafka producer usage at all**; atlas-data's two producer calls are not tx-coupled (`data/processor.go:85` is a pure `START_WORKER` command dispatch; `data/processor.go:287` `emitDataUpdated` fires after a whole worker completes across many independent transactions and is TTL-guarded — no single transaction could make it atomic). Per PRD §7, zero-site services get inventory entries and **no code change** (no drainer, no migration registration).

**Files:**
- Append: `docs/tasks/task-114-outbox-adoption/inventory.md`

- [ ] **Step 1: Re-verify the sweep** (don't trust the plan's snapshot blindly)

```bash
grep -rn "producer\.\|kafka" --include='*.go' services/atlas-gachapons/atlas.com/gachapons | grep -v _test.go | grep -iv "consumer" | head
grep -rn "producer\.\|kafka" --include='*.go' services/atlas-drop-information/atlas.com/dis | grep -v _test.go | head
grep -rn "ProviderImpl" --include='*.go' services/atlas-data/atlas.com/data | grep -v _test.go
```

Expected: no producer hits for gachapons/drop-information; exactly the two atlas-data sites. If reality differs, migrate the found sites per the recipe instead (and say so in the inventory).

- [ ] **Step 2: Inventory entries**

Append three sections: gachapons and drop-information as "Zero Kafka producer usage; no tx-coupled emit sites; no code change." atlas-data as "Zero tx-coupled emit sites (both producer calls are non-tx: START_WORKER command at data/processor.go:85; DATA_UPDATED at :287 aggregates many independent transactions and is TTL-guarded); no code change." Note in atlas-data's entry that design §7 anticipated `EnqueueBuffer` use here; the authoritative FR-3.1 sweep found no qualifying site.

- [ ] **Step 3: Commit**

```bash
git add docs/tasks/task-114-outbox-adoption/inventory.md
git commit -m "docs(task-114): inventory — gachapons, drop-information, data have zero tx-coupled sites"
```

---

## Phase 5 — CI guard

### Task 25: tools/outboxguard analyzer + wrapper + CI

Modeled exactly on tools/rediskeyguard (design D5). Rule: a call to `ProviderImpl` from a package named `producer`, lexically inside a function literal passed to `database.ExecuteTransaction(...)` or any `.Transaction(...)` call, is a diagnostic.

**Files:**
- Create: `tools/outboxguard/go.mod`, `tools/outboxguard/analyzer.go`, `tools/outboxguard/analyzer_test.go`, `tools/outboxguard/cmd/outboxguard/main.go`, `tools/outboxguard/testdata/src/gorm/gorm.go`, `tools/outboxguard/testdata/src/database/database.go`, `tools/outboxguard/testdata/src/producer/producer.go`, `tools/outboxguard/testdata/src/guardtest/example.go`, `tools/outbox-guard.sh`
- Modify: `.github/workflows/pr-validation.yml`

- [ ] **Step 1: Module + analyzer**

`tools/outboxguard/go.mod`:

```
module github.com/Chronicle20/atlas/tools/outboxguard

go 1.25.5

require golang.org/x/tools v0.47.0
```

(`go mod tidy` will add indirects.) `tools/outboxguard/analyzer.go`:

```go
package outboxguard

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name:     "outboxguard",
	Doc:      "bans direct Kafka producer construction (producer.ProviderImpl) inside DB transaction closures",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// The guard is a lexical regression tripwire, not a taint analysis: the
// fleet's only direct-producer entry point in service code is the local
// kafka/producer.ProviderImpl, and transaction entry points are uniformly
// database.ExecuteTransaction or gorm's (*DB).Transaction.
func run(pass *analysis.Pass) (interface{}, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	insp.Preorder([]ast.Node{(*ast.CallExpr)(nil)}, func(n ast.Node) {
		call := n.(*ast.CallExpr)
		if !isTxEntryPoint(call) {
			return
		}
		for _, arg := range call.Args {
			fl, ok := arg.(*ast.FuncLit)
			if !ok {
				continue
			}
			ast.Inspect(fl.Body, func(inner ast.Node) bool {
				sel, ok := inner.(*ast.SelectorExpr)
				if !ok || sel.Sel.Name != "ProviderImpl" {
					return true
				}
				ident, ok := sel.X.(*ast.Ident)
				if !ok {
					return true
				}
				pkgName, ok := pass.TypesInfo.Uses[ident].(*types.PkgName)
				if !ok || pkgName.Imported().Name() != "producer" {
					return true
				}
				pass.Reportf(sel.Pos(),
					"outboxguard: producer.ProviderImpl inside a DB transaction closure; enqueue via outbox.EmitProvider instead")
				return true
			})
		}
	})
	return nil, nil
}

func isTxEntryPoint(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	switch sel.Sel.Name {
	case "ExecuteTransaction":
		ident, ok := sel.X.(*ast.Ident)
		return ok && ident.Name == "database"
	case "Transaction":
		return true
	}
	return false
}
```

- [ ] **Step 2: Test fixtures**

`testdata/src/gorm/gorm.go`:

```go
package gorm

type DB struct{}

func (d *DB) Transaction(fn func(tx *DB) error) error { return fn(d) }
```

`testdata/src/database/database.go`:

```go
package database

import "gorm"

func ExecuteTransaction(db *gorm.DB, fn func(tx *gorm.DB) error) error { return fn(db) }
```

`testdata/src/producer/producer.go`:

```go
package producer

func ProviderImpl(l interface{}) interface{} { return nil }
```

`testdata/src/guardtest/example.go`:

```go
package guardtest

import (
	"database"
	"gorm"
	"producer"
)

func bad(db *gorm.DB) error {
	return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
		_ = producer.ProviderImpl(nil) // want "producer.ProviderImpl inside a DB transaction closure"
		return nil
	})
}

func alsoBad(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		_ = producer.ProviderImpl(nil) // want "producer.ProviderImpl inside a DB transaction closure"
		return nil
	})
}

func good(db *gorm.DB) error {
	err := database.ExecuteTransaction(db, func(tx *gorm.DB) error { return nil })
	_ = producer.ProviderImpl(nil)
	return err
}
```

`tools/outboxguard/analyzer_test.go`:

```go
package outboxguard_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	outboxguard "github.com/Chronicle20/atlas/tools/outboxguard"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), outboxguard.Analyzer, "guardtest")
}
```

`tools/outboxguard/cmd/outboxguard/main.go` (mirror `tools/rediskeyguard/cmd/rediskeyguard/main.go`'s structure):

```go
package main

import (
	"golang.org/x/tools/go/analysis/singlechecker"

	outboxguard "github.com/Chronicle20/atlas/tools/outboxguard"
)

func main() { singlechecker.Main(outboxguard.Analyzer) }
```

- [ ] **Step 3: Run the analyzer tests**

Run from `tools/outboxguard`: `GOWORK=off go mod tidy && GOWORK=off go test ./...`
Expected: PASS (both `want` diagnostics matched, `good` clean).

- [ ] **Step 4: Shell wrapper**

`tools/outbox-guard.sh` — copy `tools/redis-key-guard.sh` and swap names:

```bash
#!/usr/bin/env bash
# Build the outboxguard analyzer once, then run it over every Go service
# module. Non-empty diagnostics → non-zero exit. Run from the repo root.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
GUARD_SRC="$ROOT/tools/outboxguard"
BIN="$(mktemp -d)/outboxguard"

echo "building outboxguard..."
( cd "$GUARD_SRC" && GOWORK=off go build -o "$BIN" ./cmd/outboxguard )

rc=0
while IFS= read -r modfile; do
    moddir="$(dirname "$modfile")"
    echo "outboxguard: $moddir"
    if ! ( cd "$moddir" && "$BIN" ./... ); then
        rc=1
    fi
done < <(find "$ROOT/services" -name go.mod -not -path '*/node_modules/*')

if [ "$rc" -ne 0 ]; then
    echo "outboxguard: FAIL — direct producer calls inside DB transactions (use outbox.EmitProvider)"
fi
exit $rc
```

`chmod +x tools/outbox-guard.sh`. Run it from the repo root — expected: exit 0 (the tree is clean by now; the guard starts with no baseline).

- [ ] **Step 5: CI wiring**

In `.github/workflows/pr-validation.yml`: copy the `redis-key-guard` job block (around lines 84-98) as a sibling job `outbox-guard` (name "Outbox Guard", run step `./tools/outbox-guard.sh`); add `outbox-guard` to the final gate job's `needs:` list (around line 480); mirror the `GUARD_RESULT` consumption pattern (around line 496) with the new job's result. Follow the existing job verbatim — same checkout/setup-go steps.

- [ ] **Step 6: Commit**

```bash
git add tools/outboxguard tools/outbox-guard.sh .github/workflows/pr-validation.yml
git commit -m "feat(tools): outboxguard analyzer bans in-tx direct producer calls"
```

---

## Phase 6 — Fleet verification and closeout

### Task 26: Full verification gates, inventory completeness, CD-2 closeout

**Files:**
- Modify: `docs/architectural-improvements.md` (CD-2 section), `docs/tasks/task-114-outbox-adoption/inventory.md` (final sweep evidence)

- [ ] **Step 1: Acceptance sweep (PRD #1 evidence)**

From the worktree root:

```bash
tools/outbox-guard.sh
grep -rn "producer.ProviderImpl" --include='*.go' services/ | grep -v _test.go | grep -v "kafka/producer"
```

The guard must exit 0. Manually classify every remaining `ProviderImpl` call site from the grep: each must be a non-tx emit (command/relay/rejection/ticker) already recorded as "left direct" in inventory.md. Add the sweep date + guard exit status to inventory.md's header as the audit evidence.

- [ ] **Step 2: Per-module gates**

For every changed module (libs/atlas-outbox; the 13 migrated services + atlas-configurations; tools/outboxguard):

```bash
go test -race ./... && go vet ./... && go build ./...
```

All clean. Also from the repo root: `tools/redis-key-guard.sh` — clean.

- [ ] **Step 3: Docker bake**

From the worktree root:

```bash
docker buildx bake all-go-services
```

Expected: every target builds. This is mandatory (go.mod changed in ~15 services); fix any missing-COPY issues (none expected — atlas-outbox is already in both Dockerfile blocks at lines 39/68/92) and re-run until clean.

- [ ] **Step 4: CD-2 closeout**

In `docs/architectural-improvements.md`, update the CD-2 item: implemented by task-114 — all transactional services publish tx-coupled events through libs/atlas-outbox; consumer dedup on TransactionId remains open as CD-1. Where a migrated service has its own `docs/` directory documenting Kafka behavior, add a one-line at-least-once delivery note (check with `ls services/atlas-*/docs 2>/dev/null`; skip services without docs).

- [ ] **Step 5: Final inventory review**

inventory.md must have a section for every §7 service (17 + configurations note). Every service section lists migrated sites, left-direct sites with reasons, or the zero-sites line. No "TBD" anywhere.

- [ ] **Step 6: Commit**

```bash
git add docs/architectural-improvements.md docs/tasks/task-114-outbox-adoption/inventory.md
git commit -m "docs(task-114): CD-2 closeout, final verification evidence"
```

After this task: run the code-review step (`superpowers:requesting-code-review`) before opening the PR — mandatory per CLAUDE.md.
