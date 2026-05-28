# Commerce-Domain Packet Audit — Design

Version: v1
Status: Proposed
Created: 2026-05-15
PRD: `prd.md`
Prior art:
- `../../task-027-atlas-packet-v95-audit/{design,plan,post-phase-b}.md` (pipeline, analyzer, registry, ack pattern)
- `../../task-028-character-domain-audit/{design,plan,post-phase-b}.md` (scaling the pipeline; multi-version pass; hot-path discipline)
- `../../task-066-social-domain-packet-audit/design.md` (the template this design follows; same author conventions, same phasing skeleton)

---

## 1. Design Goals

This is the fourth audit task to use the `tools/packet-audit/` pipeline. The pipeline is mature, the per-domain scaling pattern is mature, and the cross-version pass is mature. **This design is therefore not architectural — it is scope, sequencing, and the commerce-domain-specific risks the prior tasks did not face.**

Constraints driving the decisions below:

- **The pipeline already exists and works.** Don't re-design it. Don't touch the analyzer unless a concrete commerce finding forces a fix, and even then prefer `_pending.md` deferral over a tooling excursion (see §11).
- **The PRD's scope numbers are correct this time.** Unlike task-066, the PRD's count of 78 source files (cash 30 + interaction 32 + inventory 9 + storage 7) was re-verified against the worktree (`ls libs/atlas-packet/<d>/<sd>/*.go | grep -v _test.go`). The denominator stands at **78 src files**, but with one important wrinkle: **five files declare multiple wire-shape struct types** (§3.1), so the audit *report* count is higher than the *file* count. SUMMARY.md rows are per-shape, not per-file.
- **Interaction sub-op dispatch is the best-case scenario, not the deferral pit it appears to be.** `interaction/serverbound/operation.go` writes only the leading `mode` byte (verified at `interaction/serverbound/operation.go:32`). Each sub-op (`operation_merchant_buy.go`, `operation_trade_put_item.go`, …) is a separate file containing only its post-discriminator payload. The analyzer addresses each sub-op file independently — exactly the pattern task-066 §5 celebrated for guild/party. The PRD §4.7 sub-op enum deferral concentrates in **one** `_pending.md` row keyed by "interaction mode-byte → sub-op-file routing," not 30 rows.
- **Cash is the heavy domain, not interaction.** Cash has 30 packets including the `shop_operation_buy_*` family (8 individually-named purchase variants), the `shop_operation_increase_*` family (3 capacity-increase variants), the `shop_operation_move_*` family (cash-inventory ↔ regular-inventory), and the cross-cutting `shop_operation_body.go` (78 named error/operation codes — pure constants, no encoders). It is also the domain with the most v83-vs-v95 wire-shape divergence risk (NX point category expansion, gift certificates, world-transfer, name-change, friendship/couple/package SKUs).
- **Inventory `change.go` is the silent-loss hot path.** Single file, four wire shapes (`QuantityUpdate`, `ChangeMove`, `Remove`, `Add`), each fired on every item-affecting transition. A 1-byte width error here drops items at the client. Treat with the same `member_hp.go` discipline task-066 §6 applied to party HP.
- **Storage is small but durable.** Only 7 packets, but `storage/clientbound/update_assets.go` writes the entire storage panel — one wire-shape error desyncs storage durably across sessions until the user logs out and the panel is re-issued. Small surface, high blast radius.
- **No retroactive scope creep.** Same discipline as task-027/028/066: stop at the commerce domain. Every "while we're here" gets a sibling task.
- **Bare-handler exclusion stays.** Per task-027/028/066 precedent and PRD §4.7, atlas-side handlers without a `libs/atlas-packet` decoder go to `_pending.md` — no service-code descent into atlas-cashshop, atlas-inventory, atlas-storage, atlas-channel.
- **Auto-discovery handles most sub-structs; the registry "extension" PRD §4.5 talks about is narrower than it sounds.** The TypeRegistry walks `libs/atlas-packet/` (verified at `tools/packet-audit/internal/atlaspacket/registry.go:46-79`) and auto-registers every struct with an `Encode` or `Write` method — `model.Asset`, `inventory.ChangeEntry`, and the per-storage struct types are picked up for free. The actual gap is **non-`Encode` foreign-encoder methods** like `CashInventoryItem.EncodeBytes(l)` (verified at `cash/clientbound/shop_inventory.go:25`) — the analyzer doesn't recognize these. Section §8 spells out the precise registry treatment.

---

## 2. Architecture Overview

No architectural change. Data flow established by task-027 §2 is unchanged:

```
CSV ─→ template ─→ IDA source ─→ atlas-packet analyzer ─→ diff engine ─→ report writer
                                                ↑
                                                │
                                          TypeRegistry
```

What changes for this task is **what the pieces ingest**:

| Piece                | task-066 input                                                     | task-067 input                                                                |
|----------------------|--------------------------------------------------------------------|-------------------------------------------------------------------------------|
| Atlas source         | `libs/atlas-packet/{guild,party,buddy,messenger,note,chat}/{cb,sb}/`| `libs/atlas-packet/{cash,interaction,inventory,storage}/{cb,sb}/`              |
| IDA exports          | `gms_v95.json` (social append)                                     | `gms_v95.json` (commerce append)                                              |
| IDA exports (cross)  | `gms_v83.json`, `gms_v87.json`, `gms_jms_185.json` (social rows)   | Same four — **append** commerce rows                                          |
| Templates            | Social opcodes/sub-ops                                             | Commerce opcodes/sub-ops; cash-shop "operations" enum is a 9-entry minimum    |
| TypeRegistry         | Auto-discovers GuildMember/PartyHPBar/BuddyEntry/MessengerChat     | Auto-discovers `model.Asset` + per-file structs; **`CashInventoryItem.EncodeBytes` requires explicit registration** (§8) |
| Analyzer             | Early-return walker shipped in task-028; no changes                | No analyzer change expected; one possible exception: handling `EncodeBytes` as a foreign-encoder method (§8.2) |

