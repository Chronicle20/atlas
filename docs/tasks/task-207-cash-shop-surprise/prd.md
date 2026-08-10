# Cash Shop Surprise — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-09

---

## 1. Overview

The **Cash Shop Surprise** (item `5222000`) is a purchasable cash item that acts as a
loot box: the player buys it, it lands in their cash locker (the Cash Shop's
"Cash Inventory"), and double-clicking it there consumes one and grants a
randomly-rolled **cash** item into the same locker. The in-game item text is
explicit about the context:

> "Get a random cash shop item when you open this box. Test your luck! After
> purchasing this item, open the box by double-clicking it in the cash
> inventory. Please keep in mind that you'll only be able to open the box in
> the cash inventory."

Atlas has no implementation today. A grep for `5222000` across `services/` and
`libs/` returns nothing; the only mention in the repository is the backlog entry
at `docs/research/missing-features/economy-and-trade.md:29`. The item's
classification, `ClassificationGachaponCoupon = Classification(522)`
(`libs/atlas-constants/item/constants.go:90`), is not routed by the in-field
cash-item-use dispatcher `GetCashSlotItemType`
(`services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:687-830`),
which is correct — this item is **not** used from the field. Opening it is a
Cash-Shop-context operation.

This task delivers the whole vertical slice: the serverbound decode, a
tenant-configurable weighted reward pool in `atlas-reward-pools`, a
transactional locker grant in `atlas-cashshop`, and the clientbound result — in
both of the two wire shapes the client family uses, across every client version
Atlas targets.

---

## 2. Goals

### Primary goals

1. A player can double-click a Cash Shop Surprise (`5222000`) in their cash
   locker and receive a randomly-selected cash item into that same locker,
   with the source box's quantity decremented by one (removed at zero).
2. The reward pool is tenant-scoped, weighted, and editable through the
   existing `atlas-reward-pools` service and its `atlas-ui` surface — no code
   change is required to change what the box can drop.
3. Every client version Atlas targets is handled: implemented where the client
   supports the feature, and proven `n-a` (with evidence) where it does not.
4. Failure paths report a `CCashShop::OnCashItemResult` failure arm with an
   error code, never a silent drop.
5. The packet coverage matrix (`docs/packets/audits/STATUS.md`) promotes for
   every cell this task touches, backed by byte fixtures and pinned evidence
   records.

### Non-goals

- The separate **Cash Gachapon** UI: `CASH_GACHAPON_BUTTON`
  (`CUICashGachapon::OnButtonClicked`, `STATUS.md:730`) and
  `CASHSHOP_CASH_GACHAPON_OPEN_RESULT` (`STATUS.md:486`). Different opcode,
  different UI class, different feature.
- `CASHSHOP_GACHAPON_STAMP_RESULT` (`STATUS.md:411`) — gachapon stamps.
- `USE_GACHAPON_BOX_ITEM` / `USE_REMOTE` (`STATUS.md:661,664`) — the in-field
  gachapon-remote flow.
- The `GACHAPON_COPY_SUCCESS` / `GACHAPON_COPY_FAILED` arms
  (`docs/packets/dispatchers/cash_shop_operation.yaml:197-201`) — the copy
  operation, not the open operation.
- Changing NPC gachapon behavior or the shared global reward pool.
- Rewarding non-cash (regular inventory) items. See §9 Q1.

---

## 3. User Stories

- As a **player**, I want to double-click a Cash Shop Surprise in my cash
  inventory and receive a random cash item there, so that the box I bought
  does something.
- As a **player**, when the box cannot be opened (my locker is full, the pool
  is empty, the box is gone), I want the client to tell me why with a proper
  error rather than appearing to hang.
- As a **player**, I want the granted item to appear in my cash locker
  immediately without relogging or reopening the Cash Shop.
- As a **server operator**, I want to define which cash commodities the
  Surprise box can yield, and at what weights, from the Atlas UI — the same
  place I already manage gachapon and incubator pools.
- As a **server operator**, I want a Surprise pool to be tenant-scoped so
  different tenants can run different drop tables.
- As an **Atlas maintainer**, I want the coverage matrix to reflect exactly
  which versions this works on and which are provably `n-a`.

---

## 4. Functional Requirements

### FR-1 — Client trigger (serverbound)

