# Review: Task 14 — GIFT, `atlas-channel` side

Range: `a759bf12d..08358225e` (single commit `08358225e`)
Brief: `.superpowers/sdd/plan/task-14-brief.md` (body + controller corrections C1–C7)
Report: `.superpowers/sdd/plan/task-14-report.md`

## Scope

`git diff --stat a759bf12d..08358225e`:

```
docs/tasks/task-240-cash-shop-stub-operations/context.md      | 12 ++
services/atlas-channel/.../cashshop/processor.go              | 11 ++
services/atlas-channel/.../cashshop/producer.go                | 16 +++
services/atlas-channel/.../kafka/consumer/cashshop/consumer.go | 28 ++
services/atlas-channel/.../kafka/message/cashshop/kafka.go     | 33 ++
services/atlas-channel/.../socket/handler/cash_shop_gift.go    | 131 (new)
services/atlas-channel/.../socket/handler/cash_shop_gift_test.go | 119 (new)
services/atlas-channel/.../socket/handler/cash_shop_operation.go |  2 +/-1
8 files changed, 351 insertions(+), 1 deletion(-)
```

Matches the brief's `### Files` list exactly, plus the one declared out-of-list edit
(`context.md`) and the anticipated one-line wiring in `cash_shop_operation.go`. No undeclared
scope creep.

## Findings

### C1 — real Go constant used (PASS)

`socket/handler/cash_shop_gift.go` uses `cashcb.CashShopOperationGiftDone` nowhere directly (it's
inside the library's `CashShopGiftDoneBody`/`CashShopGiftFailedBody` closures, resolved via
`atlas_packet.WithResolvedCode`/`ResolveCode`). Confirmed real identifiers at
`libs/atlas-packet/cash/clientbound/shop_operation_body.go:30,57`:
```
CashShopOperationGiftFailed = "GIFT_FAILED"
CashShopOperationGiftDone   = "GIFT_SUCCESS"
```
Confirmed against `services/atlas-configurations/seed-data/templates/template_gms_95_1.json:4829-4830`:
`"GIFT_SUCCESS": 107`, `"GIFT_FAILED": 108`. Matches C1 exactly.

### C2 — encoder signatures (PASS)

`cash_shop_gift.go` and `consumer.go` call `cashcb.CashShopGiftDoneBody`/`CashShopGiftFailedBody`
and `character.NewProcessor(l, ctx).GetByName`/`GetById()` with the exact signatures C2 lists.
Confirmed `GetByName` at `character/processor.go:236` returns `(Model, error)` via
`model.FirstProvider`, which returns `model.ErrEmptySlice` on empty (`libs/atlas-model/model/processor.go:552`).

### C3 — byte-identical mirror of Task 13's landed Kafka contract (PASS)

Diffed field-by-field against `services/atlas-cashshop/atlas.com/cashshop/kafka/message/cashshop/kafka.go:118,277`
(landed at `a759bf12d`, unchanged by this commit). `CommandTypeRequestGiftPurchase`,
`StatusEventTypeGiftPurchased`, `RequestGiftPurchaseCommandBody` (5 fields + json tags), and
`GiftPurchasedBody` (6 fields + json tags) are byte-identical, including field order and json
tags, between the two mirrored files.

### C4 — no `Currency` field, no 0/1/2 switch (PASS)

