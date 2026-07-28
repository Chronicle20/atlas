# Plan Adherence Audit — task-130-vegas-spell

**Plan:** `docs/tasks/task-130-vegas-spell/plan.md` (13 tasks)
**Design (authoritative):** `docs/tasks/task-130-vegas-spell/design.md`
**Branch:** `task-130-vegas-spell` — merge-base `38d4d0ba2`, HEAD `ba7b2799a`
**Audit date:** 2026-07-03
**Verdict:** FULL adherence — every task implemented; no silent skips. One task (Task 4/12) landed the authorized BLOCKED outcome for gms_v84 and PARKED for gms_v92. Recommendation: READY_TO_MERGE.

## Executive Summary

All 13 plan tasks are implemented with file:line evidence. The three cross-task interface contracts hold: `applyScrollCore` signature is identical between definition (`processor.go:667`) and the vega caller (`vega.go:224`); the Kafka JSON mirror tags match field-for-field between atlas-consumables and atlas-channel; and the `VegaScroll` writer name / operation bytes are consistent across the writer const, the IDA registry evidence, and all four seed templates. The v83 START_FAILURE correction (0x45, not the plan's stale 0x40) is applied uniformly across code comment, fixtures, and the gms_83 template. Spot-checked test suites (`libs/atlas-packet/cash/...`, consumables vega/buildScrollChanges) pass; controller confirmed the full per-module + matrix + bake gate is green.

## Task Completion

| # | Task | Status | Evidence |
|---|------|--------|----------|
| 1 | Vega item constants | DONE | `libs/atlas-constants/item/vegas_spell.go:10-22` (VegasSpell10/60, ClassificationVegasSpell=561, IsVegasSpell); test `vegas_spell_test.go` |
| 2 | `ItemUseVegaScroll` serverbound codec | DONE | `libs/atlas-packet/cash/serverbound/item_use_vega_scroll.go:22-78` (6 int32s, unconditional updateTime); byte fixture `item_use_vega_scroll_test.go:40-53` |
| 3 | `VegaScroll` clientbound writer | DONE | `libs/atlas-packet/cash/clientbound/vega_scroll.go:36-196` (outcome-keyed Start/Result/Invalid + WithResolvedCode) |
| 4 | IDA verification campaign | DONE | run.go splice `tools/packet-audit/cmd/run.go:1844,1929`; registry promoted to `ida-discovered` for v83/v87/v95/jms (`docs/packets/registry/*.yaml`); audit files `docs/packets/audits/*/CashVegaScroll.{md,json}`; v84 BLOCKED, v92 parked (authorized) |
| 5 | Kafka contract + producer + rate policy | DONE | `kafka/message/consumable/kafka.go:51-59,102`; `consumable/producer.go` (VegaScrollEventProvider); `consumable/vega.go` (vegaRates); `vega_test.go` |
| 6 | Extract `applyScrollCore`/`buildScrollChanges` | DONE | `consumable/processor.go:614,667`; TODO deleted (grep clean); 7 `TestBuildScrollChanges_*` at `processor_test.go:362+`; `ConsumeScroll` re-wired at `processor.go:733` |
| 7 | Vega request, chained reservations, consume | DONE | `consumable/vega.go` (RequestVegaScroll, ReserveVegaScrollStage, ConsumeVegaScroll, resolveVegaEquip, VegaScrollError); `applyScrollCore(...,boostedProb,false)` at `vega.go:224` |
| 8 | Consumer arm for REQUEST_VEGA_SCROLL | DONE | `kafka/consumer/consumable/consumer.go:37,71-75` |
| 9 | Channel command mirror, producer, processor | DONE | `kafka/message/consumable/kafka.go:44-48,72-80`; `consumable/producer.go`; `consumable/processor.go` (RequestVegaScrollUse) — JSON tags match Task 5 |
| 10 | Vega dispatch arm in cash-item-use handler | DONE | `socket/handler/character_cash_item_use.go:108-126,141-142,499-503`; test `character_cash_item_use_test.go` |
| 11 | VEGA_SCROLL event consumer + writer reg | DONE | `kafka/consumer/consumable/consumer.go:52,82-90,126-150`; writer registered `main.go:619` |
| 12 | Seed templates + rollout doc | DONE | writer entries in gms_83/87/95/jms templates (bytes match IDA table); `deployment.md`; v84/v92 correctly omitted |
| 13 | Final verification gates | DONE | controller-confirmed green (per-module test/vet/build, matrix --check exit 0, dispatcher-lint, redis-key-guard, bake all-go-services); cash_shop_entry.go:29 TODO untouched |

**Completion:** 13/13 (100%). Skipped without approval: 0. Partial: 0.

## Interface Contract Verification

- **applyScrollCore (Task 6 ↔ 7):** single definition `processor.go:667` `(l, ctx, txn, characterId, ci, scrollItem, equip *asset.Model, successProb uint32, whiteScroll bool)`; both callers (`processor.go:733` normal path with `ci.SuccessRate()`; `vega.go:224` with `boostedProb, false`) match exactly. PASS.
- **JSON mirror (Task 5 ↔ 9):** `RequestVegaScrollBody{vegaSlot, vegaItemId, scrollSlot, equipSlot}` and `VegaScrollBody{success, cursed}` identical tags in both `services/atlas-consumables/.../kafka/message/consumable/kafka.go` and `services/atlas-channel/.../kafka/message/consumable/kafka.go`. PASS.
- **Writer name / op bytes (Task 4 ↔ 12):** `VegaScrollWriter = "VegaScroll"` (`vega_scroll.go:36`) equals the template `"writer": "VegaScroll"`. Operation bytes in every template match the IDA-verified table (v83 40/45/41/43/42; v87 42/47/43/45/44; v95 44/49/45/47/42; jms 3B/40/3C/3E/3D) — including the v83 START_FAILURE=0x45(69) correction and the pinned v95 44/49 pairing. PASS.

## NewItemUpgradeEnchant verdict (carried-forward Minor)

**The plain `NewItemUpgrade(charId, success, cursed, false, false)` broadcast used by Task 11 (`consumer.go:147`) is the correct choice — not a gap.** Design §4.7 (authored with live IDA loaded) explicitly directs the vega map broadcast to use the plain writer with enchant fields zero ("its enchant fields stay zero — the enchant variant is for a different feature"), and PRD FR-5.2 restates `legendarySpirit=false, whiteScroll=false`. The observer-facing SHOW_SCROLL_EFFECT for Vega renders the ordinary scroll sparkle; the enchant variant (`NewItemUpgradeEnchant`, `enchantCategory`/`enchantResultFlag`) belongs to the separate CUIEnchantDlg/potential feature. The doc comment on `NewItemUpgradeEnchant` (`item_upgrade.go:55`) that references "Vega scroll result" is imprecise library-level naming — that constructor is not called by any task-130 code and pre-dates/sits outside this task. No functional change needed for task-130; at most a one-line doc-comment tidy (cosmetic, non-blocking).

## Notable deviations (all justified)

1. **Audit files named `CashVegaScroll`** (not `VegaScroll`): this is the packet-audit tool's `<pkg-capitalized><Type>` file-naming convention (pkg `cash`), not a renamed writer. The wire `VegaScrollWriter` const is still `"VegaScroll"`. Cosmetic.
2. **v83 START_FAILURE corrected 0x40 → 0x45** across code/fixtures/template: sanctioned IDA re-decompilation (START byte carries outcome on every version). Not a fault.
3. **Serverbound `packet-audit:verify` marker deliberately omitted** (`item_use_vega_scroll_test.go:35-39`): the serverbound cell is the shared USE_CASH_ITEM opcode whose audit report is owned by the not-yet-landed task-126; an orphan marker would fail `matrix --check`. Bytes are still fixtured and IDA-verified on 4 clients. Correct coordination, keeps matrix green.
4. **gms_v84 BLOCKED, gms_v92 PARKED:** no writer/handler template entries, registry note documents BLOCKED, `deployment.md` documents both. Authorized per the plan's BLOCKED discipline (an unverified opcode can crash the client; absence is the safe failure). PRD §10's "all six versions" criterion is met to the extent verifiable; the two gaps are documented, not silent.

## Build & Test (spot-checked; full gate controller-confirmed)

| Module | Result |
|---|---|
| libs/atlas-packet (cash/...) | PASS |
| atlas-consumables (Vega\|BuildScrollChanges) | PASS |
| Full per-module + matrix --check + dispatcher-lint + redis-key-guard + bake all-go-services | PASS (controller) |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None blocking. Optional cosmetic follow-up: tidy the `NewItemUpgradeEnchant` doc comment (`libs/atlas-packet/character/clientbound/item_upgrade.go:55`) so it doesn't imply Vega's Spell uses the enchant broadcast (design §4.7 says it does not). Non-blocking.
