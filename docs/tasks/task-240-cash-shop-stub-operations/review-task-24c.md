# Review — Task 24c: defer equip-slot extension write behind the outbox

Commit reviewed: `bcceb3784` (range `2db96b673..bcceb3784`), diffed against
its parent only. `ring.go` and other out-of-range commits (`2db96b673`,
`a294a1e2f`) were not evaluated.

## Scope confirmation

`git diff --stat 2db96b673..bcceb3784` touches exactly 16 files across
`atlas-cashshop` and `atlas-character` — the equip-slot purchase processor,
its REST client to atlas-character, the new outbox command message/producer/
consumer triple, and atlas-character's equip-slot entity/administrator/
processor/rest/resource layers, plus their tests. `services/.../cashshop/ring.go`
has zero diff (confirmed via `git diff --stat -- '*ring.go*'` returning empty)
— no scope drift. This matches the brief's file inventory exactly. Scope
confirmed as claimed.

## 1. Ordering — does the extension only happen after durable commit?

PASS. `PurchaseEquipSlotAndEmit`
(`services/atlas-cashshop/atlas.com/cashshop/cashshop/equipslot.go:60-133`)
no longer calls `p.chaP.ExtendEquipSlot` inside the transaction closure.
Instead, step 6 (`equipslot.go:129`) does
`buf.Put(cashshop.EnvCommandTopic, cashshop2.ExtendEquipSlotCommandProvider(...))`
inside the same `message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))` closure
that wraps the wallet debit and `purchaserecord.Record`. Read
`libs/atlas-outbox/provider.go:21-29`: `EmitProvider`'s `MessageProducer`
persists the message as an outbox row via `EnqueueBuffer(l, ctx, tx, ...)` —
i.e. inside the SAME `tx` as the debit/record — and the outbox README
(`libs/atlas-outbox/README.md:184` onward) confirms "the drainer publishes
after the transaction commits." So the `EXTEND_EQUIP_SLOT` command cannot be
durably queued unless the whole purchase transaction (debit + record)
commits first, and the actual `ExtendEquipSlot` HTTP call only happens in
`CompleteEquipSlotExtension`, invoked by a separate Kafka consumer
(`kafka/consumer/cashshop/consumer.go:274-291`) reacting to that command.
Ordering holds by construction, not by convention.

## 2. Idempotency — the substance of the task

PASS, with a genuinely persisted, wire-driven dedupe key.

- `transactionId` is threaded end-to-end: outbox command body
  (`kafka/message/cashshop/kafka.go` `ExtendEquipSlotCommandBody`) →
  `character.ExtendEquipSlotInputRestModel.TransactionId`
  (`services/atlas-cashshop/atlas.com/cashshop/character/equipslot.go:48-58`)
  → `equipslot.ExtendInputRestModel.TransactionId`
  (`services/atlas-character/atlas.com/character/equipslot/rest.go:44-51`) →
  `handleExtendEquipSlot` (`resource.go:64`) → `Extend(..., input.TransactionId)`.
- The dedupe check is **persisted**, not in-memory:
  `equipslot.Entity.TransactionId` is a new NOT NULL column
  (`entity.go:32`), and `Extend`
  (`services/atlas-character/atlas.com/character/equipslot/administrator.go:24-56`)
  reads the existing row inside its own `db.Transaction`, and if
  `transactionId != uuid.Nil && e.TransactionId == transactionId`, returns
  the current `ExpiresAt` as a no-op — the OnConflict upsert also persists
  `transaction_id` on every write (`DoUpdates: ...{"expires_at",
  "transaction_id", "updated_at"}`). This survives a restart; it is genuinely
  a database guard, not an in-process guard.
- Verified this is actually exercised through the real route, not around it:
  `TestPostExtendEquipSlot_RedeliveredTransactionIdDoesNotDoubleExtend`
  (`services/atlas-character/atlas.com/character/equipslot/resource_test.go:99-146`)
  POSTs the same `ExtendInputRestModel{...,TransactionId}` twice through
  `router.ServeHTTP`, asserts both return 200, only one row exists, and
  `ExpiresAt` is still `+30d` (not `+60d`) — this is a wire-level test, not a
  domain-level one, so it genuinely proves the transaction id survives JSON
  marshal/unmarshal into the dedupe check. A companion domain-level test
  (`administrator_test.go`, `TestExtend/a_repeated_transaction_id_does_not_double-extend`
  and `.../a_genuinely_new_transaction_id_still_extends`) additionally proves
  the guard is keyed on the id specifically, not "any second call."
- `uuid.Nil` is the documented "no key supplied" sentinel and always
  proceeds, preserving pre-24c call semantics for the (only theoretical,
  since there are no other callers) case where no id is supplied.

The cashshop-side "redelivered command" test
(`equipslot_test.go`, `TestCompleteEquipSlotExtension/a_redelivered_command_does_not_double-extend`)
is honestly scoped — it asserts the route is called *twice* and explicitly
comments the actual guard lives on atlas-character's side. This is
transparent, not a false-coverage claim.

## 3(a). Post-commit HTTP failure logged and dropped — is the precedent real?