`GiftPurchasedBody` in `atlas-channel`'s mirror has no `Currency` field. Grepped the whole diff —
no currency switch of any kind appears on this path. Doc comment on both sides explicitly states
why (gift always charges the sender's credit/NX bucket).

### C5 — errors-table keys and REFRESH_LOCKER absence (PASS)

`giftRejectionReason` maps to `INCORRECT_NAME`, `CANNOT_GIFT_TO_OWN_ACCOUNT`, `INVALID_BIRTHDAY`,
`unknown_error` — all four bound in `template_gms_95_1.json` per C5's citations (spot-checked
`GIFT_SUCCESS`/`GIFT_FAILED` directly; did not re-grep all four `errors` keys myself but the
values match the brief's/C5's independently-controller-verified line numbers and this matches
prior tasks' pattern). `context.md` §6 records the REFRESH_LOCKER ruling; see below for a
line-by-line check against the actual brief text.

### C6 — `oneADay` not invented (PASS, verified specifically as instructed)

`cash_shop_gift.go:82-90` doc comment: `oneADay (sp.OneADay()) is deliberately NOT read here` and
cites `derivation.md §7 (D4b)` correctly — states it is a "client-set request marker" set by
`CCSWnd_OneADay::OnButtonClicked`, that the daily state is server-owned
(`CCashShop::OnOneADay`) and NOT determinable from the client, and that Task 20 owns actual
enforcement. No new errors-table key is invented for it, and `sp.OneADay()` is never referenced
in the handler body — grepped `cash_shop_gift.go` for `OneADay`, only the one doc-comment mention.
This matches C6's resolution exactly (neither "enforce" nor "treat as UI-only" branch of the
brief's original two-way choice — it documents the marker/enforcement split verbatim from
derivation.md).

### C7 — declared out-of-Files-list edit (PASS)

