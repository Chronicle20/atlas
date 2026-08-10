# Cash Shop Coupon-Code Redemption — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-09
---

## 1. Overview

The MapleStory Cash Shop has a "Coupon" tab where a player types a promotional code and
receives NX credit, Maple Points, or cash items. Atlas has the *response* half of this
feature and none of the *request* half. Task-183 (cash-shop result family) landed both
clientbound arms — `UseCouponDone` (`USE_COUPON_SUCCESS`,
`libs/atlas-packet/cash/clientbound/shop_operation_result_gift.go:201`) and
`UseCouponFailed` (`USE_COUPON_FAILED`,
`libs/atlas-packet/cash/clientbound/shop_operation_result_failed.go:161`) — along with
their writer mode bytes in all ten cash-shop-capable tenant templates. Nothing ever
sends them, because there is no serverbound `USE_COUPON` codec, no arm in
`services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_operation.go`, no
`USE_COUPON` key in any template's `CashShopOperationHandle` `operations` table, and no
coupon-code storage anywhere in the monorepo.

This task closes the loop: a serverbound codec and channel handler arm, a `coupon`
domain in `atlas-cashshop` that owns codes / rewards / redemption records, a
saga-orchestrated redemption that credits the wallet and grants cash-locker items, the
clientbound success and failure responses, and an admin surface in `atlas-ui` for
creating codes individually or generating them in bulk.

Reward scope for this task is **wallet currency (NX credit / Maple Points / prepaid) and
cash-locker items**. Regular-inventory items and mesos are explicitly out of scope
(§2 Non-goals) even though the `UseCouponDone` codec carries a meso field — the codec
will send `0` for it.

## 2. Goals

Primary goals:

- A player who types a valid coupon code into the Cash Shop Coupon tab receives the
  code's rewards and sees the Cash Shop UI update without relogging.
- A player who types an invalid, expired, exhausted, disabled, or already-redeemed code
  receives the correct client-side error message rather than a silent no-op.
- Codes support one-time-per-account enforcement, a global max-uses counter, an
  activation/expiry window, and an active/inactive toggle.
- An administrator can create a code with an exact string, generate a batch of N random
  single-use codes sharing one reward definition, list/filter codes, disable a code
  without losing its history, and inspect who redeemed what and when.
- Redemption is atomic: a code is never marked consumed unless every reward in it landed,
  and rewards are never granted twice for one redemption.
- All ten cash-shop-capable client versions can send and receive the coupon exchange.

Non-goals:

- **Gift coupons.** `GIFT_COUPON_SUCCESS` (`CashShopOperationGiftCouponDone`) — sending a
  coupon's rewards to another account — is out of scope.
- **Regular (non-cash) inventory item rewards** via `atlas-inventory`.
- **Meso rewards.** The `UseCouponDone.meso` field is encoded as `0`.
- **Internet-café coupon variants.** The error constants
  `COUPON_INTERNET_CAFE_RESTRICTION`, `INTERNET_CAFE_COUPON_ALREADY_USED`, and
  `INTERNET_CAFE_COUPON_EXPIRED` exist but no café concept does; they are not wired.
- **NX purchase / billing / payment processing.** Codes are administratively minted, not
  bought.
- **Gender, level, job, or account-age gating** of codes (`COUPON_GENDER_RESTRICTION`
  exists as a constant but is not used).
- **`gms_12`.** `template_gms_12_1.json` has no `CashShopOperationHandle` registered at
  all — that version has no cash shop in Atlas and is not part of the ten.

## 3. User Stories

- As a player, I want to enter a promo code in the Cash Shop Coupon tab so that I receive
  the NX or item the promotion advertised.
- As a player, I want a clear on-screen reason when my code does not work so that I know
  whether to re-type it, wait, or give up.
- As a player, I want the NX balance and cash locker in the open Cash Shop window to
  reflect my reward immediately so that I can spend it without reopening the shop.
