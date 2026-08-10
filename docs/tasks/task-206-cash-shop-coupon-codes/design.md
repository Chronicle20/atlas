# Cash Shop Coupon-Code Redemption — Design

Status: Approved for planning
Created: 2026-08-09
PRD: [`prd.md`](prd.md)

---

## 1. What this document decides

The PRD (§9) left seven open questions, three of which change the shape of the
work. Those three were put to the user during this design phase and are settled
here; the remaining four are resolved below from source evidence rather than
left open.

| # | PRD question | Decision |
|---|---|---|
| Q1 | Saga vs local transaction | **Local transaction** in `atlas-cashshop`, mirroring the existing `Purchase` path. No saga. |
| Q3 | How much of the writer `errors` table to derive | **The full per-version error enum**, all ten versions, generated from the dispatcher YAML. |
| Q6 | Cash-locker capacity check timing | **Pre-flight ladder check *and* an in-transaction re-check** in the grant. |
| Q2 | Which versions have a serverbound Coupon arm | Resolved by evidence: all ten templates already carry `USE_COUPON_SUCCESS` / `USE_COUPON_FAILED` clientbound mode bytes, so the client-side coupon feature exists in all ten. The serverbound mode byte is still per-version IDB-derived (§4), and a version whose request switch genuinely lacks the arm becomes `n-a` — but the prior is "all ten", not "may shrink". |
| Q4 | Does the client echo the code back? | Resolved during codec derivation (§4). The failure arm's wire shape is already implemented and verified (`UseCouponFailed`); if the client needs more than the reason byte it will show up in the `OnCashItemResUseCouponFailed` decompile, and the answer is recorded in the codec's `packet-audit:fname` comment. No speculative extra arm is designed here. |
| Q5 | Is `UseCouponDone.maplePoint` a delta or a balance? | Resolved during derivation (§4); the handler populates whichever the decompile shows. Design assumes **absolute post-award balance** because `CashQueryResult` (the wallet-refresh packet the same service already sends) carries absolute values, but this is explicitly marked unverified and is a derivation output, not an assumption to code against. |
| Q7 | Global redemption history in the UI | **No.** Per-code and per-account only (FR-7.3). A global list has no stated user story and the account filter covers the abuse-investigation case. |

Two further findings from source that the PRD did not know about, and which
materially change FR-2/FR-3, are in §5.

---

## 2. Decision: local transaction, not a saga

**Chosen.** Redemption is one `database.ExecuteTransaction` inside
`atlas-cashshop`, with the outcome event emitted through the outbox so the
event is exactly-once with the commit.

Every reward type in this task's scope lives in `atlas-cashshop`'s own
database:

- wallet currency → `wallet` table, via `wallet.Processor.WithTransaction(tx).Update(...)`
- cash-locker item → `asset` table, via `asset.Processor.Create(mb)(...)`

The existing purchase path already pairs exactly these two writes atomically in
a single local transaction — `services/atlas-cashshop/atlas.com/cashshop/cashshop/processor.go:98`
(`Purchase`) debits the wallet at line 152 and creates the locker asset at
line 195-198, both inside one `ExecuteTransaction`, with the status event
enqueued on the message buffer inside the transaction (line 206) so it rides
the outbox.

A saga would replace that real, database-enforced atomicity with best-effort
compensation across three new actions (`reserve_coupon_redemption`,
`release_coupon_redemption`, `award_cash_shop_asset`), new orchestrator handler
and compensator arms, a new accept-event path, and a timeout budget — to buy an
atomicity guarantee that is strictly weaker than the one a single transaction
already provides. Compensation can itself fail; `ROLLBACK` cannot.

**Rejected alternatives:**

- *Saga as the PRD specifies.* Correct the moment a reward type outside
  `atlas-cashshop` is added (regular inventory, mesos, experience). It is not
  correct today, and the cost of the change at that point is bounded (§2.1).
- *Hybrid with a `RewardGranter` seam plus saga-shaped dispatch.* Was offered;
  the user chose the plain local transaction. The seam is still present in a
  lighter form (§2.1) because it costs almost nothing.

### 2.1 Keeping the saga door open cheaply

