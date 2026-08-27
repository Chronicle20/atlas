# Review — gift notice fix (bug round 2)

Range: `ae4042260..HEAD`, commits `7addf1508` (step 1/3, ordering) and
`b525249ed` (step 2, `SceneRefreshOwned` flag). Docs-only commits in between
excluded from scope.

Requirements: `bug-round-2-gift-notice-step2-ruling.md` (authoritative for
step 2), `bug-round-2-gift-notice-brief.md` (root cause, steps 1/3),
`bug-round-2-gift-notice-report.md` (implementer's claims).

## Verdict

APPROVED

## Findings

### 1. Wire contract — field declared identically on both sides

PASS.

- `services/atlas-cashshop/atlas.com/cashshop/kafka/message/wallet/kafka.go:44`
  — `SceneRefreshOwned bool `json:"sceneRefreshOwned,omitempty"``
- `services/atlas-channel/atlas.com/channel/kafka/message/wallet/kafka.go:28`
  — `SceneRefreshOwned bool `json:"sceneRefreshOwned,omitempty"``

Same field name, same JSON tag (`sceneRefreshOwned`, `omitempty`), same type
(`bool`), same doc comment verbatim on both sides. No tag mismatch.

### 2. Field is set ONLY on the gift path

PASS.

`grep -rn "SceneRefreshOwned"` across `services/` shows exactly one call site
that sets it to `true`:
`services/atlas-cashshop/atlas.com/cashshop/kafka/producer/wallet/producer.go:62`
inside `UpdateStatusEventWithTransactionSceneRefreshOwnedProvider`, invoked
only from `wallet/processor.go:177` (`UpdateWithTransactionSceneRefreshOwned`),
invoked only from `cashshop/processor_gift.go:131` (the gift sender debit).

Traced every other producer of a wallet `UPDATED` status event:

- `AdjustCurrencyWithTransaction` (`wallet/processor.go:201`) →
  `UpdateAndEmitWithTransaction` (`:186`) → still calls the *original*
  `UpdateStatusEventWithTransactionProvider` (`kafka/producer/wallet/producer.go:31`,
  unchanged), which never sets the flag (zero value `false`, and the struct
  literal at that provider does not reference the new field at all). This is
  the path used by `handleAdjustCurrencyCommand`
  (`kafka/consumer/wallet/consumer.go:50`) — the task-227 saga
  name-change/world-transfer flow, MTS auction settlement, and GM `@award`
  per the ruling's producer table. None of them were touched by this diff
  (`git diff --stat` confirms `kafka/consumer/wallet/consumer.go` and
  `saga-orchestrator`/`mts`/`messages` services are absent from the changed
  file list). They keep emitting the unflagged event and therefore keep
  their existing `CashQueryResult` refresh — matches the ruling's
  requirement and the report's claim in
  `bug-round-2-gift-notice-report.md:220-222`.
- `UpdateWithTransaction` itself (`wallet/processor.go:142`, the pre-existing
  method) is untouched and still calls the unflagged provider.

### 3. `CashSceneMts` arm untouched

PASS. `services/atlas-channel/atlas.com/channel/kafka/consumer/wallet/consumer.go:76-77`
— the `case session.CashSceneMts:` arm is a single unmodified line
(`return session.Announce(...)(fieldcb.MtsOperation2Writer)(...)`). The `if
e.Body.SceneRefreshOwned { return nil }` guard is added only inside `case
session.CashSceneCashShop:` (`:79-86`). Confirmed further by
`TestHandleWalletUpdatedMtsSceneUnaffectedBySceneRefreshOwned`
(`consumer_test.go:160-179`), which asserts `MtsOperation2Writer` fires for
both `SceneRefreshOwned` true and false.

### 4. Ordering contract

PASS.
`services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer.go:378-403`
(`handleStatusEventGiftPurchased`) now announces `CashShopOperationWriter`
(GIFT_SUCCESS) first, then fetches the wallet and announces
`CashQueryResultWriter`, in that order, mirroring the pre-existing
`handleStatusEventInventoryCapacityIncreased` (`:136-159`) shape exactly
(same fetch-with-fallback-on-error pattern, same log strings adapted).
`TestGiftPurchasedAnnouncesGiftDoneWithRecipientNameAndItem`
(`consumer_test.go:1414-1417`) asserts
`announcedWriters() == [CashShopOperationWriter, CashQueryResultWriter]` in
that exact order.

Test honesty confirmed: the pre-fix assertion (visible in the commit diff)
was `announcedWriters() == [CashShopOperationWriter]` only — a single-writer
assertion that the pre-fix handler (which announced only `GIFT_SUCCESS`)
satisfied, and that the post-fix handler (which announces two writers) would
fail. The new two-writer, ordered assertion fails against the pre-fix
handler and passes only with the fix; not a test that passes either way.

Cross-checked against the v83 IDB root-cause chain in the brief
(`CCashShop::OnQueryCashResult` drives `SendGiftsPacket`; `GIFT_SUCCESS` must
land first) — the implemented order matches the required order.

### 5. Rejected-heuristic pin

PASS.
`TestHandleWalletUpdatedNonNilTransactionIdWithoutFlagStillAnnounces`
(`services/atlas-channel/atlas.com/channel/kafka/consumer/wallet/consumer_test.go:143-158`)
sets a non-Nil `TransactionId` with `SceneRefreshOwned: false` and asserts
`CashQueryResultWriter` still announces. This is exactly the assertion the
ruling calls out to pin the rejection of the `TransactionId != uuid.Nil`
heuristic. Comment at `:139-142` states the intent explicitly. Not weak —
it directly exercises the discriminator the ruling says must NOT gate the
skip.

The full four-case suite in `consumer_test.go` also covers: flag set →
nothing announced (`:97-115`); flag unset → announces (`:117-131`); MTS
scene unaffected either way (`:160-179`). This matches the ruling's
`## Verification` section bullet-for-bullet.

### 6. `## Do not touch` held

PASS. `git diff ae4042260..HEAD --name-only | grep -iE
"giftdone|codec|seed|mode"` returns no matches. No changes to the
`GiftDone` codec, the `GIFT_SUCCESS` mode table, or any seed template.

### 7. Backend guidelines / repo conventions

PASS.

- Provider/processor idiom: `UpdateWithTransactionSceneRefreshOwned`
  (`wallet/processor.go:164-183`) is a sibling method with the identical
  curried signature shape as `UpdateWithTransaction`, not a bool parameter
  bolted onto the existing exported provider/method — matches the ruling's
  explicit instruction (step2-ruling.md:69-70) and avoids breaking existing
  call sites.
- No new domain type or numeric constant introduced (`SceneRefreshOwned` is
  a plain `bool` field); nothing that belongs in `libs/atlas-constants/`.
  Confirmed no matching type exists there (`grep -rn "SceneRefresh\|CashScene"
  libs/atlas-constants` — no output).
- `gofmt -l` on every file touched by both commits: no output (all clean).
- `go build ./...` and `go test ./...` pass in both
  `services/atlas-cashshop/atlas.com/cashshop` and
  `services/atlas-channel/atlas.com/channel` (targeted `./kafka/consumer/wallet/...
  ./kafka/consumer/cashshop/...` run for atlas-channel; full run for
  atlas-cashshop).
- Builder-pattern test setup: the new `consumer_test.go` uses a local
  `newConsumerEnv` helper following the existing pattern already used by
  `kafka/consumer/cashshop/consumer_test.go` in the same package tree, not a
  `*_testhelpers.go` file.

## Not evaluable

- Live re-test (gift an item, confirm "All the gifts have been sent…") was
  not run as part of this review — outside the module-local build/test
  surface this review covers, and the report explicitly defers it to the
  controller/verifier gate.
- `tools/lint.sh` (fix-mode formatting authority) was not run by the
  implementer per the report (`report.md:277-278`, "controller/verifier's
  gate per Contract 2"); this review ran `gofmt -l` only as a proxy, which
  is weaker than the repo's stated formatting authority. Not a blocking
  finding — `gofmt -l` came back clean and `tools/verify.sh` is a separate
  gate per this repo's CLAUDE.md, not this review's job to run.

## Scope confirmation

Reviewed exactly the two commits in the given range
(`7addf1508`, `b525249ed`) plus the wire-contract files they both touch on
each side of the service boundary. No scope mismatch: the diff matches the
work the report and ruling describe.

---

verdict: APPROVED
artifact: docs/tasks/task-240-cash-shop-stub-operations/review-bug-round-2-gift-notice.md
scope_confirmed: reviewed commits 7addf1508 and b525249ed (ae4042260..HEAD, docs-only commits excluded) — atlas-cashshop wallet message/producer/processor/gift-processor changes and atlas-channel wallet/cashshop consumer changes, both service sides of the SceneRefreshOwned wire contract, and the two commits' test additions/updates
blocking: 0
non_blocking: 0
not_evaluable: 2