- As an administrator, I want to create a coupon code with a specific string and a
  specific reward bundle so that I can run an announced promotion.
- As an administrator, I want to generate 500 unique single-use codes with one reward
  definition so that I can hand out one-per-player codes at an event.
- As an administrator, I want to set an activation and expiry time on a code so that a
  promotion opens and closes without my intervention.
- As an administrator, I want to disable a leaked code immediately without deleting its
  redemption history so that I can audit the damage.
- As an administrator, I want to see every redemption of a code — which account, which
  character, when — so that I can investigate abuse.

## 4. Functional Requirements

### FR-1 — Serverbound `USE_COUPON` codec

- **FR-1.1** Add `libs/atlas-packet/cash/serverbound/shop_operation_use_coupon.go`
  defining an immutable `ShopOperationUseCoupon` struct with both `Encode` and `Decode`,
  following the shape of the neighbouring `shop_operation_buy.go` /
  `shop_operation_gift.go` codecs.
- **FR-1.2** The field order MUST be derived from the client, not assumed. For each of
  the ten target versions, decompile the coupon branch of the Cash Shop request sender
  (`CCashShop::SendUseCouponRequest` / the `USE_COUPON` arm of the cash-shop request
  builder) in that version's IDB and record the read order in the codec's
  `packet-audit:fname` doc comment. No field may be written into the struct that is not
  observed in a decompilation.
- **FR-1.3** Where versions diverge, gate the divergent field with the `MajorAtLeast`
  idiom. Raw `> N` comparisons are banned (see
  `bug_majorversion_gt83_is_off_by_one_v87`).
- **FR-1.4** Add a round-trip byte-fixture test per version, following the existing
  `libs/atlas-packet/cash/serverbound/v48_test.go` … `v79_test.go` pattern, plus the
  modern-version equivalents.
- **FR-1.5** Existing verified cells of the packet coverage matrix MUST NOT change wire
  behaviour. Register the new op in the operation registry and promote its matrix cells
  through the standard single-cell verify procedure
  (`docs/packets/audits/VERIFYING_A_PACKET.md`).

### FR-2 — Template routing for all ten versions

- **FR-2.1** Add a `USE_COUPON` key with that version's mode byte to the
  `CashShopOperationHandle` handler's `options.operations` table in each of:
  `template_gms_48_1.json`, `template_gms_61_1.json`, `template_gms_72_1.json`,
  `template_gms_79_1.json`, `template_gms_83_1.json`, `template_gms_84_1.json`,
  `template_gms_87_1.json`, `template_gms_92_1.json`, `template_gms_95_1.json`,
  `template_jms_185_1.json`.
- **FR-2.2** Three of those templates — `gms_87`, `gms_92`, `gms_95` — currently have a
  registered `CashShopOperationHandle` with an **empty** `operations` table, and
  `jms_185` has only five entries (versus nineteen in `gms_83`). Adding a single
  `USE_COUPON` key to an otherwise-empty table would leave every other cash-shop
  operation unroutable on those versions. This task MUST populate at minimum the
  `USE_COUPON` key on all ten; where a version's table is empty or short, the missing
  keys for operations Atlas already handles MUST be filled in from that version's IDB in
  the same change, so the table is not left in a state where one op works and eighteen
  silently drop.
- **FR-2.3** The mode byte is per version and MUST be config-resolved via
  `isCashShopOperation` — never hard-coded in the handler
  (`feedback_client_wire_values_config_resolved`, DOM-25).
- **FR-2.4** `tools/template-opcode-order-guard.sh` and
  `tools/template-duplicate-binding-guard.sh` MUST pass after the edits.

### FR-3 — Clientbound response wiring, including the missing `errors` table

- **FR-3.1** On success, `atlas-channel` sends `CashShopUseCouponDoneBody` carrying the
  cash items granted (as `CashInventoryItem` records), the resulting Maple Point balance,
  the packed cash-item refs, and `0` for meso.