- **FR-1.1** `atlas-channel` MUST register a handler for the
  `CASH_ITEM_GACHAPON_BUTTON` opcode (`CUICashItemGachapon::OnButtonClicked`).
  Per-version opcodes from `docs/packets/audits/STATUS.md:723`:
  v83 `0x0A1`, v84 `0x0A5`, v87 `0x0A9`, v92 `0x0B6`, v95 `0x0B9`. The
  registry has no entry for v48/v61/v72/v79/jms_v185 — see FR-5.
- **FR-1.2** The v83 registry entry
  (`docs/packets/registry/gms_v83.yaml:2856-2860`) carries
  `provenance: csv-import`, i.e. it has **not** been IDA-verified. The design
  phase MUST re-derive the opcode and the body read order from the v83 IDB
  before any codec is written, and update the registry provenance.
- **FR-1.3** A new serverbound codec MUST be added under
  `libs/atlas-packet/cash/serverbound/`, with both `Encode` and `Decode`,
  following the immutable-struct + `New…` constructor convention used by every
  sibling in that package.
- **FR-1.4** The handler MUST be routed in **all nine** tenant socket-config
  templates under
  `services/atlas-configurations/seed-data/templates/`, at its sorted `opCode`
  position, with a non-empty validator. A handler whose validator is missing is
  silently dropped
  (see the template conventions in `docs/packets/TEMPLATE_CONVENTIONS.md`).
  Templates for versions where the feature is `n-a` do not get an entry.

### FR-2 — Ownership and eligibility validation

- **FR-2.1** The server MUST resolve the referenced locker asset from the
  requesting account's cash inventory and MUST reject the request if the asset
  does not exist, is not owned by that account, or its `templateId` is not the
  configured Surprise item id.
- **FR-2.2** The Surprise item id MUST be resolved from configuration, not
  hard-coded, consistent with DOM-25 (client wire values are config-resolved).
  `5222000` is the default. This allows a tenant to designate additional or
  alternate box template ids.
- **FR-2.3** The server MUST verify the target compartment has free capacity
  before rolling. `DefaultCapacity` is 55
  (`services/atlas-cashshop/docs/domain.md`, Compartment §Invariants). If the
  source box quantity is 1 the slot is freed by the consume, so the capacity
  check MUST account for that (net-zero slot change) rather than requiring a
  spare slot unconditionally.
- **FR-2.4** The reward MUST be granted into the **same compartment** the box
  was consumed from (Explorer / Cygnus / Legend, `CompartmentType` 1/2/3).

### FR-3 — Reward pool (`atlas-reward-pools`)

- **FR-3.1** A new pool `Kind` MUST be added to the existing closed union in
  `services/atlas-reward-pools/atlas.com/reward-pools/gachapon/builder.go:15-16`
  (currently `KindGachapon = "gachapon"`, `KindIncubator = "incubator"`). The
  new value is `KindCashSurprise = "cash-surprise"`. The builder's validation
  at `builder.go:77` MUST accept it; `DefaultKind` MUST remain `gachapon` so
  existing rows and callers are unaffected.
- **FR-3.2** A `cash-surprise` pool MUST NOT merge the shared global item pool.
  `reward/processor.go:52,107,154` currently merges `global` items for
  `KindGachapon` and excludes them for `KindIncubator`; `cash-surprise` MUST
  follow the incubator precedent (exclude), because the global pool holds
  regular item ids that would be invalid as cash rewards.
- **FR-3.3** A `cash-surprise` pool's entries MUST identify **cash shop
  commodities by commodity id (serial number)**, not raw item ids. The commodity
  catalog (`services/atlas-cashshop/atlas.com/cashshop/cashshop/commodity/`)
  is the authority for the reward's `itemId`, `count`, and `period`, and the
  commodity id is what the client's `GW_CashItemInfo` blob carries as
  `CommodityId` (`libs/atlas-packet/cash/clientbound/shop_inventory.go:33`).
  Rolling a commodity guarantees a self-consistent locker entry.
- **FR-3.4** A `cash-surprise` pool MUST support weighted selection. Whether it
  reuses the three-tier `common/uncommon/rare` weighting or a flat per-entry
  weight is a design decision (§9 Q2); either way the selection MUST be
  deterministic given a seeded RNG so it is testable.
- **FR-3.5** A `cash-surprise` pool MUST be keyed by the **box template id**
  it serves, not by `npcIds`. The existing `npcIds` field MUST be permitted to
  be empty for this kind.
- **FR-3.6** Rolling MUST be exposed through the existing selection endpoint
  shape (`POST /gachapons/{gachaponId}/rewards/select`,
  `services/atlas-reward-pools/docs/rest.md:385`), extended or paralleled as
  the design decides — a caller MUST be able to roll by box template id
  without first knowing the pool's id.
