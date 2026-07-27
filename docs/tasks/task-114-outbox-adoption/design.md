# Fleet-Wide Transactional Outbox Adoption — Design

Task: task-114-outbox-adoption
Status: Proposed
Created: 2026-07-02
PRD: docs/tasks/task-114-outbox-adoption/prd.md

---

## 1. Verified Current State

Everything below was read from source in this worktree; citations are file:line.

### 1.1 The lib is complete for enqueue/drain, but has two publish-side gaps

`libs/atlas-outbox` provides `Enqueue(tx, Message)` (outbox.go:18), a drainer with
advisory-lock leadership + NOTIFY wakeup (drainer.go:79-174), a sweeper
(drainer.go:181-211), migration, and backfill. Two gaps matter to this task:

1. **Headers are persisted but never published.** `Enqueue` stores
   `msg.Headers` as jsonb (outbox.go:29-42, entity.go:14), but
   `publishBatch` builds outgoing `kafka.Message`s with only
   `Topic`/`Key`/`Value` (drainer.go:232-239). The headers column is dead
   weight on the publish path today. This has not bitten atlas-configurations
   because its two enqueue sites pass no headers at all
   (services/atlas-configurations/atlas.com/configurations/services/processor.go:48,
   tenants/processor.go:41). For tenant-scoped services, header loss is fatal —
   every consumer parses tenant headers. **The drainer must re-attach headers.**
2. **Publish order has a tie problem.** `publishBatch` orders by
   `enqueued_at ASC` (drainer.go:220), but `enqueued_at` defaults to
   `CURRENT_TIMESTAMP` (entity.go:15), which is transaction-stable in Postgres:
   every row enqueued in one tx gets the *same* timestamp, so intra-transaction
   order (e.g. `MESO_CHANGED` before `STAT_CHANGED`) is unspecified at publish
   time. PRD FR-4.2 promises id-order publishing. **The drainer must order by
   `id ASC`** (bigserial, monotonically assigned per insert).

### 1.2 The publisher is service-local

`TopicWriterPool` (one long-lived `kafka.Writer` per topic, lazy, keyed by real
topic name) lives in
services/atlas-configurations/atlas.com/configurations/outbox/publisher.go:19-92.
It is service-agnostic already — it reads only `BOOTSTRAP_SERVERS` and mirrors
the settings of the direct path's `defaultWriterFactory`
(libs/atlas-kafka/producer/manager.go:118-126). Straight move to the lib.

### 1.3 The service-side emit seam is uniform and narrow

Every transactional service carries an identical local copy of two small
packages (verified present in character, inventory, cashshop, fame, buddies,
guilds, notes, pets, mounts, skills, quest, merchant, npc-shops, tenants):

- `kafka/message/message.go` — `Buffer` (`map[string][]kafka.Message` keyed by
  **env-var token**, `Put`, `GetAll`) plus `Emit(p producer.Provider)(closure)`
  and `EmitWithResult` which construct a Buffer, run the closure, then loop
  `p(token)(FixedProvider(msgs))` per token
  (services/atlas-character/atlas.com/character/kafka/message/message.go:44-77).
- `kafka/producer/producer.go` — `type Provider func(token string) producer.MessageProducer`
  and `ProviderImpl(l)(ctx)` which curries span+tenant header decorators onto
  the shared `producer.Produce`/`ManagerWriterProvider` pipeline
  (services/atlas-character/atlas.com/character/kafka/producer/producer.go:10-20).

Key observation: **`producer.Provider` is the single seam.** `message.Emit`
never touches Kafka itself — it hands each `(token, []kafka.Message)` pair to
whatever `Provider` it was given. A Provider that *enqueues to the outbox
inside a tx* instead of publishing makes every existing `Emit`/`EmitWithResult`
call site outbox-capable without changing the `message` package at all.

Header decoration on the direct path happens inside `Produce` at emit time
(`DecorateHeaders`, libs/atlas-kafka/producer/producer.go:52-56,74-83); buffered
`kafka.Message`s carry **no** headers until then (message.go transformer,
libs/atlas-kafka/producer/message.go:42-52). Token→topic resolution is
`topic.EnvProvider(l)(token)` — env lookup with fall-through to the raw token
(libs/atlas-kafka/topic).

