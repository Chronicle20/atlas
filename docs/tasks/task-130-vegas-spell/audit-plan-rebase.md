# Plan Audit (Post-Rebase) — task-130-vegas-spell

**Plan Path:** docs/tasks/task-130-vegas-spell/plan.md
**Audit Date:** 2026-07-13
**Branch:** task-130-vegas-spell
**Base Branch:** main
**Scope:** Confirm the VegaScroll feature survived the rebase onto current `main` intact — no silently dropped or broken behavior from conflict resolution. This is a post-rebase integrity check, not a full plan re-derivation.

## Executive Summary

The VegaScroll feature is fully intact after the rebase. All five requested verification points pass with file:line evidence: the end-to-end vega path is unbroken, all five wired seed templates carry the correct IDA-verified writer opcodes and operations tables (byte-for-byte matching `deployment.md`), the coverage matrix shows CashVegaScroll ✅ for the five wired versions and ⬜ (n-a) for the legacy versions, the vega and task-126 point-reset dispatch arms coexist and are both reachable, and no conflict markers or task-130 stubs remain. Builds, tests (`go build`/`go test`/`go vet`), and `packet-audit matrix --check` are all clean.

## Verification Results

### 1. End-to-end vega path — INTACT

| Hop | Symbol | Evidence |
|-----|--------|----------|
| Channel handler dispatch arm | `CashSlotItemTypeVegasSpellPre95 \|\| ...Spell95` arm | `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:114-133` |
| → channel processor | `RequestVegaScrollUse` | `services/atlas-channel/.../consumable/processor.go:46-49` (iface `:18`) |
| → REQUEST_VEGA_SCROLL command | `RequestVegaScrollCommandProvider` / `CommandRequestVegaScroll` | `services/atlas-channel/.../consumable/producer.go:52-61`; const `kafka/message/consumable/kafka.go:18` |
| → consumables consumer | `handleRequestVegaScroll` (registered + typed guard) | `services/atlas-consumables/.../kafka/consumer/consumable/consumer.go:37, 71-75` |
| → RequestVegaScroll | validates up front, reserves CASH first | `services/atlas-consumables/.../consumable/vega.go:95-161` |
| → chained reservations | `ReserveVegaScrollStage` (USE after CASH confirms) | `vega.go:168-185` (single-item `RequestReserve`, never batched) |
| → ConsumeVegaScroll | applies via shared `applyScrollCore`, consumes both, emits event | `vega.go:191-245` |
| → VEGA_SCROLL event | `VegaScrollEventProvider` on `EnvEventTopic` | `vega.go:242`; provider `consumable/producer.go:50` (`EventTypeVegaScroll`) |
| Channel event consumer (result path) | `handleVegaScrollConsumableEvent` → Start + Result bodies | `services/atlas-channel/.../kafka/consumer/consumable/consumer.go:52, 126-153` |
| Channel invalid arm | `ErrorTypeVegaInvalid` → `VegaScrollInvalidBody` | `consumer.go:82-88` |
| Writer registration | `cashcb.VegaScrollWriter` registered | `services/atlas-channel/.../main.go:642` |

The chained-reservation discipline (CASH → USE → consume, never batched) is preserved (`vega.go:152-156`, `:174-178`), matching design §2.8/§3.2.

### 2. VegaScroll writer + operations tables in the 5 wired templates — CORRECT

Extracted from each seed template; every value matches `deployment.md` exactly.

| version | template | opCode | START_SUCCESS | START_FAILURE | RESULT_SUCCESS | RESULT_FAILURE | INVALID |
|---|---|---|---|---|---|---|---|
| gms_83 | `template_gms_83_1.json` | 0x166 | 64 | 69 | 65 | 67 | 66 |
| gms_84 | `template_gms_84_1.json` | 0x170 | 64 | 69 | 65 | 67 | 66 |
| gms_87 | `template_gms_87_1.json` | 0x17B | 66 | 71 | 67 | 69 | 68 |
| gms_95 | `template_gms_95_1.json` | 0x1AD | 68 | 73 | 69 | 71 | 66 |
| jms_185 | `template_jms_185_1.json` | 0x183 | 59 | 64 | 60 | 62 | 61 |

