# MTS (MapleStory Trade System) — Design

Version: v1
Status: Proposed
Created: 2026-06-17
Inputs: `prd.md` (approved), `research-scaffold.md` (IDA-verified protocol facts)

---

## 1. Guiding constraints

Three constraints shape every decision below; everything else is subordinate.

1. **Single-custody / dupe-safety is the architecture, not a feature.** An
   in-transit item lives in exactly one of
   `inventory → mts:listed → mts:holding → inventory(buyer)` at every instant.
   Every cross-service item or currency move is saga-coordinated
   (reserve→confirm→commit + compensation) and idempotent by `transactionId`.
   No optimistic or direct inventory writes anywhere.

2. **Mirror existing patterns; invent nothing new at the framework level.** The
   service copies the `atlas-cashshop` shape; custody moves copy the
   `TransferToCashShop` / `WithdrawFromCashShop` composite-saga family; currency
   moves reuse the cash-shop wallet via `ADJUST_CURRENCY`; the channel wiring
   copies the `EnterCashShop` migration and the messenger/storage mode-dispatcher
   handlers; config copies the `atlas-tenants` generic JSONB resource pattern. The
   one genuinely new asset is the `atlas-mts` service itself and four new saga
   actions.

3. **Everything version- and tenant-parameterized.** Opcodes, the per-version
   `operations` mode tables, and every economic knob come from tenant config.
   Behavior is correct across gms_v83/v84/v87/v95 and jms_v185 (jms limited to its
   defined opcode set). Data is `(tenant_id, world_id)` scoped.

This design assumes the patterns mapped during grounding; the key file:line
anchors are inlined where a decision depends on them.

---

## 2. Phase 0 — serverbound packet verification (prerequisite gate)

Phase 0 is a hard gate: **no in-game MTS flow is coded until its packet cells
promote** in `docs/packets/audits/STATUS.md`. task-096 already byte-verified the
clientbound result writers (`MTS_OPERATION` / `MTS_OPERATION2`,
`libs/atlas-packet/field/clientbound/mts_operation.go`,
`services/atlas-channel/.../socket/writer/mts_operation2.go`). Phase 0 verifies
the **serverbound** request packets.

Scope (gms_v83/v84/v87/v95; jms_v185 where the opcode exists):

- Standalone: `ENTER_MTS`, `ITC_STATUS_CHARGE`, `ITC_QUERY_CASH_REQUEST`.
- `ITC_OPERATION` — **every mode arm gets its own byte fixture.** Per the
  dispatcher-family rule ([[feedback_dispatcher_mode_byte_is_false_pass]]),
  enumerating mode bytes is a false pass. Arms: register-fixed-sale (mode 2),
  sale-current (3), register-auction (0x12), buy, buy-now-on-auction, place-bid,
  set/buy/delete zzim, view/buy/cancel/register wish, cancel-sale,
  move-ITC-purchase-LtoS (take-home), change-category, change-category-sub,
  change-page.

Phase 0 is executed with the `packet-verifier` agent / `/verify-packet` playbook,
one cell per packet × version, batched per IDB
([[reference_packet_audit_serverbound_verification]]). The three verified arms in
the scaffold (§4) are re-checked, not assumed.

**Phase 0 also answers two open questions before any service code is written:**

- **§9.1 real-time bidding:** Does the verified set include a *server-pushed*
  auction-state / outbid packet (a clientbound mode the server can emit
  unsolicited, distinct from a response to a request)? The scaffold lists
  `MTS_OPERATION` cases 60–62 as `BidAuctionFailed` / "auction-state notices" —
  Phase 0 pins whether any of these is a push. **Decision rule:** if a push
  packet exists → implement live outbid notifications (§5.6 "live" path); if not →
  ship escrow-highest-bid-wins-at-expiry with buy-now early-end (§5.6 "escrow"
  path). The service is designed so this choice changes only the notification
  edge, never the custody or settlement core.
- **§9.4 jms scope:** the clientbound result packets are version-absent (⬜) for
  jms; Phase 0 records the supported jms surface. jms gets whatever its opcode set
  defines and no more.

Deliverable: matrix cells promoted + an updated decision note appended to this
design recording the §9.1 outcome.

---

## 3. `atlas-mts` service — shape & data model

