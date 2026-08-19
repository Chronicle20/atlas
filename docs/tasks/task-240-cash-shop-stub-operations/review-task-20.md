# Review: Task 20 — BUY_COUPLE / BUY_FRIENDSHIP (atlas-channel side)

**Commit reviewed:** `27d1c91e8` (range `c05b2a750..27d1c91e8`)
**Brief:** `.superpowers/sdd/plan/task-20-brief.md`
**Report:** `.superpowers/sdd/plan/task-20-report.md`

## Scope

`git show --stat 27d1c91e8` — 7 files, +446/-2, all under
`services/atlas-channel/atlas.com/channel`:

- `cashshop/processor.go` (+15)
- `cashshop/producer.go` (+24)
- `kafka/consumer/cashshop/consumer.go` (+65)
- `kafka/message/cashshop/kafka.go` (+54)
- `socket/handler/cash_shop_operation.go` (+2/-2, wiring only)
- `socket/handler/cash_shop_ring.go` (new, +160)
- `socket/handler/cash_shop_ring_test.go` (new, +126)

This matches the brief's file list exactly. Scope confirmed — no drift.

## 1. Invented identifiers — highest priority

Every identifier introduced or referenced in the diff was grepped against its
source of truth:

- `CommandTypeRequestRingPurchase = "REQUEST_RING_PURCHASE"`,
  `StatusEventTypeRingPurchased = "RING_PURCHASED"` — match
  `services/atlas-cashshop/atlas.com/cashshop/kafka/message/cashshop/kafka.go:23,176`
  literally.
- `RingTypeCouple = "COUPLE"` / `RingTypeFriendship = "FRIENDSHIP"` — match
  `services/atlas-cashshop/atlas.com/cashshop/ring/model.go:15-16`
  (`ring.TypeCouple = Type("COUPLE")`, `ring.TypeFriendship =
  Type("FRIENDSHIP")`) exactly.
- `CashShopOperationBuyCouple = "BUY_COUPLE"` (mode 29),
  `CashShopOperationBuyFriendship = "BUY_FRIENDSHIP"` (mode 35) — pre-existing
  constants in `cash_shop_operation.go:40,44`, not touched by this diff, only
  wired to the new handlers.
- `cashcb.CashShopCoupleDoneBody`, `cashcb.CashShopFriendshipDoneBody`,
  `cashcb.CashShopCoupleFailedBody`, `cashcb.CashShopFriendshipFailedBody` —
  all real exported functions in
  `libs/atlas-packet/cash/clientbound/shop_operation_body.go` (confirmed by
  grep against the resolved module cache).
- `sp.IsPoints()`, `sp.Currency()`, `sp.SPW()`, `sp.Birthday()`,
  `sp.SerialNumber()`, `sp.Name()`, `sp.Message()` on both
  `cashsb.ShopOperationBuyCouple` and `cashsb.ShopOperationBuyFriendship` —
  all real accessors in `libs/atlas-packet/cash/serverbound/shop_operation_buy_{couple,friendship}.go`.
- `cashshop.EnvCommandTopic` — matches
  `services/atlas-cashshop/.../kafka/message/cashshop/kafka.go:10`
  (`"COMMAND_TOPIC_CASH_SHOP"`), same env var every other producer call in
  `processor.go` already uses.
- No `+1` offset, no gender-derived template id, no new numeric constant
  appears anywhere in the diff. The two test-file item ids (`1112000`,
  `1112800`) are byte-encoding pin fixtures only, not production template
  ids used by any handler.

No invented identifiers found. PASS.

## 2. Message-contract fidelity (channel vs. atlas-cashshop)

Read both sides field-by-field:

`RequestRingPurchaseCommandBody` — channel
(`kafka/message/cashshop/kafka.go:154-162`) vs. cashshop
(`kafka/message/cashshop/kafka.go:154-162`, task 19): `TransactionId
uuid.UUID`, `Currency uint32`, `SerialNumber uint32`, `PartnerCharacterId
uint32`, `SenderName string`, `Message string`, `RingType string` — identical
field names, types, and `json` tags on both sides.