- **FR-3.7** Rolling against a pool with zero eligible entries MUST return a
  distinguishable "empty pool" outcome (not a 500, not a zero-value reward) so
  FR-6 can map it to a client error code.

### FR-4 — Grant and consume (`atlas-cashshop`)

- **FR-4.1** Consuming one Surprise box and creating the reward asset MUST be
  atomic: either both happen or neither does. Partial application (box consumed,
  nothing granted) is a defect.
- **FR-4.2** The reward asset MUST be created via the existing compartment
  `Accept` path so it is a normal flattened cash asset — `cashId`,
  `commodityId`, `templateId`, `quantity`, `expiration`, `purchasedBy` all
  populated. `expiration` MUST be derived from the commodity's `period`
  (zero time = permanent).
- **FR-4.3** The source box MUST be decremented by 1; when the resulting
  quantity reaches 0 the locker entry MUST be released/soft-deleted.
- **FR-4.4** The operation MUST be idempotent under retry keyed on the
  transaction id, so a redelivered command cannot double-grant.
- **FR-4.5** The reward MUST NOT itself be a Surprise box unless the operator
  explicitly configured it — the implementation MUST NOT add a recursion guard
  beyond honoring configuration, but the design MUST note the risk.

### FR-5 — Version coverage

Verified state of the two clientbound wire shapes and the serverbound trigger:

| Version | `CASHSHOP_CASH_ITEM_GACHAPON_RESULT` (`STATUS.md:409`) | `CASHSHOP_OPERATION` `GACHAPON_OPEN_*` arm (`cash_shop_operation.yaml:191-198`) | `CASH_ITEM_GACHAPON_BUTTON` (`STATUS.md:723`) |
|---|---|---|---|
| gms_v48 | `0x101` ❌ | absent | absent from registry |
| gms_v61 | `0x100` ❌ | absent | absent from registry |
| gms_v72 | `0x124` ❌ | absent | absent from registry |
| gms_v79 | **absent from registry** | absent | absent from registry |
| gms_v83 | `0x14D` ❌ | `n-a` (task-183 proof) | `0x0A1` ❌ (csv-import) |
| gms_v84 | `0x154` ❌ | modes 165 / 166 | `0x0A5` ❌ |
| gms_v87 | `0x15E` ❌ | modes 171 / 172 | `0x0A9` ❌ |
| gms_v92 | `0x180` ❌ | **not listed** | `0x0B6` ❌ |
| gms_v95 | `0x188` ❌ | modes 183 / 184 | `0x0B9` ❌ |
| jms_v185 | `0x16D` ❌ | `n-a` (task-183 proof) | absent from registry |

- **FR-5.1** For **v84, v87, v95**, the success result MUST reuse the existing
  `GachaponOpenDone` codec
  (`libs/atlas-packet/cash/clientbound/shop_operation_result_gachapon.go:33`,
  landed by task-183). No wire change may be made to that already-modeled arm.
  Note: the `GACHAPON_OPEN_FAILED` arm has dispatcher modes in the yaml
  (`cash_shop_operation.yaml:194-196`) but **no codec exists** — that file
  contains only `GachaponOpenDone` and `GachaponCopyDone`. See FR-6.2.
- **FR-5.2** For **v83**, the result MUST use the standalone
  `CASHSHOP_CASH_ITEM_GACHAPON_RESULT` opcode. task-183 recorded the
  `GACHAPON_OPEN_*` dispatcher arms as verifiably absent from the v83 binary
  (`cash_shop_operation.yaml:171-179`), so a new codec is required. Its read
  order MUST be derived from the v83 IDB (`CCashShop::OnCashItemGachaponResult`).
- **FR-5.3** For **v92**, the design MUST determine from the v92 IDB whether
  the `GACHAPON_OPEN_*` arms exist (they are absent from the yaml's mode maps,
  which is not by itself proof of absence) and route accordingly. Note that
  `CASHSHOP_OPERATION` is ❌ on v92 (`STATUS.md:406`, `0x178`), so the
  dispatcher path may need verification work on that column first.
- **FR-5.4** For **v48, v61, v72, jms_v185**, the standalone result opcode
  exists in the registry but the serverbound trigger does not. The design MUST
  determine from each IDB whether `CUICashItemGachapon` exists in that build:
  - if it does, implement both directions;
  - if it does not, record a per-version **`n-a` proof** for the serverbound
    cell (absence verified against the binary, per the `n-a` consistency gate)
    and leave the clientbound cell unimplemented with the same proof.
