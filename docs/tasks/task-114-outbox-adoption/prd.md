# Fleet-Wide Transactional Outbox Adoption — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-02
---

## 1. Overview

Atlas services persist domain state to Postgres and announce changes over Kafka, but the two writes are not atomic. `libs/atlas-outbox` is a complete transactional outbox (enqueue-in-tx, drainer with Postgres advisory-lock leadership, NOTIFY wakeup, retention sweeper, backfill) yet is adopted by exactly one service, atlas-configurations. Every other service uses a service-local `message.Buffer` + `message.Emit(producer)` pattern, which is **batching, not atomicity**: a crash between DB commit and `Emit` silently loses events, and a multi-topic buffer can partially publish.

Verified hot paths are worse than the general gap: `services/atlas-character/atlas.com/character/character/processor.go` emits `MESO_CHANGED` + `STAT_CHANGED` **inside** `database.ExecuteTransaction` before commit (`RequestChangeMeso` lines ~749-751, `AttemptMesoPickUp` line ~768), so a rollback after emit produces phantom events downstream. The same blocks assign `err = dynamicUpdate(...)` and never check it before emitting — a failed meso write still announces success.

This task closes CD-2 (docs/architectural-improvements.md) in full: fix the mid-transaction emit sites, promote the missing shared plumbing into `libs/atlas-outbox`, and migrate **every** service whose event emissions are coupled to DB mutations onto the outbox. After this task, "DB commit happened but the event was lost" and "event was published but the tx rolled back" are both structurally impossible for transactional flows fleet-wide.

## 2. Goals

Primary goals:
- No service emits a Kafka event from inside an open DB transaction via the direct producer path; tx-coupled events are persisted as outbox rows in the same transaction as the domain mutation.
- All ~17 remaining transactional services (see §7) run the outbox drainer and publish tx-coupled events through it.
- Shared, reusable adoption plumbing in `libs/atlas-outbox` (publisher + buffer bridge) so a per-service migration is mechanical: no service hand-rolls marshaling, header decoration, or topic resolution.
- The atlas-character unchecked `dynamicUpdate` error paths are fixed (error checked before any event is enqueued).
- Events published via the outbox are byte-equivalent to the current `Emit` path from a consumer's perspective: same topic, key, value, and headers (tenant + span).

Non-goals:
- Consumer-side idempotency / inbox-dedup on `TransactionId` (CD-1). Outbox delivery is at-least-once; consumer dedup is tracked as its own follow-up task.
- Migrating emits that have no DB transaction coupling (pure relay/broadcast flows, socket-event fan-out, ticker emissions with no DB write). These stay on `message.Buffer` + `Emit` / direct producer.
- Exactly-once delivery semantics.
- atlas-ui changes.
- Changing event payload schemas or topics.

## 3. User Stories

- As a game operator, I want a character's meso/stat change event to exist **iff** the DB change committed, so that downstream projections (channel, UI, saga orchestrator) never act on phantom or missing state.
- As a service developer, I want to enqueue events inside my existing GORM transaction with one call that handles topic resolution and tenant/span headers, so that adopting the outbox does not require understanding kafka-go internals.
- As an SRE, I want event relay to survive a pod crash between commit and publish, so that a restart replays unsent rows instead of losing them.
- As a future task implementer (CD-1), I want every tx-coupled event to flow through a single well-defined path, so that dedup can be added in one place.

## 4. Functional Requirements

### FR-1. atlas-character hot-path fix

1. `RequestChangeMeso`, `AttemptMesoPickUp`, and `RequestDropMeso` (`character/processor.go`) must not call `producer.ProviderImpl(...)` inside `database.ExecuteTransaction`. Success events (`MESO_CHANGED`, `STAT_CHANGED`) are enqueued via `outbox.Enqueue` within the transaction; they publish only after commit.
2. The `err = dynamicUpdate(tx)(SetMeso(...))(c)` result must be checked; on error the transaction returns the error and no event is enqueued. (Both the `RequestChangeMeso` and `AttemptMesoPickUp` occurrences.)
3. The "not enough meso" rejection events (`notEnoughMesoErrorStatusEventProvider`) reflect no state change; they may remain direct emits, but must not be emitted from inside the transaction closure. Move them outside the tx (no behavioral constraint on their timing was identified).