### 3.1 Service skeleton

Mirror `atlas-gachapons` (smallest recent service):

```
services/atlas-mts/atlas.com/mts/
  main.go                 # DB AutoMigrate + REST server + Kafka consumers + expiration ticker
  go.mod                  # module: atlas-mts
  listing/                # Listing domain (model/entity/administrator/processor/provider/rest/resource)
  bid/                    # Bid domain
  holding/                # Take-home holding domain
  wish/                   # Wish-list (zzim) domain
  configuration/          # tenant mts-config registry (lazy cache, mirrors cashshop/configuration)
  kafka/
    consumer/             # mts commands + saga custody commands + wallet/compartment status (for saga acks if needed)
    producer/             # mts status events
    message/              # Command[E]/StatusEvent[E] envelopes + Buffer/Emit
  task/                   # expiration ticker (mirrors atlas-asset-expiration task/periodic.go)
  rest/                   # base handler
  logger/
```

Unlike `atlas-gachapons` (REST-only), `atlas-mts` registers Kafka consumers and
producers (mirroring `atlas-cashshop`'s `kafka/` layout) **and** a periodic
expiration ticker.

### 3.2 Entities (GORM, AutoMigrate)

Every entity: UUID PK + `tenant_id`, `(tenant_id, id)` unique index, **explicit
name-keyed columns** (avoids the slug-only-PK collision and binary-COPY
column-order bug families — [[bug_tenant_table_slug_only_pk_collides]],
[[bug_baseline_restore_column_order_drift]]). World scoping via `world_id`.

**`Listing`** (`listing/entity.go`)
- `id`, `tenant_id`, `world_id`, `seller_id`, `seller_name`
- `sale_type` (`fixed` | `auction`), `state` (`active` | `sold` | `cancelled` | `expired`)
- item snapshot: `template_id`, `quantity`, and the full equip stat block when the
  item is equipment (str/dex/int/luk/hp/mp/watk/matk/wdef/mdef/acc/avoid/hands/
  speed/jump, upgrade slots, level, item level, item exp, ring id, vicious count,
  flags) — stored as explicit columns, not a JSON blob, to keep restore-safe.
- `list_value` (NX), `buy_now_price` (NX, nullable), `commission_rate` (captured at
  list time), `category`, `sub_category`
- auction: `ends_at`, `current_bid`, `high_bidder_id`, `min_increment`
- `created_at`, `updated_at`
- Indexes: `(tenant_id, world_id, state, category)` (browse);
  `(tenant_id, seller_id, state)` (my listings);
  `(tenant_id, world_id, ends_at)` (expiration sweep).

**`Bid`** (`bid/entity.go`)
- `id`, `tenant_id`, `listing_id`, `bidder_id`, `amount`, `escrow_txn_id`,
  `state` (`held` | `released` | `won`), `created_at`.

**`Holding`** (`holding/entity.go`) — take-home custody
- `id`, `tenant_id`, `world_id`, `owner_id`, item snapshot (same columns as
  Listing), `origin` (`purchased` | `unsold` | `cancelled` | `expired`),
  `created_at`.

**`WishEntry`** (`wish/entity.go`)
- `id`, `tenant_id`, `character_id`, `item_id` / criteria, `created_at`.

The item snapshot is duplicated across `Listing` and `Holding` rather than
normalized into a shared `mts_item` table: the snapshot is immutable once captured
and a listing→holding move is a state/row transition we want to keep as a single
local DB transaction (§5.4), so co-locating the columns avoids a join and a
second table to keep custody-consistent.

### 3.3 Immutable models + builders

Each domain follows the project pattern: private-field model + getters + Builder,
Processor `Interface`+`Impl` with `NewProcessor(l, ctx, db)`, pure `Method(mb)` and
side-effecting `MethodAndEmit()`. No `*_testhelpers.go` — Builder-based test setup
only.

---

## 4. Custody model — the dupe-safety core

```
seller inv ──list──▶ mts:listed ──buy/win──▶ mts:holding(buyer) ──take-home──▶ buyer inv
                          │                         ▲
                          └──cancel / expire────────┘ (to mts:holding(seller))
```

**atlas-mts is the sole custodian for the middle of the journey.** The buyer never
receives the item into inventory on purchase — it lands in `mts:holding` and is
pulled on demand (LtoS). Four custody transitions cross a service boundary and so
are sagas; two are internal to atlas-mts and so are single local DB transactions.

| Transition | Crosses services? | Mechanism |
|---|---|---|
| inv → listed (list) | yes (inventory) | **saga** `TransferToMts` |
| listed → holding(buyer) + currency (buy/win) | yes (inventory custody stays in mts; wallet in cashshop) | **saga** `MtsSettlePurchase` |
| holding → inv (take-home) | yes (inventory) | **saga** `WithdrawFromMts` |
| listed → holding(seller) (cancel/expire) | no — atlas-mts local | **local DB tx** (row state flip) |
| bid escrow / release | yes (wallet) | **saga** `MtsBidEscrow` (single-step wallet adjust) |

The key insight that bounds the blast radius: **cancel and expiration never leave
atlas-mts** — they move a row from `listed` to `holding` inside one Postgres
transaction, so they cannot half-complete and cannot be raced by anything except a
concurrent purchase, which is resolved by the authoritative listing `state` (§4.2).

### 4.1 New saga actions (libs/atlas-saga + orchestrator)

Following the `TransferToCashShop` precedent, add to `libs/atlas-saga/model.go`
(Action constants) and `payloads.go` (high-level payloads), with internal
expansion payloads in the orchestrator's `saga/model.go`:

- **`TransferToMts`** (composite) → expands to `ReleaseFromCharacter` (existing
  inventory release) + **`AcceptToMtsListing`** (new). atlas-mts creates the
  `Listing` row in `active` state only on `AcceptToMtsListing`; the item is gone
  from inventory first. Compensation: `AcceptToCharacter` (re-grant) if the accept
  fails.
- **`WithdrawFromMts`** (composite) → **`ReleaseFromMtsHolding`** (new) +
  `AcceptToCharacter` (existing inventory create-asset). Holding row soft-deleted
  on release; re-created on compensation.
- **`AcceptToMtsListing`** / **`ReleaseFromMtsHolding`** — atomic single-service
  steps dispatched as Kafka commands to atlas-mts, exactly mirroring how
  `AcceptToCashShop` / `ReleaseFromCashShop` dispatch to atlas-cashshop's
  compartment consumer (`saga-orchestrator/cashshop/processor.go` →
  `COMMAND_TOPIC_CASH_COMPARTMENT`). atlas-mts gets a `COMMAND_TOPIC_MTS_CUSTODY`
  consumer that performs the row transition and emits an
  `EVENT_TOPIC_MTS_CUSTODY_STATUS` ack carrying `transactionId`.
- **`MtsSettlePurchase`** (composite, the money-moving op) → a single saga whose
  ordered steps are:
  1. `AdjustCurrency(buyer, prepaid, −markedUpPrice)` (cashshop wallet) — debit
     first, so a failure grants nothing.
  2. `AdjustCurrency(seller, points, +listValue)` (cashshop wallet).
  3. `MtsMoveListingToHolding(buyer)` — atlas-mts marks the listing `sold` and
     creates the buyer `holding` row, in one local DB tx.
  Commission = `markedUpPrice − listValue` is simply never credited to anyone → it
  is the sink. Compensation walks back: re-credit buyer, debit seller, restore
  listing to `active`.

Each new action needs: a handler in the orchestrator `saga/handler.go` switch, an
entry in `saga/event_acceptance.go` (which event kind completes/fails it), a
compensator inverse in `saga/compensator.go`, and composite expansion in
`saga/processor.go` for the two `Transfer*`/`Withdraw*` actions. This is the same
set of touch-points the cash-shop family already occupies.

### 4.2 Idempotency & race resolution

- Every saga step is keyed by `transactionId`. Replayed deliveries are no-ops:
  atlas-mts custody commands check whether the target row is already in the
  destination state for that `transactionId` before acting (the orchestrator
  already drops duplicate step-completion events —
  `saga-orchestrator/saga/processor.go` idempotency guard).
- **Take-home replay** is a no-op: `ReleaseFromMtsHolding` soft-deletes by
  `holding.id`; a second delivery finds the row already released for that
  `transactionId` and acks without re-granting.
- **Cancel-vs-buy** resolves on the authoritative `Listing.state`. Cancel is a
  conditional `UPDATE ... WHERE state='active'`; a concurrent `MtsSettlePurchase`
  flips `state='sold'` under the same row lock. Exactly one wins; the loser's saga
  fails cleanly and compensates. No timing dependency.
- **List-without-remove** is impossible: the item is released from inventory
  (`ReleaseFromCharacter`) *before* `AcceptToMtsListing` creates the listing; if
  the accept fails, compensation re-grants — exactly one copy at every instant.

### 4.3 Saga timeout scaling

Sagas with data-driven step counts must scale their timeout
([[bug_preset_creation_saga_flat_timeout]]). MTS sagas are short and
fixed-length (2–3 steps), so a small fixed timeout is safe — but the builder MUST
set it explicitly via `base + perStep*N` rather than relying on the default, and
the design records `N` per saga type so a future multi-step extension doesn't
inherit a flat timeout.

---

## 5. Functional flows

### 5.1 Entry / migration (`ENTER_MTS`)
A channel serverbound handler mirroring `CashShopEntryHandleFunc`
(`socket/handler/cash_shop_entry.go`): save character, leave channel/map, mark
entered, then announce the initial browse page + the character's active listings +
their holding + wallet (`MTS_OPERATION2`). Gated on a configurable min level
(default 10) and the same map/event eligibility as cash-shop entry.

### 5.2 List — fixed price & auction
Channel handler decodes `ITC_OPERATION` mode (2 fixed / 0x12 auction), validates
client-supplied price against the **server-authoritative** floor (110 NX) and
config (cap, level), then initiates `TransferToMts` plus a meso debit for the
listing fee. **Listing fee** is charged via the existing `AwardMesos` saga action
with a negative amount (the storage-withdraw-fee precedent,
`storage_operation.go` `handleRetrieveAsset`) — this resolves PRD §9.5: character
meso is owned by atlas-character and debited through the saga, no new command.
Auction adds `buy_now_price`, `ends_at` (duration validated to 24–168h, 1h step),
`min_increment`.

### 5.3 Browse / search / paginate
Pure atlas-mts REST reads over the browse index, scoped `(tenant_id, world_id)`,
excluding `holding`-state items, paginated (default 16). Search by item id/name or
seller name. No saga.

### 5.4 Buy / buy-now
Channel validates buyer NX Prepaid ≥ `list_value × (1 + commission)` (read from
wallet), then initiates `MtsSettlePurchase` (§4.1). Item lands in buyer holding.

### 5.5 Cancel / expire
Both are atlas-mts-local row transitions (`active`→`holding(seller)`), guarded by
`WHERE state='active'`. Cancel is seller-initiated (REST/channel); expire is the
ticker (§7). No cross-service saga; emits an `EVENT_TOPIC_MTS_STATUS` notice.

### 5.6 Auction bidding — two paths, chosen by Phase 0

The custody/settlement core is identical to buy-now; only the notification edge
differs.

- **Bid:** `MtsBidEscrow` saga — `AdjustCurrency(bidder, prepaid, −bid)`; on
  success record the `Bid` row `held` and update `Listing.current_bid /
  high_bidder_id` (local tx under the listing row lock; bid must exceed
  `current_bid + min_increment`).
- **Outbid:** release prior escrow via `AdjustCurrency(prior, prepaid, +amount)`,
  mark that `Bid` `released`.
- **Settle at expiry / buy-now:** the high bid is already escrowed → credit seller
  points, mark bid `won`, move custody to winner holding, retain commission. No
  bids → listing returns to seller holding (the `expire` path).
- **Escrow path (default if no push packet):** correct and complete without any
  server push — the bidder simply doesn't get a live outbid toast.
- **Live path (only if Phase 0 finds a push packet):** on outbid, the channel
  emits the verified `MTS_OPERATION` push mode to the prior bidder. Anti-snipe
  window extension is a config knob, off by default.

Escrowing the **marked-up** amount at bid time keeps settlement pure bookkeeping
and prevents an underfunded winner.

### 5.7 Wish-list (zzim)
atlas-mts `WishEntry` CRUD + buy-from-wish (which routes into the buy flow). This
is the *only* saved-items mechanism; Cosmic's "cart" is not modeled
(research-scaffold §1). No saga for add/view/remove.

### 5.8 Take-home (LtoS)
`WithdrawFromMts` saga (§4.1) — holding → chosen inventory slot, idempotent.

### 5.9 Wallet / recharge
`ITC_QUERY_CASH_REQUEST` → read the two-bucket cash-shop wallet (Prepaid + Points;
`wallet/model.go` `Prepaid()`/`Points()`) → `MTS_OPERATION2` (2× i32).
`ITC_STATUS_CHARGE` re-opens the existing NX recharge hook (no new currency logic).

---

## 6. atlas-channel wiring

- **Serverbound handlers**, each with a validator (`LoggedInValidator`, except a
  `NoOpValidator` on the bodiless `ITC_STATUS_CHARGE`): `ENTER_MTS`,
  `ITC_STATUS_CHARGE`, `ITC_QUERY_CASH_REQUEST`, `ITC_OPERATION`. **Every handler
  entry needs a validator** or `BuildHandlerMap` silently drops it
  ([[bug_socket_handler_missing_validator_silently_dropped]]). `ITC_OPERATION` is
  a mode dispatcher reading its sub-op from the tenant `operations` table, mirror
  of `MessengerOperationHandleFunc` (`isMessengerShopOperation` resolves
  `options["operations"][KEY]`).
- **Clientbound writers**: `MTS_OPERATION` (mode dispatcher, cases 21–62, already
  built in task-096) and `MTS_OPERATION2` (2× i32). Migration handler mirrors
  `EnterCashShop`.
- **Per-version `operations` mode tables** seeded into tenant config for **all
  five** versions. Modes are version-dependent (non-uniform like the opcode
  table); a missing per-version table makes `ResolveCode` return 99 and crashes
  the client ([[bug_operations_mode_tables_missing_v87_v95_jms]]). Tables are
  populated from each version's dispatcher switch, IDA-verified in Phase 0 — not
  copied from v83.
- Channel handlers initiate sagas via the existing
  `channel/saga` producer (`Create(saga.Saga{...})` → `COMMAND_TOPIC_SAGA`), the
  same path storage/cash-shop use.

---

## 7. Expiration ticker

Mirror `atlas-asset-expiration`'s `task/periodic.go` (`PeriodicTask` with a
`time.Ticker` + `stopCh`, interval from env, registered for teardown). Each tick
queries `(tenant_id, world_id, ends_at < now, state='active')` per tenant and runs
the local cancel-equivalent transition to seller holding. The sweep is **bounded
and logged** — swept counts emitted, never silently truncated
([[feedback_no_todos_in_deliverables]] / NFR 8.3). Tenant context per iteration is
reconstructed as the expiration service does (`tenant.WithContext`).