Read-only against `libs/atlas-packet/` (except wire-bug fixes); writes only to:

- `tools/packet-audit/internal/atlaspacket/registry.go` (+ matching `registry_test.go`) — only if §8 confirms the EncodeBytes gap.
- `libs/atlas-packet/{cash,interaction,inventory,storage}/{cb,sb}/*.go` (wire-bug fixes only).
- `services/atlas-configurations/seed-data/templates/template_{gms_*,jms_185}_1.json` (opcode/enum fixes).
- `docs/packets/audits/gms_v95/{cash,interaction,inventory,storage}/` (audit reports).
- `docs/packets/ida-exports/{gms_v83,gms_v87,gms_v95,gms_jms_185}.json` (IDA evidence appends).
- `docs/packets/audits/gms_v95/_pending.md` (deferral rows).

---

## 3. Coverage matrix (PRD §4.1 verified)

Re-enumerated against the worktree:

| Domain      | Clientbound (files) | Serverbound (files) | Total files | Distinct wire shapes |
|-------------|---------------------|---------------------|-------------|----------------------|
| cash        | 6                   | 24                  | 30          | ~36 (multi-shape: shop_inventory, shop_operation_result, shop_item_moved) |
| interaction | 2                   | 30                  | 32          | 32                   |
| inventory   | 4                   | 5                   | 9           | ~12 (multi-shape: change.go has 4 shapes) |
| storage     | 3                   | 4                   | 7           | ~9 (multi-shape: error.go has 3 shapes)   |
| **Total**   | **15**              | **63**              | **78**      | **~89 wire-shape rows** |

Files match PRD §4.1 exactly. The wire-shape count is higher because of the multi-struct files itemized in §3.1. SUMMARY.md generates one row per *exposed wire shape* (struct with `Encode` or `Write`), not per file.

### 3.1 Multi-struct files (audit-row inflation)

| File                                            | Wire shapes                                                                       | Source-of-truth |
|-------------------------------------------------|-----------------------------------------------------------------------------------|-----------------|
| `cash/clientbound/shop_inventory.go`            | `CashInventoryItem` (sub-struct), `CashShopInventory`, `CashShopPurchaseSuccess`, `CashShopGifts` | verified |
| `cash/clientbound/shop_operation_result.go`     | `OperationError`, `InventoryCapacitySuccess`, `InventoryCapacityFailed`, `WishList` | verified |
| `cash/clientbound/shop_item_moved.go`           | `CashItemMovedToInventory`, `CashItemMovedToCashInventory`                        | verified |
| `storage/clientbound/error.go`                  | `ErrorSimple`, `UpdateMeso`, `ErrorMessage`                                       | verified |
| `inventory/clientbound/change.go`               | `QuantityUpdate`, `ChangeMove`, `Remove`, `Add` (each writes its own dispatch byte) | verified |

Plan-task expands the SUMMARY.md row template to use `<file>:<TypeName>` so reviewers can find each shape unambiguously. The `cash/clientbound/shop_operation_body.go` file is a 78-entry constant block plus six factory functions — it carries no `Encode` of its own; its rows are the four constructors (`CashShopWishListBody`, `CashShopCashInventoryBody`, `CashShopCashInventoryPurchaseSuccessBody`, `CashShopCashItemMovedToInventoryBody`, `CashShopCashItemMovedToCashInventoryBody`, `CashShopInventoryCapacityIncreaseSuccessBody`, `CashShopInventoryCapacityIncreaseFailedBody`, `CashShopCashGiftsBody`) which all delegate to the per-shape structs above. Treat `shop_operation_body.go` as a router: one SUMMARY.md row noting "router; per-shape rows tracked under shop_inventory.go / shop_operation_result.go / shop_item_moved.go," not seven duplicate rows.

### 3.2 PRD §10 acceptance restated

"All 78 listed packet source files have audit reports" becomes:

> All 78 commerce-domain packet src files map to at least one wire-shape audit row, and every wire-shape row carries a verdict (✅/⚠️/❌) or a `_pending.md` reference. Total expected SUMMARY.md row count is ~89, not 78.

The denominator for completeness is wire-shape count (~89), not file count (78). Plan-task captures both.

---

## 4. The hard part #1: cash-shop wire-shape variants

Cash dominates this audit by raw shape count and by version-divergence risk. Three sub-problems:

### 4.1 The `shop_operation_buy_*` family (8 variants)

`shop_operation_buy.go` is the umbrella struct; the seven `_buy_normal`, `_buy_couple`, `_buy_friendship`, `_buy_package`, `_buy_name_change`, `_buy_world_transfer`, `_buy_gift` (note: `gift` lives in `shop_operation_gift.go`, not `shop_operation_buy_gift.go`) variants each emit a different post-discriminator payload. Each variant maps to a distinct CashShop dispatcher case in IDA. Treat each as an independent ❌/⚠️/✅ row — no roll-up. Audit comments cite the IDA case-statement value for each variant.

**Risk:** these are the wire shapes most likely to differ between v83 (no NX credit categories) and v95 (Maple Points / NX Credit / NX Prepaid split). Cross-version Phase 2 v83 must inspect each `_buy_*` variant individually; do not assume a single `Region/MajorVersion` gate covers the family.

### 4.2 The `shop_operation_increase_*` family (3 variants)

`shop_operation_increase_inventory.go`, `shop_operation_increase_storage.go`, `shop_operation_increase_character_slot.go` — three capacity-increase variants. Each writes a different post-discriminator payload (inventory type byte vs storage row count vs character-slot count). Same per-variant audit treatment as §4.1.

### 4.3 The `shop_operation_body.go` constant block

