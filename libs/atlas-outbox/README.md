# atlas-outbox

Transactional outbox library for Atlas services. Lets a service atomically
persist domain state changes and outbound Kafka events in the same database
transaction, then relay those events asynchronously with **at-least-once**
delivery semantics.

## Delivery semantics — consumers MUST be idempotent

Delivery is **at-least-once**, never exactly-once. The specific failure
window: `publishBatch` publishes a batch via `WriteMessages` and only
*then*, in the same DB transaction, stamps `sent_at` on those rows
(`drainer.go`). A crash (or lost connection) after Kafka acks the write
but before the `sent_at` UPDATE commits leaves the rows looking unsent —
the next tick re-selects and republishes them. The broker not acking a
write behaves the same way from the drainer's perspective (the batch
transaction rolls back, `attempts`/`last_error` are bumped, and the rows
remain eligible for the next batch).

Every consumer of an outbox-driven topic is responsible for its own
idempotency (by message key, by tombstone-aware projection, by
domain-level dedup, etc.). Consumer-side dedup keyed on `TransactionId`
is tracked separately as task **CD-1** — it is not part of this library
and not part of adopting it; adopting this library moves a topic from
"as good as its old delivery guarantee" to "at-least-once with possible
redelivery" and callers must be able to tolerate that today, ahead of
CD-1 landing.

## Ordering guarantee

Within one flushed buffer, the single drainer leader publishes
`outbox_entries` rows in `id ASC` order (`publishBatch`'s
`Order("id ASC")`), and `id` is a `bigserial` assigned at `Enqueue` time —
so **per-service, per-transaction emission order is preserved**: if a
caller enqueues messages A then B inside the same GORM transaction, A's
row id is lower and A publishes before B.

Caveats:

- **Across concurrent transactions**, id order is *allocation* order
  (row insert order across all callers), not commit order. Two
  transactions T1 and T2 racing to commit can have T2 allocate a lower id
  than T1 if T2's `Enqueue` call lands first even though T1 commits
  first. This is the same characteristic the previous `enqueued_at`-based
  ordering had — the guarantee is per-flow (per producer/per-transaction),
  not a global cross-flow total order.
- **Cross-topic order within one flushed buffer** follows Go map
  iteration order: `EnqueueBuffer`/`EmitProvider` iterate a
  `map[string][]kafka.Message` keyed by topic token, so messages destined
  for different topics in the same buffer are enqueued (and therefore
  published) in an unspecified relative order — exactly as on the direct
  producer path, which folds the same `message.Buffer` map the same way.
  Ordering *within* a single topic's message slice is preserved.

## Headers