Unlike `atlas-asset-expiration` (which iterates in-memory sessions), the MTS ticker
is DB-driven: it must sweep listings whose seller is offline. It enumerates active
tenants (from the config registry / a tenant list) and queries each tenant's
listings.

---

## 8. atlas-tenants config resource (`mts-config`)

New generic JSONB resource `"mts-configs"` following the routes/vessels precedent —
touch-points: `configuration/rest.go` (RestModel + Transform/Extract + JsonData),
`processor.go` (+ provider/seed methods), `resource.go` (handlers + `RegisterRoutes`),
`kafka.go` (command/event constants + bodies), `seed.go`, `mock/processor.go`,
`rest/handler.go` (`ParseMtsConfigId`). The generic `Model`/`Entity` need no change.

Config payload (the §4.12 knobs): `listingFee` (5000 meso), `commissionRate`
(0.10, buyer-markup model), `maxActiveListings` (10), `minLevel` (10),
`auctionMinHours` (24), `auctionMaxHours` (168), `priceFloor` (110 NX), `pageSize`
(16), `minBidIncrement`. Loaded by atlas-mts via a lazy per-tenant cache mirroring
`cashshop/configuration/registry.go` (`GetTenantConfig`), degrading to defaults on
miss. The MTS socket opcode/handler/writer entries and per-version `operations`
tables are seeded into the channel's socket config for every templated version.