Reward application is dispatched by reward type behind one small interface in
the `coupon` package:

```go
// grant applies one reward inside the redemption transaction. Every
// implementation today writes only to atlas-cashshop's own tables; when a
// reward type owned by another service is added, that granter becomes the
// single place a saga is introduced.
type rewardGranter interface {
    Grant(mb *message.Buffer) func(tx *gorm.DB, ctx redemptionContext, r Reward) (GrantedReward, error)
}
```

with `currencyGranter` and `cashItemGranter` registered by `Reward.Type()`.
This is not an abstraction for its own sake — it is what keeps the reward loop
readable and gives the future out-of-service reward type one obvious insertion
point instead of a rewrite. No orchestrator changes, no new saga actions, no
new `libs/atlas-saga` constants. **FR-6 in the PRD is superseded in full by
this section.**

`libs/atlas-saga` is not touched by this task.

### 2.2 What replaces the saga's transaction id

FR-6.5 wanted a saga transaction id to deduplicate a double-clicked submit.
Without a saga there is still a `transaction_id` column on
`coupon_redemptions` (a plain generated UUID per redemption attempt, kept for
log correlation and for the `rewards_granted` audit trail), but deduplication
is done by the database, not by a transaction key: the unique index on
`(tenant_id, coupon_id, account_id)` makes the second concurrent submit lose
with a constraint violation, which maps to `COUPON_ALREADY_USED`. That is
strictly stronger than a transaction-id guard, which only covers duplicates
that arrive while the first is still in flight.

---

## 3. Architecture and data flow

```
client                atlas-channel                     atlas-cashshop
  |                        |                                  |
  |-- CASHSHOP_OPERATION ->|                                  |
  |   (USE_COUPON arm)     |                                  |
  |                        | decode ShopOperationUseCoupon    |
  |                        | normalize + local sanity check   |
  |                        |   (empty / too long -> reply     |
  |                        |    INVALID_COUPON_CODE directly, |
  |                        |    no round trip)                |
  |                        |                                  |
  |                        |-- COMMAND_TOPIC_CASH_SHOP ------>|
  |                        |   REQUEST_COUPON_REDEMPTION      |
  |                        |   {characterId, code}            |
  |                        |                                  | resolve accountId
  |                        |                                  | rate-limit gate
  |                        |                                  | BEGIN
  |                        |                                  |  validation ladder
  |                        |                                  |  conditional counter bump
  |                        |                                  |  insert redemption row
  |                        |                                  |  grant rewards
  |                        |                                  |  enqueue status event
  |                        |                                  | COMMIT  (outbox)
  |                        |<- EVENT_TOPIC_CASH_SHOP_STATUS --|
  |                        |   COUPON_REDEEMED | COUPON_FAILED|
  |<- CASHSHOP_OPERATION --|                                  |
  |   USE_COUPON_SUCCESS   |                                  |
  |   or USE_COUPON_FAILED |                                  |
  |<- CASH_QUERY_RESULT ---|  (wallet refresh, success only)  |
```

The request/response shape deliberately copies the existing purchase flow
(`atlas-channel` `cashshop.Processor.RequestPurchase` → `COMMAND_TOPIC_CASH_SHOP`
→ `atlas-cashshop` → `EVENT_TOPIC_CASH_SHOP_STATUS` → the channel's status
consumer at `services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer.go`).

### 3.1 New Kafka contracts

Added to `services/atlas-cashshop/atlas.com/cashshop/kafka/message/cashshop/kafka.go`
and mirrored in `services/atlas-channel/atlas.com/channel/kafka/message/cashshop/`:

```go
CommandTypeRequestCouponRedemption = "REQUEST_COUPON_REDEMPTION"

type RequestCouponRedemptionCommandBody struct {
    Code string `json:"code"`   // already normalized by the channel
}

StatusEventTypeCouponRedeemed = "COUPON_REDEEMED"
StatusEventTypeCouponFailed   = "COUPON_FAILED"

type CouponRedeemedBody struct {
    CompartmentId uuid.UUID `json:"compartmentId"`
    AssetIds      []uint32  `json:"assetIds"`
    MaplePoints   uint32    `json:"maplePoints"`   // DELTA — the Maple Points this coupon awarded, NOT a balance
}
```

