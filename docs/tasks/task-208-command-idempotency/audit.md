# Adversarial Audit — task-208 command idempotency

Scope: `07f01142ce635a851eac74560352662e313e41cc..7908d3375ae98bc474ee843d48ccfb25cad75481`
(`libs/atlas-database/idempotency.go` + wiring in atlas-inventory, atlas-cashshop, atlas-storage).

Default posture: FAIL until file:line evidence proves otherwise.

## Findings, most severe first

### 1. [BLOCKING] atlas-storage: guard now holds a live Postgres transaction open across a synchronous, retrying Kafka publish

`database.ApplyOnce`/`Once` (`libs/atlas-database/idempotency.go:73-95,114-127`) wraps the guarded work in a
**real** ambient transaction via `ExecuteTransaction` (`libs/atlas-database/transaction.go:9-16`, `db.Transaction(fn)`
when not already inside one). The three guarded atlas-storage handlers
(`services/atlas-storage/atlas.com/storage/kafka/consumer/compartment/consumer.go:56-58` for ACCEPT,
`:86-88` for RELEASE) pass the handler's root `db` (no pre-existing ambient transaction — confirmed by the
diff, which only wraps the existing call in `ApplyOnce`, `git diff` shown below) into `ApplyOnce`, so `Once`
genuinely opens a `BEGIN…COMMIT` around the whole guarded body for the first time.

Inside that now-open transaction, `AcceptAndEmit`/`ReleaseAndEmit`
(`services/atlas-storage/atlas.com/storage/storage/processor.go:345-396`) call
`emitCompartmentAcceptedEvent` / `emitCompartmentReleasedEvent` / `emitCompartmentErrorEvent`
(`storage/processor.go:403-450`), each of which calls `producer.ProviderImpl(...)` **directly** —
a live, synchronous Kafka write, not the transactional-outbox pattern. atlas-storage has no
`atlas-outbox` dependency anywhere (`grep -rn "outbox" services/atlas-storage/atlas.com/storage/storage/processor.go`
returns zero outbox hits; all 11 emit call sites are `producer.ProviderImpl`), unlike atlas-inventory
(`compartment/processor.go:1631`, `message.Emit(outbox.EmitProvider(...))`) and atlas-cashshop
(`cashshop/inventory/compartment/processor.go:146,176,208,244,252,292`, same outbox pattern).

`producer.ProviderImpl` bottoms out in `libs/atlas-kafka/producer/producer.go:60-64`:
```go
cfg := retry.DefaultConfig().WithMaxRetries(10).WithInitialDelay(100 * time.Millisecond).WithMaxDelay(10 * time.Second)
for _, m := range ms {
    err = retry.Try(context.Background(), cfg, tryMessage(l, w)(m))
```
— a synchronous, blocking retry loop: 10 attempts, 100ms initial delay, up to 10s max delay per attempt
(worst case tens of seconds), all executed **while the claim row insert and the asset
create/delete sit uncommitted** on a checked-out Postgres connection.

Before this diff, `storage.NewProcessor(l, ctx, db).AcceptAndEmit/ReleaseAndEmit` ran against the root
connection pool with no ambient transaction at all, so each GORM statement auto-committed individually —
the asset row was already durable by the time the (still-synchronous) Kafka call fired. This diff is what
newly wraps that whole sequence in one open transaction. Consequence:

- Under any Kafka degradation, every atlas-storage ACCEPT/RELEASE now holds a pooled Postgres connection
  for up to ~42s (worst case), a new failure mode this diff introduces and that can exhaust the storage
  pool during a broker hiccup that previously wouldn't have touched Postgres connection lifetime at all.
- Durability inversion: if the publish succeeds but the surrounding transaction subsequently fails to
  commit (dropped DB connection, pool timeout, etc. — a longer-held transaction is strictly more exposed
  to this), a RELEASED/ACCEPTED event has already gone out to downstream consumers (e.g.
  saga-orchestrator) while the DB mutation and the claim both roll back — an event announcing state that
  never became durable, and a subsequent legitimate retry will reprocess from scratch.

design.md's own risk section ("Known trade-off: lock ordering", `docs/tasks/task-208-command-idempotency/design.md:155-169`)
only analyzes atlas-inventory's Redis-lock-vs-transaction-duration tradeoff, and its Part 2 claims
"Kafka emission still happens outside the DB transaction (unchanged behavior)" (`design.md:152`) as a
blanket statement about all three services. That statement is true for atlas-inventory and atlas-cashshop
(outbox-based) but **false for atlas-storage**, and the doc does not analyze atlas-storage's live-publish
path at all. This is a materially worse and entirely undocumented risk compared to the one the design doc
does call out.