- **FR-3.2** On failure, `atlas-channel` sends `CashShopUseCouponFailedBody` with the
  appropriate error key.
- **FR-3.3** `CashShopUseCouponFailedBody`
  (`libs/atlas-packet/cash/clientbound/shop_operation_body.go:269`) resolves its reason
  byte via `ResolveCode(l, options, "errors", message)`. **No tenant template today has
  an `errors` table on the `CashShopOperation` writer — the writer's `options` contains
  only `operations`.** This task MUST add an `errors` table to the `CashShopOperation`
  writer options in all ten templates, populated at minimum with the coupon error keys
  in FR-3.4, with per-version byte values derived from the IDB. Without it every coupon
  failure resolves to an unconfigured code.
- **FR-3.4** The following existing constants from
  `libs/atlas-packet/cash/clientbound/shop_operation_body.go` MUST be mapped to
  redemption outcomes:

  | Outcome | Constant | Key string |
  |---|---|---|
  | Code string matches nothing | `CashShopOperationErrorInvalidCouponCode` | `INVALID_COUPON_COUPON` |
  | Now is after the code's expiry | `CashShopOperationErrorCouponExpired` | `COUPON_EXPIRED` |
  | This account already redeemed this code | `CashShopOperationErrorCouponAlreadyUsed` | `COUPON_ALREADY_USED` |
  | Global max-uses exhausted | `CashShopOperationErrorCouponUsageLimit` | `COUPON_USAGE_LIMIT` |
  | Code disabled, or now is before its activation time | `CashShopOperationErrorCouponNotRegistered` | `COUPON_NOT_REGISTERED` |
  | Cash locker has no free slot for the item reward | `CashShopOperationErrorInventoryFull` | `INVENTORY_FULL` |
  | Anything else (saga failure, timeout, internal error) | `CashShopOperationErrorUnknown` | `UNKNOWN_ERROR` |

- **FR-3.5** `CashShopOperationErrorInvalidCouponCode` is declared with the key string
  `"INVALID_COUPON_COUPON"` (`shop_operation_body.go:91`) — apparently a typo for
  `INVALID_COUPON_CODE`. Whichever string is chosen, the constant and the ten templates'
  `errors` tables MUST agree exactly; a mismatch is silent (`ResolveCode` misses and the
  reason byte is wrong). Correcting the constant to `INVALID_COUPON_CODE` is preferred
  since nothing consumes it yet.

### FR-4 — Channel handler arm

- **FR-4.1** Add a `CashShopOperationUseCoupon = "USE_COUPON"` constant to
  `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_operation.go` and an
  `isCashShopOperation(...)` arm that decodes `ShopOperationUseCoupon` and calls the
  cash-shop processor's redemption request.
- **FR-4.2** The arm normalizes the submitted code (trim surrounding whitespace,
  uppercase) before dispatching.
- **FR-4.3** A code that fails a purely local sanity check (empty after trimming, longer
  than the column limit) short-circuits to `INVALID_COUPON_CODE` without a round trip.
- **FR-4.4** The redemption request carries the session's character id; the coupon
  service resolves the owning account id from it, because wallet balances are
  account-scoped while the packet arrives on a character session.

### FR-5 — Coupon domain in `atlas-cashshop`

- **FR-5.1** Add a `coupon` package under
  `services/atlas-cashshop/atlas.com/cashshop/` following the project's domain shape:
  immutable model with private fields + getters + Builder, `Interface` + `Impl`
  processor constructed via `NewProcessor(l, ctx, db)`, GORM entity + `Migration`,
  `Make(e Entity) (Model, error)` mapper, JSON:API resource registered in `main.go` via
  `AddRouteInitializer`.
- **FR-5.2** Register the new migrations in the `database.Connect(l,
  database.SetMigrations(...))` call in
  `services/atlas-cashshop/atlas.com/cashshop/main.go`.