**Correction (task-206 derivation).** PRD Q5 is answered: `UseCouponDone.maplePoint` is a delta, not an absolute post-award balance. `CCashShop::OnCashItemResUseCouponDone` @ `0x479d8a` reads it at `0x479efb` and uses it only at `0x47a0b6`-`0x47a0df` to format `SP_587_D_MAPLEPOINTS` inside the `SP_585_YOU_HAVE_RECEIVED` sentence; the balance itself is refreshed separately by `CCashShop::OnQueryCashResult` @ `0x478f81`.

```go
type CouponFailedBody struct {
    Error string `json:"error"`   // one of the FR-3.4 key strings
}
```

**Why not reuse `StatusEventTypeError`.** The existing `ERROR` event is handled
at `consumer.go:135` by announcing `CashShopInventoryCapacityIncreaseFailedBody`
— a *different* mode byte. A coupon failure must go out on the
`USE_COUPON_FAILED` arm, so it needs its own event type; folding it into
`ERROR` would require the channel to guess which failure arm a given error
belongs to. A distinct `COUPON_FAILED` type keeps the arm selection explicit.

`CouponRedeemedBody` carries asset ids rather than the fully-built
`CashInventoryItem` records because the channel already owns the
asset-id → `CashInventoryItem` conversion (`consumer.go:105-124`, the purchase
handler); duplicating that projection into the event body would put packet
concerns in `atlas-cashshop`.

---

## 4. Packet layer (FR-1)

One new codec: `libs/atlas-packet/cash/serverbound/shop_operation_use_coupon.go`,
an immutable `ShopOperationUseCoupon` with both `Encode` and `Decode`, shaped
like `shop_operation_buy.go`.

**Derivation is a hard prerequisite, not a formality.** The struct's fields come
from decompiling the `USE_COUPON` arm of each version's cash-shop request
builder (`CCashShop::SendUseCouponRequest` / the coupon branch of
`CCashShop::OnCashItemRequest`) in that version's IDB. No field is written that
is not observed in a decompilation. The per-version fname goes in the
`packet-audit:fname` doc comment. Version divergence is gated with the
`MajorAtLeast` idiom; raw `> N` comparisons are banned
(`bug_majorversion_gt83_is_off_by_one_v87`).

Evidence that the feature exists on all ten: every one of the ten templates
already binds the clientbound coupon arms —
`USE_COUPON_SUCCESS`/`USE_COUPON_FAILED` at modes
48→54/57, 61→61/64, 72→69/72, 79→81/84, 83→89/92, 84→92/95, 87→94/97,
92→101/104, 95→102/105, jms_185→90/93. A client that can *receive* a coupon
result can send a coupon request. Any version whose request switch nonetheless
lacks the arm is recorded `n-a` with the enumeration as evidence, per the
existing n-a consistency gate.

**Matrix.** Serverbound cash-shop arms are already tracked as `sub-struct` rows
keyed by struct path (e.g. `cash/serverbound/CashShopOperationBuy` in
`docs/packets/audits/status.json`). `ShopOperationUseCoupon` gets its own row
with ten cells, each promoted through the single-cell verify procedure
(`docs/packets/audits/VERIFYING_A_PACKET.md`) with a byte-fixture test. The
serverbound `CASHSHOP_OPERATION` op already exists in
`docs/packets/registry/*.yaml` (gms_v83 opcode 229) — **no new registry op is
added**; PRD FR-1.5's "register the new op in the operation registry" is
corrected to "add the sub-struct matrix row". No previously-verified cell
changes wire behaviour.

---

## 5. Configuration: two findings that reshape FR-2 and FR-3

### 5.1 Finding — the serverbound operations tables are not tool-governed

`packet-audit operations` generates and validates template `operations` maps
from `docs/packets/dispatchers/*.yaml`, and it already supports **serverbound**
docs: `dispatcherDoc` accepts either `writer:` or `handler:`
(`tools/packet-audit/cmd/operations.go:37-77`). But no dispatcher YAML declares
`handler: CashShopOperationHandle`, so all ten serverbound tables are
hand-maintained and unvalidated — which is exactly how three of them
(`gms_87`, `gms_92`, `gms_95`) ended up **empty** and `jms_185` ended up with
five keys against `gms_83`'s nineteen:

| template | handler opCode | serverbound `operations` keys |
|---|---|---|
| gms_48 | 0xA0 | 13 |
| gms_61 | 0xC4 | 15 |
| gms_72 | 0xDB | 16 |
| gms_79 | 0xDD | 17 |
| gms_83 | 0xE5 | 19 |
| gms_84 | 0xEB | 19 |
| gms_87 | 0xF2 | **0** |
| gms_92 | 0x10C | **0** |
| gms_95 | 0x113 | **0** |
| jms_185 | 0xF5 | **5** |

**Design decision.** Author
`docs/packets/dispatchers/cash_shop_operation_handle.yaml`
(`handler: CashShopOperationHandle`, `op: CASHSHOP_OPERATION`,
`direction: serverbound`, per-version `opcodes:`) enumerating every arm of the
request switch per version from the IDBs, then let `packet-audit operations`
*generate* all ten tables. Hand-editing the ten JSON files is not the plan;
the YAML is the source of truth and CI's `--check` keeps it that way. This
satisfies FR-2.1, FR-2.2 and FR-2.3 in one mechanism and is consistent with
`feedback_dispatcher_config_drive_all_modes`.

**The all-or-nothing trap.** `expectedTable` returns the keys the YAML declares
*for that version*, and any template key not in that set is reported as
`EXTRA` (`operations.go:150-154`) — but only once the version has at least one
declared key (`len(expected) == 0` → `continue`, line 142). So declaring a
*partial* per-version arm set is worse than declaring none: adding just
`USE_COUPON` for `gms_48` would immediately fail `--check` on the thirteen keys
already in that template. Every version listed in the new YAML must be
enumerated **completely**. This is a planning constraint, not an implementation
detail.

### 5.2 Finding — `gms_v92` is absent from the clientbound dispatcher YAML

`docs/packets/dispatchers/cash_shop_operation.yaml` covers nine versions;
`gms_v92` appears zero times in it, while `template_gms_92_1.json`'s
`CashShopOperation` writer carries 57 `operations` keys. Because of the
`len(expected) == 0` short-circuit, those 57 keys are currently validated by
nothing — `packet-audit operations --check` reports `operations check OK` today
purely because v92 is invisible to it.

Since this task must open the v92 IDB anyway to derive its `errors` enum and
its serverbound arm table, **v92 is added to the clientbound YAML in the same
pass**, completely enumerated, closing a silent hole in the v92 column. The
same all-or-nothing rule applies: partial v92 coverage would fail `--check` on
the other 57 keys.

### 5.3 The `errors` table (FR-3.3) — full enum, tool-generated

No template has an `errors` table today; every `CashShopOperation` writer's
`options` contains only `operations`. `CashShopUseCouponFailedBody`
(`libs/atlas-packet/cash/clientbound/shop_operation_body.go:269`) resolves its
reason byte with `ResolveCode(l, options, "errors", message)`, so without one
every coupon failure resolves to an unconfigured code.

Per the Q3 decision, the **full per-version error enum** is derived for all ten
versions while the IDBs are open, not just the seven coupon keys.

`packet-audit operations` only knows about `options.operations`
(`setOperations` / `operationsOf`). It is extended with a second, structurally
identical table: a dispatcher YAML may declare an `errors:` section alongside
`operations:`, generated into and checked against `options.errors`. The
extension is a parameterised table name plus a second pass over the same node
plumbing — the existing generate/check/EXTRA/MISSING/DRIFT reporting is reused
verbatim. Leaving `errors` hand-maintained would recreate exactly the drift
that §5.1 documents.

`packet-audit` gets test coverage for the new table in
`tools/packet-audit/cmd/operations_test.go` before the YAML is authored.

### 5.4 The `INVALID_COUPON_COUPON` typo (FR-3.5)