> **Live-config caveat:** existing tenants do not retroactively receive new
> handler/writer opcodes from a seed template; the live tenant config must be
> patched and the channel restarted
> ([[bug_new_opcodes_not_in_live_tenant_config]]). This is an operational step in
> the rollout checklist, not code.

---

## 9. Kafka topics & messages

New (atlas-mts owns):
- `COMMAND_TOPIC_MTS` — CreateListing, CancelListing, PlaceBid, Buy, TakeHome,
  ExpireListing, RegisterWish, RemoveWish.
- `EVENT_TOPIC_MTS_STATUS` — ListingCreated, ListingCancelled, BidPlaced, Outbid,
  ListingSold, ListingExpired, ItemMovedToHolding, ItemTakenHome, WishAdded,
  WishRemoved (all carry `transactionId`, `worldId`).
- `COMMAND_TOPIC_MTS_CUSTODY` / `EVENT_TOPIC_MTS_CUSTODY_STATUS` — the saga custody
  channel (`AcceptToMtsListing` / `ReleaseFromMtsHolding` + acks), modeled on
  `COMMAND_TOPIC_CASH_COMPARTMENT`.

Reused: `COMMAND_TOPIC_WALLET` / `EVENT_TOPIC_WALLET_STATUS` (`ADJUST_CURRENCY`,
`amount int32` signed, `currencyType` 2=points/3=prepaid) for settlement and bid
escrow; `COMMAND_TOPIC_COMPARTMENT` / `EVENT_TOPIC_COMPARTMENT_STATUS`
(`REQUEST_RESERVE`/`CONSUME`/`RELEASE`/`CREATE_ASSET`) for inventory custody;
`COMMAND_TOPIC_SAGA` for saga initiation; `AwardMesos` (via saga) for the listing
fee.

