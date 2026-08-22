# Review — Task 14: seed-status consumer, destroy saga, main.go wiring

Range: `50c79fadf..0777d508c` (one commit).
Brief: `.superpowers/sdd/plan/task-14-brief.md` (body + controller addendum session 4).
Report: `.superpowers/sdd/plan/task-14-report.md`.

## Scope confirmed

`git diff --stat 50c79fadf..0777d508c`:

```
libs/atlas-saga/model.go                                                            |   6 +
services/atlas-channel/atlas.com/channel/kafka/consumer/seed/consumer.go            | 236 ++++++
services/atlas-channel/atlas.com/channel/kafka/consumer/seed/consumer_test.go       | 507 ++++++++
services/atlas-channel/atlas.com/channel/kafka/message/seed/kafka.go                |  28 +
services/atlas-channel/atlas.com/channel/main.go                                    |   8 +
services/atlas-channel/atlas.com/channel/saga/model.go                              |   1 +
services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go           |   5 +
services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/timer.go           |   2 +
8 files changed, 793 insertions(+)
```

Matches the brief's file list exactly. Confirmed no changes under `libs/atlas-packet`,
`docs/packets/`, `services/atlas-login/`, `services/atlas-character-factory/`, `deploy/`
(`git diff --stat` against those paths is empty, exit 1/no output). Confirmed
`socket/handler/maple_life_create.go` is untouched (empty diff) and no `v84`/`SelectedAL`
string appears anywhere in the range's diff. Scope matches the assignment exactly.

## 1. Two-character-id hazard — PASS, verified by mutation, not by trusting the test

`consumer.go:162`:

```go
destroySaga := buildDestroyCashItemSaga(entry.CharacterId, byte(inventory.TypeValueCash), int16(entry.Slot), uint32(entry.ItemId))
```

`entry` comes from `resolveEntry` → `maplelife.GetRegistry().TakeByTransactionId`/`Take`, i.e.
the registry `Entry.CharacterId` (the submitter). `e.Body.CharacterId` (the newly created
character) is used only in log fields (`created_char_id`) at `consumer.go:157,170,188` and in
the `createdEvent(...)` test helper — never passed to `buildDestroyCashItemSaga`.

Test seeding is non-vacuous: `testSubmitCharId = 42` (seeded into `Entry.CharacterId` via
`putSubmittedEntry`, `consumer_test.go:36,146`) and `testCreatedCharId = 99` (event
`Body.CharacterId`, `consumer_test.go:37,200`) are distinct, and
`assertDestroySagaCorrect` (`consumer_test.go:220`) asserts
`p.CharacterId != testSubmitCharId` fails with a message naming the created id it must not equal.

I additionally ran a mutation myself rather than trusting the passing suite: temporarily
swapped `entry.CharacterId` → `e.Body.CharacterId` in `buildDestroyCashItemSaga`'s call site
and reran `TestSeedCreatedConsumesItemAndAnnounces`. Result: `matched_by_transaction_id` and
`fallback_by_account_id` both failed with `CharacterId = 99, want 42 (the SUBMITTING
character, not the created one [99])`. Reverted with `git checkout` immediately after;
working tree confirmed clean. The swap-check is real, not incidental.

## 2. Destroy-saga confinement (FR-5.1) — PASS

Read every path through both handlers:

- `handleFailedStatusEvent` (`consumer.go:197-236`) never references `destroyCashItemFunc` or
  `buildDestroyCashItemSaga` at all — structurally zero destroy sagas on `FAILED`, not merely
  untested. Confirmed no destroy-related call in the function body.
- `handleCreatedStatusEvent`: a transaction-id mismatch (`resolveEntry` returns `ok=false` for
  the wrong transaction id — `TakeByTransactionId` returns `false` because no entry's
  `TransactionId` matches) hits the `!ok` early return at `consumer.go:153-160`, before
  `buildDestroyCashItemSaga` is ever reached.
- Wrong tenant: `t.Is(sc.Tenant())` guard at `consumer.go:147-150` returns before any registry
  lookup or saga construction.
- Duplicate delivery: `Take`/`TakeByTransactionId` (`registry.go:107-138`) both delete the map
  entry under the same lock that reads it, so the second delivery's `resolveEntry` call finds
  nothing and takes the `!ok` path — the safety is structural (mutex + delete-under-lock), not
  incidental ordering, exactly as the assignment required me to confirm.

All four negative paths are asserted by tests (`consumer_test.go`
`wrong_transaction_id`/`wrong_tenant`/`duplicate_delivery` subtests, and
`TestSeedFailedLeavesItemAndAnnounces`), and I reran the full `kafka/consumer/seed` suite
fresh — all pass.

## 3. Transaction-id resolution rule — PASS

`resolveEntry` (`consumer.go:125-131`):

```go
func resolveEntry(t tenant.Model, accountId uint32, transactionId string) (uint32, maplelife.Entry, bool) {
	if transactionId != "" {
		return maplelife.GetRegistry().TakeByTransactionId(t, transactionId)
	}
	e, ok := maplelife.GetRegistry().Take(t, accountId)
	return accountId, e, ok
}
```

A non-empty `transactionId` goes to `TakeByTransactionId` only; if that returns `false` (no
match), the function returns `false` directly — there is no fallback branch to `Take` by
account id in this code path. Only an *empty* `transactionId` falls back to `Take`. This
matches the rule exactly: "present but matches nothing" is a mismatch, not a fallback.

Test: `wrong transaction id` subtest (`consumer_test.go:286-311`) sends `TransactionId:
"tx-other"` against an entry seeded with `TransactionId: "tx-1"`, asserts zero destroy calls,
zero announces, the entry still present (`env.entryExists()`), and a warning-level log —
i.e. it observes the entry surviving, as required.