`shop_operation_body.go:91` declares
`CashShopOperationErrorInvalidCouponCode = "INVALID_COUPON_COUPON"`. Nothing
consumes it. The constant is corrected to `"INVALID_COUPON_CODE"` and the
`errors` YAML uses the corrected string. A mismatch here is silent — `ResolveCode`
misses and the client shows the wrong message — so the constant, the YAML, and
the ten generated tables are all driven from the one corrected string.

### 5.5 Guards

`tools/template-opcode-order-guard.sh` and
`tools/template-duplicate-binding-guard.sh` are unaffected in principle (no
opcode is added or rebound; only `options` content changes) but are run
regardless, as is `packet-audit operations --check`, after every template
regeneration.

---

## 6. Channel handler (FR-4)

`services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_operation.go`
gains one constant and one arm, in the same shape as its nineteen siblings:

```go
CashShopOperationUseCoupon = "USE_COUPON"
```

```go
if isCashShopOperation(l)(readerOptions, op, CashShopOperationUseCoupon) {
    sp := &cashsb.ShopOperationUseCoupon{}
    sp.Decode(l, ctx)(r, readerOptions)
    code := coupon.Normalize(sp.Code())
    if !coupon.PlausibleCode(code) {
        _ = session.Announce(l)(ctx)(wp)(cashcb.CashShopOperationWriter)(
            cashcb.CashShopUseCouponFailedBody(cashcb.CashShopOperationErrorInvalidCouponCode))(s)
        return
    }
    _ = cashshop.NewProcessor(l, ctx).RequestCouponRedemption(s.CharacterId(), code)
    return
}
```

- **Normalization** (`strings.ToUpper(strings.TrimSpace(code))`) happens once,
  here, so the value on the wire to `atlas-cashshop` and the value in the
  database are the same shape. The service normalizes again defensively — it
  must not trust an input it can also receive from a future caller.
- **Local sanity check** (FR-4.3): empty after trimming, or longer than the
  32-character column, short-circuits to `INVALID_COUPON_CODE` with no round
  trip. This is also the first line of brute-force defence.
- **Account resolution** (FR-4.4) happens in `atlas-cashshop`, not the channel:
  the packet arrives on a character session but wallets are account-scoped, and
  `atlas-cashshop` already resolves `characterId → accountId` via
  `character.Processor` in `Purchase` (`processor.go:112-117`). The command body
  therefore carries only `characterId` and `code`.

The normalization helper and the plausibility predicate live in one place —
a tiny `coupon` package under `services/atlas-channel/atlas.com/channel/cashshop/`
mirrored by the same rules in `atlas-cashshop` — rather than being inlined
twice with a chance of diverging.

---

## 7. Coupon domain in `atlas-cashshop` (FR-5)

New package `services/atlas-cashshop/atlas.com/cashshop/coupon/`, following the
project's standard domain shape: `model.go` (immutable, private fields, getters,
Builder), `entity.go` + `Migration`, `Make(e Entity) (Model, error)`,
`administrator.go` (writes), `provider.go` (reads), `processor.go`
(`Interface` + `Impl`, `NewProcessor(l, ctx, db)`), `resource.go` + `rest.go`
(JSON:API, registered in `main.go` via `AddRouteInitializer`).

Sub-packages `coupon/batch/` and `coupon/redemption/` keep the three entities'
CRUD from piling into one file.

### 7.1 Tables

As specified in PRD §6 — `coupons`, `coupon_batches`, `coupon_redemptions` —
with these design commitments:

- `code` is stored **normalized** (trimmed, uppercased); the unique index on
  `(tenant_id, code)` over the normalized value *is* the case-insensitive
  uniqueness guarantee. No functional index over a raw value.
- Unique index on `(tenant_id, coupon_id, account_id)` on `coupon_redemptions`
  is the one-time-per-account rule. The FR-5.4 ladder check is a
  friendly-error fast path; the index is the enforcement.
- `rewards` is `jsonb` (always read and written as a whole bundle, never
  queried by reward attribute). `rewards_granted` on the redemption row is a
  snapshot so later edits to the coupon do not rewrite history.
- All three migrations are appended to the `database.SetMigrations(...)` list in
  `services/atlas-cashshop/atlas.com/cashshop/main.go`.