All envelopes use the generic `Command[E any]` / `StatusEvent[E any]` + `Buffer`/
`Emit` pattern from `kafka/message/message.go`.

---

## 10. atlas-ui

- **Tenant config page** for the §4.12 economic knobs (react-hook-form + Zod over
  the `mts-config` JSON:API resource). POSTs use the JSON:API envelope
  (`{data:{type,attributes}}`) — bare bodies 400
  ([[bug_ui_jsonapi_envelope_required_for_input_handlers]]).
- **Read-only listings browser** per world (search/paginate) over atlas-mts REST.
- Build gate: `npm run build` type-checks tests too; update test call sites in the
  same commit; gate on build+test + no-new-lint-errors, not clean lint
  ([[reference_atlas_ui_build_typechecks_tests]],
  [[reference_atlas_ui_npm_nvm_and_lint_baseline]]).

---

## 11. Service registration & deploy

- `.github/config/services.json` — add the `atlas-mts` entry.
- `docker-bake.hcl` `go_services` — add `"atlas-mts"` (hand-synced; HCL can't read
  JSON — [[reference_docker_bake_hand_synced]]).
- `go.work` — add `./services/atlas-mts/atlas.com/mts`.
- **No new shared lib** is anticipated (saga actions/payloads go in the existing
  `libs/atlas-saga`), so **no Dockerfile COPY edits**. If that changes, add the two
  COPY lines + go.work line per CLAUDE.md.