- **FR-5.3** Codes are stored and looked up in **normalized** form: `TRIM`med and
  uppercased. Uniqueness is enforced on `(tenant_id, code)` over the normalized value.
- **FR-5.4** Validation order for a redemption attempt, first failure wins, so that the
  returned error is deterministic:
  1. code exists → else `INVALID_COUPON_CODE`
  2. code is active → else `COUPON_NOT_REGISTERED`
  3. now ≥ `starts_at` (when set) → else `COUPON_NOT_REGISTERED`
  4. now ≤ `expires_at` (when set) → else `COUPON_EXPIRED`
  5. account has no prior successful redemption → else `COUPON_ALREADY_USED`
  6. `redemption_count < max_uses` (when `max_uses` is set) → else `COUPON_USAGE_LIMIT`
  7. cash locker has free slots ≥ number of item rewards → else `INVENTORY_FULL`
- **FR-5.5** The max-uses counter MUST be decremented atomically. A conditional update
  (`UPDATE … SET redemption_count = redemption_count + 1 WHERE id = ? AND (max_uses IS
  NULL OR redemption_count < max_uses)` and check rows-affected) is required; a
  read-then-write is not acceptable. Two concurrent redemptions of a `max_uses = 1` code
  MUST result in exactly one success and one `COUPON_USAGE_LIMIT`.
- **FR-5.6** A redemption record is written for every **successful** redemption, keyed
  such that `(tenant_id, coupon_id, account_id)` is unique — this is the database-level
  enforcement of one-time-per-account, not merely the FR-5.4 check.

### FR-6 — Redemption orchestration

- **FR-6.1** Redemption is orchestrated by `atlas-saga-orchestrator` under a saga of type
  `cash_shop_operation` (`libs/atlas-saga/model.go:19`), so that a partially-applied
  redemption is compensated rather than left half-granted.
- **FR-6.2** Steps, in order: reserve the redemption (conditional counter increment +
  redemption record insert) → award wallet currency for each currency reward → award each
  cash-locker item → emit the cash-shop status event that drives the clientbound success
  packet.
- **FR-6.3** Compensations run in reverse: destroy granted cash assets, debit the awarded
  currency, release the reservation (decrement the counter and delete the redemption
  record). A failed saga MUST leave the code redeemable again and the account's balance
  and locker unchanged.
- **FR-6.4** Reuse `AwardCurrency` (`libs/atlas-saga/model.go:57`) for the wallet leg.
  New actions are needed for the reservation, the release compensation, and the
  cash-locker item grant; name them `reserve_coupon_redemption`,
  `release_coupon_redemption`, and `award_cash_shop_asset`, and register them alongside
  the existing cash-shop actions in `libs/atlas-saga/model.go:133-137`.
- **FR-6.5** The saga is keyed by a transaction id derived from
  `(tenant, account, coupon)` so a duplicate packet — a double-click on the Coupon tab's
  submit button — does not start a second saga.
- **FR-6.6** Saga failure or timeout surfaces to the player as `UNKNOWN_ERROR` via
  `CashShopUseCouponFailedBody`, never as silence.

> **Design concern (raised, and proceeding as directed).** Every reward type in this
> task's scope — wallet currency and cash-locker assets — is owned by `atlas-cashshop`
> alone. The existing purchase path (`cashshop/processor.go:98` `Purchase`) does exactly
> this pairing in a single local `database.ExecuteTransaction`, with no saga. Modelling
> coupon redemption as a saga therefore adds three new saga actions, a compensator, and
> an event-acceptance path to buy atomicity that one transaction already provides. The
> saga approach was explicitly chosen during the spec interview and is what this PRD
> specifies; it does become the clearly-correct design the moment a reward type outside
> `atlas-cashshop` (regular inventory items, mesos, experience) is added. See §9.

### FR-7 — Admin REST surface

- **FR-7.1** JSON:API CRUD for coupon codes on `atlas-cashshop`: create, list (paginated,
  filterable), read, update (reward bundle, window, max-uses, active flag), delete.