### FR-2. Shared plumbing in `libs/atlas-outbox`

1. **Publisher promotion**: the `TopicWriterPool` publisher currently local to atlas-configurations (`atlas.com/configurations/outbox/publisher.go`) moves into `libs/atlas-outbox` (straight move, not a re-export alias, per project convention). atlas-configurations switches to the lib version and deletes its local copy.
2. **Buffer bridge**: a lib function that persists an entire `message.Buffer`-shaped payload (`map[string][]kafka.Message`, keyed by **env-var token**) as outbox rows inside a caller-supplied `*gorm.DB` transaction. It must:
   - resolve each env-var token to the real topic name (same resolution the producer `ManagerWriterProvider` path uses) — outbox rows store real topic names;
   - apply the same header decoration as the direct path (`producer.SpanHeaderDecorator(ctx)`, `producer.TenantHeaderDecorator(ctx)`) before persisting, since buffered `kafka.Message`s carry no tenant/span headers;
   - preserve message key and value bytes unchanged;
   - fail the transaction if any row cannot be persisted.
3. **Emit-in-tx entry point**: an `Emit`-shaped helper (and an `EmitWithResult`-shaped one) that runs the caller's buffer-filling closure and flushes the buffer to the outbox within a given tx, so migrated processor code keeps its current structure (`message.Emit(p)(func(buf) …)` → `outbox`-flavored equivalent taking `ctx` + `tx`).
4. Behavior of existing lib features (drainer leadership, NOTIFY, sweeper, backfill, migration) is unchanged; new code is covered by unit tests at the lib's existing standard (Sqlite in-memory, no Docker).

### FR-3. Per-service migration

For **every** service in the affected list (§7):

1. Enumerate all emit sites coupled to a DB mutation — both patterns:
   - emits inside or immediately after `database.ExecuteTransaction` / `db.Transaction`;
   - `message.Emit` / `EmitWithResult` wrapping a function that performs DB writes (including single-statement `Save`/`Create`/`Update` not currently wrapped in an explicit transaction).
2. Migrate each such site: the DB mutation and the outbox enqueue occur in one transaction. Mutations not currently in an explicit transaction are wrapped in one together with the enqueue.
3. Wire the drainer in `main.go` (mirroring atlas-configurations): `outboxlib.Migration` added to `database.SetMigrations(...)`, drainer constructed with the shared publisher and `WithDSN(database.DSN())`, run in a goroutine, stopped on shutdown.
4. Emit sites with **no** DB coupling in that service are left as-is and listed in the migration inventory as explicitly out of scope (no silent skips).
5. The migration inventory (per service: sites migrated, sites intentionally left) is committed under `docs/tasks/task-114-outbox-adoption/` as the audit record.

### FR-4. Delivery-order and duplicate documentation

1. `libs/atlas-outbox/README.md` (and service docs where they exist) reflect that migrated topics are now at-least-once with possible redelivery; the CD-1 follow-up owns consumer dedup.
2. Ordering guarantee documented: single drainer leader publishes in `id` order per service, preserving per-service emission order; cross-service ordering is unchanged (never guaranteed).

## 5. API Surface

No REST API changes. New/changed Go surface (final shapes are a design-phase decision; capabilities are fixed):

- `libs/atlas-outbox`: exported `TopicWriterPool` (moved), a buffer-flush/enqueue-batch function taking `(ctx, tx, bufferContents)`, and `Emit`/`EmitWithResult`-shaped wrappers that compose the existing service-local `message.Buffer` with a tx-scoped outbox flush.
- Existing `Enqueue(tx, Message)`, `NewDrainer`, `Migration`, `Backfill` signatures unchanged (atlas-configurations must compile without changes beyond the publisher import swap).

Error cases: enqueue failure fails the enclosing tx (mutation rolls back); topic-token resolution failure fails the enqueue (never publishes to an empty/wrong topic); publish failure is retried by the drainer with `attempts`/`last_error` recorded (existing behavior).

## 6. Data Model