Header values are stored **base64-encoded** inside the `headers` jsonb
column (`headers.go`, `encodeHeaders`/`decodeHeaders`) and decoded back to
`[]byte` byte-exact at publish time. This is required, not cosmetic:
tenant-version headers are raw big-endian `uint16` bytes
(`atlas-kafka`'s `TenantHeaderDecorator`) and therefore always contain a
NUL byte — which Postgres `jsonb` rejects outright — and may not be valid
UTF-8 (e.g. version `185` = `0xB9`), which `encoding/json` would silently
mangle to `U+FFFD` if stored raw. Base64 keeps every header value,
binary or not, byte-exact through the store/publish round trip. Header
*keys* are plain ASCII and are stored unencoded.

The header set attached at publish equals the direct producer path's
span + tenant decoration: `EnqueueBuffer`'s `headerMap` folds
`kafkaproducer.SpanHeaderDecorator(ctx)` and
`kafkaproducer.TenantHeaderDecorator(ctx)` — the same two decorators, and
the same key set, the direct-path `produceHeaders` append-fold applies at
emit time. Because the two decorators' key sets are disjoint, the
map-merge `EnqueueBuffer` performs is equivalent to that append-fold.

## API

### Enqueue (producer side)

`Enqueue(tx *gorm.DB, msg Message) error` writes an `outbox_entries` row
inside the caller's existing GORM transaction. The caller is responsible
for the transaction lifecycle — `Enqueue` will **fail** if `tx` is nil.

```go
err := db.Transaction(func(tx *gorm.DB) error {
    if err := tx.Save(&domainModel).Error; err != nil {
        return err
    }
    return outbox.Enqueue(tx, outbox.Message{
        Topic: "EVENT_TOPIC_SOMETHING",
        Key:   []byte(domainModel.ID),
        Value: payloadBytes,
        Headers: map[string]string{"trace": traceId},
    })
})
```

Atomicity guarantee: if the wrapping transaction rolls back, the outbox
row never exists; if it commits, both the domain row and the outbox row
are durable. On Postgres, `Enqueue` also issues a `pg_notify` so a
listening drainer wakes immediately (see below).

A nil `Value` is permitted and stored as a SQL `NULL`, suitable for
log-compacted topic **tombstones**.

### Drainer (relayer side)

`NewDrainer(logger, db, publisher, opts...)` constructs the relayer.
Options:

| Option | Default | Purpose |
|---|---|---|
| `WithPollInterval(d)` | `1s` | Periodic poll cadence (fallback when no NOTIFY) |
| `WithBatchSize(n)` | `100` | Rows fetched per publish cycle |
| `WithSweeperInterval(d)` | `1h` | Cadence for the retention sweeper |
| `WithRetention(d)` | `7d` | How long published rows are kept before deletion |
| `WithDSN(dsn)` | `""` | Postgres DSN for the LISTEN connection |

`Drainer.Run(ctx)` is blocking; run it in its own goroutine. `Stop()` or
context cancellation terminates the loop and closes the LISTEN
connection if one was opened.

### Leadership — only one publisher per cluster

On Postgres, the drainer holds a session-scoped advisory lock
(`pg_try_advisory_lock`) before publishing. Multiple service replicas can
run a drainer concurrently; only the lock holder publishes. Lock
contention is non-blocking — followers poll and become leader on the
next tick after the leader exits or dies.

Batches are fetched with `FOR UPDATE SKIP LOCKED` inside a transaction
that also updates `sent_at`, so an in-flight batch can never be
double-published even across leader transitions.

On non-Postgres backends (Sqlite tests), leadership is bypassed —
single-process tests publish directly.

### NOTIFY / poll wakeup

`Enqueue` issues `SELECT pg_notify('atlas_outbox_new', topic)` on
Postgres. When `WithDSN(...)` is configured, the drainer opens a
`pq.Listener` at `Run` start and `runLeader` selects on both the poll
ticker and the listener's channel. Typical end-to-end latency from
`Enqueue` to Kafka publish is **sub-100ms** with NOTIFY active, falling
back to `pollInterval` if the listener disconnects or no DSN is provided.

### Sweeper

`SweepOnce(ctx)` deletes published rows whose `sent_at` is older than
`cfg.retention`. The drainer schedules it leader-only at
`cfg.sweeperInterval`; exposed publicly for operator-driven sweeps.

### Backfill

`Backfill(db, topic, loader, keyFn, valueFn)` is the bootstrap path for
fresh clusters or recovery — it enqueues an outbox row for each loader
row whose `(topic, key)` is not already present in `outbox_entries`.
**Idempotent on key**: running it on every service startup is safe.
Returns the number of rows actually inserted (zero on a steady-state
restart).

## Adoption API

Three entry points cover the ways a service's existing Kafka-emission code
can be redirected into the outbox without changing its shape:

### `EnqueueBuffer` — Buffer-less callers

```go
func EnqueueBuffer(l logrus.FieldLogger, ctx context.Context, tx *gorm.DB, contents map[string][]kafka.Message) error
```

Persists a `message.Buffer`-shaped payload (`map[env-var token][]kafka.Message]`)
as outbox rows inside `tx`. Each token is resolved to a real topic name via
`topic.EnvProvider(l)(token)()` (the same resolution the direct producer
path uses), and span + tenant headers are derived from `ctx` once per call
and applied to every message. Message key/value bytes pass through
unchanged. Any failure returns an error and fails the enclosing
transaction — nothing is enqueued on a partial failure within one call.

### `EmitProvider` — `message.Emit` / `EmitWithResult` call sites

```go
func EmitProvider(l logrus.FieldLogger, ctx context.Context, tx *gorm.DB) func(token string) kafkaproducer.MessageProducer
```

Returns a value shaped exactly like a service-local
`kafka/producer.Provider` (the unnamed `func(token string) producer.MessageProducer`
type every service already defines), so it drops into existing
`message.Emit(producer.Provider, ...)` / `message.EmitWithResult(...)` call
sites with **no signature change** at the call site — only the `Provider`
argument passed in changes, from the Kafka-writing one to
`outboxlib.EmitProvider(l, ctx, tx)`. Internally it drains the
`model.Provider[[]kafka.Message]` the emit call already builds and forwards
the result to `EnqueueBuffer` under a single-token buffer. Because it
enqueues inside `tx`, the caller must already be inside that transaction
when the emit call executes — the outbox row's atomicity is only as good
as the transaction it's enqueued under.

### `NewTopicWriterPool` — the standard drainer `Publisher`

```go
func NewTopicWriterPool() *TopicWriterPool
```

The production `Publisher` implementation for `NewDrainer`. Reads
`BOOTSTRAP_SERVERS` (comma-separated) once at construction and lazily
creates one long-lived `kafka.Writer` per real topic name on first
publish to that topic (outbox rows store real topic names, not env-var
tokens, so no token resolution happens here). Call `Close()` during
service teardown to flush and close every cached writer.

### Wiring template

The full adoption shape — migration, drainer boot, and teardown — as used
in `services/atlas-configurations/atlas.com/configurations/main.go`:

```go
db := database.Connect(l, database.SetMigrations(
    /* ...existing migrations..., */ outboxlib.Migration,
))

// Boot the outbox drainer: publishes the transactional outbox to Kafka.
// WithDSN gives sub-100ms wakeup via pq.Listener; the poll interval is
// the fallback. Leadership is gated by a postgres advisory lock, so
// multiple replicas can run the drainer safely — only the lock holder
// publishes.
publisher := outboxlib.NewTopicWriterPool()
drainer := outboxlib.NewDrainer(l, db, publisher, outboxlib.WithDSN(database.DSN()))
go drainer.Run(tdm.Context())
tdm.TeardownFunc(func() {
    drainer.Stop()
    publisher.Close()
})
```

At call sites inside processors, either enqueue directly with `Enqueue`
(see the API section above), or swap a `producer.Provider` argument for
`outboxlib.EmitProvider(l, ctx, tx)` / route a `message.Buffer` through
`outboxlib.EnqueueBuffer(l, ctx, tx, contents)` — both require the call to
execute inside the same GORM transaction that persists the domain change.

## Schema

`outbox_entries` (managed via `Migration(db)`):

| Column | Type | Notes |
|---|---|---|
| `id` | bigserial | primary key |
| `topic` | text | indexed (partial: unsent rows) |
| `message_key` | bytea | required |
| `message_value` | bytea | nullable (tombstone-friendly) |
| `headers` | jsonb | default `'{}'`; values base64-encoded, see Headers above |
| `enqueued_at` | timestamptz | default `now()` |
| `sent_at` | timestamptz | indexed (partial: sent rows, for sweeper) |
| `attempts` | int | increments on publish failure |
| `last_error` | text | last publish error message |

The table is **not tenant-scoped**; queries must use
`database.WithoutTenantFilter(ctx)` when reading it from a tenant-aware
service. Tenancy still rides through the pipeline — it's carried in the
per-message tenant headers (see Headers above), not in a table column.

## Operations

- **A wedged drainer shows up as growth in `sent_at IS NULL` rows.** No
  replica holding the advisory lock, a stuck leader, or a broker that's
  permanently unreachable all present the same way: unsent rows pile up
  and stop draining. Alert on the unsent-row count (or its growth rate)
  against `outbox_entries`, e.g. `SELECT count(*) FROM outbox_entries
  WHERE sent_at IS NULL` (the `outbox_entries_unsent_idx` partial index
  keeps this cheap even as the table grows).
- **Publish failures are logged with `attempts` and `last_error`.** When
  `WriteMessages` returns an error, `publishBatch` rolls back the
  SELECT/UPDATE transaction (the batch is not marked sent) and, in a
  separate statement against the un-locked rows, increments `attempts`
  and stores the error text in `last_error` for every row in the failed
  batch. A row with a high `attempts` and non-null `last_error` that
  isn't clearing on subsequent ticks points at a systemic publish
  problem (bad broker address, ACL, oversized message) rather than a
  transient blip — check `last_error` first.
- The drainer also logs `outbox.lock_acquired` / `outbox.lock_lost` on
  leadership transitions and `outbox.publish_failed` /
  `outbox.sweeper_failed` on the corresponding failures — useful for
  correlating a stall with a specific replica or a leadership flap.
- `SweepOnce` logs `outbox.sweeper_run` with the deleted row count each
  time it actually deletes rows, so retention behavior is observable
  without querying the table directly.

## Testing

- Unit tests use Sqlite in-memory; `go test ./...` requires no Docker.
- Integration tests are gated behind `//go:build integration`. Run them
  with `go test -tags=integration ./...` — they spin up a postgres
  testcontainer (`postgres:16-alpine`) for advisory-lock and NOTIFY
  scenarios.