- **FR-7.2** Bulk generation: one call creates N codes sharing a reward definition, a
  window, and `max_uses = 1`, grouped under a batch. Generated strings are drawn from an
  unambiguous alphabet (no `O`/`0`, `I`/`1`/`L`) and are unique within the tenant;
  collisions are retried, not silently skipped, and the response reports the exact count
  created.
- **FR-7.3** Redemption history is readable per code and filterable by account.
- **FR-7.4** All endpoints are tenant-scoped from the request's tenant context; no
  endpoint accepts a tenant id in its body.

### FR-8 — Admin UI

- **FR-8.1** A Coupons page in `atlas-ui` (`services/atlas-ui/src/pages/CouponsPage.tsx`
  plus `coupons-columns.tsx` and `CouponDetailPage.tsx`, matching the existing
  list/columns/detail trio used by e.g. `AccountsPage.tsx`) listing codes with their
  status, reward summary, uses, and window.
- **FR-8.2** Create form with react-hook-form + Zod validation covering: code string
  (optional — blank means "generate"), reward rows (currency type + amount, or cash item
  serial/template + quantity), activation and expiry timestamps, max-uses, active flag.
- **FR-8.3** Bulk-generate dialog: reward definition + count, with the generated codes
  downloadable as CSV.
- **FR-8.4** Detail page shows the code's redemption history.
- **FR-8.5** Disable/enable toggles the active flag in place; delete is confirmed and
  blocked once redemptions exist.
- **FR-8.6** All POST/PATCH bodies use the JSON:API envelope
  (`bug_ui_jsonapi_envelope_required_for_input_handlers`).

## 5. API Surface

All endpoints are served by `atlas-cashshop` under its `/api/` prefix, JSON:API via
api2go, tenant resolved from request context.

### Coupons

- `GET /coupons` — list. Query filters: `filter[code]` (exact, normalized),
  `filter[active]`, `filter[batchId]`, `filter[expiresBefore]`, `filter[expiresAfter]`.
  Paginated per `docs/rest-pagination.md`.
- `GET /coupons/{id}` — read one.
- `POST /coupons` — create one. Attributes: `code` (optional), `active`, `startsAt`,
  `expiresAt`, `maxUses`, `description`, `rewards[]`.
- `PATCH /coupons/{id}` — update `active`, `startsAt`, `expiresAt`, `maxUses`,
  `description`, `rewards[]`.
- `DELETE /coupons/{id}` — delete. `409 Conflict` when the coupon has redemptions.

Reward attribute shape (one of):

```
{ "type": "CURRENCY", "currency": 1 | 2 | 3, "amount": 10000 }
{ "type": "CASH_ITEM", "serialNumber": 50200000, "quantity": 1 }
```

`currency` matches the existing `wallet.Model.Balance` convention: `1` = credit (NX),
`2` = Maple Points, anything else = prepaid.

### Batches

- `POST /coupon-batches` — bulk generate. Attributes: `count`, `prefix` (optional),
  `length`, `startsAt`, `expiresAt`, `rewards[]`, `description`. Response includes the
  batch id and every generated code.
- `GET /coupon-batches` / `GET /coupon-batches/{id}` — list / read, including a
  generated-vs-redeemed count.

### Redemptions

- `GET /coupons/{id}/redemptions` — history for one code.
- `GET /coupon-redemptions?filter[accountId]=` — history for one account.

### Errors

| Condition | Status |
|---|---|
| Duplicate normalized code on create | `409 Conflict` |
| Reward references an unknown commodity serial | `422 Unprocessable Entity` |
| `expiresAt` ≤ `startsAt` | `422 Unprocessable Entity` |
| `maxUses` < current `redemptionCount` on PATCH | `422 Unprocessable Entity` |
| Delete with redemptions present | `409 Conflict` |

Redemption itself is **not** a REST endpoint — it is driven by the packet path through
the saga. No public endpoint may grant rewards.