Three affected-list services do not have the Buffer pattern: atlas-gachapons
and atlas-drop-information have **no producer at all** (no emit sites — their
inventory entries will be "zero tx-coupled sites" unless the FR-3.1 sweep finds
otherwise), and atlas-data has a producer but no `message` package (its emit
sites use `ProviderImpl` directly).

### 1.4 atlas-character hot paths (FR-1), exact defects

`services/atlas-character/atlas.com/character/character/processor.go`:

- `RequestChangeMeso` (:733): emits `notEnoughMeso` *inside* the tx (:742);
  assigns `err = dynamicUpdate(...)` then ignores it and emits `MESO_CHANGED` +
  `STAT_CHANGED` inside the tx regardless (:749-751). Additionally the uint32
  overflow branch does `return err` where `err` is nil (:744-747) — the
  "rejection" silently commits as a no-op success.
- `AttemptMesoPickUp` (:755): same unchecked `err = dynamicUpdate` before an
  in-tx `STAT_CHANGED` emit (:767-768); same nil-`err` return in the overflow
  branch (:762-764).
- `RequestDropMeso` (:776): `notEnoughMeso` emitted inside the tx (:785); the
  success-path `STAT_CHANGED` is already outside the tx but fire-and-forget.

The file has 25 `ExecuteTransaction` call sites total — the FR-3.1 inventory
for atlas-character extends well beyond the three meso paths (fame at :802,
etc.).

### 1.5 Infrastructure facts

- `database.ExecuteTransaction` is re-entrant-safe: if the given `*gorm.DB` is
  already a tx it runs the closure directly (libs/atlas-database/transaction.go:9-14).
  Wrapping previously-bare `Save`/`Create` sites in a tx is therefore safe even
  when called from an outer tx.
- `database.DSN()` and `SetMigrations` exist (connection.go:62,74); the
  configurations wiring template is main.go:52-60.
- Services already `replace`-reference libs; adding
  `github.com/Chronicle20/atlas/libs/atlas-outbox` to a service go.mod is the
  established pattern. The repo-root Dockerfile already COPYies
  `libs/atlas-outbox` (atlas-configurations builds today), so no Dockerfile
  change is needed; each migrating service still needs the bake gate because
  its own `go.mod` changes.
- CI-guard precedent: `tools/redis-key-guard.sh` builds a `go/analysis`
  analyzer from `tools/rediskeyguard` with `GOWORK=off` and runs it over every
  `services/*/go.mod` module (tools/redis-key-guard.sh:1-27).

## 2. Design Decisions

### D1 — Adoption seam: an outbox-backed `producer.Provider` (chosen), not a new Emit shape

Three alternatives were considered for how migrated processor code reaches the
outbox:

**A. Promote `message.Buffer` into a shared lib, add a lib-owned
`outbox.Emit`.** Kills ~15 duplicated `message` packages, but forces an
import-path migration across every service in the same task, ballooning the
diff far beyond the PRD's scope and coupling the outbox migration to an
unrelated dedup refactor. Rejected (worthwhile, but a separate task).

**B. Bridge function only; each service hand-writes a local `EmitTx`.**
Smallest lib, but ~15 copies of new boilerplate — exactly the "service
hand-rolls plumbing" the PRD forbids. Rejected.

**C. The lib exports a Provider-shaped constructor whose `MessageProducer`
enqueues instead of publishing (chosen).**

```go
// libs/atlas-outbox/provider.go
// EmitProvider returns a value assignable to each service's local
// producer.Provider type. Messages flushed through it are persisted as
// outbox rows in tx (token resolved to a real topic, span+tenant headers
// from ctx applied) instead of being written to Kafka.
func EmitProvider(l logrus.FieldLogger, ctx context.Context, tx *gorm.DB) func(token string) producer.MessageProducer
```

Because the return type is the *unnamed* func type underlying every service's
local `Provider`, existing `message.Emit` / `EmitWithResult` call sites accept
it without conversion. A migration is then purely structural:

```go
// before
return database.ExecuteTransaction(db, func(tx *gorm.DB) error { mutate(tx); ... })
... message.Emit(producer.ProviderImpl(l)(ctx))(fillBuffer)

// after
return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
    return message.Emit(outbox.EmitProvider(l, ctx, tx))(func(buf *message.Buffer) error {
        if err := mutate(tx, buf); err != nil { return err }
        return nil
    })
})
```

The service-local `message` and `producer` packages are untouched; non-tx
flows keep the direct path with zero diff. This satisfies FR-2.3's "keeps its
current structure" requirement literally — same Buffer, same Emit, different
Provider.

The bridge required by FR-2.2 is exported alongside it (used internally by
`EmitProvider`, directly by Buffer-less services like atlas-data and by tests):

```go
// libs/atlas-outbox/bridge.go
// EnqueueBuffer persists a message.Buffer-shaped payload (env-token →
// messages) as outbox rows inside tx. Any failure returns error, failing
// the enclosing transaction.
func EnqueueBuffer(l logrus.FieldLogger, ctx context.Context, tx *gorm.DB, contents map[string][]kafka.Message) error
```

Both funcs share one internal per-token path: resolve token via
`topic.EnvProvider(l)(token)()`; compute the header map once per call by
invoking `producer.SpanHeaderDecorator(ctx)` then
`producer.TenantHeaderDecorator(ctx)` and merging (same order `ProviderImpl`
passes them; span and tenant key sets are disjoint, so map-merge ≡ the direct
path's append-fold in produceHeaders, libs/atlas-kafka/producer/message.go:14-27);
call `Enqueue(tx, Message{Topic, Key, Value, Headers})` per message.

Dependency note: `libs/atlas-outbox` gains imports of
`libs/atlas-kafka` (topic + producer decorator/MessageProducer types) and
`libs/atlas-model` (MessageProducer's signature). No cycle — atlas-kafka does
not import atlas-outbox. go.work already contains both.

### D2 — Header parity: decorate at enqueue, re-attach at publish

- **Enqueue side**: headers are computed from the *request* ctx at enqueue
  time (D1). This is the semantic equivalent of the direct path, which
  decorates from the same ctx at emit time within the same request.
- **Publish side (lib fix)**: `publishBatch` unmarshals `Entity.Headers` into
  `map[string]string` and appends `kafka.Header{Key, Value}` entries to the
  outgoing message. Empty/`{}` headers → no header slice, byte-identical to
  today's output for atlas-configurations (verified: its enqueues store `{}`).
- **Equivalence claim, stated precisely**: same header *set* (keys and
  values). Header *order* is map-iteration order in both paths (headerFolder
  ranges a map too), so order was never guaranteed on the direct path and is
  not part of parity. The FR-2.2 test asserts set-equality of
  key→value against `produceHeaders(SpanHeaderDecorator(ctx), TenantHeaderDecorator(ctx))`.

### D3 — Publish ordering: switch drainer to `ORDER BY id ASC`

Fixes the intra-transaction tie described in §1.1(2) and makes FR-4.2's
documented guarantee ("single leader publishes in id order") true. `id` is
bigserial; within one tx, ids are assigned in insert order, so a flow's
enqueued sequence publishes in sequence. Caveat to document in the README:
across *concurrent* transactions id order is allocation order, not commit
order — same as today's enqueued_at behavior, and irrelevant to the per-flow
guarantee. Applies to `publishBatch` (drainer.go:220,224); backfill is audited
for the same pattern during implementation.

### D4 — Publisher promotion: straight move

`TopicWriterPool` + `NewTopicWriterPool` move verbatim to
`libs/atlas-outbox/publisher.go` (package `outbox`). atlas-configurations
deletes its copy, keeps its local `outbox` package solely for envelopes, and
its main.go swaps `outbox.NewTopicWriterPool()` →
`outboxlib.NewTopicWriterPool()`. No alias, no re-export (project convention).
The pool comment's "handful of topics" rationale is updated: per-service topic
counts stay small fleet-wide, so the design holds.

### D5 — CI guard: `tools/outboxguard` analyzer (in scope)

The PRD's open question §9 is resolved **yes**. Modeled exactly on
`tools/rediskeyguard` (go/analysis, `GOWORK=off`, shell wrapper iterating
service modules, wired next to redis-key-guard in CI):

- **Rule**: inside any function literal passed to
  `database.ExecuteTransaction(...)` or `(*gorm.DB).Transaction(...)`, a call
  to a function named `ProviderImpl` from a package named `producer` is a
  diagnostic ("direct Kafka producer inside a DB transaction — use
  outbox.EmitProvider").
- Lexical containment is sufficient: the fleet's only direct-producer entry
  point in service code is the local `producer.ProviderImpl`, and the
  migration itself removes every in-tx use, so the guard starts clean with no
  baseline file.
- Deliberately narrow (no taint tracking of MessageProducer values escaping
  the closure). The guard is a regression tripwire, not a proof; the audit
  inventory is the proof for the current tree.

### D6 — Per-service wiring template

Each migrating service's `main.go` mirrors atlas-configurations
(services/atlas-configurations/atlas.com/configurations/main.go:52-60):

1. add `outboxlib.Migration` to `database.SetMigrations(...)`;
2. `publisher := outboxlib.NewTopicWriterPool()`;
3. `drainer := outboxlib.NewDrainer(l, db, publisher, outboxlib.WithDSN(database.DSN()))`;
4. `go drainer.Run(ctx)`; `drainer.Stop()` + `publisher.Close()` on shutdown.

Lib defaults everywhere (poll 1s, batch 100, retention 7d) per the PRD's
default assumption; no per-service tuning until an operational signal says
otherwise.

### D7 — What migrates and what stays direct

Per service, the FR-3.1 inventory classifies every emit site:

- **Migrate**: any emit whose events assert a DB state change — inside
  `ExecuteTransaction`, immediately after one, or wrapping unwrapped
  `Save`/`Create`/`Update` writes (those get an explicit
  `ExecuteTransaction` wrapping mutation + enqueue; safe per §1.5
  re-entrancy).
- **Stay direct** (listed explicitly in the inventory, never silently):
  rejection/error status events reflecting *no* state change (e.g.
  `notEnoughMeso` — moved outside the tx closure per FR-1.3, emitted only on
  the no-change return path), pure relays/broadcasts, socket fan-out, ticker
  emissions without DB writes.

atlas-character is the reference implementation and lands first, including the
FR-1 defect fixes: check `dynamicUpdate` errors before enqueueing; replace the
two nil-`err` overflow returns with a real error (`errors.New("meso overflow")`
-shaped, logged as today); move `notEnoughMeso` emits outside the closures.

## 3. Data Flow (migrated flow, end to end)

1. Handler/consumer calls processor `MethodAndEmit`.
2. `ExecuteTransaction` opens tx → closure mutates domain rows via `tx` →
   `message.Emit(outbox.EmitProvider(l, ctx, tx))(fill)` runs the existing
   buffer-filling logic → per token: resolve topic, decorate headers from
   `ctx`, `Enqueue` rows (+ `pg_notify` per row, outbox.go:48-52).
3. Commit makes domain change + outbox rows atomic. Rollback discards both —
   zero events. (Rejection events, if any, are emitted directly after the tx
   returns its sentinel.)
4. Leader drainer (advisory lock) wakes on NOTIFY (≤~100ms) or 1s poll, reads
   unsent rows `ORDER BY id ASC` with `FOR UPDATE SKIP LOCKED`, re-attaches
   headers, publishes via `TopicWriterPool`, stamps `sent_at`; failures
   increment `attempts`/`last_error` and retry next tick (existing drainer
   semantics, drainer.go:213-270).

Consumers observe: same topic (env-resolved), same key/value bytes (Buffer
messages pass through untouched), same header set. Delivery becomes
at-least-once (crash between `WriteMessages` and the `sent_at` update
redelivers) — documented per FR-4; dedup is CD-1's follow-up.

## 4. Error Handling

- Token resolves to empty string → enqueue error → tx rolls back (the
  env-var fall-through in `topic.EnvProvider` returns the token itself, never
  empty; an explicitly empty env value is the error case `Enqueue` already
  rejects via its `Topic == ""` check).
- Header decorator failure → enqueue error → rollback.
- Any `Enqueue` row failure → error → rollback (FR-2.2 "fail the
  transaction").
- Publish failure → drainer records + retries; rows are never lost
  (existing).
- `EmitProvider` given a nil tx → `Enqueue`'s existing nil-tx error.

## 5. Testing Strategy

Lib (sqlite in-memory, matching existing standard; the drainer tests already
run against sqlite with the postgres-only branches gated on dialector):

- `EnqueueBuffer`: multi-token buffer → correct topics, keys/values
  byte-preserved, rows in insertion order by id.
- Header parity: enqueued header map set-equals
  `produceHeaders(SpanHeaderDecorator(ctx), TenantHeaderDecorator(ctx))` for a
  tenant-bearing ctx (FR-2.2 acceptance test).
- Drainer re-attach: enqueue with headers → published `kafka.Message.Headers`
  contains exactly the stored pairs; `{}` headers → nil/empty header slice.
- Ordering: two rows in one tx publish in id order.
- `EmitProvider`: used through a real `message.Emit`-shaped loop; error from
  `Enqueue` propagates.

Service (atlas-character reference):

- Failed `dynamicUpdate` → tx error, zero outbox rows, zero events (FR-1
  acceptance).
- Overflow branch → non-nil error.
- Rollback in a migrated flow → zero outbox rows; commit → exactly the
  expected rows (PRD acceptance #3), asserted via the outbox table.

Guard: `tools/outboxguard` gets an analysistest-style fixture (in-tx
ProviderImpl flagged; outside-tx clean), mirroring rediskeyguard's layout.

Fleet verification gates per CLAUDE.md: `go test -race`, `go vet`, `go build`
per changed module; `docker buildx bake all-go-services` (breadth makes
per-service bakes pointless); `tools/redis-key-guard.sh`; plus the new
`tools/outbox-guard.sh`.

## 6. Rollout Order

1. **Lib phase**: header re-attach (D2), id ordering (D3), publisher move
   (D4), `EnqueueBuffer` + `EmitProvider` (D1), README updates (FR-4:
   at-least-once, ordering guarantee + caveat, unsent-row growth as the
   wedged-drainer signal). atlas-configurations import swap lands here —
   it must compile unchanged otherwise (PRD §5).
2. **Reference**: atlas-character — FR-1 fixes + full FR-3 migration +
   inventory entry.
3. **Economy tier**: atlas-inventory, atlas-cashshop, atlas-fame.
4. **Standard tier**: buddies, guilds, notes, pets, mounts, skills, quest,
   merchant, npc-shops, gachapons, drop-information, data, tenants (the
   FR-3.1 sweep is authoritative; gachapons/drop-information currently show no
   producer usage and likely reduce to one-line inventory entries).
5. **Guard**: `tools/outboxguard` + `tools/outbox-guard.sh` + CI wiring, last
   (tree is clean by then; guard proves it stays clean).

Inventory doc: `docs/tasks/task-114-outbox-adoption/inventory.md`, one section
per service — sites migrated (file:line before/after), sites intentionally
left direct with reason, or "zero tx-coupled sites".

## 7. Risks & Non-Goals

- **At-least-once duplicates** on migrated topics until CD-1 lands consumer
  dedup — accepted and documented (PRD non-goal).
- **NOTIFY per enqueued row** adds one `pg_notify` per event inside the tx
  (existing `Enqueue` behavior). Notifications coalesce in the drainer's
  buffered channel; no design change, noted for awareness.
- **Latency**: enqueue→publish adds ≤~100ms steady-state (NOTIFY) —
  within saga timeout tolerances per PRD §8.
- **Buffer-less atlas-data** uses `EnqueueBuffer`/`Enqueue` directly at its
  (few) tx-coupled sites rather than adopting a Buffer it never had.
- Non-goals restated: no payload/schema/topic changes, no exactly-once, no
  consumer dedup, no atlas-ui changes, no migration of non-tx emit paths.