- **FR-5.5** For **v79**, neither the result opcode nor the trigger appears in
  `docs/packets/registry/gms_v79.yaml`, and the v72 registry note explicitly
  reads `v72-EXTRA cash op (no v79 equiv)`
  (`docs/packets/registry/gms_v72.yaml:1877`). The design MUST confirm this
  from the v79 IDB and record an `n-a` proof for both directions. v79 is
  expected to be out of implementation scope on evidence, not on convenience.
- **FR-5.6** No version may be silently skipped. Every one of the ten columns
  MUST end this task as either implemented-and-verified or `n-a`-with-proof.

### FR-6 — Failure reporting

- **FR-6.1** All failure outcomes MUST be reported to the client via a
  `CCashShop::OnCashItemResult` failure arm carrying an error code — per the
  product decision for this task. The `BUY_FAILED`
  (`OnCashItemResBuyFailed`) arm has modes on **all nine** template versions
  (`cash_shop_operation.yaml:68-70`) and its Go type is `BuyFailed`
  (`libs/atlas-packet/cash/clientbound/shop_operation_result_failed.go:128`) —
  a `mode` byte plus an `errorCode` byte. Its matrix cell,
  `cash/clientbound/CashBuyFailed`, is already ✅ on
  v48/v61/v72/v79/v83/v84/v87/v95/jms (`STATUS.md:406,408`), making it the only
  failure vehicle available on every column.
- **FR-6.2** On v84/v87/v95 the design MAY instead implement the dedicated
  `GACHAPON_OPEN_FAILED` arm where it more accurately matches client behavior.
  That is **net-new codec work, not reuse** — the arm has dispatcher modes
  (`cash_shop_operation.yaml:194-196`) but no Go type exists;
  `shop_operation_result_gachapon.go` defines only `GachaponOpenDone` (:33) and
  `GachaponCopyDone` (:100). Absent a behavioral reason from the IDB, FR-6.1's
  `BuyFailed` is the default on every column.
- **FR-6.3** Distinct error codes MUST be emitted for at minimum: locker full,
  empty/misconfigured pool, box not found or not owned, and generic internal
  failure. The concrete numeric codes MUST be derived from the client's
  error-string table, not invented.
- **FR-6.4** The failure path MUST leave the box unconsumed.

### FR-7 — Administration (`atlas-ui`)

- **FR-7.1** The existing reward-pools surface MUST accept the new kind. The
  `RewardPoolKind` type (`services/atlas-ui/src/types/models/reward-pool.ts`),
  the Zod schema (`src/lib/schemas/reward-pools.schema.ts`), and `KindBadge`
  (`src/components/features/reward-pools/KindBadge.tsx`, currently a binary
  incubator-vs-gachapon ternary) MUST all be widened. This is small but not
  zero — the badge component in particular has no default branch today.
- **FR-7.2** The pool form MUST hide or disable the `npcIds` field for
  `cash-surprise` pools (FR-3.5) and surface the box template id instead.
- **FR-7.3** Pool item entry MUST accept a commodity id (FR-3.3) and display
  the resolved item for operator sanity-checking.

---

## 5. API Surface

All endpoints are JSON:API, tenant-scoped by the usual `TENANT_ID` header
decoration.

### `atlas-reward-pools`

**Modified — `GET|POST|PATCH /gachapons` and `/gachapons/{gachaponId}`**
The `kind` attribute's accepted value set widens to
`gachapon | incubator | cash-surprise`. `kind` remains optional on the wire
with `gachapon` as the default (`gachapon/resource.go:100-103`), so existing
clients are unaffected. A new optional attribute identifies the box template
id this pool serves; it is required when `kind == "cash-surprise"` and MUST be
unique per tenant among `cash-surprise` pools.

**New — roll by box template id**
A caller (`atlas-cashshop`) MUST be able to roll without knowing the pool id.
Exact shape is a design decision; the requirement is a single request that
takes a tenant + box template id and returns one reward or a distinguishable
empty-pool result.

Reward resource fields extend to carry the **commodity id** in addition to the
existing `itemId` / `quantity` / `tier` (`reward.Model`,
`services/atlas-reward-pools/docs/domain.md`).

Error conditions:

| Status | Condition |
|---|---|
| 200 | Reward selected |
| 404 | No `cash-surprise` pool configured for that box template id |
| 409 | Pool exists but has no eligible entries (FR-3.7) |
| 400 | `kind` outside the closed union; missing box template id on a `cash-surprise` pool |