78 named operation/error codes. These are template-driven (`atlas_packet.ResolveCode("operations", ...)` and `atlas_packet.ResolveCode("errors", ...)` — verified at `cash/clientbound/shop_operation_body.go:79-138`). The code-name → wire-byte mapping lives in `template_*.json`. Two operational outcomes:

1. **The template has a code missing** for a version → atlas resolves to the zero default and emits the wrong byte. Phase 2 cross-version pass catches these by comparing the template's "operations"/"errors" sub-maps against the IDA case-statement values for `CCashShop::OnPacket` at v83/v87/JMS.
2. **The template has a code at a stale value** (the task-028 `0xE7 vs 0xB4` lesson, but for cash-shop). Each fix lands as a single template integer change, commit message citing the IDA case offset.

The 78 codes fan out across at least three IDA dispatcher functions (load-inventory, purchase, move). Cap: if a single Phase 2 version surfaces >10 stale code values, pause and triage — the template's regional split may need restructuring, not a 10-line fix.

### 4.4 `CashInventoryItem` is the cash-shop sub-struct

`CashInventoryItem.EncodeBytes(l)` (verified at `cash/clientbound/shop_inventory.go:25`) writes 49 bytes via direct writer calls. It is referenced by `CashShopInventory.Encode` (loop), `CashShopPurchaseSuccess.Encode`, and `CashItemMovedToCashInventory.Encode`. Because the method is named `EncodeBytes` (not `Encode`), the registry's auto-discovery will not pre-analyze its body — see §8 for the registry treatment decision.

---

## 5. The hard part #2: interaction operation-dispatcher family

Verified at `interaction/serverbound/operation.go:29-41`: the dispatcher writes only the `mode` byte and reads only `mode` on decode. Each sub-op file (`operation_merchant_buy.go`, `operation_trade_put_item.go`, `operation_personal_store_buy.go`, …) is a stand-alone struct with its own `Encode/Decode` carrying the post-discriminator payload. The analyzer addresses each individually.

This is the **best-case scenario** task-066 §5 named for guild/party. What changes for commerce:

- **30 sub-op files instead of 19.** Larger surface, but still per-file independent.
- **Three sub-op families:** trade (`operation_trade_*.go`, `operation_create.go` for trade open, `operation_invite.go` family), hire-merchant (`operation_merchant_*.go` + `operation_merchant_name_change.go`), personal-store (`operation_personal_store_*.go`), memory-game (`operation_memory_game_*.go`), and visit/cash-trade (`operation_visit.go`, `operation_cash_trade_open.go`, `operation_chat.go`, `operation_open.go`, `operation_transaction.go`, `operation_invite_decline.go`, `operation_field_*_black_list.go`).

Audit treatment per task-066 §5:

- **`operation.go`'s row** is ⚠️ "tool-limitation: mode byte supplied by caller; sub-op routing recorded in `_pending.md` row INTERACTION-MODE-MAP."
- **Each sub-op file's row** is a normal ✅/⚠️/❌ verdict against the IDA case body for its mode value.
- **The mode-byte → sub-op-file table** is captured once in `_pending.md` as a static reference (mode value → sub-op file → IDA case offset). This is documentation, not a tooling gap, and lives in atlas-channel routing — outside `libs/atlas-packet/`.

### 5.1 Interaction clientbound (2 files)