## 6. Data Model

Three new tables in the `atlas-cashshop` database, all tenant-scoped.

### `coupons`

| Column | Type | Notes |
|---|---|---|
| `id` | uuid | PK, generated in `BeforeCreate` |
| `tenant_id` | uuid | not null, indexed |
| `batch_id` | uuid | nullable, FK-ish to `coupon_batches.id` |
| `code` | varchar(32) | not null, **normalized** (trimmed, uppercased) |
| `description` | text | nullable, admin-facing |
| `active` | bool | not null, default true |
| `starts_at` | timestamptz | nullable; null = active immediately |
| `expires_at` | timestamptz | nullable; null = never expires |
| `max_uses` | int | nullable; null = unlimited |
| `redemption_count` | int | not null, default 0 |
| `rewards` | jsonb | not null; array of reward objects (§5) |
| `created_at` / `updated_at` | timestamptz | |

- Unique index on `(tenant_id, code)`. Because `code` is stored normalized, this is the
  case-insensitive uniqueness guarantee — do **not** rely on a functional index over a
  raw value.
- Index on `(tenant_id, batch_id)`.
- Rewards are stored as `jsonb` rather than a child table: they are always read and
  written as a whole bundle, never queried by reward attribute.

### `coupon_batches`

| Column | Type | Notes |
|---|---|---|
| `id` | uuid | PK |
| `tenant_id` | uuid | not null, indexed |
| `description` | text | nullable |
| `requested_count` | int | not null |
| `generated_count` | int | not null |
| `created_at` | timestamptz | |

### `coupon_redemptions`

| Column | Type | Notes |
|---|---|---|
| `id` | uuid | PK |
| `tenant_id` | uuid | not null, indexed |
| `coupon_id` | uuid | not null, indexed |
| `account_id` | uint32 | not null |
| `character_id` | uint32 | not null — which character submitted it |
| `transaction_id` | uuid | not null — the saga transaction id |
| `rewards_granted` | jsonb | not null — snapshot of what was actually granted |
| `redeemed_at` | timestamptz | not null |

- **Unique index on `(tenant_id, coupon_id, account_id)`** — the database-level
  one-time-per-account rule.
- Index on `(tenant_id, account_id)` for per-account history.
- `rewards_granted` is a snapshot, not a reference, so later edits to the coupon's reward
  bundle do not rewrite history.

### Migration notes

- All three go through `AutoMigrate` in
  `services/atlas-cashshop/atlas.com/cashshop/main.go`'s `database.SetMigrations(...)`
  list, alongside `wallet.Migration`, `wishlist.Migration`, `compartment.Migration`,
  `asset.Migration`, `outboxlib.Migration`.
- No backfill: all three tables start empty. No existing table is altered.

## 7. Service Impact

| Service / lib | Change |
|---|---|
| `libs/atlas-packet` | New `cash/serverbound/shop_operation_use_coupon.go` + per-version fixture tests. Possible key-string correction at `cash/clientbound/shop_operation_body.go:91`. |
| `libs/atlas-saga` | Three new `Action` constants (`reserve_coupon_redemption`, `release_coupon_redemption`, `award_cash_shop_asset`) and their payload types. |
| `services/atlas-channel` | New `USE_COUPON` arm in `socket/handler/cash_shop_operation.go`; success/failure announcement paths; consumption of the coupon status event in `kafka/consumer/cashshop/`. |
| `services/atlas-cashshop` | New `coupon` domain (model, entity, processor, provider, resource, REST), three migrations, saga step handlers for the reserve/release/award-asset actions, new status event types. |
| `services/atlas-saga-orchestrator` | Handler + compensator arms for the three new actions; the coupon saga definition. |
| `services/atlas-configurations` | `USE_COUPON` key added to the `CashShopOperationHandle` `operations` table in all ten templates; missing operations backfilled for `gms_87`/`gms_92`/`gms_95`/`jms_185`; new `errors` table on the `CashShopOperation` writer in all ten. |
| `services/atlas-ui` | New Coupons list/detail pages, columns file, create form, bulk-generate dialog, API client + types. |
| `docs/packets` | Operation-registry entry and coverage-matrix cells for the new serverbound op. |