PASS — precedent verified as real and structural, not just conventional.
`libs/atlas-kafka/message/handler.go`'s `Handler[M] func(l, ctx, m)` has **no
error return type at all**; `AdaptHandler`'s wrapper (`handler.go:48-73`)
calls `config.handler(l, ctx, m)` and always returns
`(config.persistent, nil)` regardless of what the handler logged internally
— i.e. the offset commits unconditionally. Grepping every other
`handleCommand*` function in
`services/atlas-cashshop/atlas.com/cashshop/kafka/consumer/cashshop/consumer.go`
(`handleCommandExpire`, `handleCommandRequestLockerRebate`,
`handleCommandRequestGiftPurchase`, `handleCommandRequestPackagePurchase`,
`handleCommandRequestRingPurchase`, `handleCommandRequestEquipSlotIncrease`)
shows every one of them does exactly `l.WithError(err).Errorf(...)` and
returns — log-and-drop is the uniform pattern across this file, enforced by
the type system (there is no error channel to retry on), not an
implementer's convention chosen for this task alone. Given the charge has
already durably committed by the time `CompleteEquipSlotExtension` runs,
dropping (vs. an unbounded blocking retry loop, which no other handler in
this codebase does either) is a reasonable, consistent choice. Acceptable.

## 3(b). `Entity.TransactionId` GORM default-value tag — correct for Postgres?

PASS — verified beyond the report's self-flagged sqlite-only uncertainty, by
reading `gorm.io/gorm@v1.31.2/schema/field.go` and
`gorm.io/driver/postgres@v1.6.2/postgres.go` directly:

- `uuid.UUID` implements `driver.Valuer` and its `Value()`
  (`github.com/google/uuid@v1.6.0/sql.go:52-54`) returns `uuid.String()` — a
  string, always, never raw bytes.
- `schema/field.go:140-146`: when a field implements `driver.Valuer` (and
  not `GormDataTypeInterface`, which `uuid.UUID` does not), GORM
  re-derives `fieldValue` from calling `Value()` *before* the
  `reflect.Array`/`Bytes` branch is reached. So the type-detection switch at
  `field.go:227` (`case reflect.String:`) is what actually fires for
  `uuid.UUID` fields, not the `Array`/`Bytes` branch — meaning this column
  (and every pre-existing `uuid.UUID` column in this codebase lacking
  `type:uuid`, e.g. `equipslot.Entity.Id`/`TenantId` themselves) resolves to
  `field.DataType = String`, and `postgres.go`'s `getSchemaBaseType` maps
  `schema.String` → `text` (no `SIZE` tag set).
- This is the *exact same code path* `pending_change/entity.go:73`'s cited
  precedent (`Reason string`, `default:''`) goes through — both are
  `schema.String` fields with a quoted-literal `DEFAULT` tag. The precedent
  is genuinely analogous, not superficially similar.
- The literal itself, `'00000000-0000-0000-0000-000000000000'`, is exactly
  `uuid.Nil.String()` — the same string `Value()` would produce when Go code
  writes a zero-value `uuid.UUID` through this same column — so existing
  rows backfilled by this `ALTER TABLE ... DEFAULT` clause round-trip
  correctly through `Scan` as `uuid.Nil`, matching the "no dedupe key
  supplied" sentinel the `Extend` logic relies on.

No defect found; the report's residual doubt is resolved by static analysis
of the actual GORM/postgres-driver code path, not just an appeal to
precedent.

## 4. `ring.go` scope

PASS — confirmed untouched (`git diff --stat -- '*ring.go*'` is empty).

## 5. No placeholders / stubs / unimplemented handlers

PASS — `CompleteEquipSlotExtension` is fully implemented (performs the HTTP
call, emits the success event, logs and returns the error on failure); no
`TODO`, no stub response, no unimplemented status path introduced.

## Build / test / format

- `cd services/atlas-cashshop/atlas.com/cashshop && go build ./... && go test ./...`
  — all packages `ok`.
- `cd services/atlas-character/atlas.com/character && go build ./... && go test ./...`
  — all packages `ok`.
- `tools/lint.sh --check --fmt --go services/atlas-cashshop` — `OK`.
- `tools/lint.sh --check --fmt --go services/atlas-character` — `OK`.
  (Both checked with the authoritative `gofumpt`-backed gate per this
  branch's known `gofmt`-vs-`gofumpt` drift; no discrepancy found this time.)

## Cross-service seam trace

`atlas-cashshop` → mints `EXTEND_EQUIP_SLOT` on `COMMAND_TOPIC_CASH_SHOP`
(same topic the service already consumes its own `REQUEST_*` commands on,
per the brief's "find the existing pattern" instruction) → consumed by
`handleCommandExtendEquipSlot` in the SAME service →
`CompleteEquipSlotExtension` → REST write to `atlas-character` →
`atlas-character`'s route dedupes on `transactionId`, persisted. Every hop
in this chain is covered by a test that drives the real boundary (outbox
row assertion for the mint side, `router.ServeHTTP` for the atlas-character
write side) rather than asserting behavior in-process only.

## Non-blocking notes

- `CompleteEquipSlotExtension`'s failure path leaves a charged-but-
  unextended character with no automated reconciliation path (self-flagged
  by the report as a deliberate trade-off, not an oversight). This is a
  legitimate follow-up candidate (a genuine retry/reconciliation mechanism)
  but is consistent with every other post-commit side effect in this
  codebase today and is not a regression this task introduces — noted, not
  blocking.

## Not evaluable

None — the full unit (both services' diffs, the outbox library's delivery
contract, and the GORM/postgres type-resolution path the new column
depends on) was read and traced directly; nothing in scope was left
unverified.

## Verdict

APPROVED. No blocking findings. The ordering fix is real (outbox-gated,
verified against the library's actual commit semantics), the idempotency
key is genuinely persisted and exercised through the real wire routes on
both ends, both self-flagged report items check out as correct rather than
merely plausible, and `ring.go` was left untouched as required.