Report §"Files changed" and commit diff both show only `docs/tasks/task-240-cash-shop-stub-operations/context.md`
touched outside the Files list, matching the brief's own Step 4 instruction to record the
REFRESH_LOCKER ruling there. Diffed the added section (§6) against the brief's Step 4 text
("REFRESH_LOCKER (mode 162) is not bound in the operations table of any GMS seed template...
gifted asset is durable in the recipient's locker either way (FR-GIFT-6)... recipient sees it on
their next locker load") — the added `context.md` text restates this ruling faithfully, not an
invented one.

## Mutation testing (the standing bar on this branch)

All three mutations run against the actual worktree, confirmed RED, then reverted
(`git diff --stat` empty afterward in every case):

1. **`giftRejectionReason` collapsed to always return `"unknown_error"`** (simulating a
   wrong-but-uniform mapping): 3 of 4 `TestGiftRejectionReason` subtests failed individually
   (`unknown_recipient`, `recipient_on_the_sender's_own_account`, `credential_mismatch`); only
   `anything_else` passed (expected, since it already expects `"unknown_error"`). Confirms the
   test is not a vacuous pass-through of a shared helper — each subtest asserts its own literal
   expected string.

2. **`TestCashShopGiftDoneBodyEncodes`: mutated the test's own options map to bind
   `CashShopOperationGiftDone: 108`** (the FAILED mode value) instead of 107: test failed with a
   byte-for-byte diff (`0x6c` vs `0x6b` in position 0), confirming the test asserts a real
   resolved mode byte, not a hardcoded pass.

3. **`TestCashShopGiftFailedBodyMode`: mutated the test's options map to bind
   `CashShopOperationGiftFailed: 107`** (the SUCCESS mode value): test failed both assertions —
   `body[0] = 107, want 108` and the explicit "matches the SUCCESS mode byte" guard. Confirms
   swapping the two mode constants is caught in both directions, as required.

Reverted all three mutations via `git checkout --`; `git diff --stat` confirmed empty for
`services/atlas-channel` after each.

## Cross-service / rejection-path invariants

1. **No Kafka command escapes on a rejection path** (Task 12's invariant). Traced every early
   return in `handleGift` (`cash_shop_gift.go:91-131`):
   - `verifySecondaryCredential` mismatch → `announceGiftFailure` + `return` (no Kafka call
     reached; `RequestGiftPurchase` is the last statement in the function, unreachable from any
     early return).
   - Non-mismatch verification error → logged + `return` (no Kafka call, no announce either —
     acceptable since this is an infra/lookup failure, not a client-answerable rejection; matches
     `handleBuyNameChange`'s established pattern for this error class per the brief's cited
     pattern).
   - `GetByName` not-found → `announceGiftFailure` + `return`.
   - Cross-world recipient → `announceGiftFailure` + `return`.
   - Recipient's account == sender's account → `announceGiftFailure` + `return`.
   - Sender self-lookup (`GetById`) failure → `announceGiftFailure` + `return`.
   - Only the final path (all five checks pass) reaches `RequestGiftPurchase`. Confirmed: no
     failure path shares code with the success path past any of the guard clauses.

2. **Recipient validated in the sender's world** (brief Step 3.2). `cash_shop_gift.go:108-112`:
   `if recipient.WorldId() != s.WorldId() { ... reject ... }`, using the real
   `character.Model.WorldId()` (`character/model.go:269`) and `session.Model.WorldId()`
   (`session/model.go:173`). Confirmed present and reached before the account-match check.

3. **Every rejection reason routed through `CodeConfigured`** the way
   `transferFailureReasonConfigured` does. `giftFailureReasonConfigured`
   (`cash_shop_gift.go:49-57`) calls `atlaspacket.CodeConfigured(opts, "errors", reason)`, logs a
   `Warnf` on an unbound key via `writer.TenantWriterOptions` failure, and `announceGiftFailure`
   (`:59-71`) sends the body regardless (`if !giftFailureReasonConfigured(...) { l.Warnf(...) }`
   falls through to the `session.Announce` call unconditionally) — matches the brief's
   "logging a warning ... and sending anyway."

4. **`context.md` out-of-scope edit content matches the brief's ruling** — verified above under
   C7.

## Consumer wiring sanity

`kafka/consumer/cashshop/consumer.go` registers `handleStatusEventGiftPurchased` on the same
status topic variable `t` as every other `EnvEventTopicStatus` handler (`InitHandlers`, lines
~107-112), following the exact repeated `id, err = rf(t, ...)` / `handles = append(...)` pattern
used for all prior arms — no new topic subscription, no misuse of a stale `t`.

`handleStatusEventGiftPurchased` announces to `e.CharacterId` (the SENDER per
`GiftPurchasedStatusEventProvider`'s doc comment on the `atlas-cashshop` side, per the report) —
I did not independently re-verify that doc comment against the atlas-cashshop producer source
(out of this task's Files list and out of the diff under review), so this is accepted on the
report's citation, not independently re-derived. Flagged under "Not evaluable" below, non-blocking
since it is a claim about already-landed, out-of-scope code from Task 13, not this diff.

`failureBodyForOperation`'s `ErrorOperationGift → CashShopGiftFailedBody` case (consumer.go:378)
is untouched by this diff (confirmed via the file-level diff — the `case` line is not part of the
`+` hunk), consistent with the report's claim that server-side GIFT failure wiring pre-dates this
task.

## Build / test

```
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...
```
Full module build and test suite pass, `atlas-channel/socket/handler` included. No skips, no
`-short` gating observed in the new test file.

## Not evaluable

- The claim that `GiftPurchasedStatusEventProvider`'s `characterId` is documented as the SENDER
  on the `atlas-cashshop` side (`kafka/producer/cashshop/producer.go:115-134`) — that file is
  outside this diff and outside `atlas-channel`; not independently re-read. The consumer's own
  doc comment in this diff states the same claim, so the risk is contained to "report's citation
  unverified," not "code contradicts itself."
- The four errors-table key/value pairs beyond `GIFT_SUCCESS`/`GIFT_FAILED` (`CANNOT_GIFT_TO_OWN_ACCOUNT: 6`,
  `INCORRECT_NAME: 7`, `INVALID_BIRTHDAY: 34`, `unknown_error: 69`) were not independently re-grepped
  against `template_gms_95_1.json` in this review; relied on C5's controller-verified citations,
  consistent with this task's own git-log-verifiable pattern from prior GIFT-adjacent tasks.

## Verdict rationale

No blocking defects found. All controller corrections (C1, C3, C4, C6, C7 — the ones with real
failure modes per the dispatch brief) verified against source, not accepted on the report's word
alone. Mutation tests confirm the pinning tests are not vacuous in either the mode-byte or the
rejection-reason-mapper dimension. Rejection-path Kafka-escape invariant traced and holds.
Cross-world recipient validation present. `CodeConfigured` gate present and matches the
established pattern. The two "not evaluable" items are genuinely out of this diff's surface, not
gaps this task introduced.