No service is added, so no `services.json` / `docker-bake.hcl` / `go.work` / k8s overlay
registration is required, and `tools/service-registration-guard.sh` is unaffected.

## 8. Non-Functional Requirements

**Multi-tenancy.** Every coupon table carries `tenant_id`; every query is scoped through
`tenant.MustFromContext(ctx)`. A code minted for one tenant MUST NOT be redeemable in
another, and the `(tenant_id, code)` unique index MUST allow the same string to exist in
two tenants independently.

**Concurrency.** Two simultaneous redemptions of a `max_uses = 1` code, and two
simultaneous redemptions of the same code by the same account, MUST each produce exactly
one success. The `max_uses` guard is a conditional `UPDATE … WHERE redemption_count <
max_uses` with a rows-affected check (FR-5.5); the per-account guard is the unique index
(FR-5.6). Neither may be implemented as a read-then-write.

**Security.** Coupon codes are secrets. Redemption MUST NOT be exposed as a public REST
endpoint — the packet path is the only redemption trigger. The admin CRUD endpoints sit
behind the same authentication as the rest of `atlas-cashshop`'s admin surface. Failed
redemption attempts MUST NOT leak whether a code exists but is expired versus does not
exist at all beyond what the client's error enum already distinguishes.

**Rate limiting.** Brute-forcing short codes is the obvious attack. Failed attempts are
counted per account; after a configurable threshold within a window, further attempts
short-circuit to `INVALID_COUPON_CODE` without a database lookup. The threshold and
window are tenant configuration, not hard-coded.

**Observability.** Every redemption attempt logs tenant, account, character, normalized
code, and outcome at `Info`. Saga failures and compensations log at `Error` with the
transaction id. Counters: attempts by outcome, successful redemptions, compensated
redemptions.

**Performance.** Code lookup is a single indexed point read on `(tenant_id, code)`. The
redemption path adds no N+1 queries; the reward bundle is read from the same row.

**Correctness gates.** `go test -race ./...`, `go vet ./...`, `tools/lint.sh --check`,
`tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`,
`tools/template-opcode-order-guard.sh`, `tools/template-duplicate-binding-guard.sh`, and
`docker buildx bake` for every service whose `go.mod` is touched.

## 9. Open Questions

1. **Saga versus local transaction (FR-6 design concern).** The chosen orchestration is a
   saga, but every reward type in scope lives inside `atlas-cashshop`, whose existing
   purchase path achieves the same atomicity in one local transaction. Should the design
   phase re-evaluate this trade-off, or is the saga a deliberate investment against the
   near-term addition of regular-inventory and meso rewards? **This is the single
   highest-leverage question for `/design-task`.**
2. **Serverbound mode byte and body layout per version.** Unknown until the ten IDBs are
   read. It is possible that some legacy versions (`gms_48`, `gms_61`) have no Coupon tab
   at all, in which case those columns become `n-a` rather than implemented — that
   determination is part of FR-1.2 and may shrink the ten.
3. **`errors` table byte values.** The `CashShopOperation` writer's error enum is
   completely unmapped today. Deriving the full per-version enum may be a larger job than
   this task needs; the minimum is the seven keys in FR-3.4, but the design phase should
   decide whether to derive the whole table while the IDBs are open.
4. **Does the client echo the code back?** If `UseCouponFailed`'s reason byte alone is
   insufficient for the client to re-render the Coupon tab (e.g. it expects the field
   cleared), an additional arm may be required. To be confirmed against the client.
5. **Maple Point balance in the success arm.** `UseCouponDone` carries a `maplePoint`
   field. Whether the client treats it as a delta or an absolute balance must be
   confirmed from the decompilation before the handler populates it.