### 7.2 Reward model

```go
type RewardType string
const (
    RewardTypeCurrency RewardType = "CURRENCY"
    RewardTypeCashItem RewardType = "CASH_ITEM"
)
```

`Reward` is a discriminated value with `Type()`, `Currency()`, `Amount()`,
`SerialNumber()`, `Quantity()`. Currency ids follow the existing
`wallet.Model.Balance` convention (`wallet/model.go:33-41`): `1` = credit (NX),
`2` = Maple Points, anything else = prepaid. **No new currency enum is
introduced** — reusing the existing convention avoids a second source of truth,
per DOM-21.

`wallet.Model` currently has `Purchase(currency, amount)` (a debit) but no
credit operation. A symmetric `Award(currency, amount) Model` is added
alongside it, with a saturating add so a malformed reward cannot wrap a
`uint32` balance. `Award` is pure and unit-tested next to the existing
`wallet/model_test.go` cases.

### 7.3 The redemption transaction

`coupon.Processor.RedeemAndEmit(characterId uint32, code string) error`, whose
body is one `database.ExecuteTransaction` wrapping
`message.Emit(outbox.EmitProvider(...))`, exactly as
`PurchaseAndEmit` does (`cashshop/processor.go:90-96`).

Inside the transaction, in order:

1. Resolve `characterId → accountId` (and the character's job type, for the
   locker compartment type — `TypeExplorer` / `TypeCygnus` / `TypeLegend`, the
   same branch as `Purchase` at `processor.go:130-136`).
2. Rate-limit gate (§7.5).
3. Validation ladder (FR-5.4), first failure wins:
   1. code exists → else `INVALID_COUPON_CODE`
   2. `active` → else `COUPON_NOT_REGISTERED`
   3. `now >= starts_at` when set → else `COUPON_NOT_REGISTERED`
   4. `now <= expires_at` when set → else `COUPON_EXPIRED`
   5. no prior redemption for this account → else `COUPON_ALREADY_USED`
   6. `redemption_count < max_uses` when set → else `COUPON_USAGE_LIMIT`
   7. locker free slots ≥ item-reward count → else `INVENTORY_FULL`
4. **Conditional counter bump** (FR-5.5):
   `UPDATE coupons SET redemption_count = redemption_count + 1
    WHERE id = ? AND tenant_id = ? AND (max_uses IS NULL OR redemption_count < max_uses)`,
   then check `RowsAffected`. Zero rows → `COUPON_USAGE_LIMIT`. A
   read-then-write is not acceptable and is explicitly banned in review.
5. Insert the `coupon_redemptions` row. A unique-violation on
   `(tenant_id, coupon_id, account_id)` → `COUPON_ALREADY_USED`. This is the
   race winner/loser resolution, not a redundant check.
6. Grant each reward through its `rewardGranter`. The cash-item granter
   **re-reads the compartment inside the transaction** and fails to
   `INVENTORY_FULL` if capacity has been consumed since step 3.7 (Q6:
   pre-flight *and* re-check; the ladder check gives a deterministic error
   ordering, the in-transaction check closes the TOCTOU window).
7. Enqueue `COUPON_REDEEMED` on `EnvEventTopicStatus` through the message
   buffer, so it is committed with the state change and delivered by the
   outbox.

Any ladder failure returns before any write, and the failure event is emitted
on the **direct producer path outside the transaction** — the same distinction
`Purchase` draws with its `rejectEmit` closure and the comment at
`processor.go:100-103`: an event asserting "nothing happened" must not ride an
outbox that implies a commit. A failure discovered *after* writes have begun
(step 6) rolls the transaction back and then emits on the direct path too.

### 7.4 Error mapping (FR-3.4)

| Outcome | Key string |
|---|---|
| No such code | `INVALID_COUPON_CODE` *(corrected, §5.4)* |
| Past `expires_at` | `COUPON_EXPIRED` |
| Account already redeemed | `COUPON_ALREADY_USED` |
| `max_uses` exhausted | `COUPON_USAGE_LIMIT` |
| Inactive, or before `starts_at` | `COUPON_NOT_REGISTERED` |
| No free locker slot | `INVENTORY_FULL` |
| Anything else | `UNKNOWN_ERROR` |