`interaction/clientbound/interaction.go` and `interaction_body.go`. `interaction.go` is likely the broadcast envelope; `interaction_body.go` is likely the constructor block (parallel to `cash/clientbound/shop_operation_body.go`'s pattern). Phase 1 reading confirms the structural mapping; expect both files to be routers around a per-mode struct family that already lives elsewhere or needs a `_pending.md` row for "clientbound interaction mode-body modeling."

---

## 6. The hard part #3: inventory change packets — the silent-loss hot path

`inventory/clientbound/change.go` exposes four wire shapes (`QuantityUpdate`, `ChangeMove`, `Remove`, `Add`) — each writes its own change-mode discriminator byte (1=update / 2=move / 3=remove / 0=add per `inventory.ChangeMode*` constants). All four are fired on the per-character `CharacterInventoryChange` envelope after every item-affecting transition: NPC shop buy, drop pickup, drop discard, scroll use, item use, equip, unequip, slot move, batch sort, batch merge.

Discipline (mirrors task-028 §8 / task-066 §6):

- **4-variant byte-output test sweep is mandatory** for every change-shape fix. `pt.Variants` (GMS v28/v83/v95 + JMS v185) covers the gate axis; the `silent` boolean is a separate axis that should be exercised by at least two test cases per variant (silent + not-silent) since the leading `WriteBool(!silent)` is a known wire byte.
- **Inventory type 1 (equip) special-case:** `ChangeMove` and `Remove` write an additional `addMov` byte when an equip slot crosses zero (`inventoryType == 1 && slot < 0`) — verified at `change.go:92-99` and `change.go:146-148`. The IDA dispatcher for `CWvsContext::OnInventoryOperation` is the source of truth for the addMov byte's encoding; audit must confirm the byte is emitted at exactly the v95 boundaries.
- **`change_batch.go` is the multi-op shape.** Verifies that batched `ChangeEntry` lists in `inventory/change_entry.go` are recognized by the analyzer (auto-discovery should pick up `ChangeEntry.Encode` if present; verify in Phase 0).
- **`inventory/clientbound/compartment_merge.go` and `compartment_sort.go`** are the panel-refresh shapes — fired on full-inventory recompaction. These are not per-item but their wire shape includes the inventory-type byte; gate-boundary risk exists (the v95 inventory expansion may have widened the slot-count field).
- **No `reflect.*`, no new `interface{}` options.** Variant axis from `tenant.Model` only.

`inventory/serverbound/move.go` is the client-driven move handler — it reads what the client sends, the server then echoes via `change.go`'s `ChangeMove`. Wire-shape mismatch between the serverbound `Decode` and the clientbound `Encode` for the equip-slot `addMov` byte is the single most likely silent-corruption bug in this domain. Phase 1c (inventory) audits both halves of the move flow and asserts symmetry.

---

## 7. The hard part #4: storage durability

7 packets. Small surface, high blast radius — a wire-shape bug in `storage/clientbound/update_assets.go` desyncs the storage panel until the player logs out and the panel is re-issued. `storage/clientbound/show.go` is the initial open; `error.go` carries three shapes (simple error code, meso-update reflow, message-string error).

Discipline:

- 4-variant test sweep per fix.
- `storage/clientbound/error.go`'s three shapes each get individual rows; the dispatch byte is emitted by each shape (`ErrorSimple` writes one mode byte, `UpdateMeso` writes a different mode byte, `ErrorMessage` writes a third).
- `update_assets.go` is the durable-state writer — Phase 2 cross-version verification is mandatory; v83 and v95 storage-slot widths almost certainly differ (storage was widened from 16 → 24 slots somewhere between v62 and v95).

`storage/serverbound/operation.go` is the dispatcher equivalent of the interaction operation pattern: writes only the mode byte, with `operation_meso.go`, `operation_retrieve_asset.go`, `operation_store_asset.go` as the three sub-ops. Same per-sub-op audit treatment as §5.

---

## 8. The hard part #5: TypeRegistry registration for `EncodeBytes`

The registry's auto-discovery in `tools/packet-audit/internal/atlaspacket/registry.go:79-114` only switches on `Encode` and `Write` method names. `CashInventoryItem.EncodeBytes(l logrus.FieldLogger) []byte` (verified at `cash/clientbound/shop_inventory.go:25`) is invisible to this discovery — when the analyzer encounters `w.WriteByteArray(item.EncodeBytes(l))` inside `CashShopInventory.Encode`, it cannot recurse into the item's body and will report the call as an opaque `Decode1`/`WriteByteArray` writer.

Two options:

**Option A — Extend auto-discovery to `EncodeBytes`** (preferred, ~10 LOC change in `registry.go:101-114`)

Add an `EncodeBytes` case to the method-name switch, treating its body as a flat `Write` body (no closure return). Trade-offs:

- ✅ One-shot fix; future commerce-style sub-structs benefit automatically.
- ✅ No per-call manual tagging.
- ⚠️ Counts as an analyzer change. Per §1 design constraint, prefer not. But this is a *registry* extension (additive method recognition), not an analyzer behavior change — the analyzer's call-collection logic is unchanged. Argument for greenlight: the change is mechanical, ships with a fixture in `registry_test.go`, and unblocks the cash audit cleanly.
- ⚠️ Risk: some other `EncodeBytes` method in `libs/atlas-packet/` may exist with non-flat semantics. Phase 0 survey: `grep -rn "func.*EncodeBytes" libs/atlas-packet/` to enumerate. If only `CashInventoryItem.EncodeBytes` exists, Option A is safe.

**Option B — `_pending.md` row + per-call ack** (status-quo deferral)

Each call site (`shop_inventory.go:88`, `shop_operation_body.go:131-135` chain into `CashItemMovedToCashInventory`, `shop_inventory.go:131-141` into `CashShopPurchaseSuccess`) gets a `⚠️ tool-limitation` annotation in its audit report citing the analyzer's blindness. One `_pending.md` row keyed by "EncodeBytes-style foreign encoders" with sub-list.

- ✅ Zero analyzer/registry change.
- ⚠️ Three audit rows lose meaningful sub-shape verification (the cash-shop item *is* the most security-critical commerce sub-struct).
- ⚠️ Recurring problem — if the next commerce-style audit (storage gift-receive flow, NPC-shop locker) introduces another `EncodeBytes`-shaped sub-struct, the deferral compounds.

**Recommendation:** Option A, gated on the Phase 0 `grep` confirming `CashInventoryItem.EncodeBytes` is the only such method. If multiple `EncodeBytes` methods exist with divergent semantics, fall back to Option B.

### 8.1 Auto-discovered registry coverage (no action required)

These structs already register correctly via `Encode`:

- `model.Asset.Encode` (`libs/atlas-packet/model/asset.go:164`) — the inventory item slot sub-struct, used by `inventory/clientbound/change.go:Add`, `cash/clientbound/shop_item_moved.go:CashItemMovedToInventory`, and likely `storage/clientbound/{show,update_assets}.go`.
- `inventory.ChangeEntry.Encode` (verify Phase 0; if file exposes `Encode`, registry picks it up).
- `interaction/{room,visitor}.go` and `mini_room.go` — top-level `interaction/` package files; auto-discovered.
- All per-shape structs in cash/clientbound, storage/clientbound, inventory/clientbound except `CashInventoryItem`.

### 8.2 Registration discipline (carried from task-028 §4.1 / task-066 §7.1)

For each *manually-added* registration (i.e., the `EncodeBytes` extension if Option A lands):

1. The registry change ships with a fixture in `registry_test.go` asserting the analyzed primitive-field list.
2. Don't pre-emptively register every `EncodeBytes`-style method in the codebase — register only the body of `CashInventoryItem.EncodeBytes` once the extension is in place; the registry's *auto-discovery* will then pick up any future `EncodeBytes` method without further code change.
3. The analyzer-change commit is its own commit, dispatched and verified before any cash audit row is written. Don't mix tooling and audit commits.

### 8.3 Cross-domain ripple (carried from task-028 §4.3 / task-066 §7.2)

Registry additions are additive. Login (task-027), character (task-028), social (task-066) verdicts must not regress. Phase 3 closing memo confirms by re-running prior audits.

---

## 9. The hard part #6: cross-version cash-shop divergence

Cash-shop's wire shape is the most version-sensitive surface in the commerce domain. Concrete known/predicted divergences:

- **NX point category:** v83 likely uses a single NX integer; v95 splits into NX Credit / NX Prepaid / Maple Points. The `shop_operation_buy_*` family's leading "payment type" byte may not exist in v83. Cross-version Phase 2 must inspect `cash/serverbound/check_wallet.go` and `shop_entry.go` against v83 IDA before assuming a single gate suffices.
- **Cash-shop SKU expansion:** `shop_operation_buy_couple.go`, `shop_operation_buy_friendship.go`, `shop_operation_buy_world_transfer.go` may not exist as v83 features at all. If the IDA dispatcher for v83 has no case for these mode bytes, the v83 audit row reads "✅ N/A — feature absent in v83; atlas behavior gated correctly" or "❌ — atlas attempts to encode; v83 client will reject."
- **JMS v185 NX point system:** PRD §9 flags this as uncharted. Likely outcome: a structural divergence requires either a `Region() == "JMS"` gate added at file-top (if scope is small) or a sibling task (if scope crosses 2 nested gates per task-028 §7).
- **`CashInventoryItem` 49-byte width:** the `WritePaddedString(GiftFrom, 13)` field plus two trailing `WriteInt(0)` padding bytes (verified at `shop_inventory.go:33-36`) are version-sensitive. Strong candidate for v83 divergence (gift-from field may be absent or shorter).
- **Inventory item-slot width:** parallel to task-028's `GW_CharacterStat` HP/MP widening. Every `int16` slot field in inventory/storage encoders is a cross-version gate candidate.

Hard cap (carried from task-028 §7): no encoder grows beyond 2 nested `Region/MajorVersion` guards. 3+ → STOP, log to `_pending.md`, do not refactor under audit cover.

---

## 10. Phasing — concrete artifacts

Phasing follows task-028 / task-066, scaled to 78 packets across 4 sub-domains.

### Phase 0 — Survey + (optional) registry batch (gate)

Three sub-tasks:

0a. **`EncodeBytes` audit (10 minutes).** `grep -rn "func.*EncodeBytes(" libs/atlas-packet/`. If the result is `CashInventoryItem` and nothing else, proceed with Option A from §8. Otherwise enumerate the divergent semantics and decide A vs B per-call.

0b. **`model.Asset` and `inventory.ChangeEntry` confirm.** `grep -n "func.*Asset.*Encode\|func.*ChangeEntry.*Encode" libs/atlas-packet/`. Confirm both are auto-discovered. If `ChangeEntry` exposes a different method (e.g., `EncodeForeign` like the `CharacterTemporaryStat` precedent), record as a follow-up registration.

0c. **Registry change** (only if 0a chose Option A). One commit: registry.go method-switch extension + registry_test.go fixture for `CashInventoryItem.EncodeBytes`. Exit: `go test -race ./tools/packet-audit/...` clean.

A short note in `<task-folder>/phase-0-survey.md` records 0a/0b findings (transient; folded into `post-phase-b.md` at Phase 4).

Exit gate: registry tests green; the `EncodeBytes` extension is either landed or explicitly deferred to Option B.

### Phase 1 — v95 audit by sub-domain

Four sub-phases, one per commerce sub-domain. Suggested ordering (small → big, durable → ephemeral, simple → cross-cutting):

- **1a — storage** (7 packets, ~9 wire shapes). Smallest. Confirms the §5/§7 dispatcher pattern reading on the `storage/serverbound/operation.go` family. Exercises `model.Asset` recursion through `update_assets.go`.
- **1b — inventory** (9 packets, ~12 wire shapes). Hot path per §6. The change.go four-shape sweep is the marquee item; equip-slot `addMov` symmetry is the marquee verification.
- **1c — interaction** (32 packets, 32 wire shapes). Biggest by file count, simplest by per-file shape (each sub-op file is one shape). Both clientbound files (`interaction.go`, `interaction_body.go`) get individual treatment per §5.1. Sub-op routing → one `_pending.md` row per §5.
- **1d — cash** (30 packets, ~36 wire shapes). Highest density of multi-shape files (§3.1) and the most version-sensitive. The `shop_operation_buy_*` and `shop_operation_increase_*` families per §4.1/§4.2; `shop_operation_body.go` constants per §4.3; `CashInventoryItem` recursion per §4.4.

Each sub-phase ends with:

- Audit reports for every wire shape in that sub-domain.
- `SUMMARY.md` rows added (verdict + IDA address + notes/citation per row).
- Real wire-bug fixes committed individually with 4-variant test sweep per fix.
- Template opcode/sub-op fixes committed individually with the case-statement value in the commit message.
- `_pending.md` rows for bare handlers + sub-op enum spaces + (if Option B) per-`EncodeBytes` ack.

Sub-phase done when SUMMARY.md shows a verdict or `_pending.md` entry for every wire shape in that sub-domain. No silent skips.

### Phase 2 — Cross-version pass (v83 → v87 → JMS v185)

One commit batch per version, user-driven IDA swap (PRD §4.6).

For each version, for each commerce FName:

1. Populate `docs/packets/ida-exports/gms_{v83,v87}.json` or `gms_jms_185.json` (commerce rows append).
2. Re-run audit with that version's IDA source + template.
3. For divergences vs v95:
   - Existing `Region/MajorVersion` gate handles it → no code change; export row captures evidence.
   - Atlas gate is wrong → fix the gate, 4-variant sweep, document.
   - Template opcode/enum drift → fix the template, cite case-statement value.
   - Feature absent in this version (e.g., world-transfer in v83) → atlas should not emit; verify gate or `_pending.md` row.
4. Hard cap: 2 nested `if t.Region()` / `if t.MajorVersion()` levels per encoder. 3+ → STOP, `_pending.md`, do not refactor under audit cover.

Special attention per §9: cash-shop SKU absences in v83, JMS NX system divergence, `CashInventoryItem` width changes.

Each version commit batch named `phase-2-v83`, `phase-2-v87`, `phase-2-jms-185` for review traceability.

### Phase 3 — Login + character + social regression confirm

Mechanical re-run of task-027, task-028, task-066 audits, verifying no verdict regression. Commands:

```
go run ./tools/packet-audit \
  --csv-clientbound docs/packets/MapleStory\ Ops\ -\ ClientBound.csv \
  --template services/atlas-configurations/seed-data/templates/template_gms_95_1.json \
  --atlas-packet libs/atlas-packet \
  --ida-source docs/packets/ida-exports/gms_v95.json \
  --output docs/packets/audits/gms_v95
```

Diff `SUMMARY.md` against task-066's closing snapshot. Cap: 2 new ❌s across login + character + social before stop-and-split (task-028 §3.5 logic). The §8 registry extension is the most likely source of regression — if `EncodeBytes` recognition surfaces a previously-hidden recurse mismatch in some character or social packet, that flip is in-scope to triage.

### Phase 4 — post-phase-b.md + verification + code review

Mirror the task-028 / task-066 closing pattern:

- Write `docs/tasks/task-067-commerce-domain-packet-audit/post-phase-b.md` with five sections (final state, real wire bugs fixed, template fixes, tooling improvements, remaining work / deferrals).
- Run the four verification commands per PRD §10:
  - `go build ./...`
  - `go vet ./libs/atlas-packet/...`
  - `go test -race ./libs/atlas-packet/...`
  - `go test -race ./tools/packet-audit/...`
- Run `docker build -f services/atlas-configurations/Dockerfile .` only if seed-data structure (not values) changed.
- `gitleaks` scrub: `grep -rn '/home/' docs/packets/audits/gms_v95/{cash,interaction,inventory,storage}/` must be empty.
- Invoke `superpowers:requesting-code-review` (plan-adherence + backend-guidelines).
- Open PR.

---

## 11. Templates — opcode vs sub-op (carried from task-028 §9 / task-066 §9)

Two surfaces:

- **Opcode drift** = the `case N` in the dispatch switch changed between versions. Single integer fix in `template_*.json`. Trivial.
- **Sub-op (enum) drift** = a writer that routes correctly emits a sub-op byte whose value-to-meaning mapping shifted. Requires reading the IDA function's internal switch table.

Commerce-domain sub-op enum suspects (prioritised):

- `cash/clientbound/shop_operation_body.go` — the 78-entry `operations`/`errors` enum block (§4.3). The richest sub-op surface in the audit.
- `interaction/serverbound/operation.go` mode-byte map (covered by §5's `_pending.md` row).
- `interaction/clientbound/interaction_body.go` mode-byte map (likely a parallel `_pending.md` row per §5.1).
- `storage/serverbound/operation.go` mode-byte map (3 sub-ops; small, may inline directly).
- `storage/clientbound/error.go` per-shape mode bytes (3 shapes; small, may inline).
- `inventory.ChangeMode*` enum (4 values: 0=Add, 1=QuantityUpdate, 2=Move, 3=Remove — verified by `change.go:43,88,143,192`). Already in `libs/atlas-packet/inventory/`. Verify against IDA.

Sub-op enum modeling is not in scope for tooling fixes. Document the limitation in one `_pending.md` row keyed by "sub-op enum modeling — commerce domain" with a sub-list of affected files. Single row, not one per file.

---

## 12. Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The `EncodeBytes` registry extension (§8 Option A) breaks an existing audit row | Low-medium | Medium | Phase 0a confirms `CashInventoryItem.EncodeBytes` is the only method with that name. Phase 3 regression catches any login/character/social verdict flip. Cap: 2 new ❌s before stop-and-split. |
| Cash-shop sub-shape count (§3.1) is undercounted — some `shop_operation_body.go` constructor maps to a previously-unidentified struct in another file | Medium | Low | Phase 1d opens with a constructor-to-struct enumeration commit (`grep "atlas_packet.WithResolvedCode\|NewCashShop\|NewCashItem" cash/clientbound/`). Update `SUMMARY.md` denominator if the wire-shape count grows past 36 for cash. |
| Interaction sub-op family proves to have shared payload bytes (mode written by `operation.go`, but some sub-files re-write the mode for re-routing inside hire-merchant or trade) | Low | Medium | §5's reading is from `operation.go:32`. If Phase 1c finds IDA evidence of a shared payload byte, that becomes a new finding in the report; if a sub-file re-writes mode, that's a wire-bug fix candidate, not a structural design failure. |
| Inventory equip-slot `addMov` byte (§6) has a v83-vs-v95 width change that the current `inventoryType == 1 && slot < 0` gate misses | Medium | High | Phase 1b 4-variant sweep is mandatory and exercises both silent/non-silent × all variants. Cross-version Phase 2 v83 specifically inspects equip-slot move/remove. |
| Storage update_assets desync — a wire-bug fix shifts byte layout for v83 in a way that desyncs an active session before the player logs out and re-issues the panel | Medium | High | §7 4-variant byte-output asserts (not just round-trip). Cross-version Phase 2 v83 inspects update_assets specifically; the byte-output assertion catches mid-encoding drift the round-trip test would miss. |
| Cash-shop `_buy_*` variants share enough shape that "fix one, break another" cascades during Phase 1d | Medium | Medium | §4.1: each variant is its own row, its own commit, its own 4-variant test. No multi-variant fix commits. If a fix would touch 2+ variant files, treat as "shared encoder needs split" and pause for triage. |
| JMS v185 cash-shop is a structural rewrite, not a gate adjustment (PRD §9) | Medium | High | §9 hard cap on 2 nested guards. If JMS-only NX system requires a 3+ gate, `_pending.md` and sibling task. Don't ship a 3-gate encoder under cover of an audit task. |
| `_pending.md` row inflation — one row per sub-op file + one per bare handler + one per `EncodeBytes` ack + one per template constant = noise that obscures real deferrals | Medium | Low | Group deferrals by *cause*, not by *file*. One row per limitation with a sub-list (carry from task-066 §10). |
| Cross-version pass surfaces a v83 regression introduced by a v95 fix in a hot commerce packet (inventory move) | Medium | High | §6 fix-discipline + Phase 2 v83 re-run. Plan owner reviews every Phase 2 diff. |
| Login + character + social regression (Phase 3 finds verdict flip) | Low-medium | Medium | Phase 3 is a gate, not an afterthought. Run before Phase 4. Cap: 2 new ❌s before stop-and-split. The §8 registry extension is the most likely source. |
| Branch hygiene drift across 4 sub-phases × dozens of commits | Medium | Low | One commit per wire-bug fix or per template-constant fix group. `superpowers:finishing-a-development-branch` rebase-cleans before PR. |
| gitleaks catches absolute paths in audit reports (task-027/028/066 all had this) | High | Low | Phase 4 pre-PR check: `grep -rn '/home/' docs/packets/audits/gms_v95/<commerce>/` must be empty. Plan ledger row. |
| Retroactive scope expansion (task-027 ballooning pattern) | Medium | Medium | This design explicitly disallows mid-task pivots into atlas-channel routing logic, atlas-cashshop business logic, sub-op enum tooling work, or service-layer fixes beyond minimum constructor ripple. Every "while we're here" gets a sibling task. |
| Constructor-signature ripple lands a `services/atlas-cashshop/` or `services/atlas-inventory/` Dockerfile rebuild requirement that's missed | Medium | Medium | PRD §8 mandates `docker build -f services/<svc>/Dockerfile .` if `go.mod` or `Dockerfile` is touched. Plan ledger row per service. |

---

## 13. Out of scope (explicit)

- Bare-handler descent into atlas-channel, atlas-cashshop, atlas-inventory, atlas-storage service code (PRD §3 non-goal).
- Sub-op enum modeling in the audit pipeline (task-028 §9 limitation — acknowledged, not fixed).
- Sub-struct registry coverage for any type the commerce domain doesn't reference (e.g., chair/quest/buff sub-structs).
- Performance work on hot packets (inventory `change.go`, party HP equivalent for storage). Wire-shape fixes only.
- Analyzer extensions of any kind. The §8 Option A registry change is **registry method-name recognition**, not analyzer behavior change. If §8 surfaces a need to actually rewrite analyzer call-collection logic, that's a sibling task.
- atlas-cashshop Kafka CREATED/DELETED event handling (PRD §3 non-goal — service is N/A for map/field packets).
- v28 binary integration (task-028 §6 — defer to a sibling task if a binary surfaces).
- Domains outside cash/interaction/inventory/storage. Map (warp, drop, mob spawn), NPC, quest, monster-book, chair are sibling tasks.
- Generic packet-DSL or schema-first encoder rewrite (task-027 §12).
- Stock-Nexon clientVariant axis additions beyond what task-027 already shipped.
- Service-layer logic changes (atlas-cashshop NX accounting, atlas-inventory move atomicity, atlas-storage durability) — fixes are wire-shape only; if a fix requires upstream caller changes, document and split.
- Cash-shop operations/errors enum *expansion* (adding new code names) — this audit fixes existing-code value drift only. New regional codes are sibling tasks.
- Interaction trade flow business logic (slot reservation, fairness invariants) — wire-shape only.

---

## 14. File enumeration (canonical input list)

Recorded so plan-task does not re-derive. Excludes `_test.go`. Verified against `ls libs/atlas-packet/<d>/<sd>/*.go | grep -v _test.go` on 2026-05-15.

**cash/clientbound (6):** `query_result.go`, `shop_inventory.go`, `shop_item_moved.go`, `shop_open.go`, `shop_operation_body.go`, `shop_operation_result.go`

**cash/serverbound (24):** `check_wallet.go`, `item_use.go`, `item_use_chalkboard.go`, `item_use_field_effect.go`, `item_use_pet_consumable.go`, `shop_entry.go`, `shop_operation.go`, `shop_operation_buy.go`, `shop_operation_buy_couple.go`, `shop_operation_buy_friendship.go`, `shop_operation_buy_name_change.go`, `shop_operation_buy_normal.go`, `shop_operation_buy_package.go`, `shop_operation_buy_world_transfer.go`, `shop_operation_enable_equip_slot.go`, `shop_operation_get_purchase_record.go`, `shop_operation_gift.go`, `shop_operation_increase_character_slot.go`, `shop_operation_increase_inventory.go`, `shop_operation_increase_storage.go`, `shop_operation_move_from_cash_inventory.go`, `shop_operation_move_to_cash_inventory.go`, `shop_operation_rebate_locker_item.go`, `shop_operation_set_wishlist.go`

**interaction/clientbound (2):** `interaction.go`, `interaction_body.go`

**interaction/serverbound (30):** `operation.go`, `operation_cash_trade_open.go`, `operation_chat.go`, `operation_create.go`, `operation_field_add_to_black_list.go`, `operation_field_remove_from_black_list.go`, `operation_invite.go`, `operation_invite_decline.go`, `operation_memory_game_flip_card.go`, `operation_memory_game_move_stone.go`, `operation_memory_game_retreat_answer.go`, `operation_memory_game_tie_answer.go`, `operation_merchant_add_to_black_list.go`, `operation_merchant_buy.go`, `operation_merchant_name_change.go`, `operation_merchant_put_item.go`, `operation_merchant_remove_from_black_list.go`, `operation_merchant_remove_item.go`, `operation_open.go`, `operation_personal_store_add_to_black_list.go`, `operation_personal_store_buy.go`, `operation_personal_store_put_item.go`, `operation_personal_store_remove_item.go`, `operation_personal_store_set_black_list.go`, `operation_personal_store_set_visitor.go`, `operation_trade_add_meso.go`, `operation_trade_confirm.go`, `operation_trade_put_item.go`, `operation_transaction.go`, `operation_visit.go`

**inventory/clientbound (4):** `change.go`, `change_batch.go`, `compartment_merge.go`, `compartment_sort.go`

**inventory/serverbound (5):** `compartment_merge.go`, `compartment_sort.go`, `item_use.go`, `move.go`, `scroll_use.go`

**storage/clientbound (3):** `error.go`, `show.go`, `update_assets.go`

**storage/serverbound (4):** `operation.go`, `operation_meso.go`, `operation_retrieve_asset.go`, `operation_store_asset.go`

**Multi-struct files (per §3.1):** `cash/clientbound/shop_inventory.go` (4), `cash/clientbound/shop_operation_result.go` (4), `cash/clientbound/shop_item_moved.go` (2), `storage/clientbound/error.go` (3), `inventory/clientbound/change.go` (4).

---

## 15. Reference points in the existing tree

- `tools/packet-audit/internal/atlaspacket/registry.go:79-114` — method-name switch (§8 Option A extension site).
- `tools/packet-audit/internal/atlaspacket/registry_test.go` — fixture format for §8 registry change.
- `tools/packet-audit/internal/atlaspacket/analyzer.go` — early-return walker shipped in task-028; no changes expected.
- `libs/atlas-packet/interaction/serverbound/operation.go:29-41` — minimal dispatcher (mode byte only); confirms §5 reading.
- `libs/atlas-packet/interaction/serverbound/operation_merchant_buy.go` — typical sub-op file (one byte + one short).
- `libs/atlas-packet/inventory/clientbound/change.go:18-210` — four-shape hot path (§6).
- `libs/atlas-packet/cash/clientbound/shop_inventory.go:14-170` — multi-shape file with `EncodeBytes` recursion (§3.1, §4.4, §8).
- `libs/atlas-packet/cash/clientbound/shop_operation_body.go:12-77` — 78-entry constants block (§4.3).
- `libs/atlas-packet/storage/clientbound/error.go` — three-shape error file (§3.1, §7).
- `libs/atlas-packet/storage/clientbound/update_assets.go` — durable storage panel writer (§7).
- `libs/atlas-packet/storage/serverbound/operation.go` — storage dispatcher (parallels §5).
- `libs/atlas-packet/model/asset.go:164` — `model.Asset.Encode` (auto-discovered sub-struct).
- `libs/atlas-packet/inventory/change_entry.go` — `ChangeEntry` sub-struct (verify auto-discovery in Phase 0b).
- `docs/packets/audits/gms_v95/SUMMARY.md` — top-level audit index; receives ~89 commerce wire-shape rows.
- `docs/packets/audits/gms_v95/_pending.md` — deferral ledger.
- `docs/packets/ida-exports/{gms_v83,gms_v87,gms_v95,gms_jms_185}.json` — IDA evidence appends.
- `services/atlas-configurations/seed-data/templates/template_gms_{12,28,83,87,92,95}_1.json` + `template_jms_185_1.json` — opcode/enum fixes.
- `docs/tasks/task-066-social-domain-packet-audit/design.md` — template this design follows.
- `docs/tasks/task-028-character-domain-audit/post-phase-b.md` — template for this task's closing memo.

---

## 16. What plan-task should do next

Split this design into explicit, sequenced, small tasks. Suggested structure:

- **3 tasks for Phase 0** (0a `EncodeBytes` survey, 0b `model.Asset`/`ChangeEntry` confirm, 0c registry change if Option A).
- **One task per sub-phase 1a–1d** (storage, inventory, interaction, cash). 4 sub-tasks. Each ends with verdict-triaged commit set; fix commits inside a sub-task are individual.
- **One task per version for Phase 2** (3 sub-tasks: v83, v87, JMS-185).
- **One task for Phase 3** (login + character + social regression confirm).
- **One task for Phase 4** (post-phase-b.md + verification + code review + PR).

Total target: **12 plan tasks**. More than that is the plan re-deriving the audit per-packet; fewer hides scope.

Plan-task should specifically resolve:

- Phase 0a's `EncodeBytes` enumeration result and the §8 A/B decision.
- The exact wire-shape count for cash (currently estimated at ~36; locked by Phase 1d's constructor-to-struct enumeration commit).
- Whether `interaction/clientbound/{interaction.go,interaction_body.go}` is one router shape or two independent shapes (§5.1).
- The IDA function-address mapping for `cash/clientbound/shop_operation_body.go`'s 78 constants (drives the `_pending.md` reference table per §4.3 — 78 entries is a lot; group by dispatcher function).
- Whether Phase 1 sub-phase ordering 1a → 1d stands or whether the user prefers cash-first (highest version risk) before storage. Default: keep small-to-big to build executor familiarity with the multi-shape file pattern before facing cash's 36 rows.
- The bundling convention for fix commits within a sub-phase (suggestion: one commit per wire-shape fix; one template-fix commit per dispatcher function audited).
- Phase 2 commit naming convention (`phase-2-v83`, `phase-2-v87`, `phase-2-jms-185`) carried from task-066.
- The post-phase-b ledger row format (carry task-028 / task-066 table headers verbatim).
- Whether `services/atlas-cashshop/` and `services/atlas-inventory/` Dockerfile rebuilds are pre-emptive (run once per sub-phase) or constructor-ripple-driven (run only when ripple lands). Default: ripple-driven, with a Phase 4 belt-and-suspenders rebuild if any encoder constructor changed shape.