6. **Cash locker capacity check timing.** FR-5.4 step 7 checks free slots before the
   saga starts; the locker could fill between the check and the grant. Should the grant
   step re-check and fail the saga, or should the check move entirely into the grant
   step?
7. **Should `atlas-ui` surface redemption history globally**, or only per-code and
   per-account as specified in FR-7.3?

## 10. Acceptance Criteria

Packet layer:

- [ ] `libs/atlas-packet/cash/serverbound/shop_operation_use_coupon.go` exists with
      `Encode` and `Decode`, an immutable struct, and a `packet-audit:fname` comment
      naming the client function each version's layout was derived from.
- [ ] Round-trip fixture tests pass for every version determined applicable in FR-1.2.
- [ ] The op is registered in the operation registry and its matrix cells are promoted
      through `/verify-packet`; `packet-audit` matrix and fname-doc checks exit 0.
- [ ] No previously-verified matrix cell changed wire behaviour.

Templates:

- [ ] All ten cash-shop templates carry a `USE_COUPON` key in the
      `CashShopOperationHandle` `operations` table with a version-correct mode byte.
- [ ] `gms_87`, `gms_92`, `gms_95`, and `jms_185` operations tables are no longer empty or
      short — every cash-shop operation `atlas-channel` handles is routable on them.
- [ ] All ten templates carry an `errors` table on the `CashShopOperation` writer
      containing the seven keys in FR-3.4.
- [ ] `tools/template-opcode-order-guard.sh` and
      `tools/template-duplicate-binding-guard.sh` exit 0.

Domain and orchestration:

- [ ] `coupons`, `coupon_batches`, and `coupon_redemptions` tables migrate cleanly on a
      fresh database and on an existing one.
- [ ] Unique index on `(tenant_id, code)` and on `(tenant_id, coupon_id, account_id)`
      exist and are exercised by tests.
- [ ] A race test proves two concurrent redemptions of a `max_uses = 1` code yield exactly
      one success and one `COUPON_USAGE_LIMIT`.
- [ ] A race test proves two concurrent redemptions of the same code by the same account
      yield exactly one success and one `COUPON_ALREADY_USED`.
- [ ] A forced failure in the item-grant step leaves the wallet balance unchanged, the
      locker unchanged, `redemption_count` back at its prior value, and the code
      redeemable again.
- [ ] The seven validation outcomes in FR-5.4 each map to the error key in FR-3.4, covered
      by a test per outcome.
- [ ] Codes differing only in case or surrounding whitespace resolve to the same coupon.

End to end:

- [ ] On a live v83 tenant, entering a valid code in the Cash Shop Coupon tab credits the
      wallet and/or the cash locker and updates the open Cash Shop window without a
      relog.
- [ ] Entering the same code again on the same account shows the "already used" client
      message.
- [ ] Entering a garbage code shows the "invalid code" client message.
- [ ] Entering an expired code shows the "expired" client message.

Admin surface:

- [ ] `POST /coupons` with an explicit code creates it; a duplicate returns `409`.
- [ ] `POST /coupon-batches` with `count: 500` returns 500 unique codes and a batch whose
      `generated_count` is 500.
- [ ] `DELETE /coupons/{id}` returns `409` once a redemption exists.
- [ ] The `atlas-ui` Coupons page lists, filters, creates, bulk-generates (with CSV
      download), toggles active, and shows redemption history.
- [ ] All UI mutations send JSON:API-enveloped bodies.

Build and verification:

- [ ] `go test -race ./...` and `go vet ./...` clean in every changed module.
- [ ] `tools/lint.sh --check`, `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`
      clean from the repo root.
- [ ] `docker buildx bake atlas-cashshop atlas-channel atlas-saga-orchestrator
      atlas-configurations` succeeds from the worktree root.
- [ ] `atlas-ui` builds (`npm run build`, which type-checks tests) and its tests pass.