## 4. Tenant scoping — PASS

`t.Is(sc.Tenant())` guard present in both handlers (`consumer.go:147-150`, `:203-206`),
identical in form to the established `cashshop` consumer pattern
(`kafka/consumer/cashshop/consumer.go:111-116`: `t := tenant.MustFromContext(ctx); if !t.Is(sc.Tenant()) { return }`).
The `wrong tenant` subtest (`consumer_test.go:313-333`) constructs a genuine second
`tenant.Model` via `mustTenant` with its own UUID, dispatches under that tenant's context
against tenant A's server model/session, and asserts zero destroy calls, zero announces, and
tenant A's entry intact — a real two-tenant exercise, not a same-tenant no-op.

## 5. Saga classification — PASS, ran the test myself

`libs/atlas-saga/model.go` adds `MapleLifeUse Type = "maple_life_use"`. Both
`noReverseWalkSagaTypes` and `allSagaTypes` in
`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/timer.go` gained
`MapleLifeUse` (lines 253+1, 264+1 in the diff). Ran:

```
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./saga/... -run TestEverySagaTypeIsClassified -v
--- PASS: TestEverySagaTypeIsClassified (0.00s)
```

The step used is `saga.DestroyAssetFromSlot` (`consumer.go:151` in `buildDestroyCashItemSaga`),
structurally identical to `character_cash_item_use.go:626-637`'s `consume_sacrifice` step
(same payload shape: `CharacterId`, `InventoryType`, `Slot`, `Quantity: 1`, `TemplateId`).
`RequestItemConsume` is not used anywhere in the diff (grep confirms no reference).
`msb.MapleLifeUseHandle` does not exist anywhere in the tree (`grep -rn
"MapleLifeUseHandle"` — zero hits), matching addendum §3.

## 6. Disconnected-session and destroy-failure paths — PASS

Disconnected session (`TestSeedCreatedWithDisconnectedSessionStillConsumes`,
`consumer_test.go:436-461`): no `connectSession()` call; asserts `destroyCalls == 1` (saga
still created), `announced == 0` (no client write attempted), and an info-level log present.
Matches `consumer.go:176-190`'s `found := false` / `IfPresentByAccountId` / `!found` branch,
which is reached only after the destroy call at line 163 — destroy happens unconditionally of
session presence.

Destroy failure (`TestSeedCreatedDestroyFailureIsLoggedNotRolledBack`,
`consumer_test.go:463-507`): `destroyCashItemFunc` returns an error; asserts an error-level
log entry carrying `account_id`, `submitting_char_id`, `created_char_id`, `item_id`,
`submit_transaction_id` (all present at `consumer.go:167-173`), and that the SUCCESS arm is
still announced (the character genuinely exists). Grepped the whole `kafka/consumer/seed`
package for a character-delete call — none exists; the absence of a compensating rollback is
structural, matching design §5.4.

## 7. `main.go` wiring — PASS, all three corrections verified

- `mlcb.MapleLifeResultWriter` / `mlcb.MapleLifeErrorWriter` added to `produceWriters()`
  (`main.go` diff, lines 690-691 area) beside `cashcb.CashShopCheckNameChangePossibleResultWriter`.
  Grepped `main.go` — both present exactly once.
- `handlerMap[msb.MapleLifeCheckNameHandle]` appears exactly once, at `main.go:1015`
  (pre-existing, from Task 12) — not duplicated by this diff (`git diff ... | grep msb.` is
  empty — the diff never touches an `msb.` line).
- `msb.MapleLifeUseHandle` does not exist anywhere (`grep -rn "MapleLifeUseHandle"` across
  `services/atlas-channel/` and `libs/atlas-packet/` — zero hits) and was correctly not added.

`InitHandlers(sc server.Model, ...)` deviation from the brief's abstract `(ten tenant.Model)`
line: checked against `cashshop.InitHandlers` (`kafka/consumer/cashshop/consumer.go:62`) —
identical signature shape (`sc server.Model` → `wp writer.Producer` → `rf ...`). The tenant
guard is not dropped: `handleCreatedStatusEvent`/`handleFailedStatusEvent` still derive
`t := tenant.MustFromContext(ctx)` and check `t.Is(sc.Tenant())` inline, exactly matching
`cashshop`'s own `handleStatusEventInventoryCapacityIncreased` (`cashshop/consumer.go:116-119`).
The deviation is a correct match to established convention, and the brief's own concrete
Step 5 line (`register(seedConsumer.InitHandlers(fl)(sc)(wp)(rh))`) already resolves the
ambiguity in the implementer's favor. Not a defect.

## Build/test verification (fresh, not from the report)

```
cd libs/atlas-saga && go build ./... && go test ./...            -> ok (cached, matches report)
cd services/atlas-saga-orchestrator/.../saga-orchestrator &&
  go test ./saga/... -run TestEverySagaTypeIsClassified -v        -> PASS
cd services/atlas-channel/atlas.com/channel &&
  go build ./... && go test ./kafka/consumer/seed/... -v          -> all PASS
```

Mutation test (character-id swap) — see §1 — confirms the suite is non-vacuous on the one
subtle correctness point this task exists to get right.

## Findings

None blocking. No non-blocking notes beyond what's already covered above.

## Not evaluable

None — every item in the assigned checklist was traceable within the diff's surface plus the
one contract file it depends on (`maplelife/registry.go`, read for `Take`/`TakeByTransactionId`
locking semantics) and the one comparison file used to confirm the `InitHandlers` idiom
(`kafka/consumer/cashshop/consumer.go`, an existing sibling pattern the brief itself pointed at).