All five rows are byte-identical to `deployment.md` (opcode table and operations table). No drift introduced by the rebase.

### 3. Packet coverage matrix (STATUS.md) — ✅ for all 5 wired versions

`docs/packets/audits/STATUS.md:492` (VEGA_SCROLL / `cash/clientbound/CashVegaScroll` row), mapped against the header (`STATUS.md:20`):

- v48 ⬜, v61 ⬜, v72 ⬜, v79 ⬜ (n-a; no opcode)
- v83 `0x166` ✅, v84 `0x170` ✅, v87 `0x17B` ✅, v95 `0x1AD` ✅, JMS185 `0x183` ✅

`go run ./tools/packet-audit matrix --check` exits 0.

### 4. Vega arm and task-126 point-reset arm coexist — CONFIRMED

Both dispatch arms are present and reachable in the same handler:
- Vega arm: `character_cash_item_use.go:114-133` (`CashSlotItemTypeVegasSpellPre95 == 68` / `CashSlotItemTypeVegasSpell95 == 71`, consts `:160-161`).
- Point-reset arm (task-126): `character_cash_item_use.go:135-140` (`CashSlotItemTypePointResetTier1 == 24` / `CashSlotItemTypePointResetShared == 23`).
- Both classification branches feed distinct type bytes in `GetCashSlotItemType` (point-reset `:182-190`, vega `:518-523`), so neither shadows the other. Handler test `TestGetCashSlotItemTypeVegasSpell` passes.

### 5. No conflict markers, no TODOs/stubs in landed vega files — CONFIRMED

- `git grep` for conflict markers (`<<<<<<<`, `=======`, `>>>>>>>`) over tracked files: none.
- No `TODO`/`FIXME`/`panic("not…")`/`501`/`not implemented` in any landed vega source file (`vega.go`, `vega_scroll.go`, `item_use_vega_scroll.go`, channel producer/processor/consumer, templates).
- Two pre-existing TODO comments matched an `-i vega` grep but are NOT task-130 stubs: `cash_shop_entry.go:29` ("TODO block when performing vega scrolling") and `main.go:279` ("wire session.Processor lookup-by-key") both exist verbatim on `main` (verified via `git show main:…`). They are out of scope for this feature.

### Legacy versions (gms_v48/61/72/79) + gms_v92 — correctly version-absent

Grep for `vega` (case-insensitive) in `template_gms_{48,61,72,79,92}_1.json`: **0 matches each**. No VegaScroll writer/registry/operations entry was added for any legacy version, matching `deployment.md`'s n-a disposition (the `CUIVega` dialog does not exist in those clients). The matrix grades these ⬜ automatically via registry-absence.

## Build & Test Results

| Module / Package | Build | Tests | Vet | Notes |
|---|---|---|---|---|
| libs/atlas-constants (item) | — | PASS | — | `go test ./item/ -run Vegas` ok |
| libs/atlas-packet (cash/…) | — | PASS | — | clientbound + serverbound ok |
| services/atlas-consumables (consumable, kafka) | PASS | PASS | PASS | `go build ./...` clean; consumable tests ok |
| services/atlas-channel (handler, consumable, kafka) | PASS | PASS | PASS | `go build ./...` clean; `socket/handler` tests ok |
| tools/packet-audit `matrix --check` | — | PASS (exit 0) | — | all cells consistent |

## Overall Assessment

- **Plan Adherence (post-rebase):** FULL — every implemented behavior from the plan is present and correct after the rebase; nothing silently dropped or broken by conflict resolution.
- **Recommendation:** READY_TO_MERGE (from the rebase-integrity standpoint).

## Action Items

None. The rebase preserved the feature end-to-end. (The two pre-existing, unrelated TODO comments in `cash_shop_entry.go` and `main.go` are on `main` already and are out of task-130 scope.)