`RingPurchasedBody` — channel (`kafka.go:360-369`) vs. cashshop
(`kafka.go:355-364`): `TransactionId uuid.UUID`, `CompartmentId uuid.UUID`,
`AssetId uint32`, `PartnerName string`, `TemplateId uint32`, `Quantity
uint16`, `RingType string`, `PairId uuid.UUID` — identical on both sides.

Topic (`EnvCommandTopic` / `EnvEventTopicStatus`) and command/event type
strings (`REQUEST_RING_PURCHASE` / `RING_PURCHASED`) match. Producer key
(`producer.CreateKey(int(characterId))`) matches the existing keying pattern
every other command in this producer file uses. PASS — no cross-service
field-name divergence.

## 3. `giftRejectionReason` reuse

Confirmed independently, not just taken from the implementer's report:
`git show 27d1c91e8` touches only `kafka.go`, `producer.go`, `processor.go`,
`consumer.go`, `cash_shop_operation.go`, and the two new `cash_shop_ring*.go`
files — `cash_shop_gift.go` (home of `giftRejectionReason`,
`errGiftOwnAccount`, `ErrCredentialMismatch`) is untouched in this commit.
`grep` confirms `giftRejectionReason`'s `switch` in `cash_shop_gift.go:30-40`
has exactly 4 cases (`ErrEmptySlice → INCORRECT_NAME`, `errGiftOwnAccount →
CANNOT_GIFT_TO_OWN_ACCOUNT`, `ErrCredentialMismatch → INVALID_BIRTHDAY`,
default → `unknown_error`) and none were added or modified. `cash_shop_ring.go`
calls the function unchanged, feeding it the same four sentinel classes
(`ErrCredentialMismatch`, `atlasmodel.ErrEmptySlice` for unknown/cross-world
partner, `errGiftOwnAccount` for the own-account case, and a raw resolve
error for the sender lookup, which falls into the `default → unknown_error`
branch). No new constant, no new enum value. PASS.

## 4. Rejection paths (`cash_shop_ring.go:101-135`)

Traced `handleRingPurchase` line by line:

1. `verifySecondaryCredential` (FR-RING-5) — on `ErrCredentialMismatch`,
   announces `INVALID_BIRTHDAY` and returns; on any other error, logs and
   returns (no command sent either way). Matches `handleGift`'s pattern.
2. Unknown partner name — `character.NewProcessor(...).GetByName(name)`
   error path announces `giftRejectionReason(err)` and returns (falls to
   `unknown_error` since the underlying error class isn't
   `atlasmodel.ErrEmptySlice` — a minor imprecision vs. the gift path, which
   explicitly forces `atlasmodel.ErrEmptySlice`; noted below as non-blocking).
3. Cross-world partner (FR-RING-2 — "same world" requirement, confirmed in
   `prd.md:179-180`) — announces `INCORRECT_NAME` (via
   `atlasmodel.ErrEmptySlice`) and returns.
4. Partner on sender's own account — announces `CANNOT_GIFT_TO_OWN_ACCOUNT`
   and returns.
5. Sender self-resolve failure — logs, announces `unknown_error`, returns.
6. Success path — mints `uuid.New()`, calls `RequestRingPurchase(...)`.

Every rejection announces a failure body (`announceRingFailure`) before
returning; none fall through silently. PASS.

**Non-blocking note:** step 2 (unknown-name path) does not explicitly wrap
its error to `atlasmodel.ErrEmptySlice` the way step 3 (cross-world) does, so
if `character.GetByName`'s not-found path ever returns something other than
`atlasmodel.ErrEmptySlice` (e.g., a different sentinel or a wrapped HTTP
404), the client would see the generic `unknown_error` key instead of
`INCORRECT_NAME` for a genuinely-unknown name. This is pre-existing behavior
inherited unchanged from `handleGift`'s identical pattern
(`cash_shop_gift.go:100-106`), not something task 20 introduced, so it is not
attributed to this unit — flagged for completeness only.

## 5. Idempotency