### 2. [IMPORTANT] atlas-storage RELEASE: pre-guard asset read defeats the guard on redelivery (confirms FOCUS AREA 6)

`handleReleaseCommand` (`services/atlas-storage/atlas.com/storage/kafka/consumer/compartment/consumer.go:70-97`)
reads the asset **before** calling the guard:
```go
assetModel, err := asset.GetById(db.WithContext(ctx))(uint32(c.Body.AssetId))   // line 77, unchanged by this diff
...
err = database.ApplyOnce(l, ctx, db, c.Body.TransactionId, compartment.CommandRelease, c.Body, func(tx *gorm.DB) error {
    return storage.NewProcessor(l, ctx, tx).ReleaseAndEmit(...)                  // line 86
})
```
Confirmed via `git diff` that line 77's read predates this branch — only the `ApplyOnce` wrap at line 86 was
added. `storage.Release` (`services/atlas-storage/atlas.com/storage/storage/processor.go:357-376`) **deletes**
the asset row for the common case (`quantity == 0 || !a.IsStackable()` or a full-quantity release, lines
366-372, `asset.Delete(...)`). On a redelivered RELEASE for the same transaction, the asset row is already
gone, so the line-77 `asset.GetById` fails first, logs an Errorf (`consumer.go:79`), and returns — **the
function never reaches `database.ApplyOnce` at all.**

Net effect: the double-release doesn't happen, but only by accident of an unrelated failure, not through the
guard. This directly contradicts the design doc's stated contract — "A duplicate logs at Info and returns
nil — the message is acked, not retried" (`design.md:150`) — for this specific handler: a duplicate RELEASE
in atlas-storage instead logs an Error on every routine redelivery, indistinguishable in the logs from a
genuine asset-lookup failure (e.g., a bad `AssetId` from a real bug), defeating the operational purpose of
adding the guard.

### 3. [IMPORTANT] Idempotency key omits CharacterId/InventoryType — collapses any same-transaction commands with byte-identical bodies, not just redeliveries

`Key` (`libs/atlas-database/idempotency.go:52-64`) hashes only `(transactionId, operation, payload)`. At every
guarded ACCEPT/RELEASE call site in atlas-inventory
(`services/atlas-inventory/atlas.com/inventory/kafka/consumer/compartment/consumer.go:225,305,324`) the
payload passed is `c.Body`, and `AcceptCommandBody`/`ReleaseCommandBody`/`CreateAssetCommandBody`
(`kafka/message/compartment/kafka.go:103-132`) carry **no** `CharacterId` or `InventoryType`, even though
`Command[E]` (`kafka.go:38-44`) carries both in the envelope alongside `Body`. Same gap in atlas-storage's
`AcceptCommandBody`/`ReleaseCommandBody` (`services/atlas-storage/atlas.com/storage/kafka/message/compartment/kafka.go:28-39`,
no `AccountId`/`CharacterId`).

`AcceptToCharacterPayload.TransactionId` is explicitly documented as the **saga**-wide transaction id
(`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go:917`, comment: `// Saga
transaction ID`), not a per-command id. design.md accepts this risk ("two byte-identical accepts inside one
transaction collapse into one," `design.md:70-72`) and defends it by inspecting the three current call sites
(`saga/processor.go:1386,1545,1745`), each of which emits exactly one `accept_to_character` step per saga
today (verified: `expandWithdrawFromStorage`/`expandWithdrawFromCashShop`/`expandWithdrawFromMts` each build
exactly one `AcceptToCharacterPayload`). But the guard's key does not structurally enforce "one accept per
transaction" — it is a fact about today's three call sites, not a property `Key` derives from the message.
Any future saga step, or any other producer sharing a saga's `transactionId`, that emits two structurally
distinct compartment commands (different character, different compartment) with an identical body would have
the second one silently dropped as a "duplicate" rather than applied — a correctness bug that would not
reproduce in any test targeting today's shapes. atlas-cashshop is comparatively safer here because its bodies
include `CompartmentId` (`services/atlas-cashshop/atlas.com/cashshop/kafka/message/cashshop/compartment/kafka.go:23,34`),
which is unique per character/compartment and would differentiate such cases; atlas-inventory and
atlas-storage have no equivalent field in the hashed payload.

### 4. [MINOR] Test coverage gaps

- **No concurrency test.** `databasetest.NewInMemoryTenantDB` (`libs/atlas-database/databasetest/testdb.go:22-44`)
  and the inventory-level test DB (`services/atlas-inventory/atlas.com/inventory/kafka/consumer/compartment/idempotency_test.go:47-68`)
  both cap the connection pool to a single connection (`testdb.go:37` `SetMaxOpenConns(1)`;
  `idempotency_test.go:54` `SetMaxIdleConns(1)`), so no test in this diff can exercise two real concurrent
  transactions racing the same claim key. The design doc's concurrency argument (`design.md:162-164` — the
  loser blocks on the `ON CONFLICT` insert until the winner commits, then sees `RowsAffected == 0`) is a
  correct description of Postgres unique-index insert semantics but is asserted, not verified by anything
  in this branch.