- `deploy/k8s/base/atlas-mts.yaml` — mirror `atlas-gachapons.yaml` (Deployment +
  Service, `DB_NAME=atlas-mts`, `atlas-env`). **No new socket ports**: MTS is a
  channel-migrated stage (cash-shop model), so no login/channel port additions
  ([[bug_new_version_lb_socket_ports]] does not apply).
- Readiness probe path must be `/api/readyz` if a readiness mount is added (base
  path is `/api/` — [[bug_readiness_probe_path_under_api_basepath]]).

---

## 12. Alternatives considered

**A. Custody home: cash-shop compartment vs atlas-mts-owned tables.**
*Rejected:* parking listed items in the cash-shop compartment (reusing
`AcceptToCashShop`). It would avoid two new saga actions, but it breaks the
single-custody invariant's clarity (the cash-shop compartment is the player's
*own* cash inventory; a listed item is *not* the seller's anymore) and entangles
MTS state machine with cash-shop semantics. *Chosen:* atlas-mts owns
`listed`/`holding` as first-class states. The cost (two new custody saga actions)
is small and the boundary stays clean.

**B. Wallet: extract a shared `atlas-wallet` lib/service now vs route through
cash-shop.** *Chosen (per PRD):* route MTS currency through atlas-cashshop's
existing `ADJUST_CURRENCY`. It is a deliberate design smell — the wallet logically
belongs to neither service — but extraction is a cross-cutting refactor that would
balloon this task. *Follow-up (PRD §9.2):* file a wallet-extraction task after
task-102. This is the same "audit existing libs before a new one" discipline
([[feedback_audit_existing_libs_before_new_module]]) resolved in favor of reuse.