### `atlas-cashshop`

The open operation is a **command**, not a REST call — it is initiated by a
packet and must be transactional across a consume and a create. It follows the
existing saga/command precedent in this service rather than adding a REST
endpoint. The design phase specifies whether it is a single in-service
transaction or a saga; §9 Q3.

No existing `atlas-cashshop` REST endpoint changes shape.

---

## 6. Data Model

### `atlas-reward-pools` — modified

`gachapon` entity (`gachapon/entity.go`):

- `Kind` (`string`, `not null`, `default:gachapon`) — **existing column**, no
  migration needed; the widening is validation-only
  (`gachapon/builder.go:77`).
- **New** `box_template_id` (`uint32`, nullable, default `0`) — the cash item
  whose opening draws from this pool. Nullable/zero for `gachapon` and
  `incubator` kinds. Unique per `(tenant_id, box_template_id)` among rows where
  `kind = 'cash-surprise'` and `box_template_id <> 0`.

`item` entity (per-pool reward entries):

- **New** `commodity_id` (`uint32`, nullable, default `0`) — the cash shop
  commodity (serial number) awarded. Required for `cash-surprise` pool entries;
  zero for other kinds, which continue to use `item_id`.

Migration notes: both additions are additive nullable/defaulted columns.
Existing rows are untouched and continue to read as `kind = 'gachapon'` with
zero-valued new columns. No backfill required.

### `atlas-cashshop` — unchanged

The reward is an ordinary `asset` row created through the existing
compartment `Accept` path. No schema change.

### Tenant configuration

The Surprise box template id default (`5222000`) and the client error codes
(FR-6.3) are tenant configuration, not constants in code, per DOM-25.

---

## 7. Service Impact

| Service / library | Change |
|---|---|
| `libs/atlas-packet/cash/serverbound` | New `CashItemGachaponButton` codec (Encode + Decode), version-gated per FR-5 |
| `libs/atlas-packet/cash/clientbound` | New `CashItemGachaponResult` codec for the standalone-opcode versions (v83 at minimum). **No change** to the existing `GachaponOpenDone` / `GachaponOpenFailed` |
| `services/atlas-channel` | New handler (`cash_shop_surprise.go` or similar) + writer wiring; validation, roll orchestration, result emission |
| `services/atlas-configurations` | New handler + writer entries in every applicable seed template, at sorted `opCode` positions with validators |
| `services/atlas-reward-pools` | New `Kind`, new `box_template_id` and `commodity_id` columns, global-pool exclusion, roll-by-box-template-id, empty-pool outcome |
| `services/atlas-cashshop` | Transactional open: consume box + resolve commodity + `Accept` reward; idempotency by transaction id |
| `services/atlas-saga-orchestrator` | Only if the design chooses a saga over an in-service transaction (§9 Q3) |
| `services/atlas-ui` | Widen `RewardPoolKind`, Zod schema, `KindBadge`; conditional form fields |
| `docs/packets/audits/` | Fixtures, evidence records, `n-a` proofs, matrix regeneration for every touched cell |
| `docs/packets/registry/` | Provenance upgrade for the v83 serverbound entry (FR-1.2); new entries wherever RE discovers them |

---

## 8. Non-Functional Requirements

- **Multi-tenancy.** Every read and write is tenant-scoped via
  `tenant.MustFromContext(ctx)`. Pools, box template ids, and error codes are
  per-tenant. A pool from one tenant must never be reachable from another.
- **Atomicity.** FR-4.1 is the hard non-functional constraint of this task. A
  consumed box with no reward is a player-visible loss of paid currency.
- **Idempotency.** FR-4.4 — Kafka redelivery must not double-grant.
- **Determinism in test.** The weighted roll must be injectable/seedable so
  tests assert exact selections rather than statistical distributions.
- **Observability.** Every open logs tenant, account, character, box asset id,
  resolved pool id, rolled commodity id, and outcome. Failures log the emitted
  error code. This is a currency-adjacent path; an unexplained grant must be
  reconstructable from logs alone.
- **No wire regressions.** Versions already ✅ in the matrix must not change
  bytes. `packet-audit` matrix regeneration must show promotions only.
- **Grounding.** No opcode, error code, or field order in the implementation
  may come from general MapleStory knowledge. Each is derived from the IDB, the
  registry, or ingested WZ data, and cited.

---

## 9. Open Questions

