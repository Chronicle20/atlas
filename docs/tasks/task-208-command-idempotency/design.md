# task-208 — Idempotent compartment commands

## Problem

On `atlas-main`, character 12 withdrew cash item template **1812000 (Meso Magnet)**
from the cash shop and received **two** copies in the EQUIP compartment.

Traced chain (2026-08-10 13:23, tenant `ec876921-…`, saga `45da5862-…`):

| Time | Where | Fact |
|---|---|---|
| 13:23:26.991 | atlas-saga-orchestrator | `Progressing saga step [accept_to_character]` — emitted **once**, no retry, no version conflict |
| — | Kafka `COMMAND_TOPIC_COMPARTMENT-main` | `kafka-console-consumer --from-beginning` finds **exactly 1** message for that transactionId |
| 13:23:27.045 | atlas-inventory `…-ts4sq` | ACCEPT consumed → `assetId 257`, slot 4 |
| 13:23:29.256 | atlas-inventory `…-ts4sq` | **same message consumed again** → `assetId 258`, slot 5 |

So: one produce, two deliveries, two rows. Two independent defects.

### Defect 1 — write handlers are not idempotent

`services/atlas-inventory/.../kafka/consumer/compartment/consumer.go:254`
(`handleAcceptCommand`) unconditionally builds an asset and calls
`AcceptAndEmit`. Nothing keys off `TransactionId`. The same shape exists for
`RELEASE` and `CREATE_ASSET`, and in every other compartment holder
(atlas-cashshop, atlas-storage, atlas-merchant, atlas-mts).

The delivery layer is explicitly at-least-once — `libs/atlas-kafka/consumer/manager.go:571`
("Could not commit message offset, it may be redelivered") and the outbox
drainer's publish-then-mark (`libs/atlas-outbox/drainer.go:249`). Handlers that
create durable state are therefore *required* to be idempotent, and these are not.

### Defect 2 — the consumer watchdog recreates readers that were never broken

`readerMadeProgress` (`libs/atlas-kafka/consumer/manager.go:65`) classifies a
reader as healthy when `Stats()` shows `Fetches|Dials|Messages > 0`. A group
member that holds **no partition assignment** issues no fetches at all, so it is
indistinguishable from a stalled reader. Every topic in the cluster is
single-partition, so the second replica of every service is permanently
unassigned:

```
Inventory Service  COMMAND_TOPIC_COMPARTMENT-main  0  …  CONSUMER-ID …-ts4sq
```

`…-dsh9j` owns nothing, wedges after 3 × 60s no-progress ticks, and recreates —
**attempt #435** at 13:23:12, ~15s before the dupe. Each recreate rejoins the
group and forces a rebalance, which is when the active replica's offset commit
is lost and the message is redelivered.

Defect 2 is what makes defect 1 fire routinely rather than never.

**Defect 2 is NOT fixed on this branch.** It is recorded here because it is the
trigger that turned a latent bug into a live one, and the evidence above is worth
keeping — but the fix is being done separately and properly. Nothing in
`libs/atlas-kafka` is touched by this task.

Consequence to keep in mind: until that separate task lands, redelivery remains
frequent, so the guard added here is load-bearing rather than belt-and-braces.

## Decisions

Taken with the requester:

1. **Scope:** all compartment holders — atlas-inventory, atlas-cashshop,
   atlas-storage, atlas-merchant, atlas-mts. Same latent bug in each.
2. **Idempotency key:** `transactionId + command + payload hash`, recorded in a
   table with a UNIQUE constraint, inserted **inside the same DB transaction** as
   the mutation. Catches redelivery *and* true producer-side duplicates.
   Accepted trade-off: two byte-identical accepts inside one transaction collapse
   into one. No current saga expansion emits that shape — the three
   `AcceptToCharacter` sites (`saga/processor.go:1386,1545,1745`) each emit one
   accept per transaction.

## Design

### Part 1 — `database.Once` in libs/atlas-database

New home rather than a new module: every affected service already depends on
`libs/atlas-database` for GORM/postgres, and this is a persistence concern.

```go
// idempotency_keys, composite PK (tenant_id, key)
type IdempotencyEntity struct {
    TenantId  uuid.UUID `gorm:"primaryKey"`
    Key       string    `gorm:"primaryKey"`
    Operation string
    CreatedAt time.Time `gorm:"index"`
}

// Key derives the stable key: sha256 over transactionId, operation and the
// JSON encoding of the command body.
func Key(transactionId uuid.UUID, operation string, payload any) (string, error)

// Once runs fn exactly once per (tenant, key). On a repeat it skips fn and
// returns ErrDuplicate. The marker insert and fn share one transaction, so a
// rollback un-claims the key. The tenant comes from context, matching the
// tenant callbacks that scope every other query in this lib.
func Once(ctx context.Context, db *gorm.DB, key, operation string,
    fn func(tx *gorm.DB) error) error

// ApplyOnce is the handler front door: derives the key, turns a repeat into a
// logged no-op, and — if the key cannot be derived — applies the work
// unguarded rather than dropping the item.
func ApplyOnce(l logrus.FieldLogger, ctx context.Context, db *gorm.DB,
    transactionId uuid.UUID, operation string, payload any,
    fn func(tx *gorm.DB) error) error
```