A `redemptionError` sentinel type carries the key string, so the mapping lives
once at the ladder and the transport layer never re-derives it.

### 7.5 Rate limiting

Per PRD §8: failed attempts are counted per account; past a threshold within a
window, further attempts short-circuit to `INVALID_COUPON_CODE` with no
database lookup. The counter is a Redis key through `libs/atlas-redis` (the
raw `go-redis` client is banned outside that library by
`tools/redis-key-guard.sh`) with a TTL equal to the window. Threshold and
window come from the tenant configuration resource, not constants
(`feedback_client_wire_values_config_resolved`); a tenant that has not set them
gets a documented default resolved in the configuration layer, never a magic
number at the call site.

Returning `INVALID_COUPON_CODE` (rather than a distinct "rate limited" error)
is deliberate: it leaks nothing about whether the attempted code exists, which
is the point of the limiter.

---

## 8. REST surface (FR-7)

Exactly PRD §5. Nothing is added or removed. Notes on shape:

- Three api2go resources: `coupons`, `coupon-batches`, `coupon-redemptions`,
  each with `GetName()` returning the resource type and registered via
  `RegisterHandler(l)(si)`.
- Every query is scoped from `tenant.MustFromContext(ctx)`; no endpoint accepts
  a tenant id in a body (FR-7.4).
- **There is no redemption endpoint.** The packet path is the only trigger. A
  REST redeem endpoint would be an unauthenticated reward faucet.
- Bulk generation (FR-7.2) draws from the unambiguous alphabet
  `ABCDEFGHJKMNPQRSTUVWXYZ23456789` (no `O`/`0`, `I`/`1`/`L`), inserts with the
  unique index as the collision detector, and **retries a collision rather than
  skipping it**, so the response's created count always equals `count`. The
  generator uses `crypto/rand`; a code drawn from `math/rand` is guessable and
  these are secrets.
- Deleting a coupon with redemptions returns `409`, so history is never
  silently destroyed; disabling (`active = false`) is the intended way to kill
  a leaked code.
- `maxUses` on PATCH below the current `redemptionCount` is `422` — it would
  make the conditional bump permanently unsatisfiable while claiming the coupon
  is live.

---

## 9. Admin UI (FR-8)

`services/atlas-ui/src/pages/CouponsPage.tsx`, `coupons-columns.tsx`,
`CouponDetailPage.tsx` — the existing list/columns/detail trio pattern used by
`AccountsPage.tsx`. TanStack Query for fetching, react-hook-form + Zod for the
create form and the bulk-generate dialog, shadcn/ui components, Tailwind.

- The create form's `code` field is optional; blank means "generate one".
- The reward-rows editor is a field array switching between the two reward
  shapes on `type`; Zod discriminated union, so an invalid combination cannot
  reach the API.
- Bulk-generate returns every code; the dialog offers a CSV download built
  client-side from the response (no extra endpoint).
- All POST/PATCH bodies use the JSON:API envelope
  (`bug_ui_jsonapi_envelope_required_for_input_handlers`).
- Delete is confirmed and disabled once `redemptionCount > 0`, matching the
  server's `409`.
- No global redemption list (Q7).

---

## 10. Testing

**Packet.** Round-trip byte-fixture tests per applicable version with a
`packet-audit:verify` marker, following
`libs/atlas-packet/cash/serverbound/v48_test.go`…`v79_test.go` and the modern
equivalents. A cell that does not promote in the matrix is a failure, not a
prose claim.

**Tooling.** `packet-audit` tests for the new `errors` table (generate, check,
drift, missing, extra) before any YAML is authored.

**Domain — the tests that actually matter here are the concurrency ones:**

- Two concurrent redemptions of a `max_uses = 1` code → exactly one success,
  one `COUPON_USAGE_LIMIT`. Real goroutines against a real database, not a
  mocked counter — the whole point is the conditional `UPDATE`'s `RowsAffected`.
- Two concurrent redemptions of the same code by the same account → exactly one
  success, one `COUPON_ALREADY_USED`, driven through the unique index.