- **Only atlas-inventory has a handler-level idempotency test.** `services/atlas-inventory/.../kafka/consumer/compartment/idempotency_test.go`
  exercises the real `handleAcceptCommand` end to end; no equivalent file exists for atlas-storage or
  atlas-cashshop. A redelivery test against atlas-storage's real `handleReleaseCommand` (the same pattern
  already used for inventory's ACCEPT) would have caught finding #2 directly.
- **RELEASE is untested everywhere.** All committed idempotency tests (`idempotency_test.go` in
  `libs/atlas-database` and in atlas-inventory) exercise ACCEPT/generic `Once` semantics; no test drives a
  RELEASE redelivery through any of the three wired services.

## Verified correct (with evidence)

### FOCUS AREA 1 — transaction nesting: correct where verified

`ExecuteTransaction` (`libs/atlas-database/transaction.go:9-16`) detects an already-open transaction via
`isTransaction` (checks whether `Statement.ConnPool` implements `gorm.TxCommitter`, matching GORM's own
idiom) and calls `fn(db)` directly instead of opening a nested transaction or savepoint. A processor's own
`ExecuteTransaction` call therefore joins the transaction `Once` opened rather than nesting one inside it.
Verified end-to-end (not just at the library level) for atlas-inventory's ACCEPT path by
`TestAcceptCommandRedeliveryDoesNotDuplicateTheAsset` / `TestDistinctAcceptCommandsBothApply`
(`idempotency_test.go:110-147`), and by reading the call graph:
`compartment.NewProcessor(l, ctx, tx).AcceptAndEmit` → `database.ExecuteTransaction(p.db...)` (already `tx`,
joins) → `Accept(mb)(...)` → another `database.ExecuteTransaction(p.db...)` (still `tx`, joins again)
(`services/atlas-inventory/atlas.com/inventory/compartment/processor.go:1629-1648`). Same pattern confirmed
for atlas-cashshop (`cashshop/inventory/compartment/processor.go:145,175,207,217,243,251,291`, all
`database.ExecuteTransaction`, outbox emission via `message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))` joining
the same `tx`).

### FOCUS AREA 4 — multi-tenancy: correct

`IdempotencyEntity` uses composite primary key `(TenantId, Key)` (`idempotency.go:31-36`). `Once` derives
`TenantId` from `tenant.FromContext(ctx)()` and hard-fails (does not silently proceed unscoped) when no
tenant is present (`idempotency.go:74-77`), verified by `TestOnceRequiresATenant` and
`TestOnceIsScopedPerTenant` (`idempotency_test.go:68-82,99-108`). The sweeper's use of
`WithoutTenantFilter` (`idempotency.go:143`) is the documented, correct escape hatch for a cross-tenant
maintenance job (retention sweep), not a violation of per-request tenant scoping.

### FOCUS AREA 5 — ApplyOnce's unguarded fallback: acceptable, and effectively unreachable at current call sites

The fallback (`idempotency.go:114-119`) applies the work unguarded and logs at Error level when
`json.Marshal(payload)` fails. Every payload type passed at the three services' call sites
(`AcceptCommandBody`, `ReleaseCommandBody`, `CreateAssetCommandBody` — `kafka.go:103-132` in atlas-inventory,
equivalents in atlas-storage/atlas-cashshop) consists only of plain JSON-marshalable fields (uuid.UUID,
numeric types, `time.Time`, `*time.Time`, strings) — none can fail `json.Marshal`. The fallback is real,
correctly reasoned (losing the item is worse than risking a duplicate under the same rare conditions that
already existed pre-branch), and dead code at every currently-wired call site — not a defect.