Claim is `INSERT … ON CONFLICT DO NOTHING`; `RowsAffected == 0` means an earlier
delivery already applied it. Because `database.ExecuteTransaction` joins an
existing transaction, the processor's own `ExecuteTransaction` nests into the
same one — the marker and the asset row commit or roll back together.

Growth is bounded by `SweepIdempotency(ctx, db, retention)` plus
`StartIdempotencySweeper`, mirroring the outbox sweeper (default retention 7d,
sweep hourly, leader-agnostic — the delete is idempotent, so every replica may
run it).

### Part 2 — wire the guard into the compartment holders

Surveying the five candidates narrowed the actual work to three. The other two
were verified, not assumed:

| Service | Handler | Outcome |
|---|---|---|
| atlas-inventory | `CREATE_ASSET`, `ACCEPT`, `RELEASE` | **guarded** |
| atlas-cashshop | `ACCEPT`, `RELEASE` (cash compartment) | **guarded** |
| atlas-storage | `ACCEPT`, `RELEASE` | **guarded** |
| atlas-cashshop | `CREATE` (`COMMAND_TOPIC_CASH_ITEM`) | **left unguarded** — see below |
| atlas-mts | `ACCEPT_TO_MTS_LISTING`, `RELEASE_FROM_MTS_HOLDING` | already idempotent (task-114): accept short-circuits on an existing `ListingId` (`listing/processor_custody.go:90`), release is a soft-delete keyed by `HoldingId` |
| atlas-merchant | compartment status events | no durable writes — the handlers only log (`kafka/consumer/compartment/consumer.go:34-58`). It *produces* ACCEPT/RELEASE, which the guarded inventory consumer now dedupes |

`COMMAND_TOPIC_CASH_ITEM`'s CREATE command carries no transaction id — the
envelope is `{characterId, type, body}` with a body of
`{templateId, commodityId, quantity, purchasedBy}` — so two legitimate identical
purchases are byte-identical and a payload-only key would wrongly collapse them.
No producer for this topic exists anywhere in the repo, so there is nothing to
derive an identity from yet. Guarding it correctly requires adding a transaction
id at the producer, which does not exist to change.

For atlas-inventory and atlas-cashshop, wrapping happens at the consumer-handler
boundary, so no processor API changes:

```go
_ = database.ApplyOnce(l, ctx, db, transactionId, compartment2.CommandAccept, c.Body,
    func(tx *gorm.DB) error {
        return compartment.NewProcessor(l, ctx, tx).AcceptAndEmit(transactionId, …)
    })
```

A duplicate logs at Info and returns nil — the message is acked, not retried.

A duplicate delivery re-announces nothing, because `fn` never runs.

**atlas-storage is different, and the difference matters.** It has no
`atlas-outbox` dependency: `AcceptAndEmit`/`ReleaseAndEmit` publish straight to
Kafka via `producer.ProviderImpl`, a blocking write that retries with backoff
(worst case tens of seconds). Wrapping those at the handler boundary would put
that network call inside the claim transaction, so a broker hiccup would pin a
pooled postgres connection — and the event would fire before the commit was
guaranteed. So atlas-storage instead grows `AcceptOnceAndEmit` /
`ReleaseOnceAndEmit` on its processor: the claim covers only the DB write, and
emission stays exactly where it was, outside the transaction.

The same restructure fixes a second wart. `handleReleaseCommand` read the source
asset *before* the guard, so a redelivery (asset already gone) failed the lookup
and logged an Error before ever reaching the claim — the double-release was
prevented by accident. The read now happens inside the claim; the handler's
remaining pre-read exists only to learn the inventory type for the projection
refresh, and treats "already gone" as an ordinary duplicate at Info level.

### Key composition

The hash covers the **whole command envelope**, not just the body. Several
bodies (`AcceptCommandBody`, `ReleaseCommandBody`) carry neither the character
nor the account even though the envelope does, so hashing the body alone would
let two structurally distinct commands sharing one saga transaction collide on a
single key — and the loser would be silently dropped. atlas-storage, whose
processor never sees the envelope, hashes an explicit
`acceptClaim`/`releaseClaim` struct carrying world, account, character and body.

## Known trade-off: lock ordering

`atlas-inventory`'s `Accept` takes a Redis per-`(character, inventoryType)` lock
and *then* opens its DB transaction. Wrapping the handler in `Once` inverts that
for the guarded paths: the claim transaction is open while `Accept` waits on the
Redis lock.

This does not deadlock — the two lock domains cannot form a cycle. A waiter
holds only its own claim row, and two deliveries contending for the *same* claim
key serialize on the `ON CONFLICT` insert, with the loser doing no work at all.
The cost is transaction duration: a contended character now holds a pooled
connection for the length of the Redis lock wait. The lock is short and scoped
to one character's one compartment, so this is accepted rather than restructured;
moving the claim inside the processor would push idempotency into the domain
layer for no correctness gain.

## Out of scope

- Removing the surplus replicas / raising partition counts. That is a deployment
  decision, not a code fix, and the watchdog must be correct regardless.
- The duplicate `assetId 258` already in character 12's inventory on
  `atlas-main` — cleaned up separately as live data, not by this branch.