- Each migrating service's database gains the existing `outbox_entries` table via `outboxlib.Migration` (bigserial id, topic, key, nullable value, jsonb headers, enqueued_at, sent_at, attempts, last_error). No schema changes to the lib's table.
- The table is intentionally **not** tenant-scoped (README-documented); tenancy rides in the persisted headers. Reads from tenant-aware services use `database.WithoutTenantFilter(ctx)` — already handled inside the lib.
- No changes to any domain tables. No data backfill required: pre-migration events were published synchronously; cutover starts with an empty outbox.

## 7. Service Impact

Known transactional services (from `ExecuteTransaction` usage; the FR-3.1 inventory is the authoritative final list and must also sweep DB-writing services that emit without explicit transactions):

| Service | Impact |
|---|---|
| atlas-character | FR-1 hot-path fix + full migration (reference implementation, done first) |
| atlas-inventory, atlas-cashshop, atlas-fame | Economy-integrity tier — migrated immediately after atlas-character |
| atlas-buddies, atlas-guilds, atlas-notes, atlas-pets, atlas-mounts, atlas-skills, atlas-quest, atlas-merchant, atlas-npc-shops, atlas-gachapons, atlas-drop-information, atlas-data, atlas-tenants | Standard migration per FR-3 |
| atlas-configurations | Publisher import swap only (already on outbox) |
| libs/atlas-outbox | FR-2 additions |

Services whose inventory finds zero tx-coupled emit sites get a one-line inventory entry and no code change.

## 8. Non-Functional Requirements

- **Latency**: NOTIFY-driven relay keeps enqueue→publish under ~100ms steady-state (lib's existing behavior); poll interval fallback 1s. No user-visible gameplay regression expected; the saga orchestrator's timeouts (base + per-step) already tolerate this.
- **Multi-tenancy**: tenant headers on outbox-published messages are byte-identical to the direct-producer path; a consumer cannot distinguish the two. This is test-asserted in the lib (FR-2.2).
- **Replica safety**: drainer leadership via advisory lock means services at `replicas: 2` publish exactly one stream; no replica pinning needed.
- **Observability**: drainer logs publish failures with `attempts`/`last_error` (existing). Unsent-row growth is the operational signal for a wedged drainer; note this in the lib README.
- **Verification gates** (per CLAUDE.md): `go test -race ./...`, `go vet ./...`, `go build ./...` clean in every changed module; `docker buildx bake` for every touched service (in practice `all-go-services` given the breadth); `tools/redis-key-guard.sh` clean.

## 9. Open Questions

- Should a CI guard (à la `tools/redis-key-guard.sh`) ban `producer.ProviderImpl`/direct producer calls inside `ExecuteTransaction` closures to prevent regression? Recommended, but not committed scope — decide at design time.
- Whether any service's emit volume warrants non-default drainer tuning (`WithBatchSize`, `WithRetention`). Default assumption: lib defaults everywhere.

## 10. Acceptance Criteria

- [ ] No `producer.ProviderImpl(...)` (or equivalent direct producer) call executes inside a `database.ExecuteTransaction` / `db.Transaction` closure anywhere in `services/` — verified by sweep, cited in the inventory doc.
- [ ] atlas-character `RequestChangeMeso` / `AttemptMesoPickUp` check the `dynamicUpdate` error; a failed update emits nothing (unit-tested).
- [ ] A rollback in any migrated flow produces zero Kafka events; a commit produces exactly the events the flow enqueued (lib + service tests demonstrate both).
- [ ] `TopicWriterPool` lives in `libs/atlas-outbox`; atlas-configurations uses it; its local `outbox/publisher.go` is deleted.
- [ ] Buffer bridge resolves env-token→topic and applies tenant/span header decoration; a test asserts header parity with the direct producer path.
- [ ] Every §7 service: outbox migration registered, drainer wired in `main.go`, tx-coupled emit sites migrated, inventory entry committed (including explicit "left as direct emit" lists).
- [ ] Migration inventory document committed under `docs/tasks/task-114-outbox-adoption/`.
- [ ] All CLAUDE.md verification gates pass (test -race / vet / build / bake / redis-key-guard) for every changed module.