## Summary

### Blocking (must fix)
- **Finding #1** — atlas-storage's ACCEPT/RELEASE guard now holds an open Postgres transaction across a
  synchronous, up-to-~42s-retrying Kafka publish (`storage/processor.go:403-450` via
  `producer/producer.go:60-64`), a new failure mode not present before this diff and not analyzed anywhere
  in design.md. Needs either: move atlas-storage's compartment status events onto the transactional outbox
  (matching atlas-inventory/atlas-cashshop), or restructure the guard so the claim commits before the live
  publish runs.

### Non-blocking (should fix)
- **Finding #2** — atlas-storage RELEASE's pre-guard `asset.GetById` (`consumer.go:77`) bypasses the
  idempotency guard on redelivery instead of hitting the intended "Info: duplicate skipped" path; move the
  inventory-type lookup inside the guarded closure (before the delete) or derive it from the command body.
- **Finding #3** — the idempotency key's omission of CharacterId/InventoryType/AccountId is a latent
  landmine that relies on today's saga shapes rather than the key itself; consider adding those fields to
  the hashed payload (or the `Key(...)` call) for the ACCEPT/RELEASE call sites in atlas-inventory and
  atlas-storage.
- **Finding #4** — add a concurrency test (even a synthetic one using two contending transactions with a
  larger sqlite connection pool, or a Postgres-backed test if available), a RELEASE redelivery test, and an
  atlas-storage/atlas-cashshop equivalent of the atlas-inventory `idempotency_test.go`.

---

## Resolution (post-review)

Reviewed by `backend-guidelines-reviewer` and a general-purpose reviewer, run in
parallel against `07f01142c..7908d3375`. Their findings and what was done:

| # | Finding | Resolution |
|---|---|---|
| 1 | **BLOCKING** — atlas-storage publishes directly to Kafka (no outbox), so wrapping its handlers put a blocking, retrying network call inside the claim transaction: pool exhaustion risk plus a durability inversion. | **Fixed.** Added `AcceptOnceAndEmit`/`ReleaseOnceAndEmit` to the storage processor; the claim now covers only the DB write and emission stays outside the transaction. Verified independently before acting: `services/atlas-storage/atlas.com/storage/go.mod` has no `atlas-outbox`, and `storage/processor.go:417` emits via `producer.ProviderImpl`. |
| 2 | atlas-storage RELEASE read the asset before the guard, so redelivery failed the lookup and logged an Error; the guard was unreached and double-release prevented only by accident. | **Fixed.** The source read moved inside the claim. The handler's remaining pre-read serves only the projection refresh and now treats "already gone" as a duplicate at Info level. |
| 3 | The key hashed only `(transactionId, operation, body)`; bodies omit character/account, so two distinct commands in one saga transaction could collide and the second be silently dropped. | **Fixed.** Inventory and cashshop now hash the whole command envelope. Storage hashes an explicit `acceptClaim`/`releaseClaim` carrying world, account, character and body. |
| 4 | Test coverage asymmetric — only inventory ACCEPT had a handler-level test. | **Partly fixed.** Added inventory RELEASE (handler-level) and three storage tests (`AcceptOnceAndEmit` redelivery + distinct, `ReleaseOnceAndEmit` redelivery). See the gap below. |

### Remaining gap, accepted

- **CREATE_ASSET has no handler-level test.** Its processor fetches equipment
  stats from atlas-data over HTTP, so the handler cannot run without
  data-service mocks. An attempted test failed at the create, not at the guard.
  The guard mechanism itself is covered by the `libs/atlas-database` suite and
  by the two inventory handler tests that use the identical call shape.
- **No concurrency test.** Both harnesses cap the sqlite pool at one connection,
  so a genuine two-writer race cannot be expressed; the serialization guarantee
  comes from postgres `ON CONFLICT`, not from anything this suite can exercise.
- The general reviewer's point that CREATE_ASSET's collapse risk was not audited
  the way ACCEPT was still stands as a reasoning gap, now narrowed by the
  envelope-wide key. No currently reachable path emits two byte-identical
  CREATE_ASSET commands under one transaction.
