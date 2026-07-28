# CashShopOperation Result Family — Design

Task: task-183-cashshop-result-family
Status: Approved (design phase)
Date: 2026-07-27
Playbooks this design defers to (not restated here):
[`DISPATCHER_FAMILY.md`](../../packets/DISPATCHER_FAMILY.md) ·
[`VERIFYING_A_PACKET.md`](../../packets/audits/VERIFYING_A_PACKET.md) ·
[`PROCESS.md`](../../packets/PROCESS.md)

---

## 1. Problem & scope

`CCashShop::OnCashItemResult` is a **mode-prefix dispatcher**: one clientbound
opcode (`CASHSHOP_OPERATION`, v95 `0x180`) whose leading `Decode1` byte selects
one of ~57 per-mode handler arms. Atlas models **9** of those arms as discrete
writer structs in `libs/atlas-packet/cash/clientbound/` and enumerates them in
`docs/packets/dispatchers/cash_shop_operation.yaml` across only the 5 modern
versions. This task closes the gap: **every arm of the switch, every applicable
version, modeled + enumerated + verified.**

### 1.1 The authoritative arm set (v95 reference — grounded)

The design anchors on the user-supplied v95 `CCashShop::OnCashItemResult`
decompile (the canonical read-order reference per FR-1, FR-8 grounding). The
full switch is reproduced in [Appendix A](#appendix-a-v95-onCashItemResult-switch--the-canonical-arm-set)
as the backbone of `arm-catalog.md`. It has **56 arms** (plus `default:
return`), of which **9 are already modeled** and **47 are new**.

### 1.2 Scope decisions (locked with the user, 2026-07-27)

| # | Decision | Consequence |
|---|---|---|
| **D1** | **In-switch arms only.** Scope = exactly the cases in the v95 `OnCashItemResult` switch. | Gachapon **open/copy** (`0xB7`–`0xBA`), **world-transfer** (`0xB5`/`0xB6`), **name-change-buy** (`0xB3`), **maple-point** (`0xBB`/`0xBC`), **free-item** (`0xAA`), **purchase-record** (`0xAF`/`0xB0`) **are in** — they are genuine switch cases. The sibling **separate-opcode** functions (`OnCashItemGachaponResult`, `OnGiftMateInfoResult`, `OnCheckDuplicatedIDResult`, `OnCheckNameChangePossibleResult`, `OnCheckTransferWorldPossibleResult`, `OnChargeParamResult`, `OnPurchaseExpChanged`, `OnOneADay`, `OnNoticeFreeCashItem`, `CASHSHOP_REGISTER_NEW_CHARACTER_RESULT`) are **out** — each is its own matrix row/dispatcher, deferred to follow-up tasks. See [§7](#7-explicitly-out-of-scope). |
| **D2** | **Full legacy RE, subset arms + `n-a`.** RE all 4 legacy IDBs (v48/v61/v72/v79); model+verify whichever arms genuinely exist per version, record the rest `n-a` with enumeration evidence. | Legacy is structurally divergent (v48/v61 = `COutPacket(160)+Enc1(sub-op)` cash-purchase subset; v72 op 291 / v79 op 303 = divergent `CCashShop` block). See [§6](#6-legacy-strategy-v48v61v72v79). |
| **D3** | **Single aggregate op-row.** Keep one `CASHSHOP_OPERATION` matrix row, worst-of-all-arms (the established FIELD_EFFECT dispatcher model). | Adding ~47 arms **regresses** the currently-`✅` v83/v84/v87/v95/jms cells; the op-row re-greens only when every arm×version is verified or `n-a`. No matrix-tooling change. See [§5.1](#51-op-row-aggregation--the-regression). |

### 1.3 Non-goals (from PRD, reaffirmed)

- No new cash-shop **feature logic** (no real gifting/coupon/gachapon/transfer/
  maple-point domain flows). Codecs + body-func API only; producers wired only
  where a domain flow already emits the corresponding result (default: none new).
- No serverbound request-codec changes beyond keeping an existing request/result
  pair consistent.
- No wire change to an already-verified arm/version except to fix an RE-discovered
  correctness defect (documented + fixed on-branch).
- No `atlas-ui` changes.

---

## 2. Architecture — four layers, one branch

The work is four coordinated layers (mirroring the PRD), each a distinct
artifact class, executed as sequenced **waves** on the single task worktree
(CLAUDE.md one-worktree rule — no mid-task forks):

```
Layer 1  RE (all 9 IDBs, v95 ref)  ──►  arm-catalog.md  +  named IDB functions
Layer 2  Codecs (cash/clientbound) ──►  discrete per-mode structs + body funcs + run.go #-entries
Layer 3  Enumeration & templates   ──►  cash_shop_operation.yaml + regenerated template operations maps
Layer 4  Verification              ──►  per-arm byte-fixtures + pinned evidence + matrix promotion
```

Layer 1 is the **critical path and the sole source of truth** for everything
downstream. A wrong read-order in the catalog propagates into a wrong codec, a
wrong template byte, and a false-verified cell. The design therefore gates Layer
2 behind a **completed, reviewed `arm-catalog.md`** (see [§8](#8-work-sequencing--waves)).

### 2.1 Component boundaries

| Unit | Responsibility | Depends on | Consumed by |
|---|---|---|---|
| `arm-catalog.md` | Per-arm × per-version wire shape: mode byte, field list, version divergences, `n-a` versions, IDB fname + address. | The 9 IDBs. | Layers 2–4 (single source of wire truth). |
| `cash/clientbound/*` structs | One discrete struct per arm: `mode byte` + body; `Encode`+`Decode`; `Operation()`; `String()`. | `arm-catalog.md`; `response.Writer`/`request.Reader`; `model.Asset`. | body funcs; verifier fixtures. |
| `shop_operation_body.go` funcs | One body func per arm: fixes the operation key, resolves mode from `operations` (and reason from `errors`), passes it into the constructor. | struct constructors; `atlas_packet.WithResolvedCode`/`ResolveCode`. | future features (the usable API). |
| `run.go candidatesFromFName` | One `#<Mode>` synthetic-FName entry per arm → its struct. | struct names. | verifier / matrix report-gen. |
| `cash_shop_operation.yaml` | Per-version mode table (source of truth for template `operations`). | `arm-catalog.md`. | `packet-audit operations`. |
| version templates | `CashShopOperation` writer `operations` map (regenerated). | the yaml. | live tenants / verifier. |

Each unit is independently testable: catalog by RE review, structs by round-trip
unit tests, body funcs by `dispatcher-lint`, yaml↔template by `operations --check`,
cells by the verifier fixture procedure.

---

## 3. Arm taxonomy & the RE catalog (Layer 1)

### 3.1 `arm-catalog.md` — schema

Supporting artifact required by PRD §5 and Acceptance. One row per arm; per
version, the mode byte + shape delta. Schema (markdown table + per-arm notes):

```
| operation key | v95 handler fname | shape group | v48 | v61 | v72 | v79 | v83 | v84 | v87 | v95 | jms | fields (v95) |
```

- **operation key** — the `SCREAMING_SNAKE` const shared by the struct, the
  yaml, the `operations` map, and `run.go`. New keys extend the existing 9 (e.g.
  `LOAD_GIFT_SUCCESS`, `GIFT_SUCCESS`, `BUY_PACKAGE_SUCCESS`, `DESTROY_SUCCESS`,
  `TRANSFER_WORLD_SUCCESS`, `GACHAPON_OPEN_SUCCESS`, `CHANGE_MAPLE_POINT_SUCCESS`,
  and the `_FAILED` counterparts). Final names fixed during Layer 1 from the
  handler fname.
- **mode byte per version** — DECIMAL in the yaml (tool reads ints); hex in the
  catalog notes. `n-a` where the arm is absent in that version's switch (proven
  by enumerating every case at the switch address — the party.yaml precedent).
- **shape group** — the wire-body taxonomy (below); recorded, but each arm still
  gets its **own discrete struct** (DISPATCHER_FAMILY AP-1 bans shape-sharing).
- **fields** — the exact `Decode*` read order from the v95 handler, each cited to
  a decompile line; per-version deltas noted inline.

### 3.2 Wire-shape groups (for reasoning; NOT for struct-sharing)

From the arm names + the existing 9, arms fall into recognizable shapes. This is
a *reasoning aid* — the RE confirms each, and every arm is a discrete struct:

- **Failure arms** (`…Failed`, ~24 of them) — hypothesis `mode + reason byte`
  (like `LoadInventoryFailure` / `InventoryCapacityFailed` today). FR-2.3: the RE
  decides per arm whether the arm carries extra fields beyond the reason; if so it
  gets those fields. The reason byte is config-resolved from the writer `errors`
  table, never a Go literal.
- **Locker/item-blob arms** (`LoadGiftDone`, `Move*Done`, `Buy*Done`,
  `Gift*Done`, `Package*Done`, `Rebate*`, `Couple*`, `Gachapon*Done`) — carry a
  `GW_CashItemInfo` (55-byte) or `GW_ItemSlotBase` (`model.Asset`) blob, possibly
  a count-prefixed list. Reuse the existing `CashInventoryItem` / `model.Asset`
  helpers.
- **Counter arms** (`IncSlotCountDone`, `IncTrunkCountDone`,
  `IncCharacterSlotCountDone`, `IncBuyCharacterCountDone`,
  `EnableEquipSlotExtDone`) — `mode + inventoryType/slotType + count` (like
  `InventoryCapacitySuccess` today).
- **Scalar/notice arms** (`LimitGoodsCountChanged`, `ExpireDone`,
  `FreeCashItemDone`, `ChangeMaplePointDone`, `PurchaseRecord`,
  `NameChangeResBuyDone`, `TransferWorldDone`) — small scalar bodies or lists;
  RE-derived individually.

**No shared struct across arms**, even wire-identical ones (the two wishlist
arms are already split precisely for this reason). Bodyless notice arms are still
their own `struct { mode byte }`.

### 3.3 IDB naming (FR-1.3)

Every arm handler and the root dispatcher is **named in each IDB** using the v95
names as the baseline (`OnCashItemRes…`). Confirmed-but-unnamed arms are renamed
and the IDB saved (`idb_save`). An unresolved fname is a **stop-and-ask**
escalation (project fname rule), never a guess.

---

## 4. Codec design (Layer 2)

Follows the canonical discrete-per-mode pattern
([DISPATCHER_FAMILY.md](../../packets/DISPATCHER_FAMILY.md) §"canonical
pattern") — this design does not restate it, only records the task-specific
choices:

### 4.1 File organization

Extend the **existing consolidated files**, do not sprawl (AP-8):

- `shop_operation_result.go` — the discrete structs. Given ~47 new structs, this
  file will grow large. **Split by wire-shape group into sibling files** in the
  same package to keep each file focused (the brainstorming "focused files"
  principle), e.g.:
  - `shop_operation_result.go` (existing 9 + shared helpers)
  - `shop_operation_result_failed.go` (the `…Failed` discrete structs)
  - `shop_operation_result_gift.go`, `…_gachapon.go`, `…_slots.go`, `…_transfer.go`
  Grouping is by domain, one struct still = one arm. (This is *file* grouping for
  readability, **not** struct-sharing — INV-1 still holds.)
- `shop_operation_body.go` — one body func per arm (existing file).
- `run.go` — one `case "CCashShop::OnCashItemResult#<Mode>":` per arm.

### 4.2 Config-resolved mode & reason (DOM-25, INV-2/INV-3)

- Mode byte: `atlas_packet.WithResolvedCode("operations", <FIXED_KEY>, func(mode byte)…)`
  — the constructor's first param is `mode byte`; the body func passes the
  **resolved** value. Zero `mode: 0x` literals, zero `func(_ byte)`.
- Failure reason: `atlas_packet.ResolveCode(l, options, "errors", <message key>)`.
  The `errors` const table already has ~55 entries
  (`shop_operation_body.go:24-77`); RE-discovered new reasons extend it.
- No body func takes a caller-supplied `op`/`code`/`mode`/`key`/reason-as-key
  (AP-4 / INV-3 semantic) — the arm maps to ONE operation, so the key is a fixed
  const.

### 4.3 Version gating (FR-2.4)

Version-divergent fields use the `MajorAtLeast`/`MajorVersion` idiom (never raw
`> N`), consistent with `libs/atlas-packet`. Any gate straddling two versions
needs a verified byte-fixture on **both** adjacent versions (the `gates.yaml`
`gate-check` CI gate). No wire change to an already-verified arm/version.

### 4.4 The `0x4D` gift TODO (FR-2.6)

`CashShopCashGiftsBody()` currently returns `NewCashShopGifts(0x4D).Encode` with
a `// TODO map codes for JMS`. This resolves naturally: the gift arm
(`OnCashItemResGiftDone`, v95 `0x6B`) gets its config-resolved body func like
every other arm; the hardcoded `0x4D` is deleted. Reconcile `CashShopGifts` with
the RE'd `GiftDone` shape (they may be the same struct — RE confirms; if so, one
canonical struct, the old constructor updated to take `mode byte`).

### 4.5 Testing (FR-4.3)

Every new struct gets an `Encode`/`Decode` round-trip unit test (Builder pattern,
no `*_testhelpers.go`), plus the per-version `vNN_test.go` byte-fixture files
already present in the package pick up the new arms.

---

## 5. Enumeration, templates & verification (Layers 3–4)

### 5.1 Op-row aggregation & the regression (D3)

`cash` is **not** in `families.yaml`, so `CASHSHOP_OPERATION` does **not** cap at
`🧩`; the op-row aggregates worst-of-all-`#`-entry-arms (the FIELD_EFFECT model,
DISPATCHER_FAMILY step 5). Because the 9 existing arms are `✅` today, adding ~47
unverified arms will drop the v83/v84/v87/v95/jms op-row cells out of `✅` for
the duration of the task. This is **expected and accepted** (D3): the row
re-greens only when *every* arm is verified-or-`n-a` across every applicable
version. Wave ordering ([§8](#8-work-sequencing--waves)) front-loads the modern-5
so the biggest column re-greens first.

### 5.2 yaml → templates

- Extend `cash_shop_operation.yaml` to the full arm set: `key`, `handler`
  (IDB-verified fname), `modes` for all 9 versions or `n-a`. v95 modes come from
  [Appendix A](#appendix-a-v95-onCashItemResult-switch--the-canonical-arm-set)
  directly; other versions from their RE'd switch.
- `packet-audit operations` regenerates every version template's
  `CashShopOperation` writer `operations` map. A new key must land in **every**
  version template that supports it (no silent drop — the "new opcodes missing
  from live tenant config" bug class). `packet-audit operations --check` must pass
  (yaml↔template sync); the template-opcode-order guard must pass (ascending
  `opCode` — but note: the `operations` map is keyed, not the writers array; the
  order guard applies to the writers/handlers arrays, which this task does not
  reorder).

### 5.3 Per-arm verification

Each arm×version cell is promoted via the verifier procedure
([VERIFYING_A_PACKET.md](../../packets/audits/VERIFYING_A_PACKET.md)) — this
design does not restate it. `n-a` cells pass the n-a consistency gate (arm
enumerated-absent in that version's switch, address cited). The verification
fan-out is large (~47 arms × up to 9 versions ≈ 250–300 cells); it is
**batched per IDB** (all v95 arms in one pass, etc.) to amortize decompile setup,
and dispatched to `packet-verifier` sub-agents pinned to a cheaper model
(review/verify cost rule).

### 5.4 `coverage-manifest.yaml`

Authored up front (PROCESS.md §"Coverage manifest") so the
`packet-completeness-critic` can diff the branch:

```yaml
ops:
  - CASHSHOP_OPERATION            # the single aggregate op-row (D3)
versions: [gms_v48, gms_v61, gms_v72, gms_v79, gms_v83, gms_v84, gms_v87, gms_v95, jms_v185]
out_of_scope:
  - CASHSHOP_CASH_ITEM_GACHAPON_RESULT       # separate opcode (D1)
  - CASHSHOP_GIFT_INFO_RESULT
  - CASHSHOP_CHECK_NAME_CHANGE
  - CASHSHOP_CHECK_TRANSFER_WORLD_POSSIBLE_RESULT
  - CHARGE_PARAM_RESULT
  - CASHSHOP_PURCHASE_EXP_CHANGED
  - ONE_A_DAY
  - CASHSHOP_NOTICE_FREE_CASH_ITEM
  - CASHSHOP_REGISTER_NEW_CHARACTER_RESULT
  - CASHSHOP_GACHAPON_STAMP_RESULT
```

---

## 6. Legacy strategy (v48/v61/v72/v79) — D2

Legacy cash is **structurally divergent**, so the modern arm mapping does not
transfer by offset:

- **v48 / v61** — the cash-purchase family is sent via `COutPacket(160)+Enc1(sub-op)`
  (`OnBuy`/`OnBuyCouple`/`OnBuyFriendship`/`OnBuyPackage`/`OnGift`/`OnSetWish`);
  `v61 CASHSHOP_OPERATION=196`. `OnCashItemResult` exists (v48 IDB address
  4535510) but with a collapsed arm set. RE enumerates the actual switch; arms
  present get modes, everything else `n-a`.
- **v72 / v79** — `CCashShop::OnPacket` op **291** (v72) / **303** (v79) =
  `OnCashItemResult`; the v72 `CCashShop` block is "structurally divergent vs v79"
  with extra sibling ops (out of scope per D1). RE each switch independently.

**Approach:** Layer 1 RE produces the legacy switch enumeration first; the design
assumes a **subset** of the modern arms exist (buy/gift/wish/inc-slot families
likely; gachapon/world-transfer/maple-point/name-change/purchase-record likely
`n-a` in the oldest clients, per PRD Q1). Every `n-a` is proven by full case
enumeration at the switch address, not assumed. Version-gating in the codecs
handles any legacy field-shape divergence via `MajorAtLeast`.

Legacy is **Wave 3** (after the modern-5 land) to de-risk — the highest-RE-cost,
lowest-certainty column, isolated so a legacy surprise doesn't block the modern
deliverable.

---

## 7. Explicitly out of scope

The **sibling separate-opcode** cash-result dispatchers — each is its own matrix
row with its own fname, **not** a case in the `OnCashItemResult` switch (confirmed
against the v95 decompile):

`CASHSHOP_CASH_ITEM_GACHAPON_RESULT` (`OnCashItemGachaponResult`),
`CASHSHOP_GIFT_INFO_RESULT` (`OnGiftMateInfoResult`),
`CASHSHOP_CHECK_NAME_CHANGE` (`OnCheckDuplicatedIDResult`/`OnCheckNameChangePossibleResult`),
`CASHSHOP_CHECK_TRANSFER_WORLD_POSSIBLE_RESULT`, `CHARGE_PARAM_RESULT`,
`CASHSHOP_PURCHASE_EXP_CHANGED`, `ONE_A_DAY`, `CASHSHOP_NOTICE_FREE_CASH_ITEM`,
`CASHSHOP_REGISTER_NEW_CHARACTER_RESULT`, `CASHSHOP_GACHAPON_STAMP_RESULT`.

These stay at their current matrix state; they are listed in
`coverage-manifest.yaml` `out_of_scope` so the completeness critic does not flag
incidental churn.

> **Note the naming trap:** `OnCashItemResCashGachaponOpenDone` (in-switch, **in
> scope**) vs `OnCashItemGachaponResult` (separate opcode, **out**); likewise
> `OnCashItemNameChangeResBuyDone` (in) vs `OnCheckNameChangePossibleResult`
> (out), and `OnCashItemResTransferWorldDone` (in) vs
> `OnCheckTransferWorldPossibleResult` (out). The catalog records the fname
> verbatim so the boundary is unambiguous.

---

## 8. Work sequencing — waves

One branch, sequenced waves with hard checkpoints. Each wave is
build/vet/test-clean before the next starts.

- **Wave 0 — RE & catalog (critical path).** RE `OnCashItemResult` + every arm in
  all 9 IDBs; name functions; save IDBs. Produce `arm-catalog.md` (full wire
  truth) + `coverage-manifest.yaml`. Extend `cash_shop_operation.yaml` (modes for
  every version, `n-a` proven). **Checkpoint: catalog reviewed before any codec.**
- **Wave 1 — modern-5 codecs.** Discrete struct + body func + `run.go` `#`-entry +
  round-trip test per new arm (v83/v84/v87/v95/jms shapes). Delete the `0x4D`
  gift TODO. `dispatcher-lint`, `operations --check`, `go build/vet/test -race`
  clean.
- **Wave 2 — modern-5 verification.** Per-arm byte-fixtures + pinned evidence;
  promote the modern columns of the `CASHSHOP_OPERATION` op-row back to `✅`.
- **Wave 3 — legacy.** RE-confirmed legacy arms modeled + verified; `n-a` cells
  finalized with enumeration evidence. Legacy columns of the op-row reach `✅`.
- **Wave 4 — close-out.** Full verification suite (below) + code review
  (`plan-adherence` + `backend-guidelines` + `packet-completeness-critic`) before
  PR.

Rationale: Layer 1 gates everything (a wrong catalog poisons all layers), so it is
its own wave with a review checkpoint. Modern-5 before legacy front-loads the
largest, highest-certainty columns and re-greens the op-row's biggest cells
first; legacy — the divergent, high-RE-cost, `n-a`-heavy column — is isolated
last so a surprise there can't stall the primary deliverable.

---

## 9. Alternatives considered

- **Per-arm matrix rows (rejected, D3).** Would keep already-green arms green and
  show per-arm progress, but diverges from every other graduated dispatcher family
  (all use the aggregate op-row + `#`-entry model) and would need matrix-tooling
  changes. Consistency + zero-tooling-change won.
- **Separate task per wire-shape group (rejected).** Cleaner PRs, but violates the
  one-worktree rule and would fracture the RE pass (the catalog is one artifact).
  Kept as internal waves instead.
- **Shared failure struct for all `…Failed` arms (rejected).** Tempting (most are
  `mode+reason`), but AP-1/INV-1 ban a struct serving >1 mode, and the
  wishlist-split precedent shows the project's stance: discrete per mode always.
  File-grouping gives the readability without the shared struct.
- **Defer legacy entirely (rejected).** The user chose full legacy RE (D2);
  isolating it as Wave 3 captures the de-risking benefit without dropping scope.
- **Pull the sibling separate-opcodes in (rejected, D1).** They are distinct
  dispatchers; folding them in would nearly double the task and blur the
  arm-vs-opcode boundary the catalog exists to keep sharp.

---

## 10. Risks & mitigations

| Risk | Mitigation |
|---|---|
| Wrong read-order in catalog poisons all layers | Wave-0 review checkpoint; v95 decompile is user-grounded; each field decompile-cited. |
| Op-row red for most of the task reads as "broken" | D3 documented as expected; wave order re-greens modern-5 first; PR description states the aggregate-row semantics. |
| Legacy switch has no clean symbols → unresolved fnames | Stop-and-ask escalation per project fname rule; `n-a` only via full case enumeration, never assumption. |
| Verification fan-out (~250–300 cells) is huge | Batch per IDB; dispatch to `packet-verifier` on a cheap model; op-row aggregates so no per-cell hand-grading. |
| `GiftDone` vs existing `CashShopGifts` struct duplication | Wave-0 RE reconciles; one canonical struct, `0x4D` literal deleted. |
| Naming trap (in-switch `…ResTransferWorldDone` vs separate `OnCheckTransferWorldPossibleResult`) | Catalog records fname verbatim; `out_of_scope` list in manifest; §7 note. |

---

## 11. Acceptance & verification

Acceptance criteria are the PRD §10 checklist (unchanged). Close-out verification
(CLAUDE.md Build & Verification + packet CI gates), run from the worktree root:

- `go test -race ./...`, `go vet ./...`, `go build ./...` clean in `libs/atlas-packet`
  and any touched module.
- `docker buildx bake` for any service whose `go.mod` was touched — **expected
  none** (only `libs/atlas-packet` code + `atlas-configurations` seed JSON, no
  `go.mod` change; confirm at close-out).
- `tools/lint.sh --check`, `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`,
  `tools/template-opcode-order-guard.sh` clean.
- `packet-audit fname-doc --check`, `operations --check`, `dispatcher-lint`,
  `gate-check --check`, `matrix --check` exit 0.
- Code review (`plan-adherence-reviewer` + `backend-guidelines-reviewer` +
  `packet-completeness-critic`) before PR.

---

## Appendix A — v95 `OnCashItemResult` switch — the canonical arm set

Grounded from the user-supplied v95 decompile (`CCashShop::OnCashItemResult`).
Mode bytes are v95 (hex). `existing` = already modeled (the 9). `NEW` = added by
this task. Non-v95 mode bytes are RE-derived per version in Wave 0 (not shown
here — the catalog owns them).

| v95 mode | handler | status |
|---|---|---|
| 0x54 | OnCashItemResLimitGoodsCountChanged | NEW |
| 0x58 | OnCashItemResLoadLockerDone | existing (LOAD_INVENTORY_SUCCESS) |
| 0x59 | OnCashItemResLoadLockerFailed | existing (LOAD_INVENTORY_FAILURE) |
| 0x5A | OnCashItemResLoadGiftDone | NEW |
| 0x5B | OnCashItemResLoadGiftFailed | NEW |
| 0x5C | OnCashItemResLoadWishDone | existing (LOAD_WISHLIST) |
| 0x5D | OnCashItemResLoadWishFailed | NEW |
| 0x62 | OnCashItemResSetWishDone | existing (UPDATE_WISHLIST) |
| 0x63 | OnCashItemResSetWishFailed | NEW |
| 0x64 | OnCashItemResBuyDone | existing (PURCHASE_SUCCESS) |
| 0x65 | OnCashItemResBuyFailed | NEW |
| 0x66 | OnCashItemResUseCouponDone | NEW |
| 0x68 | OnCashItemResGiftCouponDone | NEW |
| 0x69 | OnCashItemResUseCouponFailed | NEW |
| 0x6B | OnCashItemResGiftDone | NEW (resolves 0x4D TODO) |
| 0x6C | OnCashItemResGiftFailed | NEW |
| 0x6D | OnCashItemResIncSlotCountDone | existing (INVENTORY_CAPACITY_INCREASE_SUCCESS) |
| 0x6E | OnCashItemResIncSlotCountFailed | existing (INVENTORY_CAPACITY_INCREASE_FAILED) |
| 0x6F | OnCashItemResIncTrunkCountDone | NEW |
| 0x70 | OnCashItemResIncTrunkCountFailed | NEW |
| 0x71 | OnCashItemResIncCharacterSlotCountDone | NEW |
| 0x72 | OnCashItemResIncCharacterSlotCountFailed | NEW |
| 0x73 | OnCashItemResIncBuyCharacterCountDone | NEW |
| 0x74 | OnCashItemResIncBuyCharacterCountFailed | NEW |
| 0x75 | OnCashItemResEnableEquipSlotExtDone | NEW |
| 0x76 | OnCashItemResEnableEquipSlotExtFailed | NEW |
| 0x77 | OnCashItemResMoveLtoSDone | existing (CASH_ITEM_MOVED_TO_INVENTORY) |
| 0x78 | OnCashItemResMoveLtoSFailed | NEW |
| 0x79 | OnCashItemResMoveStoLDone | existing (CASH_ITEM_MOVED_TO_CASH_INVENTORY) |
| 0x7A | OnCashItemResMoveStoLFailed | NEW |
| 0x7B | OnCashItemResDestroyDone | NEW |
| 0x7C | OnCashItemResDestroyFailed | NEW |
| 0x7D | OnCashItemResExpireDone | NEW |
| 0x96 | OnCashItemResRebateDone | NEW |
| 0x97 | OnCashItemResRebateFailed | NEW |
| 0x98 | OnCashItemResCoupleDone | NEW |
| 0x99 | OnCashItemResCoupleFailed | NEW |
| 0x9A | OnCashItemResBuyPackageDone | NEW |
| 0x9B | OnCashItemResBuyPackageFailed | NEW |
| 0x9C | OnCashItemResGiftPackageDone | NEW |
| 0x9D | OnCashItemResGiftPackageFailed | NEW |
| 0x9E | OnCashItemResBuyNormalDone | NEW |
| 0x9F | OnCashItemResBuyNormalFailed | NEW |
| 0xA2 | OnCashItemResFriendShipDone | NEW |
| 0xA3 | OnCashItemResFriendShipFailed | NEW |
| 0xAA | OnCashItemResFreeCashItemDone | NEW |
| 0xAF | OnCashItemResPurchaseRecord | NEW |
| 0xB0 | OnCashItemResPurchaseRecordFailed | NEW |
| 0xB3 | OnCashItemNameChangeResBuyDone | NEW |
| 0xB5 | OnCashItemResTransferWorldDone | NEW |
| 0xB6 | OnCashItemResTransferWorldFailed | NEW |
| 0xB7 | OnCashItemResCashGachaponOpenDone | NEW |
| 0xB8 | OnCashItemResCashGachaponOpenFailed | NEW |
| 0xB9 | OnCashItemResCashGachaponCopyDone | NEW |
| 0xBA | OnCashItemResCashGachaponCopyFailed | NEW |
| 0xBB | OnCashItemResChangeMaplePointDone | NEW |
| 0xBC | OnCashItemResChangeMaplePointFailed | NEW |

56 arms (`default: return` is not an arm). 9 existing + **47 new**.