`transactionId := uuid.New()` is minted once, inside `handleRingPurchase`,
per invocation (`cash_shop_ring.go:136`), and passed straight through
`RequestRingPurchase` → `RequestRingPurchaseCommandProvider` with no
additional wrapping, caching, or dedup layer in the channel. This is the same
mint-once-per-click pattern `RequestGiftPurchase`/`RequestPackagePurchase`
already use (`processor.go:245-286`), and idempotency enforcement itself
lives entirely on the `atlas-cashshop` side (task 11's ledger, confirmed in
task 19's review). No second mechanism found. PASS.

## 6. OQ-R1 — typed distinct-halves rejection branch

Confirmed absent: `handleRingPurchase` has no branch that inspects
resolved-commodity or partner data to decide "this ring needs distinct
halves" and answer a typed `*_FAILED`. The doc comment at
`cash_shop_ring.go:88-91` cites `context.md:188-208` (verified to exist,
section 7, "Task 19 — ring purchase: same-template halves shipped,
distinct-halves rejection left undetectable") and gives the same reasoning
context.md gives: no data source in this service (or reachable from it)
distinguishes a couple-ring commodity needing distinct templates from an
ordinary same-template ring. No invented data source, no `+1` offset, no
gender-derived id anywhere near this branch. PASS — the absence is correct.

## 7. Tenant scoping

- `handleStatusEventRingPurchased` (`consumer.go`) checks `t :=
  tenant.MustFromContext(ctx); if !t.Is(sc.Tenant()) { return }` before doing
  any work — matches every sibling status-event handler in this file
  (`handleStatusEventGiftPurchased`, `handleStatusEventPackagePurchased`,
  etc., all of which use the identical guard).
- `handleRingPurchase`'s `ringFailureReasonConfigured` and
  `announceRingFailure` both pull `tenant.MustFromContext(ctx)` to resolve
  writer options and log the tenant identity on an unconfigured errors-table
  key — same as `giftFailureReasonConfigured`/`announceGiftFailure`.
- The synchronous handler path itself is scoped implicitly through
  `session.Model`/`ctx`, consistent with every other BUY arm in this file (no
  handler in `cash_shop_operation.go` does its own explicit tenant check
  before dispatch — that's a repo-wide convention, not something this task
  should have changed).

PASS.

## Test honesty

`TestCashShopRingResultBodies` — full byte-slice assertions for both
`CashShopCoupleDoneBody`/`CashShopFriendshipDoneBody`, including reciprocal
mode-byte checks (`body[0] == 162` fails the couple subtest, `body[0] == 152`
fails the friendship subtest) that would catch a mode-byte swap between the
two arms. `TestRingTypeForOperation` pins the pure mapper. Ran both locally:

```
cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/... -run 'TestCashShopRing|TestRingTypeForOperation' -v
=== RUN   TestCashShopRingResultBodies
=== RUN   TestCashShopRingResultBodies/couple
=== RUN   TestCashShopRingResultBodies/friendship
--- PASS: TestCashShopRingResultBodies (0.00s)
    --- PASS: TestCashShopRingResultBodies/couple (0.00s)
    --- PASS: TestCashShopRingResultBodies/friendship (0.00s)
=== RUN   TestRingTypeForOperation
--- PASS: TestRingTypeForOperation (0.00s)
PASS
```

The implementer's reported RED state (`undefined: ringTypeForOperation`
compile failure before implementation) is consistent with a genuine
pre-implementation failure, not a vacuously-passing test. PASS.

## Build/vet/format

```
go build ./...   → clean
go vet ./...     → clean (no output)
gofmt -l <touched files>  → empty (pristine)
```

`git status --porcelain` shows only pre-existing untracked task docs and the
pre-existing `go.work.sum` diff — no stray mutation from this commit or from
this review.

## Findings

None blocking.

**Non-blocking:**
1. `cash_shop_ring.go:114-116` — the unknown-partner-name error is passed to
   `giftRejectionReason` raw (not wrapped to `atlasmodel.ErrEmptySlice`), so
   it may resolve to the generic `unknown_error` key rather than
   `INCORRECT_NAME` depending on what `character.GetByName`'s not-found path
   actually returns. This is inherited unchanged from `handleGift`'s
   identical, pre-existing pattern (`cash_shop_gift.go:100-106`) — not
   introduced by task 20, so not attributed here, but worth a follow-up
   sweep across all three copies (gift/package/ring) if it's ever confirmed
   wrong.

## Not evaluable

None — the full review surface (message contracts, both new handler files,
the consumer addition, the producer/processor plumbing, and the wiring
change) was covered directly from repo source and passing local test/build
output.
