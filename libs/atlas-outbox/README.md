# atlas-outbox

Transactional outbox library for Atlas services. Lets a service atomically
persist domain state changes and outbound Kafka events in the same database
transaction, then relay those events asynchronously with **at-least-once**
delivery semantics.

## Delivery semantics — consumers MUST be idempotent

Messages may be redelivered if the relayer crashes between publishing to
Kafka and marking the row `sent_at`, or if the broker doesn't ack a write.
Every consumer of an outbox-driven topic is responsible for idempotency
(by message key, by tombstone-aware projection, by domain-level dedup,
etc.). The library does not attempt exactly-once.

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

## Schema

`outbox_entries` (managed via `Migration(db)`):

| Column | Type | Notes |
|---|---|---|
| `id` | bigserial | primary key |
| `topic` | text | indexed (partial: unsent rows) |
| `message_key` | bytea | required |
| `message_value` | bytea | nullable (tombstone-friendly) |
| `headers` | jsonb | default `'{}'` |
| `enqueued_at` | timestamptz | default `now()` |
| `sent_at` | timestamptz | indexed (partial: sent rows, for sweeper) |
| `attempts` | int | increments on publish failure |
| `last_error` | text | last publish error message |

The table is **not tenant-scoped**; queries must use
`database.WithoutTenantFilter(ctx)` when reading it from a tenant-aware
service.

## Testing

- Unit tests use Sqlite in-memory; `go test ./...` requires no Docker.
- Integration tests are gated behind `//go:build integration`. Run them
  with `go test -tags=integration ./...` — they spin up a postgres
  testcontainer (`postgres:16-alpine`) for advisory-lock and NOTIFY
  scenarios.
