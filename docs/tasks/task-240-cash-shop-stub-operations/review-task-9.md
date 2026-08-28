# Review: Task 9 — BUY_NORMAL (commit 0d239bf36)

Range reviewed: `3cc3c8962..0d239bf36` (single commit).
Reviewed against `.superpowers/sdd/plan/task-9-brief.md` **as corrected by its
C1-C4 controller corrections**, `.superpowers/sdd/plan/task-9-report.md`, and
`docs/tasks/task-240-cash-shop-stub-operations/derivation.md` (confirmed: no
BUY_NORMAL entry, so C4's "derivation wins" clause never triggers).

## Scope confirmation

`git diff --stat 3cc3c8962..0d239bf36` — 14 files, 307 insertions / 37
deletions, exactly the C2-corrected file inventory (both `kafka/message/cashshop/kafka.go`
mirrors, both `consumer.go`/`producer.go`/`processor.go` pairs, the handler,
the new test file, `cash_shop_credential.go`). No `libs/atlas-packet/` file
touched, no `docs/tasks/` file touched. Scope matches the corrected brief.

## Findings

### 1. BLOCKING — `SlotPos: 0` is an invented value, not a documented default; it is client-constrained AND a real, available replacement value was ignored

The implementer's report and the code comment both call `SlotPos: 0` a
"documented default... no derivation source... assigns this list entry any
other position." That claim is false on two independent grounds:

**(a) The field is constrained by IDA-derived evidence already in this repo.**
`libs/atlas-packet/cash/clientbound/shop_operation_result_gift.go:57-62`
documents `PackedCashItemRef.SlotPos` as "Bit layout resolved from raw
disassembly... NOT from the (unreliable, mis-scaled) Hex-Rays pseudocode
pointer arithmetic — see task-0.3f report arms 2/3." That report is
`docs/tasks/task-183-cashshop-result-family/arm-catalog.md:51`, which states
for `BUY_NORMAL_SUCCESS` specifically:

> slotPos offset 2 (u16, `movzx ebp, word ptr [esi]`@0x495394, **passed as
> `nPos` to `CCSWnd_Inventory::SetSelectedNo`**@0x4953d2)

This is not a cosmetic/unused field for this exact arm — the client passes it
to a UI call that selects a slot in the cash inventory window. A hard-coded
`0` will cause the client to always select/highlight the first slot in the
cash-shop window regardless of where the purchased item actually landed,
which is observable client behavior, not a harmless placeholder.

**(b) A real, non-invented value was available in scope and not used.**
The consumer already holds the fetched asset, `a`, at
`services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer.go:156`
(`a, err := asset.NewProcessor(l, ctx).GetById(...)`), and the same `a` is
used two lines away for `a.Item().Quantity()`/`a.Item().TemplateId()` in the
new `PackedCashItemRef` literal (`:205-207`). `asset.Model` exposes
`Slot() int16` (`services/atlas-channel/atlas.com/channel/asset/model.go:66`).
The implementer did not attempt to read `a.Slot()` into the new ref; it wrote
a bare `0` instead, one line away from `a.Item()...` calls on the same
variable.

Given (a) a verified IDA-derived read order constrains this field's client
use, and (b) a real derivable value existed in scope and was bypassed for a
literal `0`, this is an invented value wearing a "documented default"
comment, which the plan's Global Constraints forbid. **Per the task
instructions, this is BLOCKING.**

### 2. BLOCKING — the new tests cannot fail the actual cross-service wiring change (TDD concern is real, not just a process nit)

I mechanically verified this rather than trusting the implementer's report.
I removed the entire new `if e.Body.Operation == cashshop2.ErrorOperationBuyNormal { ... }`
branch from `handleStatusEventPurchase` in
`services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer.go`
(the branch that picks `CashShopBuyNormalDoneBody` over the generic
`CashShopCashInventoryPurchaseSuccessBody` for a BUY_NORMAL purchase — this
*is* the production behavior change on the success path) and ran:

```
go build ./...                                            # clean
go test ./kafka/consumer/cashshop/... ./socket/handler/... # all PASS, no FAIL
```

Result: `ok atlas-channel/kafka/consumer/cashshop 0.037s`, `ok
atlas-channel/socket/handler (cached)`. **Nothing failed.** I restored the
file afterward (`git checkout --`) and confirmed `git status` shows no diff.

Root cause: all four new tests in
`socket/handler/cash_shop_buy_normal_test.go` exercise either (a) the
currency-derivation/command-emission path on the request side
(`TestBuyNormalPurchaseCurrency`, `TestBuyNormalHandleEmitsPurchase`), or (b)
the `cashcb.CashShopBuyNormalDoneBody`/`CashShopBuyNormalFailedBody` packet
encoder/decoder directly, in isolation
(`TestCashShopBuyNormalDoneBodyMode`, `TestCashShopBuyNormalFailedBodyMode`).
None of them drive `handleStatusEventPurchase` and assert it picks the
BUY_NORMAL body over the generic one. The success-arm routing logic added in
`consumer.go` — the actual cross-service seam this task exists to wire up —
has zero test coverage and can be silently deleted or misrouted (e.g.
swapped to always take the generic fallback) without any test in the diff
failing.

This is exactly the class of gap C4 warns about ("A prior task in this plan
shipped a client-wedging bug... The earlier bug survived a green suite
precisely because no test covered the failure path"), except here it is the
**success-routing** path that is uncovered, not the failure path (the
failure path IS covered — see Non-blocking / Verified-correct #3 below). The
implementer's own "Concerns" section flags the TDD-literalness question but
frames it as a process deviation ("I did not perform a manufactured RED
run"); the actual, more serious finding is that even a manufactured RED run
against the shipped diff would not go red for this branch, because no test
targets it.

## Verified correct (non-blocking / no finding)

1. **C1 identifiers** — grepped `cashcb.CashShopOperationBuyNormalDone`
   (`libs/atlas-packet/cash/clientbound/shop_operation_body.go:63`, value
   `"BUY_NORMAL_SUCCESS"`) and `CashShopOperationBuyNormalFailed` (`:42`,
   `"BUY_NORMAL_FAILED"`) — both exist verbatim and are used, not the
   brief's non-existent `CashShopOperationBuyNormalSuccess`. No hard-coded
   mode byte in production code; both the handler's success branch
   (`consumer.go:204-214`) and the pre-existing `failureBodyForOperation`
   switch (`consumer.go:307-308`, wired by Task 3) resolve through
   `cashcb.CashShopBuyNormalDoneBody`/`CashShopBuyNormalFailedBody`, which
   themselves resolve mode via `WithResolvedCode`/`ResolveCode` — confirmed
   by reading `shop_operation_body.go:585` and `:421`.

2. **Success/failure pairing (C4)** — `BUY_NORMAL_SUCCESS`/158 and
   `BUY_NORMAL_FAILED`/159 is the correct, unambiguous pair per C1, and both
   sides land in this diff: success in the new `consumer.go` branch, failure
   already present from Task 3 at `consumer.go:307-308`
   (`case cashshop2.ErrorOperationBuyNormal: return
   cashpkt.CashShopBuyNormalFailedBody(reason)`). Confirmed the failure side
   is genuinely pre-existing (not newly added and silently mismatched) via
   `git diff` — `consumer.go`'s only new hunk in this commit is the 20-line
   success-branch insertion at `:194-214`; the `failureBodyForOperation`
   function is untouched by this commit.

3. **Failure arm has real test coverage from Task 3.**
   `services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer_failure_routing_test.go:48`
   (pre-existing, not part of this diff) already has a `"buy normal"` case
   asserting `failureBodyForOperation("BUY_NORMAL", ...)` returns mode 159.
   This test does discriminate: it is driven through the real routing
   function, not a direct encoder call, unlike the new tests' handling of
   the success arm (see Finding 2).

4. **Kafka schema byte-compatibility** — diffed both
   `kafka/message/cashshop/kafka.go` files side by side: both add
   `Operation string \`json:"operation,omitempty"\`` to
   `RequestPurchaseCommandBody` and `PurchaseEventBody`, identical field
   name, identical tag, identical position (appended, not inserted), both
   `omitempty`. A message emitted by a pre-task-9 build unmarshals to
   `Operation: ""` on either side; a pre-task-9 build reading a task-9
   message ignores the unknown field. No in-flight message is broken.

5. **C1 threading is complete and additive-safe.** All three pre-existing
   `RequestPurchase` call sites (`cash_shop_operation.go`: the plain
   `BUY` arm at `:57`, `handleBuyNameChange` at `:335`,
   `handleBuyWorldTransfer` at `:387`) now pass `""` for `operation`,
   preserving today's behavior byte-for-byte (verified: none of the changed
   lines alter any other argument). `atlas-cashshop`'s `Purchase` threads
   `operation` into every one of its seven `ErrorStatusEventProvider` →
   `ErrorStatusEventForOperationProvider` call-site conversions and into the
   success emit (`PurchaseStatusEventProvider`), leaving the sibling
   `PurchaseInventoryIncrease` path's own unrelated `ErrorStatusEventProvider`
   call untouched, as the report states — confirmed by reading
   `services/atlas-cashshop/atlas.com/cashshop/cashshop/processor.go` in
   full diff.

6. **C3 lint fix** — exact text match against the controller's specified
   directive at `cash_shop_credential.go:36-37`. No change to the function
   body, no deletion, no fake caller added; the diff for this file is a
   2-line comment insertion only.

7. **No stub / placeholder / unimplemented status response** — the BUY_NORMAL
   arm now performs a real `RequestPurchase` call with error handling
   (`cash_shop_operation.go:161-163`), replacing the prior log-only stub.

8. **No `libs/atlas-packet/` or `docs/tasks/` changes** — confirmed via
   `git diff --name-only` filtered against both path prefixes: empty.

## Not evaluable

- Whether the client genuinely renders/uses `SlotPos` incorrectly at runtime
  (i.e., an end-to-end client repro) is outside this review's reach — the
  finding above is grounded in the repo's own disassembly-derived
  documentation (`arm-catalog.md`), not a live client test.
- `resolvePurchaseCurrency`'s correctness itself is out of scope (read-only,
  pre-existing from task-227); only its invocation from the new BUY_NORMAL
  arm was checked.

## Verdict rationale

Two independent BLOCKING findings: an invented/under-derived `SlotPos` value
where a constrained field and a real replacement value both existed in
scope, and a genuinely un-discriminating test suite for the success-arm
routing change that is the primary point of this task (mechanically
confirmed by reverting the branch and re-running the suite — all green).
Everything else — identifier correctness, mode-byte resolution discipline,
failure-arm pairing and its pre-existing test coverage, kafka schema
byte-compatibility, and the C3 lint fix — is correct and verified against
source.