- A forced failure in the item-grant step leaves the wallet unchanged, the
  locker unchanged, `redemption_count` at its prior value, and the code
  redeemable. With a local transaction this is a rollback assertion, and it is
  the single test that proves the §2 decision.
- One test per FR-5.4 outcome → FR-3.4 key.
- Codes differing only in case or surrounding whitespace resolve to the same
  coupon.
- `wallet.Model.Award` saturation.

Test setup uses the project's Builder pattern; no `*_testhelpers.go`
test-only constructors.

**End to end** (PRD §10): live v83 tenant — valid code credits and updates the
open Cash Shop window without relog; repeat shows "already used"; garbage shows
"invalid"; expired shows "expired".

---

## 11. Verification gates

Per `CLAUDE.md`, before the branch is called done:

- `go test -race ./...` and `go vet ./...` clean in `libs/atlas-packet`,
  `services/atlas-cashshop`, `services/atlas-channel`, `tools/packet-audit`.
- `tools/lint.sh --check`, `tools/redis-key-guard.sh`,
  `tools/goroutine-guard.sh` clean from the repo root.
- `tools/template-opcode-order-guard.sh`,
  `tools/template-duplicate-binding-guard.sh`,
  `tools/template-movement-types-guard.sh` clean (templates changed).
- `packet-audit operations --check`, `packet-audit matrix`, fname-doc and n-a
  consistency checks exit 0.
- `docker buildx bake atlas-cashshop atlas-channel atlas-configurations` from
  the worktree root. **`atlas-saga-orchestrator` is not in this list** — §2
  removed it from the blast radius.
- `atlas-ui`: `npm run build` (type-checks tests) plus its test suite.

---

## 12. Risks

| Risk | Mitigation |
|---|---|
| Serverbound arm enumeration for ten versions is the bulk of the work, and §5.1's all-or-nothing rule means a version cannot be half-done. | Sequence the plan version-by-version: one IDB open per version producing that version's *complete* serverbound arm set **and** its complete `errors` enum, both landing in the YAML together. Do not interleave. |
| Adding `gms_v92` to the clientbound YAML (§5.2) forces a complete 57-arm enumeration that the existing template may not agree with. | Treat any disagreement between the derived v92 enumeration and the template's current 57 keys as a **bug in the template**, fixed by regeneration — the template values were never validated against anything. Record the diff in the task folder. |
| `UseCouponDone.maplePoint` delta-vs-absolute (Q5) is unverified. | Blocking on the decompile before the success handler is written; the codec's fname comment records the answer. Sending the wrong one shows the player a wrong balance until the next `CashQueryResult`. |
| The full per-version error enum (Q3) is a large IDA job. | It is bounded and it is the user's explicit choice; it is also work that only has to be done once, and doing it while each IDB is already open is strictly cheaper than a second pass. |
| Rate-limit config defaults could silently become magic numbers. | The default is resolved in the configuration layer with the other tenant defaults, and review checks the call site takes a resolved value (DOM-25). |

---

## 13. Departures from the PRD

Recorded explicitly so the plan does not reintroduce them:

1. **FR-6 (saga orchestration) is superseded in full** by §2. No new
   `libs/atlas-saga` actions, no `atlas-saga-orchestrator` changes. FR-6.1–6.6
   are replaced by the local transaction and the unique index.
2. **FR-1.5's registry registration** is corrected to a sub-struct matrix row;
   the serverbound `CASHSHOP_OPERATION` op already exists in the registry.
3. **FR-2.1/2.2 are satisfied by generation, not hand-editing** — a new
   serverbound dispatcher YAML plus `packet-audit operations`.
4. **FR-3.3 is widened** from the seven coupon keys to the full per-version
   error enum, and the `errors` table becomes tool-generated (requires a
   `packet-audit` extension).
5. **`gms_v92` is added to the clientbound dispatcher YAML**, which the PRD did
   not know was missing.
6. **PRD §7's Service Impact row for `atlas-saga-orchestrator` is dropped**, and
   `tools/packet-audit` is added.
7. **Q7 answered "no"** — no global redemption history view.