**Q1 — Cash-only, or eventually regular items?**
Resolved for v1: **locker-only**, cash items only. The `GachaponOpenDone` codec
carries a conditional `GW_CashItemInfo` blob and an `isCashItem` flag
(`shop_operation_result_gachapon.go:32-40`), which implies the client can also
represent a non-cash outcome. Out of scope now; the data model should not
foreclose it.

**Q2 — Tier weights or flat weights?**
The existing pool model has three tier weights (`commonWeight`,
`uncommonWeight`, `rareWeight`) plus per-item weights within a tier. Reusing
tiers keeps the UI and roller untouched; a flat per-entry weight is a better
fit for a cash box but adds a second selection path. Design phase decides.

**Q3 — In-service transaction or saga?**
The consume and the grant both live in `atlas-cashshop`, which argues for a
single DB transaction. But the roll lives in `atlas-reward-pools`, making the
sequence roll → consume+grant. If the roll is done first and the grant then
fails, nothing is lost (the box is untouched), which suggests a plain
transaction is sufficient. Design phase confirms and documents the ordering.

**Q4 — `isCashItem` flag semantics on v84+.**
`GachaponOpenDone.IsCashItem()` gates whether the 55-byte blob is written.
What value the client expects for a cash reward, and what the `resultCode` /
`resultParam2` values passed to `CUICashGachapon::OnCashGachaponOpenResult`
mean, must be read out of the IDB during design.

**Q5 — Client error codes (FR-6.3).**
The numeric codes for "locker full" / "empty pool" / "not found" must be
sourced from the client's cash-shop error-string table. Unknown until the RE
pass.

**Q6 — Legacy and JMS reachability (FR-5.4).**
Whether `CUICashItemGachapon` exists at all in the v48/v61/v72/jms_v185
builds is unknown. The clientbound result opcode being present in the registry
is suggestive but not decisive — the handler may be dead code in those builds.

**Q7 — Does the box exist in ingested WZ for every target version?**
Item `5222000`'s presence and its `Item/Cash` spec node have not been checked
against the ingested WZ data for any version. If the item is absent from a
version's data, that version is `n-a` regardless of packet support.

---

## 10. Acceptance Criteria

### Behavior

- [ ] Double-clicking a Cash Shop Surprise in the cash locker on a live v83
      tenant grants a random cash item into the same compartment and decrements
      the box, with no relog required.
- [ ] The same flow verified live on at least one v84+ tenant, exercising the
      `GACHAPON_OPEN_SUCCESS` arm rather than the standalone opcode.
- [ ] Opening with a full locker produces a client-visible error and leaves the
      box intact.
- [ ] Opening a box whose pool has no entries produces a client-visible error
      and leaves the box intact.
- [ ] A forged request naming an asset the account does not own is rejected and
      logged, with no state change.
- [ ] Granted rewards carry a correct `commodityId`, `templateId`, `quantity`,
      and `expiration` derived from the commodity catalog.

### Configuration

- [ ] A `cash-surprise` pool can be created, edited, and deleted from
      `atlas-ui`, and the kind renders its own badge.
- [ ] A `cash-surprise` pool does not draw from the shared global pool.
- [ ] Two tenants with different pools for the same box template id roll from
      their own pool only.
- [ ] Existing `gachapon` and `incubator` pools are byte-identical in behavior
      after the change (regression).

### Packets

- [ ] Every matrix cell in the FR-5 table is either promoted with a byte
      fixture and pinned evidence record, or carries an `n-a` proof.
- [ ] `packet-audit` matrix regeneration shows promotions only — no cell
      degrades.
- [ ] `tools/template-opcode-order-guard.sh` and
      `tools/template-duplicate-binding-guard.sh` exit 0.
- [ ] The v83 serverbound registry entry's provenance is no longer
      `csv-import`.
- [ ] No already-✅ version's encoded bytes changed.

### Build & verification

- [ ] `go test -race ./...` clean in every changed module.
- [ ] `go vet ./...` clean in every changed module.
- [ ] `docker buildx bake atlas-cashshop atlas-channel atlas-reward-pools`
      (plus any other service whose `go.mod` was touched) succeeds from the
      worktree root.
- [ ] `tools/lint.sh --check`, `tools/redis-key-guard.sh`,
      `tools/goroutine-guard.sh` all clean.
- [ ] `atlas-ui`: `npm run build` and `vitest` clean.
- [ ] Code review run (`plan-adherence-reviewer`, `backend-guidelines-reviewer`,
      `frontend-guidelines-reviewer`) before the PR is opened.