**C. Cancel/expire as saga vs local transaction.** *Chosen:* local DB transaction.
The transition never leaves atlas-mts, so a distributed saga would add latency and
failure surface for nothing. The only race (cancel-vs-buy) is resolved by the
authoritative listing state under a row lock — strictly simpler and safer than a
two-phase cross-service dance.

**D. Real-time bidding: build live-push speculatively vs gate on Phase 0.**
*Chosen:* gate. The escrow-at-expiry path is a complete, correct auction on its
own; live push is a notification enhancement layered on *only* if Phase 0 proves a
server-push packet exists. Building a broadcast path for an unverified packet would
be the "passes on enumeration" false pass this project explicitly bans.

**E. Settlement ordering: debit-buyer-first vs grant-custody-first.** *Chosen:*
debit buyer → credit seller → move custody. Debiting first means a mid-saga crash
never grants an item without payment; compensation re-credits. The reverse ordering
risks the canonical grant-before-debit dupe vector.

---

## 13. Open questions — resolution

| PRD §9 | Resolution in this design |
|---|---|
| 9.1 real-time bidding | **Gated on Phase 0** (§2/§5.6). Escrow-at-expiry is the committed baseline; live push added iff a verified server-push packet exists. |
| 9.2 wallet extraction | Route through cash-shop now; file follow-up task (§12-B). |
| 9.3 cart / history tabs | Out of scope (StringPool labels unresolved); not modeled — wish-list is the only saved-items mechanism. |
| 9.4 jms scope | Determined in Phase 0; jms limited to its defined opcode set, clientbound-absent flows omitted. |
| 9.5 listing-fee meso owner | **Resolved:** atlas-character owns meso; debit via the existing `AwardMesos` saga action (negative amount), storage-fee precedent (§5.2). |

---

## 14. Testing strategy

- **Dupe-safety suite (acceptance-critical, NFR 8.1):** byte-fixtured/unit +
  integration tests for crash-mid-list, grant-before-debit, double-grant replay,
  cancel-racing-purchase, take-home replay — each asserting the single-custody
  invariant (exactly one copy, currency balanced) after compensation/replay.
  Modeled on the orchestrator's `preset_integration_test.go` reverse-walk
  assertions.
- **Saga compensation** unit tests per new action (inverse fires, idempotent).
- **Packet tests:** Phase 0 byte fixtures, one per mode arm (the verification
  deliverable).
- **Service unit tests:** Builder-based model/processor tests (no testhelpers).
- **Standard gates** (CLAUDE.md): `go test -race ./...`, `go vet ./...`,
  `go build ./...`, `docker buildx bake atlas-mts` (+ any other touched module),
  `tools/redis-key-guard.sh` — all clean before PR.

---

## 15. Sequencing

1. **Phase 0** — serverbound packet verification + §9.1/§9.4 determination (gate).
2. **atlas-mts skeleton** — service, entities, models, REST reads (browse/search),
   config registry, registration (services.json/bake/go.work/k8s).
3. **Custody sagas** — new actions in libs/atlas-saga + orchestrator
   handlers/expansion/compensation/event-acceptance; atlas-mts custody consumer.
4. **List + take-home + cancel/expire** flows end-to-end (TransferToMts,
   WithdrawFromMts, local transitions, ticker).
5. **Buy + settlement** (MtsSettlePurchase) + wallet/recharge query.
6. **Auction + bidding** (escrow path; live push only if Phase 0 unlocked it).
7. **Wish-list.**
8. **atlas-channel wiring** — handlers + validators + writers + migration +
   per-version operations tables; tenant config seed.
9. **atlas-ui** — config page + listings browser.
10. **Dupe-safety test suite** + full verification gates + code review before PR.

Each phase is independently buildable; custody/settlement (3–6) is the risk core
and gets the heaviest test investment.
